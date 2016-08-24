package parsing

import (
	"fmt"
	"strings"

	"github.com/bradfitz/iter"
)

type Error struct {
	Context *Context
	Err     error
}

func stringIndent(indent int) string {
	s := ""
	for range iter.N(indent) {
		s += " "
	}
	return s
}

func indentedLines(lines []string) (ret []string) {
	for _, l := range lines {
		ret = append(ret, "  "+l)
	}
	return
}

func (me Error) lines(parent *Context) (lines []string) {
	if me.Context.p != nil {
		lines = append(lines, fmt.Sprintf("while parsing %s at %s", ParserName(me.Context.p), me.Context.Stream().Position()))
	}
	if me.Err != nil {
		lines = append(lines, me.Err.Error())
	}
	for _, e := range me.Context.errs {
		lines = append(lines, "after")
		lines = append(lines, indentedLines(e.lines(me.Context))...)
	}
	if me.Context.Parent != parent {
		lines = append(lines, Error{Context: me.Context.Parent}.lines(parent)...)
	}
	return
}

func (me Error) Error() string {
	return strings.Join(append([]string{"syntax error"}, me.lines(nil)...), "\n")
}
