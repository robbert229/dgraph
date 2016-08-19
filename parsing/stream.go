package parsing

import "github.com/joeshaw/gengen/generic"

// Represents a stream of tokens of some kind. (Not necessarily just bytes).
// Can be type-specialized with gengen, but usable without it.
type Stream interface {
	// The current token.
	Token() generic.T
	// Return a stream pointing at the next token.
	Next() Stream
	// Any stream error that occurred at the current position.
	Err() error
	// Return whether there is a token available in the current position. Kind
	// of just Err() != nil for now.
	Good() bool
	// Returns the source location of the current token, or error.
	Position() interface{}
}
