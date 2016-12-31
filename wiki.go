package main

import (
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"
	"os"
	"regexp"

	log "github.com/Sirupsen/logrus"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

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

	e.Use(middleware.Logger())
	e.GET("/err", func(c echo.Context) (err error) {
		return errors.New("some error")
	})
	e.GET("/login", h.loginPageHandler)
	e.POST("/login", h.loginHandler)
	e.GET("/signup", h.signupPageHandler)
	e.POST("/signup", h.signupHandler)
	e.GET("/logout", h.logoutHandler)

	e.GET("/auth/callback", h.authCallbackHandler)
	e.GET("/auth", h.authHandler)
	e.File("/500", "style/500.html")
	e.File("/404", "style/404.html")
	e.File("/layout.css", "style/layout.css")
	e.File("/favicon.ico", "style/favicon.ico")

	auth := e.Group("")
	auth.Use(authMiddleware())
	auth.GET("/", func(c echo.Context) (err error) {
		// For first access, title query should be passed.
		return c.Redirect(http.StatusFound, "/page/"+s3.titleHash("Home")+"?title=Home")
	})
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

type breadcrumb struct {
	List []([]string) `json:"list"`
}

func (h *handler) fileHandler(c echo.Context) (err error) {
	titleHash := c.Param("titleHash")
	filename := c.Param("filename")

	fileData, err := h.db.loadFileAsync(fileDataKey{
		filename:  filename,
		titleHash: titleHash,
	})
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
	version := c.QueryParam("history")
	log.Println(version)
	var md *pageData
	if version == "" {
		md, err = h.db.loadMarkdownAsync(titleHash)
	} else {
		md, err = h.db.loadMarkdown(titleHash, version)
	}
	if err != nil {
		// If no object found, title cannot get from metadata, so it must be passed via query.
		if version == "" {
			title := c.QueryParam("title")
			return c.Redirect(http.StatusFound, "/page/"+titleHash+"/edit?title="+title)
		}
		return c.Redirect(http.StatusFound, "/404")
	}

	html := renderHTML(h.db, md)
	title := md.title

	public := h.db.checkPublic(titleHash)
	publicURL := h.db.publicURL(titleHash)

	cookie, err := c.Cookie("breadcrumb")
	var jsonStr string
	if err != nil {
		jsonStr = ""
	} else {
		jsonStr = cookie.Value
	}
	var u breadcrumb
	u.List = []([]string){}
	if jsonStr != "" {
		err = json.Unmarshal([]byte(jsonStr), &u)
		if err != nil {
			u.List = []([]string){}
		}
	}

	var array []([]string)

	for _, l := range u.List {
		if len(l) != 2 {
			// cookie is malformed, may be old version.
			break
		}
		if l[0] != title {
			array = append(array, l)
		}
	}

	maxSize := 5

	u.List = append(array, []string{title, titleHash})

	// Cut down the size
	if len(u.List) > maxSize {
		u.List = u.List[1 : maxSize+1]
	}
	jsonOut, _ := json.Marshal(&u)
	c.SetCookie(&http.Cookie{Name: "breadcrumb", Value: string(jsonOut)})

	return c.Render(http.StatusOK, "view.html", map[string]interface{}{
		"Title":        title,
		"TitleHash":    titleHash,
		"Body":         template.HTML(html),
		"Breadcrumb":   array,
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
