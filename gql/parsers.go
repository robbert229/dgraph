package gql

import (
	"errors"
	"strings"
	"unicode"

	p "github.com/dgraph-io/dgraph/parsing"
)

const (
	query    = "query"
	mutation = "mutation"
	first    = "first"
	after    = "after"
	offset   = "offset"
)

type whitespace struct {
}

func (whitespace) Parse(c *p.Context) {
	for c.Stream().Err() == nil {
		if !unicode.IsSpace(rune(c.Stream().Token().(byte))) {
			return
		}
		c.Advance()
	}
}

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

func (d *Document) Parse(c *p.Context) {
	d.Operations = nil
	for {
		c.Parse(whitespace{})
		if c.Stream().Err() == nil {
			return
		}
		var op Operation
		c.Parse(&op)
		d.Operations = append(d.Operations, op)
	}
}

type Operation struct {
	Type       string
	Name       string
	Variables  struct{}
	Directives struct{}
	Selection  Selection
}

func (op *Operation) Parse(c *p.Context) {
	oo := p.OneOf(p.Bytes(query), p.Bytes(mutation))
	if c.TryParse(oo) {
		switch oo.Index {
		case 0:
			op.Type = query
		case 1:
			op.Type = mutation
		}
	}
	c.Parse(whitespace{})
	c.Parse(p.Byte('{'))
	c.Parse(whitespace{})
	c.Parse(&op.Selection)
	c.Parse(whitespace{})
	c.Parse(p.Byte('}'))
}

// Leaves the stream at the terminating newline, or stream error.
func discardComment(_s *p.Stream) {
	s := *_s
	for s.Good() {
		b := s.Token().(byte)
		if b == '\n' {
			break
		}
		s = s.Next()
	}
	*_s = s
}

type SelectionSet []Selection

func (ss *SelectionSet) Parse(c *p.Context) {
	*ss = nil
	c.Parse(p.Byte('{'))
	for {
		c.Parse(whitespace{})
		var sel Selection
		if !c.TryParse(&sel) {
			break
		}
		*ss = append(*ss, sel)
	}
	c.Parse(p.Byte('}'))
}

type Selection struct {
	Field
}

type Field struct {
	Name       Name
	Args       Arguments
	Selections SelectionSet
}

func (f *Field) Parse(c *p.Context) {
	c.Parse(&f.Name)
	c.Parse(whitespace{})
	c.TryParse(&f.Args)
	c.Parse(whitespace{})
	c.TryParse(&f.Selections)
}

type Arguments []Argument

func (args *Arguments) Parse(c *p.Context) {
	*args = nil
	c.Parse(p.Byte('('))
	for {
		c.Parse(whitespace{})
		var arg Argument
		if !c.TryParse(&arg) {
			break
		}
		*args = append(*args, arg)
	}
	c.Parse(p.Byte(')'))
}

type Argument struct {
	Name  Name
	Value Value
}

type Value string

func (v *Value) Parse(c *p.Context) {
	bw := p.BytesWhile{
		Pred: func(b byte) bool {
			r := rune(b)
			return !unicode.IsSpace(r) && !strings.ContainsRune("(),", r)
		},
	}
	c.Parse(&bw)
	*v = Value(bw.B)
}

func (arg *Argument) Parse(c *p.Context) {
	c.Parse(&arg.Name)
	c.Parse(whitespace{})
	c.Parse(p.Byte(':'))
	c.Parse(whitespace{})
	c.Parse(&arg.Value)
}

type Name string

func (n *Name) Parse(c *p.Context) {
	re := p.Regexp(`([_A-Za-z.][._0-9A-Za-z]*)`)
	c.Parse(re)
	*n = Name(re.Submatches[0])
}
