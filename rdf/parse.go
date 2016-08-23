package rdf

import (
	"bytes"

	p "github.com/dgraph-io/dgraph/parsing"
)

func Parse(line string) (rnq NQuad, err error) {
	s := p.NewByteStream(bytes.NewBufferString(line))
	c := p.NewContext(s)
	var nqp nQuadParser
	err = c.ParseErr(&nqp)
	rnq = NQuad(nqp)
	return
}

func ParseDoc(doc string) (ret []NQuad, err error) {
	var nqd nQuadsDoc
	err = p.NewContext(p.NewByteStream(bytes.NewBufferString(doc))).ParseErr(&nqd)
	ret = nqd
	return
}
