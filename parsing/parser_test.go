package parsing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type someParser struct{}

func (*someParser) Parse(*Context) {
}

func TestParserName(t *testing.T) {
	p := ParseFunc(func(*Context) {})
	assert.Equal(t, "ParseFunc", ParserName(p))
	assert.Equal(t, "someParser", ParserName(&someParser{}))
	// var i Parser
	// assert.Equal(t, "", ParserName(i))
}
