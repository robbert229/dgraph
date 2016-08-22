package parsing

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
	p  Parser
	Ok bool
}

func (me *Opt) Parse(c *Context) {
	me.Ok = c.TryParse(me.p)
}

func Maybe(p Parser) Opt {
	return Opt{
		p: p,
	}
}

type Repeats struct {
	min, max int
	p        Parser
	Values   []interface{}
}

func (me *Repeats) Parse(c *Context) {
	for i := 0; me.max != 0 && i < me.max; i++ {

	}
}

func Repeat(min, max int, p Parser) Repeats {
	return Repeats{
		min: min,
		max: max,
		p:   p,
	}
}

func Star(p Parser) Repeats {
	return Repeat(0, 0, p)
}
