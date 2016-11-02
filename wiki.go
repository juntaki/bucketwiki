package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
)

func main() {
	s3 := Wikidata{
		bucket: os.Getenv("AWS_BUCKET_NAME"),
		region: os.Getenv("AWS_BUCKET_REGION"),
	}
	s3.connect()

	router := gin.Default()
	store := sessions.NewCookieStore([]byte("secret"))
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
	router.StaticFile("/layout.css", "style/layout.css")
	router.StaticFile("/favicon.ico", "style/favicon.ico")

	auth := router.Group("/")
	auth.Use(authMiddleware())
	{
		auth.GET("/", func(c *gin.Context) {
			// For first access, title query should be passed.
			c.Redirect(http.StatusFound, "/page/"+s3.titleHash("Home")+"?title=Home")
		})
		auth.GET("/page/:titleHash/edit", editfunc)
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

func editfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	title := c.Query("title")
	body := "# " + title + "\n"
	markdown, err := s3.loadMarkdown(titleHash)
	if err == nil {
		body = markdown.body
	}
	c.HTML(http.StatusOK, "edit.html", gin.H{
		"Title":     title,
		"TitleHash": titleHash,
		"Body":      body,
	})
}

func aclfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	acl, _ := c.GetPostForm("acl")
	switch acl {
	case "public":
		s3.aclPublic(titleHash)
	case "private":
		s3.aclPrivate(titleHash)
	}
	c.Redirect(http.StatusFound, "/page/"+titleHash)
}

type breadcrumb struct {
	List []([]string) `json:"list"`
}

func getfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	html, err := s3.loadHTML(titleHash)
	if err != nil {
		// If no object found, title cannot get from metadata, so it must be passed via query.
		title := c.Query("title")
		c.Redirect(http.StatusFound, "/page/"+titleHash+"/edit?title="+title)
		return
	}
	title := html.title

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
		"Body":         template.HTML(html.body),
		"Breadcrumb":   array,
		"Public":       public,
		"PublicURL":    publicURL,
		"LastModified": html.lastUpdate,
		"Author":       html.author,
	})
}

func postfunc(c *gin.Context) {
	method, _ := c.GetPostForm("_method")
	fmt.Println(method)
	switch method {
	case "put":
		putfunc(c)
	case "delete":
		deletefunc(c)
	}
}

func deletefunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	s3.deleteMarkdown(titleHash)
	s3.deleteHTML(titleHash)
	c.Redirect(http.StatusFound, "/")
}

func putfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	title, _ := c.GetPostForm("title")
	if titleHash != s3.titleHash(title) {
		fmt.Println("title not match")
		fmt.Println("title:", title)
		fmt.Println("generated:", s3.titleHash(title))
		fmt.Println("titleHash:", titleHash)
		c.Redirect(http.StatusFound, "/500")
		return
	}

	session := sessions.Default(c)
	user := session.Get("user")

	id, err := s3.loadDocumentID(titleHash)
	if err != nil {
		id, err = randomString()
		if err != nil {
			fmt.Println("random string cannot be generated", err)
			c.Redirect(http.StatusFound, "/500")
			return
		}
	}

	var markdown, html pageData

	markdown.titleHash = titleHash
	markdown.title = title
	markdown.author = user.(string)
	markdown.body, _ = c.GetPostForm("body")
	markdown.id = id
	err = s3.saveMarkdown(markdown)
	if err != nil {
		fmt.Println("save Markdown", err)
		c.Redirect(http.StatusFound, "/500")
		return
	}

	html.titleHash = titleHash
	html.title = title
	html.author = user.(string)
	html.body, _ = MarkdownToHTML(s3, []byte(markdown.body))
	html.id = id
	err = s3.saveHTML(html)
	if err != nil {
		fmt.Println("save HTML", err)
		c.Redirect(http.StatusFound, "/500")
		return
	}

	c.Redirect(http.StatusFound, "/page/"+titleHash)
}
