package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
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
	router.LoadHTMLGlob("*.html")

	router.Use(s3Middleware(&w))
	router.GET("/login", getloginfunc)
	router.POST("/login", postloginfunc)

	router.GET("/auth/callback", authCallback)
	router.GET("/auth", authenticate)

	auth := router.Group("/")
	auth.Use(authMiddleware())
	{
		auth.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusFound, "/page/top")
		})
		auth.GET("/page/:title/edit", editfunc)
		auth.GET("/page/:title", getfunc)
		auth.POST("/page/:title", postfunc)
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

func getfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	title := c.Param("title")
	html, err := s3.loadHTML(title)
	if err != nil {
		c.Redirect(http.StatusFound, "/page/"+title+"/edit")
		return
	}

	c.HTML(http.StatusOK, "view.html", gin.H{
		"Title": title,
		"Body":  template.HTML(string(html)),
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

	id, err := s3.loadUUID(title)
	if err != nil {
		id = uuid.NewV4().String()
	}

	markdown, _ := c.GetPostForm("body")
	s3.saveMarkdown(title, id, markdown)

	html, _ := MarkdownToHTML([]byte(markdown))
	s3.saveHTML(title, id, html)
	c.Redirect(http.StatusFound, "/page/"+title)
}
