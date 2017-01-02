package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

type mockS3 struct {
	s3iface.S3API
}

func (m *mockS3) GetObject(i *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	body := []byte("test")
	meta := map[string]*string{}
	contentType := "application/octet-stream"
	switch strings.Split(*i.Key, "/")[0] {
	case "page":
		switch path.Base(*i.Key) {
		case "index.md":
			contentType = "text/x-markdown"
			meta["Title"] = aws.String(base64.StdEncoding.EncodeToString([]byte("testTitle")))
			meta["ID"] = aws.String("testID")
			meta["Author"] = aws.String("testauthor")
		case "index.html":
			contentType = "text/html"
			meta["Title"] = aws.String(base64.StdEncoding.EncodeToString([]byte("testTitle")))
			meta["ID"] = aws.String("testID")
			meta["Author"] = aws.String("testauthor")
		default:
			contentType = "image/jpeg"
		}
	case "user":
		contentType = "text/plain"
		user := userData{
			Name: "user",
			ID:   "userID",
		}
		body, _ = json.Marshal(user)
	case "session":
		contentType = "text/plain"
		session := sessionData{
			ID:        "id",
			Challange: "challange",
			User:      "user",
			Login:     true,
		}
		body, _ = json.Marshal(session)
	}
	return &s3.GetObjectOutput{
		Body:         ioutil.NopCloser(bytes.NewReader(body)),
		Metadata:     meta,
		LastModified: aws.Time(time.Now()),
		ContentType:  aws.String(contentType),
	}, nil
}

func (m *mockS3) HeadObject(*s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	meta := map[string]*string{}
	meta["Title"] = aws.String(base64.StdEncoding.EncodeToString([]byte("testTitle")))
	meta["ID"] = aws.String("testID")
	meta["Author"] = aws.String("testauthor")
	return &s3.HeadObjectOutput{
		Metadata:     meta,
		LastModified: aws.Time(time.Now()),
	}, nil
}

func (m *mockS3) PutObjectAcl(*s3.PutObjectAclInput) (*s3.PutObjectAclOutput, error) {
	return &s3.PutObjectAclOutput{}, nil
}

func (m *mockS3) GetObjectAcl(*s3.GetObjectAclInput) (*s3.GetObjectAclOutput, error) {
	return &s3.GetObjectAclOutput{}, nil
}

func (m *mockS3) PutObject(i *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3) DeleteObject(i *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	return &s3.DeleteObjectOutput{}, nil
}

func (m *mockS3) ListObjectsV2(*s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
	return &s3.ListObjectsV2Output{}, nil
}
