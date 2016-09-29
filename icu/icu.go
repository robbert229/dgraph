package icu

// #include <stdint.h>
// #include <stdlib.h>
// #include "icuc.h"
import "C"

import (
	"unsafe"

	"github.com/dgraph-io/dgraph/x"
)

type Tokenizer struct {
	c *C.Tokenizer
}

const (
	bufLength = 1000
)

func NewTokenizer(s []byte) (*Tokenizer, error) {
	var err C.UErrorCode
	c := C.NewTokenizer(byteToChar(s), C.int(len(s)), &err)
	if int(err) > 0 {
		return nil, x.Errorf("ICU new tokenizer error %d", int(err))
	}
	return &Tokenizer{c}, nil
}

func (t *Tokenizer) Destroy() {
	C.DestroyTokenizer(t.c)
}

func (t *Tokenizer) Next() string {
	// C.TokenizerNext returns a null-terminated C string.
	return C.GoString(C.TokenizerNext(t.c))
}

// byteToChar returns *C.char from byte slice.
func byteToChar(b []byte) *C.char {
	var c *C.char
	if len(b) > 0 {
		c = (*C.char)(unsafe.Pointer(&b[0]))
	}
	return c
}
