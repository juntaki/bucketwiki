package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/k0kubun/pp"
	"github.com/labstack/echo"
)

var h handler
var e *echo.Echo

func init() {
	wikidata := &Wikidata{
		svc:        &mockS3{},
		bucket:     "testbucket",
		region:     "testregion",
		wikiSecret: "testSecret",
	}
	wikidata.initializeCache()
	h = handler{db: wikidata}

	e = echo.New()
	t := &Template{
		templates: template.Must(template.ParseGlob("style/*.html")),
	}
	e.Renderer = t
}

func TestLogin(t *testing.T) {
	req, err := http.NewRequest("POST", "/login", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Form = url.Values{
		"username": []string{"user"},
		"password": []string{"74b9ae037d3d2c12acedff18b2b7bd22b8de586f32a9860ad6fa9276f1a942f7"},
	}

	cookie := &http.Cookie{Name: "sessionID", Value: "id"}
	req.Header["Cookie"] = []string{cookie.String()}

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_, err = c.Cookie("sessionID")
	if err != nil {
		t.Fatal(err)
	}

	err = h.loginHandler(c)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusFound ||
		rec.HeaderMap["Location"][0] != "/" {
		pp.Print(rec)
		t.Fatal(rec.Code)
	}
}

func CheckStatus(status int, path string, handler echo.HandlerFunc) error {
	req, err := http.NewRequest("", path, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	sess := &sessionData{
		Login: true,
	}
	c.Set("session", sess)
	err = handler(c)
	if err != nil {
		if he, ok := err.(*echo.HTTPError); ok {
			if he.Code != status {
				return fmt.Errorf("code expected %d, but got %d", status, he.Code)
			}
			return nil
		}
		return err
	}
	if status >= 400 {
		return fmt.Errorf("error should be passed by echo.HTTPError, rec.Code=%d", rec.Code)
	}
	if rec.Code != status {
		return fmt.Errorf("code expected %d, but got %d", status, rec.Code)
	}
	return nil
}

func TestPageHandler(t *testing.T) {
	var err error
	err = CheckStatus(http.StatusOK, "/login", h.loginPageHandler)
	if err != nil {
		t.Error(err)
	}
	err = CheckStatus(http.StatusFound, "/login", h.loginHandler)
	if err != nil {
		t.Error(err)
	}
	err = CheckStatus(http.StatusOK, "/signup", h.signupPageHandler)
	if err != nil {
		t.Error(err)
	}
	err = CheckStatus(http.StatusFound, "/signup", h.signupHandler)
	if err != nil {
		t.Error(err)
	}
	err = CheckStatus(http.StatusFound, "/logout", h.logoutHandler)
	if err != nil {
		t.Error(err)
	}
	err = CheckStatus(http.StatusFound, "/auth/callback", h.authCallbackHandler)
	if err != nil {
		t.Error(err)
	}
	err = CheckStatus(http.StatusBadRequest, "/auth", h.authHandler)
	if err != nil {
		t.Error(err)
	}
	err = CheckStatus(http.StatusBadRequest, "/page/titleHash/upload", h.uploadHandler)
	if err != nil {
		t.Error(err)
	}
	err = CheckStatus(http.StatusOK, "/page/titleHash/edit", h.editorHandler)
	if err != nil {
		t.Error(err)
	}
	//err = CheckStatus(http.StatusOK,"/page/titleHash/history", h.historyPageHandler)
	err = CheckStatus(http.StatusOK, "/page/titleHash/file/:filename", h.fileHandler)
	if err != nil {
		t.Error(err)
	}
	err = CheckStatus(http.StatusOK, "/page/titleHash", h.pageHandler)
	if err != nil {
		t.Error(err)
	}
	err = CheckStatus(http.StatusOK, "/page/titleHash", h.postPageHandler)
	if err != nil {
		t.Error(err)
	}
	err = CheckStatus(http.StatusBadRequest, "/page/titleHash/acl", h.aclHandler)
	if err != nil {
		t.Error(err)
	}
}
