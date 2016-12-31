package main

import (
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/labstack/echo"
)

func (h *handler) editorHandler(c echo.Context) (err error) {
	titleHash := c.Param("titleHash")
	title := c.QueryParam("title")
	body := "# " + title + "\n"
	markdown, err := h.db.loadMarkdownAsync(titleHash)
	if err == nil {
		body = markdown.body
	}
	return c.Render(http.StatusOK, "edit.html", map[string]interface{}{
		"Title":     title,
		"TitleHash": titleHash,
		"Body":      body,
	})
}

func (h *handler) uploadHandler(c echo.Context) (err error) {
	titleHash := c.Param("titleHash")
	header, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	filename := header.Filename
	contentType := header.Header["Content-Type"][0]

	key := fileDataKey{
		filename:  filename,
		titleHash: titleHash,
	}
	file, err := header.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	body, _ := ioutil.ReadAll(file)
	fileData := &fileData{
		fileDataKey: key,
		contentType: contentType,
		filebyte:    body,
	}
	err = h.db.saveFileAsync(key, fileData)
	return echo.NewHTTPError(http.StatusInternalServerError, err)
}

func (h *handler) postPageHandler(c echo.Context) (err error) {
	method := c.FormValue("_method")
	log.Println(method)
	switch method {
	case "put":
		return h.putPageHandler(c)
	case "delete":
		return h.deletePageHandler(c)
	}
	return nil
}

func (h *handler) deletePageHandler(c echo.Context) (err error) {
	titleHash := c.Param("titleHash")
	err = h.db.deleteMarkdownAsync(titleHash)
	if err != nil {
		return err
	}
	err = h.db.deleteHTML(titleHash)
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, "/")
}

func (h *handler) putPageHandler(c echo.Context) (err error) {
	titleHash := c.Param("titleHash")
	title := c.FormValue("title")
	if titleHash != h.db.titleHash(title) {
		log.Println("title not match")
		log.Println("title:", title)
		log.Println("generated:", h.db.titleHash(title))
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
	err = h.db.saveMarkdownAsync(titleHash, &markdown)
	if err != nil {
		return err
	}

	// Async upload compiled HTML
	if h.db.checkPublic(markdown.titleHash) {
		go func(s3 *Wikidata, markdown pageData) {
			html := markdown
			html.body = string(renderHTML(h.db, &markdown))

			err = s3.saveHTML(&html)
			log.Println("HTML uploaded")
		}(h.db, markdown)
	}
	return c.Redirect(http.StatusFound, "/page/"+titleHash)
}
