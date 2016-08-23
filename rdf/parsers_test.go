package rdf

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	p "github.com/dgraph-io/dgraph/parsing"
)

func TestBNLabel(t *testing.T) {
	s := p.NewByteStream(bytes.NewBufferString("_:hello"))
	c := p.NewContext(s)
	var bn bnLabel
	c.Parse(&bn)
}

func TestLineWS(t *testing.T) {
	c := p.NewContext(p.NewByteStream(bytes.NewBufferString("")))
	c.Parse(&lineWS)
	assert.EqualValues(t, io.EOF, c.Stream().Err())

	c = p.NewContext(p.NewByteStream(bytes.NewBufferString("a")))
	c.Parse(&lineWS)
	assert.NoError(t, c.Stream().Err())
	t.Log(c.Stream().Position())

	c = p.NewContext(p.NewByteStream(bytes.NewBufferString(" a")))
	c.Parse(&lineWS)
	assert.NoError(t, c.Stream().Err())
	t.Log(c.Stream().Position())

	c = p.NewContext(p.NewByteStream(bytes.NewBufferString(" # comment\n a")))
	c.Parse(&lineWS)
	assert.NoError(t, c.Stream().Err())
	t.Log(c.Stream().Position())
}
