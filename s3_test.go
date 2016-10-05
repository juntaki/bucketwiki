package main

import (
	"fmt"
	"os"
	"testing"
)

func TestList(t *testing.T) {
	w := Wikidata{
		bucket: os.Getenv("AWS_BUCKET_NAME"),
		region: os.Getenv("AWS_BUCKET_REGION"),
	}
	w.connect()

	list, _ := w.list()
	fmt.Println(list)
}
