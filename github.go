package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"
)

// Markdown convert markdown to html
func Markdown(text []byte) (string, error) {
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

	return string(body), err
}
