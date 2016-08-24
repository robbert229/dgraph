package parsing

import (
	"fmt"

	"github.com/joeshaw/gengen/generic"
)

func NewContext(s Stream) *Context {
	return &Context{
		s: s,
	}
}

type Context struct {
	errs   []Error
	s      Stream
	Parent *Context
	p      Parser
}

func (me *Context) Stream() Stream {
	return me.s
}

func (me *Context) Token() generic.T {
	if me.s.Err() != nil {
		me.Fatal(fmt.Errorf("stream error: %s", me.s.Err()))
	}
	return me.s.Token()
}

func (me *Context) Advance() {
	me.s = me.s.Next()
}

func (me *Context) Good() bool {
	return me.s.Good()
}

func (me *Context) parse(p Parser, c *Context) {
	p.Parse(c)
	// if c.s == me.s {
	// 	panic(fmt.Sprintf("operation did not advance stream: %v", ParserName(c.p)))
	// }
	me.s = c.s
	me.errs = append(me.errs, c.errs...)
	c.errs = nil
}

func (me *Context) newChild(p Parser) *Context {
	return &Context{
		s:      me.s,
		p:      p,
		Parent: me,
	}
}

func (me *Context) TryParse(ps ...Parser) bool {
	err := me.ParseErr(ps...)
	if err != nil {
		me.errs = append(me.errs, err.(Error))
		return false
	}
	return true
}

func (me *Context) Parse(ps ...Parser) {
	for _, p := range ps {
		me.parse(p, me.newChild(p))
	}
}

func recoverError(f func(Error)) {
	r := recover()
	if r == nil {
		return
	}
	se, ok := r.(Error)
	if !ok {
		panic(r)
	}
	f(se)
}

// Only returns Error.
func (me *Context) ParseErr(ps ...Parser) (err error) {
	defer recoverError(func(e Error) {
		err = e
	})
	for _, p := range ps {
		child := me.newChild(p)
		me.parse(p, child)
		if err != nil {
			break
		}
	}
	return
}

func (me *Context) Fatal(err error) {
	panic(Error{
		Context: me,
		Err:     err,
	})
}

func (me *Context) FailNow() {
	panic(Error{
		Context: me,
	})
}
