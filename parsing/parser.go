package parsing

import (
	"fmt"
	"reflect"
)

// Implemented by types that can consume some tokens from a stream.
type Parser interface {
	// Returns the stream after parsing the current value, or panics with
	// SyntaxError.
	Parse(Stream) Stream
}

// Optional interface for Parsers with custom names.
type Namer interface {
	Name() string
}

// Runs the Parser, recovering any SyntaxError, and returning it, and the
// passed in Stream if one occurs.
func ParseErr(s Stream, p Parser) (_s Stream, err SyntaxError) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		_s = s
		if se, ok := r.(SyntaxError); ok {
			err = se
			return
		}
		panic(r)
	}()
	_s = Parse(s, p)
	return
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

func recoverSyntaxError(f func(SyntaxError)) {
	r := recover()
	if r == nil {
		return
	}
	se, ok := r.(SyntaxError)
	if !ok {
		panic(r)
	}
	f(se)
}

func Parse(s Stream, p Parser) Stream {
	defer recoverSyntaxError(func(se SyntaxError) {
		se.AddContext(SyntaxErrorContext{
			Parser: p,
			Stream: s,
		})
		panic(se)
	})
	return p.Parse(s)
}

type ParseFunc func(Stream) Stream

func (pf ParseFunc) Parse(s Stream) Stream {
	return pf(s)
}

func (pf ParseFunc) Name() string {
	return fmt.Sprintf("%v", pf)
}
