package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
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
	wikiID string
}

func (w *Wikidata) titleID(title string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(title+w.wikiID)))
}

func (w *Wikidata) publicURL(title string) string {
	titleID := w.titleID(title)
	return "http://" + w.region + "s3-website-" + w.bucket + ".amazonaws.com/page/" + titleID
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

func (w *Wikidata) loadDocumentID(title string) (string, error) {
	r, err := w.head(title)
	if err != nil {
		return "", err
	}
	return *r["documentID"], nil
}

func (w *Wikidata) loadHTML(title string) ([]byte, error) {
	return w.loadObject("page/" + w.titleID(title) + "/index.html")
}

func (w *Wikidata) saveHTML(title, documentID string, html string) {
	w.put("page/"+w.titleID(title)+"/index.html", "text/html", documentID, []byte(html))
}

func (w *Wikidata) deleteHTML(title string) {
	w.delete("page/" + w.titleID(title) + "/index.html")
}

func (w *Wikidata) loadMarkdown(title string) ([]byte, error) {
	return w.loadObject("page/" + w.titleID(title) + "/index.md")
}

func (w *Wikidata) saveMarkdown(title, documentID string, markdown string) {
	w.put("page/"+w.titleID(title)+"/index.md", "text/x-markdown", documentID, []byte(markdown))
}

func (w *Wikidata) deleteMarkdown(title string) {
	w.delete("page/" + w.titleID(title) + "/index.md")
}

func (w *Wikidata) checkPublic(title string) bool {
	resp, _ := w.getacl("page/" + w.titleID(title) + "/index.html")
	for _, g := range resp.Grants {

		if g.Grantee.URI != nil &&
			*g.Grantee.URI == "http://acs.amazonaws.com/groups/global/AllUsers" &&
			g.Permission != nil &&
			*g.Permission == "READ" {
			return true
		}
	}
	return false
}

func (w *Wikidata) aclPublic(title string) error {
	return w.putacl("page/"+w.titleID(title)+"/index.html", s3.ObjectCannedACLPublicRead)
}

func (w *Wikidata) aclPrivate(title string) error {
	return w.putacl("page/"+w.titleID(title)+"/index.html", s3.ObjectCannedACLPrivate)
}

func (w *Wikidata) loadUser(username string) ([]byte, error) {
	return w.loadObject("users/" + username)
}

func (w *Wikidata) saveUser(username string, userdata string) {
	w.put("users/"+username, "text/plain", "", []byte(userdata))
}

func (w *Wikidata) put(key, mime, documentID string, payload []byte) error {
	params := &s3.PutObjectInput{
		Bucket:      aws.String(w.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(payload),
		ContentType: aws.String(mime),
		Metadata: map[string]*string{
			"DocumentID": aws.String(documentID),
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

func (w *Wikidata) putacl(key string, acl string) error {
	params := &s3.PutObjectAclInput{
		Bucket: aws.String(w.bucket),
		Key:    aws.String(key),
		ACL:    aws.String(acl),
	}
	_, err := w.svc.PutObjectAcl(params)

	if err != nil {
		return err
	}
	return nil
}

func (w *Wikidata) getacl(key string) (*s3.GetObjectAclOutput, error) {
	params := &s3.GetObjectAclInput{
		Bucket: aws.String(w.bucket),
		Key:    aws.String(key),
	}
	return w.svc.GetObjectAcl(params)
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
	w.wikiID = os.Getenv("WIKI_ID")
	return nil
}
