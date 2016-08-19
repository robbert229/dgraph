package gql

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	p "github.com/dgraph-io/dgraph/parsing"
)

const (
	query    = "query"
	mutation = "mutation"
)

type Document struct {
	Operations []Operation
	// Fragments  []Fragment
}

func (d *Document) query() (ret *Operation, err error) {
	for _, op := range d.Operations {
		if op.Type == query || op.Type == "" {
			if ret != nil {
				err = errors.New("document contains multiple query operations")
				return
			}
			ret = &op
		}
	}
	return
}

func (d *Document) Parse(s p.Stream) p.Stream {
	d.Operations = nil
	for {
		p.DiscardWhitespace(&s)
		if !s.Good() {
			return s
		}
		var op Operation
		s = p.Parse(s, &op)
		d.Operations = append(d.Operations, op)
	}
}

type Operation struct {
	Type       string
	Name       string
	Variables  struct{}
	Directives struct{}
	Selections SelectionSet
}

func (op *Operation) Parse(s p.Stream) p.Stream {
	oo := p.OneOf(p.Bytes(query), p.Bytes(mutation))
	s, err := p.ParseErr(s, &oo)
	if err == nil {
		switch oo.Index {
		case 0:
			op.Type = query
		case 1:
			op.Type = mutation
		}
	}
	return p.Parse(s, &op.Selections)
}

type SelectionSet []Selection

func (ss *SelectionSet) Parse(s p.Stream) p.Stream {
	*ss = nil
	p.DiscardWhitespace(&s)
	s = p.Byte('{').Parse(s)
	for {
		p.DiscardWhitespace(&s)
		var sel Selection
		oo := p.OneOf(p.Byte('}'), &sel)
		s = p.Parse(s, &oo)
		switch oo.Index {
		case 0:
			return s
		case 1:
			*ss = append(*ss, sel)
		}
		p.DiscardWhile(&s, func(b byte) bool {
			return unicode.IsSpace(rune(b)) || b == ','
		})
	}
}

type Selection struct {
	Field
}

type Field struct {
	Name       Name
	Args       Arguments
	Selections SelectionSet
}

func (f *Field) Parse(s p.Stream) p.Stream {
	s = p.Parse(s, &f.Name)
	p.DiscardWhitespace(&s)
	s = p.Maybe(&f.Args).Parse(s)
	p.DiscardWhitespace(&s)
	s = p.Maybe(&f.Selections).Parse(s)
	p.DiscardWhitespace(&s)
	return s
}

type Arguments []Argument

func (args *Arguments) Parse(s p.Stream) p.Stream {
	*args = nil
	s = p.Byte('(').Parse(s)
	for {
		p.DiscardWhitespace(&s)
		var arg Argument
		oo := p.OneOf(p.Byte(')'), &arg)
		s = oo.Parse(s)
		switch oo.Index {
		case 0:
			return s
		case 1:
			*args = append(*args, arg)
		}
	}
}

type Argument struct {
	Name  Name
	Value Value
}

type Value string

func (v *Value) Parse(s p.Stream) p.Stream {
	bw := p.BytesWhile{
		Pred: func(b byte) bool {
			r := rune(b)
			return !unicode.IsSpace(r) && !strings.ContainsRune("()", r)
		},
	}
	s = bw.Parse(s)
	*v = Value(bw.B)
	return s
}

func (arg *Argument) Parse(s p.Stream) p.Stream {
	s = arg.Name.Parse(s)
	p.DiscardWhitespace(&s)
	s = p.Byte(':').Parse(s)
	p.DiscardWhitespace(&s)
	return arg.Value.Parse(s)
}

type Name string

func (n *Name) Parse(s p.Stream) p.Stream {
	w := p.BytesWhile{
		Pred: func(b byte) bool {
			r := rune(b)
			return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' || r == '.'
		},
	}
	s = p.Parse(s, &w)
	if len(w.B) < 1 || unicode.IsNumber(rune(w.B[0])) {
		panic(p.NewSyntaxError(p.SyntaxErrorContext{
			Err: fmt.Errorf("bad len or starts with number"),
		}))
	}
	*n = Name(w.B)
	return s
}
