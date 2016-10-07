package main

import (
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

	router.GET("/list", func(c *gin.Context) {
		list, err := w.listBasenameWithSuffix(".md")
		if err != nil {
			return
		}
		c.HTML(http.StatusOK, "list.html", gin.H{
			"List": list,
		})
	})

	router.GET("/view/:title", func(c *gin.Context) {
		title := c.Param("title")
		html, err := loadHTML(title, &w)
		if err != nil {
			c.Redirect(http.StatusFound, "/edit/"+title)
			return
		}

		c.HTML(http.StatusOK, "view.html", gin.H{
			"Title": title,
			"Body":  template.HTML(string(html)),
		})
	})

	router.GET("/edit/:title", func(c *gin.Context) {
		title := c.Param("title")
		body := []byte("")
		markdown, err := loadMarkdown(title, &w)
		if err == nil {
			body = markdown
		}
		c.HTML(http.StatusOK, "edit.html", gin.H{
			"Title": c.Param("title"),
			"Body":  body,
		})
	})

	router.POST("/save/:title", func(c *gin.Context) {
		title := c.Param("title")
		markdown, _ := c.GetPostForm("body")
		w.put(title+".md", []byte(markdown))

		html, _ := Markdown([]byte(markdown))
		w.put(title+".html", []byte(html))
		c.Redirect(http.StatusFound, "/view/"+title)
	})

	router.Run(":8080")
}
