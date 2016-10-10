package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	uuid "github.com/satori/go.uuid"
)

// Wikidata is storing data in S3
type Wikidata struct {
	svc    *s3.S3
	bucket string
	region string
	uuid   uuid.UUID
}

func (w *Wikidata) delete(key string) error {
	params := &s3.DeleteObjectInput{
		Bucket: aws.String(w.bucket),
		Key:    aws.String(key),
	}
	_, err := w.svc.DeleteObject(params)
	if err != nil {
		return err
	}

	return nil
}

func (w *Wikidata) loadObject(title string) ([]byte, error) {
	r, err := w.get(title)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}

func (w *Wikidata) loadUUID(title string) (string, error) {
	r, err := w.head(title)
	if err != nil {
		return "", err
	}
	return *r["uuid"], nil
}

func (w *Wikidata) loadHTML(title string) ([]byte, error) {
	uuidTitle := uuid.NewV5(w.uuid, title).String()
	return w.loadObject("page/" + uuidTitle + "/index.html")
}

func (w *Wikidata) saveHTML(title, uuidStatic string, html string) {
	uuidTitle := uuid.NewV5(w.uuid, title).String()
	w.put("page/"+uuidTitle+"/index.html", "text/html", uuidStatic, []byte(html))
}

func (w *Wikidata) deleteHTML(title string) {
	uuidTitle := uuid.NewV5(w.uuid, title).String()
	w.delete("page/" + uuidTitle + "/index.html")
}

func (w *Wikidata) loadMarkdown(title string) ([]byte, error) {
	uuidTitle := uuid.NewV5(w.uuid, title).String()
	return w.loadObject("page/" + uuidTitle + "/index.md")
}

func (w *Wikidata) saveMarkdown(title, uuidStatic string, markdown string) {
	uuidTitle := uuid.NewV5(w.uuid, title).String()
	w.put("page/"+uuidTitle+"/index.md", "text/x-markdown", uuidStatic, []byte(markdown))
}

func (w *Wikidata) deleteMarkdown(title string) {
	uuidTitle := uuid.NewV5(w.uuid, title).String()
	w.delete("page/" + uuidTitle + "/index.md")
}

func (w *Wikidata) loadUser(username string) ([]byte, error) {
	return w.loadObject("users/" + username)
}

func (w *Wikidata) saveUser(username string, userdata string) {
	w.put("users/"+username, "text/plain", "", []byte(userdata))
}

func (w *Wikidata) put(key, mime, uuid string, payload []byte) error {
	params := &s3.PutObjectInput{
		Bucket:      aws.String(w.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(payload),
		ContentType: aws.String(mime),
		Metadata: map[string]*string{
			"uuid": aws.String(uuid),
		},
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

func (w *Wikidata) head(key string) (map[string]*string, error) {
	paramsGet := &s3.HeadObjectInput{
		Bucket: aws.String(w.bucket),
		Key:    aws.String(key),
	}
	resp, err := w.svc.HeadObject(paramsGet)
	if err != nil {
		return nil, err
	}

	return resp.Metadata, nil
}

func (w *Wikidata) list() ([]string, error) {
	params := &s3.ListObjectsV2Input{
		Bucket:    aws.String(w.bucket),
		MaxKeys:   aws.Int64(30),
		Prefix:    aws.String("page/"),
		Delimiter: aws.String("/"),
	}
	resp, err := w.svc.ListObjectsV2(params)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, c := range resp.CommonPrefixes {
		item := strings.TrimRight(*c.Prefix, "/")
		item = strings.TrimLeft(item, "page/")
		result = append(result, item)
	}
	return result, nil
}

func (w *Wikidata) connect() error {
	sess, err := session.NewSession()
	if err != nil {
		return err
	}
	w.svc = s3.New(sess, &aws.Config{Region: aws.String(w.region)})
	w.uuid, _ = uuid.FromString(os.Getenv("UUID"))
	return nil
}
