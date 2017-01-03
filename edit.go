package main

import (
	"io/ioutil"
	"net/http"
	"reflect"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/labstack/echo"
)

func (h *handler) editorHandler(c echo.Context) (err error) {
	titleHash := c.Param("titleHash")
	title := c.QueryParam("title")
	body := "# " + title + "\n"

	md := &pageData{
		titleHash: titleHash,
	}
	err = h.db.loadBare(md)
	if err == nil {
		body = md.body
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

	file, err := header.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	body, _ := ioutil.ReadAll(file)

	fileData := &fileData{
		filename:    filename,
		titleHash:   titleHash,
		contentType: contentType,
		filebyte:    body,
	}
	err = h.db.saveBare(fileData)
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
	md := &pageData{titleHash: titleHash}
	html := &htmlData{titleHash: titleHash}

	err = h.db.deleteBare(md)
	if err != nil {
		return err
	}
	err = h.db.deleteBare(html)
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

	sess := c.Get("session").(*sessionData)

	public := h.db.checkPublic(titleHash)

	// Upload markdown
	markdown := &pageData{
		titleHash:  titleHash,
		title:      title,
		author:     sess.User,
		body:       c.FormValue("body"),
		lastUpdate: time.Now(),
		public:     public,
	}

	err = h.db.saveBare(markdown)
	if err != nil {
		return err
	}

	if public {
		err = h.db.setACL(titleHash, true)
		if err != nil {
			return err
		}
	}
	return c.Redirect(http.StatusFound, "/page/"+titleHash)
}

func (w *Wikidata) uploadHTML(markdown *pageData) error {
	html := &htmlData{
		titleHash: markdown.titleHash,
		body:      string(w.renderHTML(markdown)),
	}
	err := w.saveBare(html)
	if err != nil {
		return err
	}
	w.cacheStack[reflect.TypeOf(htmlData{})].Sync()
	err = w.putacl(html.getKey(), s3.ObjectCannedACLPublicRead)
	if err != nil {
		return err
	}
	return nil
}
