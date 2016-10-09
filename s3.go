package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Wikidata is storing data in S3
type Wikidata struct {
	svc    *s3.S3
	bucket string
	region string
}

func (w *Wikidata) delete(basename, suffix string) error {
	params := &s3.DeleteObjectInput{
		Bucket: aws.String(w.bucket),
		Key:    aws.String(basename + suffix),
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

func (w *Wikidata) loadHTML(title string) ([]byte, error) {
	return w.loadObject("files/" + title + ".html")
}

func (w *Wikidata) saveHTML(title string, html string) {
	w.put("files/"+title, ".html", []byte(html))
}

func (w *Wikidata) deleteHTML(title string) {
	w.delete("files/"+title, ".html")
}

func (w *Wikidata) loadMarkdown(title string) ([]byte, error) {
	return w.loadObject("files/" + title + ".md")
}

func (w *Wikidata) saveMarkdown(title string, markdown string) {
	w.put("files/"+title, ".md", []byte(markdown))
}

func (w *Wikidata) deleteMarkdown(title string) {
	w.delete("files/"+title, ".md")
}

func (w *Wikidata) loadUser(username string) ([]byte, error) {
	return w.loadObject("users/" + username)
}

func (w *Wikidata) saveUser(username string, userdata string) {
	w.put("users/"+username, "", []byte(userdata))
}

func (w *Wikidata) put(basename, suffix string, payload []byte) error {
	params := &s3.PutObjectInput{
		Bucket:      aws.String(w.bucket),
		Key:         aws.String(basename + suffix),
		Body:        bytes.NewReader(payload),
		ContentType: aws.String(mime.TypeByExtension(suffix)),
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

func (w *Wikidata) listBasenameWithSuffix(suffix string) ([]string, error) {
	params := &s3.ListObjectsV2Input{
		Bucket:  aws.String(w.bucket),
		MaxKeys: aws.Int64(30),
	}
	resp, err := w.svc.ListObjectsV2(params)

	if err != nil {
		return nil, err
	}

	var result []string
	for _, c := range resp.Contents {
		if strings.HasSuffix(*c.Key, suffix) &&
			strings.HasPrefix(*c.Key, "files/") {
			item := strings.TrimRight(*c.Key, suffix)
			item = strings.TrimLeft(item, "files/")
			result = append(result, item)
		}
	}
	return result, nil
}

func (w *Wikidata) connect() error {
	sess, err := session.NewSession()
	if err != nil {
		return err
	}
	w.svc = s3.New(sess, &aws.Config{Region: aws.String(w.region)})
	return nil
}
