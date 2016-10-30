package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"
)

// MarkdownToHTML convert markdown to html
func MarkdownToHTML(s3 *Wikidata, text []byte) (string, error) {
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
	rep := regexp.MustCompile(`\[\[.*?\]\]`)

	str := string(body)

	str = rep.ReplaceAllStringFunc(str, func(a string) string {
		title := a[2 : len(a)-2]
		return "<a href=\"/page/" + s3.titleHash(title) + "?title=" + title + "\">" + title + "</a>"
	})
	return str, err
}
