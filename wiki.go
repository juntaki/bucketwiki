package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
)

// Wikidata is storing data in S3
type Wikidata struct {
	svc    *s3.S3
	bucket string
	region string
}

func (w *Wikidata) put(key string, payload []byte) error {
	params := &s3.PutObjectInput{
		Bucket: aws.String(w.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(payload),
	}
	_, err := w.svc.PutObject(params)
	if err != nil {
		return err
	}

	return nil
}

func (w *Wikidata) get(key string) (io.Reader, error) {
	paramsGet := &s3.GetObjectInput{
		Bucket: aws.String(w.bucket),
		Key:    aws.String(key),
	}
	respGet, err := w.svc.GetObject(paramsGet)
	if err != nil {
		return nil, err
	}

	return respGet.Body, nil
}

func (w *Wikidata) connect() error {
	sess, err := session.NewSession()
	if err != nil {
		return err
	}
	w.svc = s3.New(sess, &aws.Config{Region: aws.String(w.region)})
	return nil
}

// Page is single wikipage
type Page struct {
	Title string
	Body  []byte
}

func (p *Page) save(w *Wikidata) error {
	return w.put(p.Title, p.Body)
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
		p := &Page{Title: title, Body: []byte(body)}
		p.save(&w)
		c.Redirect(http.StatusFound, "/view/"+title)
	})

	router.Run(":8080")
}
