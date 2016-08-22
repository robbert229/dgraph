package parsing

import (
	"bytes"
	"fmt"
	"regexp"
	"unicode"

	"github.com/bradfitz/iter"
)

func Bytes(bs string) Parser {
	return ParseFunc(func(s Stream) Stream {
		for _, b := range []byte(bs) {
			if !s.Good() {
				panic(NewSyntaxError(SyntaxErrorContext{
					Stream: s,
					Err:    fmt.Errorf("expected %q but got %s", b, s.Err()),
				}))
			}
			_b := s.Token().(byte)
			if _b != b {
				panic(NewSyntaxError(SyntaxErrorContext{
					Stream: s,
					Err:    fmt.Errorf("expected %q but got %q", b, _b),
				}))
			}
			s = s.Next()
		}
		return s
	})
}

func Byte(b byte) Parser {
	return ParseFunc(func(s Stream) Stream {
		if !s.Good() {
			panic(NewSyntaxError(SyntaxErrorContext{Err: s.Err()}))
		}
		_b := s.Token().(byte)
		if _b != b {
			panic(NewSyntaxError(SyntaxErrorContext{
				Err:    fmt.Errorf("wanted %q", b),
				Stream: s,
			}))
		}
		return s.Next()
	})
}

func DiscardWhitespace(s *Stream) {
	DiscardWhile(s, func(b byte) bool {
		return unicode.IsSpace(rune(b))
	})
}

func DiscardWhile(s *Stream, pred func(byte) bool) {
	_s := *s
	for _s.Good() {
		b := _s.Token().(byte)
		if !pred(b) {
			break
		}
		_s = _s.Next()
	}
	*s = _s
}

type BytesWhile struct {
	B    []byte
	Pred func(byte) bool
}

func (me *BytesWhile) Parse(s Stream) Stream {
	for s.Good() {
		b := s.Token().(byte)
		if !me.Pred(b) {
			break
		}
		me.B = append(me.B, b)
		s = s.Next()
	}
	return s
}

type re struct {
	re         *regexp.Regexp
	Submatches []string
}

type streamRuneReader struct {
	s Stream
}

func (me *streamRuneReader) ReadRune() (r rune, size int, err error) {
	err = me.s.Err()
	if err != nil {
		return
	}
	r = rune(me.s.Token().(byte))
	size = 1
	me.s = me.s.Next()
	return
}

func (re *re) Parse(s Stream) Stream {
	locs := re.re.FindReaderSubmatchIndex(&streamRuneReader{s})
	if locs == nil {
		panic(NewSyntaxError(SyntaxErrorContext{
			Err: fmt.Errorf("no regexp match"),
		}))
	}
	var buf bytes.Buffer
	for range iter.N(locs[1]) {
		buf.WriteByte(s.Token().(byte))
		s = s.Next()
	}
	for i := 2; i < len(locs); i += 2 {
		re.Submatches = append(re.Submatches, string(buf.Bytes()[locs[i]:locs[i+1]]))
	}
	return s
}

func Regexp(pattern string) (re re) {
	re.re = regexp.MustCompile(pattern)
	return
}
