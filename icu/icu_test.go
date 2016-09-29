package icu

import (
	"log"
	"testing"
)

func TestBasic(t *testing.T) {
	tokenizer, err := NewTokenizer([]byte("hello world"))
	defer tokenizer.Destroy()
	if err != nil {
		t.Error(err)
		return
	}

	log.Printf("~~[%s]\n", tokenizer.Next())
}
