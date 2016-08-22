package parsing

import (
	"fmt"
	"reflect"
)

type ParseResult struct {
	Stream  Stream
	HaltErr Error
}

// Implemented by types that can consume some tokens from a stream.
type Parser interface {
	// Returns the stream after parsing the current value, or panics with
	// SyntaxError.
	Parse(*Context)
}

// Optional interface for Parsers with custom names.
type Namer interface {
	Name() string
}

func ParserName(p Parser) string {
	if n, ok := p.(Namer); ok {
		return n.Name()
	}
	t := reflect.ValueOf(p).Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Name() == "" {
		panic(p)
	}
	return t.Name()
}

type ParseFunc func(*Context)

func (pf ParseFunc) Parse(c *Context) {
	pf(c)
}

func (pf ParseFunc) Name() string {
	return fmt.Sprintf("%v", pf)
}
