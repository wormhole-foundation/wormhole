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

func (f FinalityLevel) String() string {
	if f == Latest {
		return "Latest"
	}
	if f == Safe {
		return "Safe"
	}
	if f == Finalized {
		return "Finalized"
	}
	return fmt.Sprintf("unknown(%d)", f)
}
