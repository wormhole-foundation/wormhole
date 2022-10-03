package near

import (
	"testing"

	"github.com/test-go/testify/assert"
)

func TestSuccessValueToInt(t *testing.T) {

	type test struct {
		input  string
		output int
	}

	testsPositive := []test{
		{"MjU=", 25},
		{"MjQ4", 248},
	}

	testsNegative := []test{
		{"", 0},
		{"?", 0},
		{"MjQ4=", 0},
		{"eAo=", 0},
		{"Cg==", 0},
	}

	for _, tc := range testsPositive {
		t.Run(tc.input, func(t *testing.T) {
			i, err := successValueToInt(tc.input)
			assert.Equal(t, tc.output, i)
			assert.NoError(t, err)
		})
	}

	for _, tc := range testsNegative {
		t.Run(tc.input, func(t *testing.T) {
			i, err := successValueToInt(tc.input)
			assert.Equal(t, tc.output, i)
			assert.NotNil(t, err)
		})
	}
}
