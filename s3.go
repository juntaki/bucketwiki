package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

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

type pageData struct {
	titleHash  string
	title      string
	author     string
	body       string
	lastUpdate time.Time // only for GET
	id         string    // internal use
}

type userData struct {
	Name             string `json:"name"`
	ID               string `json:"id"`
	AuthenticateType string `json:"authtype"`
	Token            string `json:"token"`
	Secret           string `json:"secret"`
}

func (w *Wikidata) titleHash(titleHash string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(titleHash+w.wikiID)))
}

func (w *Wikidata) publicURL(titleHash string) string {
	return "http://" + w.bucket + ".s3-website-" + w.region + ".amazonaws.com/page/" + titleHash
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

func (w *Wikidata) loadDocumentID(titleHash string) (string, error) {
	r, err := w.head(titleHash)
	if err != nil {
		return "", err
	}
	return *r["ID"], nil
}

func (w *Wikidata) deleteHTML(titleHash string) {
	w.delete("page/" + titleHash + "/index.html")
}

func (w *Wikidata) deleteMarkdown(titleHash string) {
	w.delete("page/" + titleHash + "/index.md")
}

func (w *Wikidata) checkPublic(titleHash string) bool {
	resp, _ := w.getacl("page/" + titleHash + "/index.html")
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

func (w *Wikidata) aclPublic(titleHash string) error {
	return w.putacl("page/"+titleHash+"/index.html", s3.ObjectCannedACLPublicRead)
}

func (w *Wikidata) aclPrivate(titleHash string) error {
	return w.putacl("page/"+titleHash+"/index.html", s3.ObjectCannedACLPrivate)
}

// PUT request

func (w *Wikidata) saveHTML(page pageData) error {
	params := &s3.PutObjectInput{
		Bucket:      aws.String(w.bucket),
		Key:         aws.String("page/" + page.titleHash + "/index.html"),
		Body:        strings.NewReader(page.body),
		ContentType: aws.String("text/html"),
		Metadata: map[string]*string{
			"Id":     aws.String(page.id),
			"Author": aws.String(page.author),
			"Title":  aws.String(base64.StdEncoding.EncodeToString([]byte(page.title))),
		},
	}
	_, err := w.svc.PutObject(params)
	if err != nil {
		return err
	}

	return nil
}

func (w *Wikidata) saveMarkdown(page pageData) error {
	params := &s3.PutObjectInput{
		Bucket:      aws.String(w.bucket),
		Key:         aws.String("page/" + page.titleHash + "/index.md"),
		Body:        strings.NewReader(page.body),
		ContentType: aws.String("text/x-markdown"),
		Metadata: map[string]*string{
			"Id":     aws.String(page.id),
			"Author": aws.String(page.author),
			"Title":  aws.String(base64.StdEncoding.EncodeToString([]byte(page.title))),
		},
	}
	_, err := w.svc.PutObject(params)
	if err != nil {
		return err
	}

	return nil
}

func (w *Wikidata) saveUser(user userData) error {
	body, err := json.Marshal(user)
	if err != nil {
		return err
	}

	params := &s3.PutObjectInput{
		Bucket:      aws.String(w.bucket),
		Key:         aws.String("user/" + user.Name),
		Body:        bytes.NewReader(body),
		ContentType: aws.String("text/plain"),
	}
	_, err = w.svc.PutObject(params)
	if err != nil {
		return err
	}

	return nil
}

// GET
func (w *Wikidata) loadUser(name string) (*userData, error) {
	fmt.Println("key:", "user/"+name)
	paramsGet := &s3.GetObjectInput{
		Bucket: aws.String(w.bucket),
		Key:    aws.String("user/" + name),
	}
	respGet, err := w.svc.GetObject(paramsGet)
	if err != nil {
		return nil, err
	}

	body, _ := ioutil.ReadAll(respGet.Body)

	var user userData
	json.Unmarshal(body, &user)

	return &user, nil
}

func (w *Wikidata) loadHTML(titleHash string) (*pageData, error) {
	fmt.Println("key:", "page/"+titleHash+"/index.html")
	paramsGet := &s3.GetObjectInput{
		Bucket: aws.String(w.bucket),
		Key:    aws.String("page/" + titleHash + "/index.html"),
	}
	respGet, err := w.svc.GetObject(paramsGet)
	if err != nil {
		return nil, err
	}

	body, _ := ioutil.ReadAll(respGet.Body)

	page := pageData{
		titleHash:  titleHash,
		lastUpdate: *respGet.LastModified,
		body:       string(body),
	}
	if respGet.Metadata["Title"] != nil {
		title, err := base64.StdEncoding.DecodeString(*respGet.Metadata["Title"])
		if err != nil {
			return nil, err
		}
		page.title = title
	}
	if respGet.Metadata["Id"] != nil {
		page.id = *respGet.Metadata["Id"]
	}
	if respGet.Metadata["Author"] != nil {
		page.author = *respGet.Metadata["Author"]
	}
	return &page, nil
}

func (w *Wikidata) loadMarkdown(titleHash string) (*pageData, error) {
	fmt.Println("key:", "page/"+titleHash+"/index.md")
	paramsGet := &s3.GetObjectInput{
		Bucket: aws.String(w.bucket),
		Key:    aws.String("page/" + titleHash + "/index.md"),
	}
	respGet, err := w.svc.GetObject(paramsGet)
	if err != nil {
		return nil, err
	}

	body, _ := ioutil.ReadAll(respGet.Body)

	page := pageData{
		titleHash:  titleHash,
		lastUpdate: *respGet.LastModified,
		body:       string(body),
	}
	if respGet.Metadata["Title"] != nil {
		title, err := base64.StdEncoding.DecodeString(*respGet.Metadata["Title"])
		if err != nil {
			return nil, err
		}
		page.title = title
	}
	if respGet.Metadata["Id"] != nil {
		page.id = *respGet.Metadata["Id"]
	}
	if respGet.Metadata["Author"] != nil {
		page.author = *respGet.Metadata["Author"]
	}
	return &page, nil
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
