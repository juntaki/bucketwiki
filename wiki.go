package main

import (
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
	router.LoadHTMLGlob("*.html")

	router.Use(s3Middleware(&w))
	router.GET("/login", getloginfunc)
	router.POST("/login", postloginfunc)

	auth := router.Group("/")
	auth.Use(authMiddleware())
	{
		auth.GET("/", listfunc)
		auth.GET("/files/:title/edit", editfunc)
		auth.GET("/files/:title", getfunc)
		auth.POST("/files/:title", postfunc)
		auth.PUT("/files/:title", putfunc)
		auth.DELETE("/files/:title", deletefunc)
	}

	router.Run(":8080")
}

func s3Middleware(s3 *Wikidata) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("S3", s3)
		c.Next()
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user")
		if user == nil {
			fmt.Println("get failed")
			c.Redirect(http.StatusFound, "/login")
		}
		c.Next()
	}
}

func postloginfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	username, ok := c.GetPostForm("username")
	if !ok {
		fmt.Println("postform failed")
		return
	}
	_, err := s3.loadUser(username)
	if err != nil {
		fmt.Println("loadUser failed")
		return
	}
	session := sessions.Default(c)
	session.Set("user", username)
	session.Save()
	c.Redirect(http.StatusFound, "/")
}

func getloginfunc(c *gin.Context) {
	c.HTML(http.StatusOK, "auth.html", gin.H{})
}

func listfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	list, err := s3.listBasenameWithSuffix(".md")
	if err != nil {
		return
	}
	c.HTML(http.StatusOK, "list.html", gin.H{
		"List": list,
	})
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
		c.Redirect(http.StatusFound, "/files/"+title+"/edit")
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
	markdown, _ := c.GetPostForm("body")
	s3.saveMarkdown(title, markdown)

	html, _ := Markdown([]byte(markdown))
	s3.saveHTML(title, html)
	c.Redirect(http.StatusFound, "/files/"+title)
}
