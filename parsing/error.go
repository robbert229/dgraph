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

func (me Error) lines(depth int, parent *Context) (lines []string) {
	lines = []string{"syntax error"}
	if me.Err != nil {
		lines = append(lines, me.Err.Error())
	}
	for _, e := range me.Context.errs {
		lines = append(lines, "after")
		lines = append(lines, e.lines(depth+1, me.Context)...)
	}
	for c := me.Context; c != parent; c = c.Parent {
		if c.p != nil {
			lines = append(lines, fmt.Sprintf("while parsing %s", ParserName(c.p)))
		}
	}
	s := ""
	for range iter.N(depth * 2) {
		s += " "
	}
	for i := range lines {
		lines[i] = s + lines[i]
	}
	return
}

func (me Error) Error() string {
	return strings.Join(me.lines(0, nil), "\n")
}
