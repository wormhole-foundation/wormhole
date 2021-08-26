package processor

import (
	"fmt"
	"testing"
)

func TestCalculateQuorum(t *testing.T) {
	tests := []struct {
		have int
		want int
	}{
		{have: 1, want: 1},
		{have: 2, want: 2},
		{have: 3, want: 3},
		{have: 4, want: 3},
		{have: 5, want: 4},
		{have: 6, want: 5},
		{have: 7, want: 5},
		{have: 8, want: 6},
		{have: 9, want: 7},
		{have: 10, want: 7},
		{have: 11, want: 8},
		{have: 12, want: 9},
		{have: 20, want: 14},
		{have: 25, want: 17},
		{have: 100, want: 67},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d have", tt.have), func(t *testing.T) {
			if got := CalculateQuorum(tt.have); got != tt.want {
				t.Errorf("CalculateQuorum(%d) = %v, want %v", tt.have, got, tt.want)
			}
		})
	}
}
