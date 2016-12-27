package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/k0kubun/pp"
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
			meta["Id"] = aws.String("testID")
			meta["Author"] = aws.String("testauthor")
		case "index.html":
			contentType = "text/html"
			meta["Title"] = aws.String(base64.StdEncoding.EncodeToString([]byte("testTitle")))
			meta["Id"] = aws.String("testID")
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
	meta["Id"] = aws.String("testID")
	return &s3.HeadObjectOutput{
		Metadata: meta,
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

var wikidata *Wikidata

func init() {
	wikidata = &Wikidata{
		svc:        &mockS3{},
		bucket:     "testbucket",
		region:     "testregion",
		wikiSecret: "testSecret",
	}
}

func TestLoadMarkdown(t *testing.T) {
	page, err := wikidata.loadMarkdown("test", "")
	if err != nil {
		t.Fatal(err)
	}
	expected := &pageData{
		titleHash: "test",
		title:     "testTitle",
		author:    "testauthor",
		body:      "test",
		id:        "testID",
	}

	if page.title != expected.title ||
		page.titleHash != expected.titleHash ||
		page.author != expected.author ||
		page.body != expected.body ||
		page.id != expected.id {
		pp.Print(page)
		t.Fail()
	}
}

func TestLoadUser(t *testing.T) {
	user, err := wikidata.loadUser("testUser")
	if err != nil {
		t.Fatal(err)
	}

	expected := &userData{
		Name:             "user",
		ID:               "userID",
		AuthenticateType: "",
	}

	if user.Name != expected.Name ||
		user.ID != expected.ID {
		pp.Print(user)
		t.Fail()
	}
}

func TestLoadFile(t *testing.T) {
	fkey := fileDataKey{
		filename:  "test",
		titleHash: "test",
	}
	file, err := wikidata.loadFile(fkey)
	if err != nil {
		t.Fatal(err)
	}

	expected := &fileData{
		fileDataKey: fileDataKey{
			filename:  "test",
			titleHash: "test",
		},
		filebyte: []uint8{
			0x74, 0x65, 0x73, 0x74,
		},
		contentType: "image/jpeg",
	}

	if file.filename != expected.filename ||
		len(file.filebyte) != len(expected.filebyte) {
		pp.Print(file)
		t.Fail()
	}
}

func TestSaveFile(t *testing.T) {
	file := &fileData{
		fileDataKey: fileDataKey{
			filename:  "test",
			titleHash: "test",
		},
		filebyte: []uint8{
			0x74, 0x65, 0x73, 0x74,
		},
		contentType: "image/jpeg",
	}

	err := wikidata.saveFile(file)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSaveHTML(t *testing.T) {
	page := &pageData{
		titleHash: "test",
		title:     "testTitle",
		author:    "testauthor",
		body:      "test",
		id:        "testID",
	}
	err := wikidata.saveHTML(page)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSaveMarkdown(t *testing.T) {
	page := &pageData{
		titleHash: "test",
		title:     "testTitle",
		author:    "testauthor",
		body:      "test",
		id:        "testID",
	}
	err := wikidata.saveMarkdown(page)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSaveUser(t *testing.T) {
	user := &userData{
		Name:             "user",
		ID:               "userID",
		AuthenticateType: "",
	}
	err := wikidata.saveUser(user)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDelete(t *testing.T) {
	fkey := fileDataKey{
		filename:  "test",
		titleHash: "test",
	}
	err := wikidata.deleteUser("test")
	if err != nil {
		t.Fatal(err)
	}
	err = wikidata.deleteFile(fkey)
	if err != nil {
		t.Fatal(err)
	}
	err = wikidata.deleteMarkdown("test")
	if err != nil {
		t.Fatal(err)
	}
	err = wikidata.deleteHTML("test")
	if err != nil {
		t.Fatal(err)
	}
}
