package main

import (
	"fmt"
	"testing"
)

func TestRandomString(t *testing.T) {
	rand, err := randomString()
	if err != nil {
		t.Error("Failed")
	}
	fmt.Println(rand)

	if len(rand) != 64 {
		t.Error("Failed")
	}
}

func TestDocumentID(t *testing.T) {
	title := "title"
	secret := "secret"
	rand := documentID(title, secret)
	fmt.Println(rand)
	if len(rand) != 64 {
		t.Error("Failed")
	}
}
