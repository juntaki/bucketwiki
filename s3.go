package main

import (
	"bytes"
	"io"
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
		if strings.HasSuffix(*c.Key, suffix) {
			result = append(result, strings.TrimRight(*c.Key, suffix))
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
