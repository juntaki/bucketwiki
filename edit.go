package main

import (
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/labstack/echo"
)

func editorHandler(c echo.Context) (err error) {
	s3 := c.Get("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	title := c.QueryParam("title")
	body := "# " + title + "\n"
	markdown, err := s3.loadMarkdownAsync(titleHash)
	if err == nil {
		body = markdown.body
	}
	return c.Render(http.StatusOK, "edit.html", map[string]interface{}{
		"Title":     title,
		"TitleHash": titleHash,
		"Body":      body,
	})
}

func uploadHandler(c echo.Context) (err error) {
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
	return s3.saveFileAsync(key, fileData)
}

func postPageHandler(c echo.Context) (err error) {
	method := c.FormValue("_method")
	log.Println(method)
	switch method {
	case "put":
		return putPageHandler(c)
	case "delete":
		return deletePageHandler(c)
	}
	return nil
}

func deletePageHandler(c echo.Context) (err error) {
	s3 := c.Get("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	err = s3.deleteMarkdownAsync(titleHash)
	if err != nil {
		return err
	}
	err = s3.deleteHTML(titleHash)
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, "/")
}

func putPageHandler(c echo.Context) (err error) {
	s3 := c.Get("S3").(*Wikidata)
	titleHash := c.Param("titleHash")
	title := c.FormValue("title")
	if titleHash != s3.titleHash(title) {
		log.Println("title not match")
		log.Println("title:", title)
		log.Println("generated:", s3.titleHash(title))
		log.Println("titleHash:", titleHash)
		return c.Redirect(http.StatusFound, "/500")
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
	err = s3.saveMarkdownAsync(titleHash, &markdown)
	if err != nil {
		return err
	}

	// Async upload compiled HTML
	if s3.checkPublic(markdown.titleHash) {
		go func(s3 *Wikidata, markdown pageData) {
			html := markdown
			html.body = string(renderHTML(s3, &markdown))

			err = s3.saveHTML(&html)
			log.Println("HTML uploaded")
		}(s3, markdown)
	}
	return c.Redirect(http.StatusFound, "/page/"+titleHash)
}
