package rdf

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	p "github.com/dgraph-io/dgraph/parsing"
)

type subject string

func (me *subject) Parse(c *p.Context) {
	var (
		iriRef  iriRef
		bnLabel bnLabel
	)
	oo := p.OneOf(&iriRef, &bnLabel)
	c.Parse(oo)
	switch oo.Index {
	case 0:
		*me = subject(iriRef)
	case 1:
		*me = subject(bnLabel)
	}
}

type object struct {
	Id      string
	Literal literal
}

func (me *object) Parse(c *p.Context) {
	var (
		iriRef  iriRef
		bnLabel bnLabel
	)
	oo := p.OneOf(&iriRef, &bnLabel, &me.Literal)
	c.Parse(oo)
	switch oo.Index {
	case 0:
		me.Id = string(iriRef)
	case 1:
		me.Id = string(bnLabel)
	}
}

type eChar byte

func (me *eChar) Parse(c *p.Context) {
	c.Parse(p.Byte('\\'))
	b := c.Token().(byte)
	// ECHAR ::= '\' [tbnrf"'\]
	switch b {
	case 't':
		*me = '\t'
	case 'b':
		*me = '\b'
	case 'n':
		*me = '\n'
	case 'r':
		*me = '\r'
	case 'f':
		*me = '\f'
	case '"':
		*me = '"'
	case '\'':
		*me = '\''
	default:
		c.Fatal(fmt.Errorf("can't escape %q", b))
	}
	c.Advance()
}

type quotedStringLiteral string

func (me *quotedStringLiteral) Parse(c *p.Context) {
	c.Parse(p.Byte('"'))
	var bs []byte
	for {
		b := c.Token().(byte)
		switch b {
		case '"':
			*me = quotedStringLiteral(string(bs))
			c.Advance()
			return
		case '\\':
			var e eChar
			c.Parse(&e)
			bs = append(bs, byte(e))
		default:
			bs = append(bs, b)
			c.Advance()
		}
	}
}

type literal struct {
	Value   string
	LangTag string
}

func (l *literal) Parse(c *p.Context) {
	var qsl quotedStringLiteral
	c.Parse(&qsl)
	l.Value = string(qsl)
	var (
		langTag langTag
		iriRef  iriRef
	)
	oo := p.OneOf(
		&langTag,
		p.ParseFunc(func(c *p.Context) {
			c.Parse(p.Bytes("^^"))
			c.Parse(&iriRef)
		}),
	)
	if c.TryParse(oo) {
		switch oo.Index {
		case 0:
			l.LangTag = string(langTag)
		case 1:
			l.Value += "@@" + string(iriRef)
		}
	}
}

type untilByte struct {
	b  byte
	bs []byte
}

func (me *untilByte) Parse(c *p.Context) {
	for {
		b := c.Token().(byte)
		c.Advance()
		if b == me.b {
			return
		}
		me.bs = append(me.bs, b)
	}
}

type langTag string

func (me *langTag) Parse(c *p.Context) {
	c.Parse(p.Byte('@'))
	bw := p.BytesWhile{
		Pred: func(b byte) bool { return unicode.IsLetter(rune(b)) },
	}
	c.Parse(&bw)
	if len(bw.B) < 1 {
		c.Fatal(errors.New("require at least one letter"))
	}
	bw.Pred = func(b byte) bool {
		return b == '-' || unicode.IsLetter(rune(b)) || unicode.IsNumber(rune(b))
	}
	c.Parse(&bw)
	*me = langTag(bw.B)
}

type iriRef string

func (me *iriRef) Parse(c *p.Context) {
	c.Parse(p.Byte('<'))
	bw := p.BytesWhile{
		Pred: func(b byte) bool {
			return b > 0x20 && !strings.ContainsRune("<>\"{}|^`\\", rune(b))
		},
	}
	c.Parse(&bw)
	c.Parse(p.Byte('>'))
	*me = iriRef(bw.B)
}

type bnLabel string

func (me *bnLabel) Parse(c *p.Context) {
	c.Parse(p.Byte('_'))
	beforeColon := p.BytesWhile{
		Pred: func(b byte) bool {
			return b != ':' && !unicode.IsSpace(rune(b))
		},
	}
	c.Parse(&beforeColon)
	c.Parse(p.Byte(':'))
	rest := p.BytesWhile{
		Pred: func(b byte) bool {
			return !unicode.IsSpace(rune(b))
		},
	}
	c.Parse(&rest)
	*me = bnLabel(fmt.Sprintf("_%s:%s", beforeColon.B, rest.B))
}

type predicate struct {
	iriRef
}

type label struct {
	subject
}

type nQuadParser NQuad

func (me *nQuadParser) Parse(c *p.Context) {
	var (
		ws = p.BytesWhile{Pred: func(b byte) bool {
			return unicode.IsSpace(rune(b)) && b != '\n'
		}}
		sub      subject
		pred     predicate
		obj      object
		label    label
		optLabel = p.Maybe(&label)
	)
	c.Parse(p.Seq{&sub, &ws, &pred, &ws, &obj, &ws, &optLabel, &ws, p.Byte('.')})
	me.Subject = string(sub)
	me.Predicate = string(pred.iriRef)
	me.ObjectId = obj.Id
	if obj.Literal.Value != "" {
		me.ObjectValue = []byte(obj.Literal.Value)
	}
	if obj.Literal.LangTag != "" {
		me.Predicate += "." + obj.Literal.LangTag
	}
	if optLabel.Ok {
		me.Label = string(label.subject)
	}
}

var lineWS = p.Star(p.OneOf(
	p.Regexp("#[^\n]*"),
	&p.BytesWhile{Pred: func(b byte) bool {
		return unicode.IsSpace(rune(b))
	}},
))

type nQuadsDoc []NQuad

func (me *nQuadsDoc) Parse(c *p.Context) {
	for {
		c.Parse(&lineWS)
		if !c.Good() {
			break
		}
		var nqp nQuadParser
		c.Parse(&nqp)
		*me = append(*me, NQuad(nqp))
	}
}
