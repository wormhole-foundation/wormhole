package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEnvironment(t *testing.T) {
	type test struct {
		input  string
		output Environment
		err    bool
	}

	tests := []test{
		{input: "MainNet", output: MainNet, err: false},
		{input: "Prod", output: MainNet, err: false},

		{input: "TestNet", output: TestNet, err: false},
		{input: "test", output: TestNet, err: false},

		{input: "UnsafeDevNet", output: UnsafeDevNet, err: false},
		{input: "devnet", output: UnsafeDevNet, err: false},
		{input: "dev", output: UnsafeDevNet, err: false},

		{input: "GoTest", output: GoTest, err: false},
		{input: "unit-test", output: GoTest, err: false},

		{input: "AccountantMock", output: AccountantMock, err: false},
		{input: "accountant-mock", output: AccountantMock, err: false},

		{input: "junk", output: UnsafeDevNet, err: true},
		{input: "", output: UnsafeDevNet, err: true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			output, err := ParseEnvironment(tc.input)
			if err != nil {
				if tc.err == false {
					assert.NoError(t, err)
				}
			} else if tc.err {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tc.output, output)
			}
		})
	}
}
