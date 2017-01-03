package main

import (
	"errors"
	"html/template"
	"io"
	"net/http"
	"os"
	"regexp"

	log "github.com/Sirupsen/logrus"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/twitter"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

type handler struct {
	db *Wikidata
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) (err error) {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	if os.Getenv("AWS_BUCKET_NAME") == "" ||
		os.Getenv("AWS_BUCKET_REGION") == "" ||
		os.Getenv("AWS_ACCESS_KEY_ID") == "" ||
		os.Getenv("AWS_SECRET_ACCESS_KEY") == "" ||
		os.Getenv("WIKI_SECRET") == "" {
		log.Println("Error at environment variable")
		os.Exit(1)
	}

	s3 := &Wikidata{
		bucket: os.Getenv("AWS_BUCKET_NAME"),
		region: os.Getenv("AWS_BUCKET_REGION"),
	}
	err := s3.connect()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	e := echo.New()
	e.Debug = true
	t := &Template{
		templates: template.Must(template.ParseGlob("style/*.html")),
	}
	e.Renderer = t

	h := handler{db: s3}

	goth.UseProviders(
		twitter.New(
			os.Getenv("TWITTER_KEY"),
			os.Getenv("TWITTER_SECRET"),
			os.Getenv("URL")+"/auth/callback?provider=twitter",
		),
	)

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.GET("/err", func(c echo.Context) (err error) {
		return errors.New("some error")
	})
	e.GET("/login", h.loginPageHandler)
	e.POST("/login", h.loginHandler)
	e.GET("/signup", h.signupPageHandler)
	e.POST("/signup", h.signupHandler)

	e.GET("/auth/callback", h.authCallbackHandler)
	e.GET("/auth", h.authHandler)
	e.File("/500", "style/500.html")
	e.File("/404", "style/404.html")
	e.File("/layout.css", "style/layout.css")
	e.File("/favicon.ico", "style/favicon.ico")

	auth := e.Group("")
	auth.Use(h.authMiddleware())
	auth.GET("/", func(c echo.Context) (err error) {
		// For first access, title query should be passed.
		return c.Redirect(http.StatusFound, "/page/"+s3.titleHash("Home")+"?title=Home")
	})
	auth.GET("/logout", h.logoutHandler)
	auth.POST("/page/:titleHash/upload", h.uploadHandler)
	auth.GET("/page/:titleHash/edit", h.editorHandler)
	auth.GET("/page/:titleHash/history", h.historyPageHandler)
	auth.GET("/page/:titleHash/file/:filename", h.fileHandler)
	auth.GET("/page/:titleHash", h.pageHandler)
	auth.POST("/page/:titleHash", h.postPageHandler)
	auth.POST("/page/:titleHash/acl", h.aclHandler)
	auth.PUT("/page/:titleHash", h.putPageHandler)
	auth.DELETE("/page/:titleHash", h.deletePageHandler)

	port := ":" + os.Getenv("PORT")
	if port == ":" {
		port = ":8080"
	}
	e.Logger.Fatal(e.Start(port))
}

func (h *handler) aclHandler(c echo.Context) (err error) {
	titleHash := c.Param("titleHash")
	acl := c.FormValue("acl")
	switch acl {
	case "public":
		err = h.db.setACL(titleHash, true)
	case "private":
		err = h.db.setACL(titleHash, false)
	default:
		err = echo.NewHTTPError(http.StatusBadRequest, "unknown ACL")
	}
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, "/page/"+titleHash)
}

func (h *handler) fileHandler(c echo.Context) (err error) {
	titleHash := c.Param("titleHash")
	filename := c.Param("filename")

	fileData := &fileData{
		filename:  filename,
		titleHash: titleHash,
	}

	err = h.db.loadBare(fileData)
	if err != nil {
		log.Println(err)
		return nil
	}
	return c.Blob(http.StatusOK, fileData.contentType, fileData.filebyte)
}

func (h *handler) historyPageHandler(c echo.Context) (err error) {
	titleHash := c.Param("titleHash")
	title := c.QueryParam("title")

	history, _ := h.db.listhistory(titleHash)
	return c.Render(http.StatusOK, "history.html", map[string]interface{}{
		"Title":     title,
		"TitleHash": titleHash,
		"List":      history,
	})
}

func (h *handler) pageHandler(c echo.Context) (err error) {
	titleHash := c.Param("titleHash")
	versionId := c.QueryParam("history")

	md := &pageData{
		titleHash: titleHash,
		versionId: versionId,
	}

	err = h.db.loadBare(md)
	if err != nil {
		// If no object found, title cannot get from metadata, so it must be passed via query.
		if versionId == "" {
			title := c.QueryParam("title")
			return c.Redirect(http.StatusFound, "/page/"+titleHash+"/edit?title="+title)
		}
		return c.Redirect(http.StatusFound, "/404")
	}

	html := renderHTML(h.db, md)
	title := md.title

	public := h.db.checkPublic(titleHash)
	publicURL := h.db.publicURL(titleHash)

	sess := c.Get("session").(*sessionData)

	var tmpList []([]string)

	for _, l := range sess.BreadCrumb {
		if l[0] != title {
			tmpList = append(tmpList, l)
		}
	}
	tmpList = append(tmpList, []string{title, titleHash})

	maxSize := 5
	// Cut down the size
	if len(tmpList) > maxSize {
		tmpList = tmpList[1 : maxSize+1]
	}

	sess.BreadCrumb = tmpList

	err = h.setSession(c, sess)
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, "view.html", map[string]interface{}{
		"Title":        title,
		"TitleHash":    titleHash,
		"Body":         template.HTML(html),
		"Breadcrumb":   sess.BreadCrumb,
		"Public":       public,
		"PublicURL":    publicURL,
		"LastModified": md.lastUpdate,
		"Author":       md.author,
	})
}

func renderHTML(s3 *Wikidata, md *pageData) []byte {
	rep := regexp.MustCompile(`\[\[.*?\]\]`)

	str := md.body
	str = rep.ReplaceAllStringFunc(str, func(a string) string {
		title := a[2 : len(a)-2]
		return "[" + title + "](/page/" + s3.titleHash(title) + "?title=" + title + ")"
	})

	unsafe := blackfriday.MarkdownCommon([]byte(str))
	html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	return html
}
