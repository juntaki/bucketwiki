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
	w := Wikidata{
		bucket: os.Getenv("AWS_BUCKET_NAME"),
		region: os.Getenv("AWS_BUCKET_REGION"),
	}
	w.connect()

	router := gin.Default()
	store := sessions.NewCookieStore([]byte("secret"))
	router.Use(sessions.Sessions("mysession", store))
	router.LoadHTMLGlob("style/*.html")

	router.Use(s3Middleware(&w))
	router.GET("/login", getloginfunc)
	router.POST("/login", postloginfunc)
	router.GET("/signup", getsignupfunc)
	router.POST("/signup", postsignupfunc)
	router.GET("/logout", getlogoutfunc)

	router.GET("/auth/callback", authCallback)
	router.GET("/auth", authenticate)
	router.StaticFile("/layout.css", "style/layout.css")
	router.StaticFile("/favicon.ico", "style/favicon.ico")

	auth := router.Group("/")
	auth.Use(authMiddleware())
	{
		auth.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusFound, "/page/Home")
		})
		auth.GET("/page/:title/edit", editfunc)
		auth.GET("/page/:title", getfunc)
		auth.POST("/page/:title", postfunc)
		auth.POST("/page/:title/acl", aclfunc)
		auth.PUT("/page/:title", putfunc)
		auth.DELETE("/page/:title", deletefunc)
	}

	router.Run(":8080")
}

func s3Middleware(s3 *Wikidata) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("S3", s3)
		c.Next()
	}
}

func editfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	title := c.Param("title")
	body := []byte("")
	markdown, err := s3.loadMarkdown(title)
	if err == nil {
		body = markdown
	}
	c.HTML(http.StatusOK, "edit.html", gin.H{
		"Title": c.Param("title"),
		"Body":  body,
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

func aclfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	title := c.Param("title")
	acl, _ := c.GetPostForm("acl")
	switch acl {
	case "public":
		s3.aclPublic(title)
	case "private":
		s3.aclPrivate(title)
	}
	c.Redirect(http.StatusFound, "/page/"+title)
}

type breadcrumb struct {
	List []string `json:"list"`
}

func getfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	title := c.Param("title")
	html, err := s3.loadHTML(title)
	if err != nil {
		c.Redirect(http.StatusFound, "/page/"+title+"/edit")
		return
	}

	public := s3.checkPublic(title)
	publicURL := s3.publicURL(title)

	session := sessions.Default(c)
	jsonStr := session.Get("breadcrumb")

	var u breadcrumb
	u.List = []string{}
	if jsonStr != nil {
		json.Unmarshal(jsonStr.([]byte), &u)
	}

	var array []string

	for _, l := range u.List {
		if l != title {
			array = append(array, l)
		}
	}

	maxSize := 5

	u.List = append(array, title)
	if len(u.List) > maxSize {
		u.List = u.List[1 : maxSize+1]
	}
	jsonOut, _ := json.Marshal(&u)
	session.Set("breadcrumb", jsonOut)
	session.Save()

	c.HTML(http.StatusOK, "view.html", gin.H{
		"Title":      title,
		"Body":       template.HTML(string(html)),
		"Breadcrumb": array,
		"Public":     public,
		"PublicURL":  publicURL,
	})
}

func deletefunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	title := c.Param("title")
	s3.deleteMarkdown(title)
	s3.deleteHTML(title)
	c.Redirect(http.StatusFound, "/")
}

func putfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	title := c.Param("title")

	id, err := s3.loadDocumentID(title)
	if err != nil {
		id, err = randomString()
		if err != nil {
			return
		}
	}

	markdown, _ := c.GetPostForm("body")
	s3.saveMarkdown(title, id, markdown)

	html, _ := MarkdownToHTML([]byte(markdown))
	s3.saveHTML(title, id, html)
	c.Redirect(http.StatusFound, "/page/"+title)
}
