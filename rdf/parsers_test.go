package rdf

import (
	"bytes"
	"testing"

	p "github.com/dgraph-io/dgraph/parsing"
)

func TestBNLabel(t *testing.T) {
	s := p.NewByteStream(bytes.NewBufferString("_:hello"))
	c := p.NewContext(s)
	var bn bnLabel
	c.Parse(&bn)
}
