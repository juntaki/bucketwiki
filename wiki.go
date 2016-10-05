package main

import (
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// Page is single wikipage
type Page struct {
	Title string
	Body  []byte
}

func loadPage(title string, w *Wikidata) (*Page, error) {
	r, err := w.get(title)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func main() {
	w := Wikidata{
		bucket: os.Getenv("AWS_BUCKET_NAME"),
		region: os.Getenv("AWS_BUCKET_REGION"),
	}
	w.connect()

	router := gin.Default()
	router.LoadHTMLGlob("*.html")

	router.GET("/list", func(c *gin.Context) {
		list, err := w.list()
		if err != nil {
			return
		}
		c.HTML(http.StatusOK, "list.html", gin.H{
			"List": list,
		})
	})

	router.GET("/view/:title", func(c *gin.Context) {
		title := c.Param("title")
		p, err := loadPage(title, &w)
		if err != nil {
			c.Redirect(http.StatusFound, "/edit/"+title)
			return
		}
		c.HTML(http.StatusOK, "view.html", gin.H{
			"Title": p.Title,
			"Body":  p.Body,
		})
	})

	router.GET("/edit/:title", func(c *gin.Context) {
		title := c.Param("title")
		body := []byte("")
		p, err := loadPage(title, &w)
		if err == nil {
			body = p.Body
		}
		c.HTML(http.StatusOK, "edit.html", gin.H{
			"Title": c.Param("title"),
			"Body":  body,
		})
	})

	router.POST("/save/:title", func(c *gin.Context) {
		title := c.Param("title")
		body, _ := c.GetPostForm("body")
		w.put(title, []byte(body))
		c.Redirect(http.StatusFound, "/view/"+title)
	})

	router.Run(":8080")
}
