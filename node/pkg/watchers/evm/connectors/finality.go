package connectors

import (
	"fmt"
)

type FinalityLevel uint8

const (
	Latest FinalityLevel = iota
	Safe
	Finalized
)

// String() formats the finality as a string. Note that these strings are used in the RPC calls, so they cannot be changed.
func (f FinalityLevel) String() string {
	if f == Latest {
		return "latest"
	}
	if f == Safe {
		return "safe"
	}
	if f == Finalized {
		return "finalized"
	}
	return fmt.Sprintf("unknown(%d)", f)
}
