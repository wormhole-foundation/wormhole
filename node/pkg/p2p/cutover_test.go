package p2p

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// We want to be able to test the cutover conversion stuff so force us into cutover mode.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestCutOverBootstrapAddrs(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	bootstrappers, isBootstrapNode := BootstrapAddrs(logger, oldBootstrapPeers, "12D3KooWHHzSeKaY8xuZVzkLbKFfvNgPPeKhFBGrMbNzbm5akpqu")
	assert.Equal(t, 2, len(bootstrappers))
	assert.False(t, isBootstrapNode)
	for _, ba := range bootstrappers {
		assert.True(t, strings.Contains(ba.String(), "/quic-v1"))
	}
}

func TestCutOverListeningAddresses(t *testing.T) {
	components := DefaultComponents()

	las := components.ListeningAddresses()
	require.Equal(t, len(components.ListeningAddressesPatterns), len(las))
	for _, la := range las {
		assert.True(t, strings.Contains(la, "/quic-v1"))
	}
}

const oldBootstrapPeers = "/dns4/guardian-0.guardian/udp/8999/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw,/dns4/guardian-0.guardian/udp/8999/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jx"
