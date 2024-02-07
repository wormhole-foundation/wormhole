package common

import (
	"testing"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/stretchr/testify/assert"
)

func TestMustRegisterReadinessSyncing(t *testing.T) {
	// The first time should work.
	assert.NotPanics(t, func() {
		MustRegisterReadinessSyncing(vaa.ChainIDEthereum)
	})

	// A second time should panic.
	assert.Panics(t, func() {
		MustRegisterReadinessSyncing(vaa.ChainIDEthereum)
	})

	// An invalid chainID should panic.
	assert.Panics(t, func() {
		MustRegisterReadinessSyncing(vaa.ChainIDUnset)
	})
}
