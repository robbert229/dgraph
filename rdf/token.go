package rdf

import "fmt"

type TokenType int

const (
	iriRef TokenType = iota + 1
	bnLabel
	literal
	period
	langTag
	doubleHat
)

func (me TokenType) String() string {
	switch me {
	case iriRef:
		return "IRIREF"
	case bnLabel:
		return "BLANK_NODE_LABEL"
	case literal:
		return "literal"
	case period:
		return "."
	case langTag:
		return "LANGTAG"
	default:
		panic(me)
	}
}

type Token struct {
	Type   TokenType
	Value  string
	Origin Location
}

type Location struct {
	Line, Col int
}

func (l Location) String() string {
	return fmt.Sprintf("%d:%d", l.Line, l.Col)
}
