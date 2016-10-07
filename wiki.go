package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func loadObject(title string, w *Wikidata) ([]byte, error) {
	r, err := w.get(title)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}

func loadHTML(title string, w *Wikidata) ([]byte, error) {
	return loadObject(title+".html", w)
}

func loadMarkdown(title string, w *Wikidata) ([]byte, error) {
	return loadObject(title+".md", w)
}

func main() {
	w := Wikidata{
		bucket: os.Getenv("AWS_BUCKET_NAME"),
		region: os.Getenv("AWS_BUCKET_REGION"),
	}
	w.connect()

	router := gin.Default()
	router.LoadHTMLGlob("*.html")

	router.Use(s3Middleware(&w))
	router.GET("/list", listfunc)
	router.GET("/files/:title/edit", editfunc)
	router.GET("/files/:title", getfunc)
	router.POST("/files/:title", postfunc)
	router.PUT("/files/:title", putfunc)
	router.DELETE("/files/:title", deletefunc)

	router.Run(":8080")
}

func s3Middleware(s3 *Wikidata) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("S3", s3)
		c.Next()
	}
}

func listfunc(c *gin.Context) {
	s3, ok := c.MustGet("S3").(*Wikidata)
	if !ok {
		return
	}
	list, err := s3.listBasenameWithSuffix(".md")
	if err != nil {
		return
	}
	c.HTML(http.StatusOK, "list.html", gin.H{
		"List": list,
	})
}

func editfunc(c *gin.Context) {
	s3, ok := c.MustGet("S3").(*Wikidata)
	if !ok {
		return
	}
	title := c.Param("title")
	body := []byte("")
	markdown, err := loadMarkdown(title, s3)
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
	s3, ok := c.MustGet("S3").(*Wikidata)
	if !ok {
		return
	}
	title := c.Param("title")
	html, err := loadHTML(title, s3)
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
	s3, ok := c.MustGet("S3").(*Wikidata)
	if !ok {
		return
	}
	title := c.Param("title")
	s3.delete(title, ".md")
	s3.delete(title, ".html")
	c.Redirect(http.StatusFound, "/list")
}

func putfunc(c *gin.Context) {
	s3, ok := c.MustGet("S3").(*Wikidata)
	if !ok {
		return
	}
	title := c.Param("title")
	markdown, _ := c.GetPostForm("body")
	s3.put(title, ".md", []byte(markdown))

	html, _ := Markdown([]byte(markdown))
	s3.put(title, ".html", []byte(html))
	c.Redirect(http.StatusFound, "/files/"+title)
}
