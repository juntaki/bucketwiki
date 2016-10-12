package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

// MarkdownToHTML convert markdown to html
func MarkdownToHTML(text []byte) (string, error) {
	req, err := http.NewRequest(
		"POST",
		"https://api.github.com/markdown/raw",
		bytes.NewBuffer(text),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: time.Duration(15 * time.Second)}
	resp, err := client.Do(req)

	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	// Wiki Link: convert "[[Page title]]" to Linked text
	rep := regexp.MustCompile(`\[\[(.*?)\]\]`)
	title := "$1"
	escapedTitle := (&url.URL{Path: title}).String()
	str := string(body)

	str = rep.ReplaceAllString(str, "<a href=\"/page/"+escapedTitle+"\">"+title+"</a>")
	return str, err
}
