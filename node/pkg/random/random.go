package random

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

// ErrFailedToGenerate is returned when random number generation fails
type ErrFailedToGenerate struct {
	Op  string // Operation that failed
	Err error  // Underlying error
}

// Error implements the error interface
func (e *ErrFailedToGenerate) Error() string {
	return fmt.Sprintf("failed to generate random %s: %v", e.Op, e.Err)
}

func Uint32() (uint32, error) {
	var n uint32
	err := binary.Read(rand.Reader, binary.BigEndian, &n)
	if err != nil {
		return 0, &ErrFailedToGenerate{
			Op:  "uint32",
			Err: err,
		}
	}
	return n, nil
}

func Uint64() (uint64, error) {
	var n uint64
	err := binary.Read(rand.Reader, binary.BigEndian, &n)
	if err != nil {
		return 0, &ErrFailedToGenerate{
			Op:  "uint64",
			Err: err,
		}
	}
	return n, nil
}
