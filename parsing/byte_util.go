package parsing

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/bradfitz/iter"
)

func Bytes(bs string) Parser {
	return ParseFunc(func(c *Context) {
		for _, b := range []byte(bs) {
			_b := c.Token().(byte)
			if _b != b {
				c.Fatal(fmt.Errorf("expected %q but got %q", b, _b))
			}
			c.Advance()
		}
	})
}

func Byte(b byte) Parser {
	return NamedParseFunc{fmt.Sprintf("%q", b), func(c *Context) {
		if c.Token().(byte) != b {
			c.FailNow()
		}
		c.Advance()
	}}
}

type BytesWhile struct {
	B    []byte
	Pred func(byte) bool
}

func (me *BytesWhile) Parse(c *Context) {
	for c.Stream().Err() == nil {
		b := c.Token().(byte)
		if !me.Pred(b) {
			break
		}
		me.B = append(me.B, b)
		c.Advance()
	}
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

func (re *re) Parse(c *Context) {
	locs := re.re.FindReaderSubmatchIndex(&streamRuneReader{c.Stream()})
	if locs == nil {
		c.FailNow()
	}
	if locs[0] != 0 {
		// Doesn't make sense if it wasn't matching from the start of the
		// stream. This is a requirement.
		panic(locs[0])
	}
	var buf bytes.Buffer
	for range iter.N(locs[1]) {
		buf.WriteByte(c.Token().(byte))
		c.Advance()
	}
	for i := 2; i < len(locs); i += 2 {
		re.Submatches = append(re.Submatches, string(buf.Bytes()[locs[i]:locs[i+1]]))
	}
}

func Regexp(pattern string) *re {
	return &re{
		re: regexp.MustCompile("^" + pattern),
	}
}
