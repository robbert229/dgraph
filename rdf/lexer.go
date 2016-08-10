package rdf

import (
	"bufio"
	"fmt"
	"io"
	"unicode"

	"github.com/pkg/errors"
)

type lexer struct {
	r   *bufio.Reader
	cur Token
	// b byte
	err         error
	lastReadLoc Location
	unreadLoc   Location
}

func newLexer(r io.Reader) (l *lexer) {
	l = &lexer{
		r:           bufio.NewReader(r),
		lastReadLoc: Location{Line: 1},
	}
	l.Next()
	return
}

func (l *lexer) Err() error {
	return l.err
}

func (l *lexer) Token() Token {
	if l.err != nil {
		panic(l.err)
	}
	return l.cur
}

func (l *lexer) readByte() (b byte, err error) {
	b, err = l.r.ReadByte()
	if err != nil {
		return
	}
	l.unreadLoc = l.lastReadLoc
	if b == '\n' {
		l.lastReadLoc.Line++
		l.lastReadLoc.Col = 1
	} else {
		l.lastReadLoc.Col++
	}
	return
}

func (l *lexer) unreadByte() {
	err := l.r.UnreadByte()
	if err != nil {
		panic(err)
	}
	l.lastReadLoc = l.unreadLoc
}

func (l *lexer) discardWhitespace() {
	for {
		b, err := l.readByte()
		if err != nil {
			break
		}
		if !unicode.IsSpace(rune(b)) {
			break
		}
	}
	l.unreadByte()
}

func (l *lexer) Next() bool {
	l.discardWhitespace()
	var b byte
	b, l.err = l.readByte()
	if l.err != nil {
		return false
	}
	switch b {
	case '<':
		return l.iriRef()
	case '_':
		return l.bnLabel()
	case '"':
		return l.literal()
	case '.':
		l.cur = Token{
			Type: period,
		}
		return true
	default:
		l.setInputError(fmt.Errorf("unexpected character %q", b))
		return false
	}
}

func (l *lexer) setInputError(err error) {
	l.err = errors.Wrap(err, "input error")
}

func (l *lexer) startToken(_type TokenType) {
	l.cur = Token{
		Type:   _type,
		Origin: l.lastReadLoc,
	}
}

func (l *lexer) iriRef() bool {
	l.startToken(iriRef)
	for {
		b, err := l.readByte()
		if err != nil {
			l.setInputError(err)
			return false
		}
		if b == '>' {
			return true
		}
		l.cur.Value += string(b)
	}
}

func (l *lexer) consumeByte(b byte) {
	_b, err := l.readByte()
	if err != nil {
		l.setInputError(fmt.Errorf("error consuming byte %q: %s", b, err))
		return
	}
	if _b != b {
		l.setInputError(fmt.Errorf("expected %q but got %q", b, _b))
		return
	}
}

func (l *lexer) accumulateUntilWhitespace() (s string) {
	return l.accumulateUntilPred(func(b byte) bool {
		return unicode.IsSpace(rune(b))
	})
}

func (l *lexer) accumulateUntilPred(pred func(byte) bool) (s string) {
	for {
		b, err := l.readByte()
		if err != nil {
			l.setInputError(err)
			return
		}
		if pred(b) {
			l.unreadByte()
			return
		}
		s += string(b)
	}
}
func (l *lexer) bnLabel() bool {
	l.startToken(bnLabel)
	l.consumeByte(':')
	if l.err != nil {
		l.err = errors.Wrapf(l.err, "while parsing BLANK_NODE_LABEL at %s", l.lastReadLoc)
		return false
	}
	l.cur.Value = "_:" + l.accumulateUntilWhitespace()
	return l.err == nil
}

func (l *lexer) literal() bool {
	l.startToken(literal)
	l.cur.Value = l.accumulateUntilPred(func(b byte) bool {
		return b == '"'
	})
	l.readByte()
	return l.err == nil
}
