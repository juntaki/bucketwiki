package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
)

func editfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	title := c.Query("title")
	body := "# " + title + "\n"
	markdown, err := s3.loadMarkdown(titleHash, "")
	if err == nil {
		body = markdown.body
	}
	c.HTML(http.StatusOK, "edit.html", gin.H{
		"Title":     title,
		"TitleHash": titleHash,
		"Body":      body,
	})
}

func uploadfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	page, err := s3.loadMarkdownMetadata(titleHash)
	if err != nil {
		fmt.Println(err)
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		fmt.Println(err)
	}
	filename := header.Filename
	contentType := header.Header["Content-Type"][0]

	s3.saveFile(page, file, filename, contentType)
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

	var markdown pageData

	markdown.titleHash = titleHash
	markdown.title = title
	markdown.author = user.(string)
	markdown.body, _ = c.GetPostForm("body")
	err := s3.saveMarkdown(markdown)
	if err != nil {
		fmt.Println("save Markdown", err)
		c.Redirect(http.StatusFound, "/500")
		return
	}

	c.Redirect(http.StatusFound, "/page/"+titleHash)

	// Async upload compiled HTML
	if s3.checkPublic(markdown.titleHash) {
		go func(s3 *Wikidata, markdown pageData) {
			html := markdown
			html.body = string(renderHTML(s3, &markdown))

			s3.saveHTML(html)
			fmt.Println("HTML uploaded")
		}(s3, markdown)
	}
}
