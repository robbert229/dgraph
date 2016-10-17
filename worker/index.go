package worker

import (
	"golang.org/x/net/context" // Need this for the time being.

	"github.com/dgraph-io/dgraph/index"
	"github.com/dgraph-io/dgraph/x"
)

func init() {
	x.AddInit(func() {
		go processIndexMutations()
	})
}

func printMutations(m x.Mutations) {
	x.Printf("~~~")
	if len(m.Set) > 0 {
		x.Printf("SET: ")
		a := m.Set[0]
		x.Printf("[%s] [%s] [%d] [%s]", a.Attribute, string(a.Value), a.ValueId, string(a.Key))
	} else {
		x.Printf("DEL: ")
		a := m.Del[0]
		x.Printf("[%s] [%s] [%d] [%s]", a.Attribute, string(a.Value), a.ValueId, string(a.Key))
	}
}

func processIndexMutations() {
	ctx := context.Background()
	for m := range index.MutateChan {
		printMutations(m)
		err := MutateOverNetwork(ctx, m)
		if err != nil {
			x.Printf("%+v", err)
		}
	}
}
