package parsing

import "fmt"

type Error struct {
	Context *Context
	Err     error
}

func (me Error) Error() string {
	return fmt.Sprintf("%#v", me)
}
