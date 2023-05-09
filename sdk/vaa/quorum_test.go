package vaa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateQuorum(t *testing.T) {
	type Test struct {
		numGuardians int
		quorumResult int
		shouldPanic  bool
	}

	tests := []Test{
		// Positive Test Cases
		{numGuardians: 0, quorumResult: 1},
		{numGuardians: 1, quorumResult: 1},
		{numGuardians: 2, quorumResult: 2},
		{numGuardians: 3, quorumResult: 3},
		{numGuardians: 4, quorumResult: 3},
		{numGuardians: 5, quorumResult: 4},
		{numGuardians: 6, quorumResult: 5},
		{numGuardians: 7, quorumResult: 5},
		{numGuardians: 8, quorumResult: 6},
		{numGuardians: 9, quorumResult: 7},
		{numGuardians: 10, quorumResult: 7},
		{numGuardians: 11, quorumResult: 8},
		{numGuardians: 12, quorumResult: 9},
		{numGuardians: 13, quorumResult: 9},
		{numGuardians: 14, quorumResult: 10},
		{numGuardians: 15, quorumResult: 11},
		{numGuardians: 16, quorumResult: 11},
		{numGuardians: 17, quorumResult: 12},
		{numGuardians: 18, quorumResult: 13},
		{numGuardians: 19, quorumResult: 13},
		{numGuardians: 50, quorumResult: 34},
		{numGuardians: 100, quorumResult: 67},
		{numGuardians: 1000, quorumResult: 667},

		// Negative Test Cases
		{numGuardians: -1, quorumResult: 1, shouldPanic: true},
		{numGuardians: -1000, quorumResult: 1, shouldPanic: true},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			if tc.shouldPanic {
				assert.Panics(t, func() { CalculateQuorum(tc.numGuardians) }, "The code did not panic")
			} else {
				num := CalculateQuorum(tc.numGuardians)
				assert.Equal(t, tc.quorumResult, num)
			}
		})
	}
}
