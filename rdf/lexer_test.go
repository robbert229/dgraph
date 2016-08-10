package rdf

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLexer(t *testing.T) {
	l := newLexer(bytes.NewBufferString("<a> <b> <c> ."))
	tok := l.Token()
	require.EqualValues(t, Token{iriRef, "a", Location{1, 1}}, tok)
	require.True(t, l.Next())
}
