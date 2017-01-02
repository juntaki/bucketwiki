package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"reflect"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/juntaki/transparent"
	"github.com/juntaki/transparent/lru"
	ts3 "github.com/juntaki/transparent/s3"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

// Wikidata is storing data in S3
type Wikidata struct {
	svc        s3iface.S3API
	bucket     string
	region     string
	wikiSecret string
	cacheStack map[reflect.Type]*transparent.Stack
	bareStack  *transparent.Stack
}

func (w *Wikidata) titleHash(titleHash string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(titleHash+w.wikiSecret)))
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

func (w *Wikidata) checkPublic(titleHash string) bool {
	_, err := w.getacl("page/" + titleHash + "/index.html")
	if err != nil {
		return false
	}
	return true
}

func (w *Wikidata) setACL(titleHash string, public bool) error {
	// public: Upload HTML and set file permission as public
	// private: Delete HTML and set file permission as private
	var err error
	if public {
		markdown := &pageData{titleHash: titleHash}
		err = w.loadBare(markdown)
		if err != nil {
			return err
		}
		html := &htmlData{titleHash: titleHash}

		unsafe := blackfriday.MarkdownCommon([]byte(markdown.body))
		html.body = string(bluemonday.UGCPolicy().SanitizeBytes(unsafe))

		w.saveBare(html)
		log.Println("HTML uploaded")
	} else {
		html := &htmlData{titleHash: titleHash}
		w.deleteBare(html)
	}
	if err != nil {
		return err
	}

	params := &s3.ListObjectsV2Input{
		Bucket:    aws.String(w.bucket),
		MaxKeys:   aws.Int64(30),
		Prefix:    aws.String("page/" + titleHash + "/file/"),
		Delimiter: aws.String("/"),
	}
	resp, err := w.svc.ListObjectsV2(params)
	if err != nil {
		return err
	}
	for _, c := range resp.Contents {
		if public {
			err = w.putacl(*c.Key, s3.ObjectCannedACLPublicRead)
		} else {
			err = w.putacl(*c.Key, s3.ObjectCannedACLPrivate)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Wikidata) loadMarkdownMetadata(titleHash string) (*pageData, error) {
	log.Println("key:", "page/"+titleHash+"/index.md")
	paramsGet := &s3.HeadObjectInput{
		Bucket: aws.String(w.bucket),
		Key:    aws.String("page/" + titleHash + "/index.md"),
	}
	respGet, err := w.svc.HeadObject(paramsGet)
	if err != nil {
		return nil, err
	}

	page := pageData{
		titleHash:  titleHash,
		lastUpdate: *respGet.LastModified,
	}
	if respGet.Metadata["Title"] != nil {
		title, err := base64.StdEncoding.DecodeString(*respGet.Metadata["Title"])
		if err != nil {
			return nil, err
		}
		page.title = string(title)
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

func (w *Wikidata) listhistory(titleHash string) ([][]string, error) {
	params := &s3.ListObjectVersionsInput{
		Bucket:  aws.String(w.bucket),
		Prefix:  aws.String("page/" + titleHash + "/index.md"),
		MaxKeys: aws.Int64(10),
	}
	resp, err := w.svc.ListObjectVersions(params)
	if err != nil {
		return nil, err
	}

	var result [][]string
	for _, v := range resp.Versions {
		version := []string{v.LastModified.String(), *v.VersionId}
		result = append(result, version)
	}
	return result, nil
}

func (w *Wikidata) connect() error {
	sess, err := session.NewSession()
	if err != nil {
		return err
	}
	w.svc = s3.New(sess, &aws.Config{
		Region: aws.String(w.region),
	})

	w.wikiSecret = os.Getenv("WIKI_SECRET")

	err = w.initializeCache()
	if err != nil {
		return err
	}
	return nil
}

func (w *Wikidata) initializeCache() error {
	bare, err := ts3.NewBareSource(w.svc)
	if err != nil {
		return err
	}

	w.bareStack = transparent.NewStack()
	w.bareStack.Stack(bare)
	w.bareStack.Start()

	w.cacheStack = make(map[reflect.Type]*transparent.Stack)
	w.newCacheStack(bare, reflect.TypeOf(pageData{}))
	w.newCacheStack(bare, reflect.TypeOf(htmlData{}))
	w.newCacheStack(bare, reflect.TypeOf(userData{}))
	w.newCacheStack(bare, reflect.TypeOf(fileData{}))
	w.newCacheStack(bare, reflect.TypeOf(sessionData{}))
	return nil
}

func (w *Wikidata) newCacheStack(bare transparent.Layer, t reflect.Type) error {
	lruL, err := lru.NewCache(10, 10)
	if err != nil {
		return err
	}
	w.cacheStack[t] = transparent.NewStack()
	w.cacheStack[t].Stack(bare)
	w.cacheStack[t].Stack(lruL)
	w.cacheStack[t].Start()
	return nil
}
