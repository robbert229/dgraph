package parsing

import "fmt"

type Seq []Parser

func (me Seq) Parse(c *Context) {
	for _, p := range me {
		c.Parse(p)
	}
}

type Which struct {
	ps    []Parser
	Index int
}

func OneOf(ps ...Parser) *Which {
	return &Which{
		ps: ps,
	}
}

func (me *Which) Parse(c *Context) {
	for i, p := range me.ps {
		if c.TryParse(p) {
			me.Index = i
			return
		}
	}
	c.FailNow()
}

type Opt struct {
	ps []Parser
	Ok bool
}

func (me *Opt) Parse(c *Context) {
	me.Ok = c.TryParse(me.ps...)
}

func Maybe(ps ...Parser) *Opt {
	return &Opt{
		ps: ps,
	}
}

type Repeats struct {
	min, max int
	p        Parser
	// Values   []interface{}
}

func (me *Repeats) Parse(c *Context) {
	for i := 0; me.max == 0 || i < me.max; i++ {
		s := c.Stream()
		if !c.TryParse(me.p) {
			if i < me.min {
				c.Fatal(fmt.Errorf("got %d repetitions, minimum is %d", i+1, me.min))
			}
			return
		}
		if me.max == 0 && c.Stream() == s {
			panic("no advance")
		}
	}
}

func Repeat(min, max int, p Parser) *Repeats {
	return &Repeats{
		min: min,
		max: max,
		p:   p,
	}
}

func Star(p Parser) *Repeats {
	return Repeat(0, 0, p)
}

func Plus(p Parser) *Repeats {
	return Repeat(1, 0, p)
}
