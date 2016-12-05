package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"regexp"

	log "github.com/Sirupsen/logrus"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

func main() {

	if os.Getenv("AWS_BUCKET_NAME") == "" ||
		os.Getenv("AWS_BUCKET_REGION") == "" ||
		os.Getenv("AWS_ACCESS_KEY_ID") == "" ||
		os.Getenv("AWS_SECRET_ACCESS_KEY") == "" ||
		os.Getenv("WIKI_SECRET") == "" {
		log.Println("Error at environment variable")
		os.Exit(1)
	}

	s3 := Wikidata{
		bucket: os.Getenv("AWS_BUCKET_NAME"),
		region: os.Getenv("AWS_BUCKET_REGION"),
	}
	s3.connect()

	router := gin.Default()
	store := sessions.NewCookieStore([]byte(s3.wikiSecret))
	router.Use(sessions.Sessions("mysession", store))
	router.LoadHTMLGlob("style/*.html")

	router.Use(s3Middleware(&s3))
	router.GET("/login", getloginfunc)
	router.POST("/login", postloginfunc)
	router.GET("/signup", getsignupfunc)
	router.POST("/signup", postsignupfunc)
	router.GET("/logout", getlogoutfunc)

	router.GET("/auth/callback", authCallback)
	router.GET("/auth", authenticate)
	router.StaticFile("/500", "style/500.html")
	router.StaticFile("/404", "style/404.html")
	router.StaticFile("/layout.css", "style/layout.css")
	router.StaticFile("/favicon.ico", "style/favicon.ico")

	auth := router.Group("/")
	auth.Use(authMiddleware())
	{
		auth.GET("/", func(c *gin.Context) {
			// For first access, title query should be passed.
			c.Redirect(http.StatusFound, "/page/"+s3.titleHash("Home")+"?title=Home")
		})
		auth.POST("/page/:titleHash/upload", uploadfunc)
		auth.GET("/page/:titleHash/edit", editfunc)
		auth.GET("/page/:titleHash/history", gethistory)
		auth.GET("/page/:titleHash/file/:filename", getfilefunc)
		auth.GET("/page/:titleHash", getfunc)
		auth.POST("/page/:titleHash", postfunc)
		auth.POST("/page/:titleHash/acl", aclfunc)
		auth.PUT("/page/:titleHash", putfunc)
		auth.DELETE("/page/:titleHash", deletefunc)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	router.Run(":" + port)
}

func s3Middleware(s3 *Wikidata) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("S3", s3)
		c.Next()
	}
}

func aclfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	acl, _ := c.GetPostForm("acl")
	switch acl {
	case "public":
		s3.setACL(titleHash, true)
	case "private":
		s3.setACL(titleHash, false)
	}
	c.Redirect(http.StatusFound, "/page/"+titleHash)
}

type breadcrumb struct {
	List []([]string) `json:"list"`
}

func getfilefunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	filename := c.Param("filename")

	fileData, err := s3.loadFileAsync(fileDataKey{
		filename:  filename,
		titleHash: titleHash,
	})
	if err != nil {
		log.Println(err)
		return
	}
	c.Data(http.StatusOK, fileData.contentType, fileData.filebyte)
}

func gethistory(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	title := c.Query("title")

	history, _ := s3.listhistory(titleHash)
	c.HTML(http.StatusOK, "history.html", gin.H{
		"Title":     title,
		"TitleHash": titleHash,
		"List":      history,
	})
}

func getfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	version := c.Query("history")
	log.Println(version)
	var md *pageData
	var err error
	if version == "" {
		md, err = s3.loadMarkdownAsync(titleHash)
	} else {
		md, err = s3.loadMarkdown(titleHash, version)
	}
	if err != nil {
		// If no object found, title cannot get from metadata, so it must be passed via query.
		if version == "" {
			title := c.Query("title")
			c.Redirect(http.StatusFound, "/page/"+titleHash+"/edit?title="+title)
			return
		} else {
			c.Redirect(http.StatusFound, "/404")
			return
		}
	}

	html := renderHTML(s3, md)
	title := md.title

	public := s3.checkPublic(titleHash)
	publicURL := s3.publicURL(titleHash)

	session := sessions.Default(c)
	jsonStr := session.Get("breadcrumb")

	var u breadcrumb
	u.List = []([]string){}
	if jsonStr != nil {
		err = json.Unmarshal(jsonStr.([]byte), &u)
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
	session.Set("breadcrumb", jsonOut)
	session.Save()

	c.HTML(http.StatusOK, "view.html", gin.H{
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
