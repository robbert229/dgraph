package parsing

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegexp(t *testing.T) {
	re := Regexp(`\s+`)
	c := NewContext(NewByteStream(bytes.NewBufferString("\n\thello")))
	c.Parse(re)
	assert.Len(t, re.Submatches, 0)

	c = NewContext(NewByteStream(bytes.NewBufferString(" abc ")))
	c.Parse(re)
	assert.Len(t, re.Submatches, 0)
	t.Log(c.Stream().Position())

	c = NewContext(NewByteStream(bytes.NewBufferString("abc  ")))
	assert.Panics(t, func() { c.Parse(re) })
}
