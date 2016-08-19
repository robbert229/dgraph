package gql

import (
	"log"

	_ "github.com/anacrolix/envpprof"
)

func init() {
	log.SetFlags(log.Flags() | log.Lshortfile)
}
