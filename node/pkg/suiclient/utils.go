package suiclient

import (
	mystenbcs "github.com/block-vision/sui-go-sdk/mystenbcs"
)

// DecodeBcs wraps the Bcs decoding, and returns nil if the unmarshalling failed. This is
// useful in cases where knowing the decoding failed is sufficient to proceed.
func DecodeBcs[T any](bcs []byte) *T {
	var t T
	_, err := mystenbcs.Unmarshal(bcs, &t)

	if err != nil {
		return nil
	}

	return &t
}
