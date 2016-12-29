package main

import (
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/labstack/echo"
)

func editfunc(c echo.Context) (err error) {
	s3 := c.Get("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	title := c.QueryParam("title")
	body := "# " + title + "\n"
	markdown, err := s3.loadMarkdownAsync(titleHash)
	if err == nil {
		body = markdown.body
	}
	c.Render(http.StatusOK, "edit.html", map[string]interface{}{
		"Title":     title,
		"TitleHash": titleHash,
		"Body":      body,
	})
	return nil
}

func uploadfunc(c echo.Context) (err error) {
	s3 := c.Get("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	file, header, err := c.Request().FormFile("file")
	if err != nil {
		log.Println(err)
	}
	filename := header.Filename
	contentType := header.Header["Content-Type"][0]

	key := fileDataKey{
		filename:  filename,
		titleHash: titleHash,
	}
	body, _ := ioutil.ReadAll(file)
	fileData := &fileData{
		fileDataKey: key,
		contentType: contentType,
		filebyte:    body,
	}
	s3.saveFileAsync(key, fileData)
	return nil
}

func postfunc(c echo.Context) (err error) {
	method := c.FormValue("_method")
	log.Println(method)
	switch method {
	case "put":
		putfunc(c)
	case "delete":
		deletefunc(c)
	}
	return nil
}

func deletefunc(c echo.Context) (err error) {
	s3 := c.Get("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	s3.deleteMarkdownAsync(titleHash)
	s3.deleteHTML(titleHash)
	c.Redirect(http.StatusFound, "/")
	return nil
}

func putfunc(c echo.Context) (err error) {
	s3 := c.Get("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	title := c.FormValue("title")
	if titleHash != s3.titleHash(title) {
		log.Println("title not match")
		log.Println("title:", title)
		log.Println("generated:", s3.titleHash(title))
		log.Println("titleHash:", titleHash)
		c.Redirect(http.StatusFound, "/500")
		return
	}

	cookie, err := c.Cookie("user")
	if err != nil {
		return err
	}
	user := cookie.Value

	var markdown pageData

	markdown.titleHash = titleHash
	markdown.title = title
	markdown.author = user
	markdown.body = c.FormValue("body")
	s3.saveMarkdownAsync(titleHash, &markdown)

	c.Redirect(http.StatusFound, "/page/"+titleHash)

	// Async upload compiled HTML
	if s3.checkPublic(markdown.titleHash) {
		go func(s3 *Wikidata, markdown pageData) {
			html := markdown
			html.body = string(renderHTML(s3, &markdown))

			s3.saveHTML(&html)
			log.Println("HTML uploaded")
		}(s3, markdown)
	}
	return nil
}
