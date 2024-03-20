package p2p

import (
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMainnetNetworkId(t *testing.T) {
	networkId := GetNetworkId(common.MainNet)
	require.Equal(t, MainnetNetworkId, networkId)
}

func TestValidateTestnetNetworkId(t *testing.T) {
	networkId := GetNetworkId(common.TestNet)
	require.Equal(t, TestnetNetworkId, networkId)
}

func TestValidateDevnetNetworkId(t *testing.T) {
	networkId := GetNetworkId(common.UnsafeDevNet)
	require.Equal(t, DevnetNetworkId, networkId)
}

func TestValidateMainnetBootstrapPeers(t *testing.T) {
	bootstrapPeers, err := GetBootstrapPeers(common.MainNet)
	require.NoError(t, err)
	require.NotEqual(t, "", bootstrapPeers)

	// Make sure we can parse the result.
	logger := zap.NewNop()
	bootStrappers, _ := BootstrapAddrs(logger, bootstrapPeers, "somePeerID")
	assert.Equal(t, 3, len(bootStrappers))
}

func TestValidateMainnetCcqBootstrapPeers(t *testing.T) {
	bootstrapPeers, err := GetCcqBootstrapPeers(common.MainNet)
	require.NoError(t, err)
	require.NotEqual(t, "", bootstrapPeers)

	// Make sure we can parse the result.
	logger := zap.NewNop()
	bootStrappers, _ := BootstrapAddrs(logger, bootstrapPeers, "somePeerID")
	assert.Equal(t, 3, len(bootStrappers))
}

func TestValidateTestnetBootstrapPeers(t *testing.T) {
	bootstrapPeers, err := GetBootstrapPeers(common.TestNet)
	require.NoError(t, err)
	require.NotEqual(t, "", bootstrapPeers)

	// Make sure we can parse the result.
	logger := zap.NewNop()
	bootStrappers, _ := BootstrapAddrs(logger, bootstrapPeers, "somePeerID")
	assert.Equal(t, 3, len(bootStrappers))
}

func TestValidateTestnetCcqBootstrapPeers(t *testing.T) {
	bootstrapPeers, err := GetCcqBootstrapPeers(common.TestNet)
	require.NoError(t, err)
	require.NotEqual(t, "", bootstrapPeers)

	// Make sure we can parse the result.
	logger := zap.NewNop()
	bootStrappers, _ := BootstrapAddrs(logger, bootstrapPeers, "somePeerID")
	assert.Equal(t, 3, len(bootStrappers))
}

func TestGetBootstrapPeersFailsForUnsupportedEnvironment(t *testing.T) {
	bootstrapPeers, err := GetBootstrapPeers(common.UnsafeDevNet)
	require.ErrorContains(t, err, "unsupported environment")
	require.Equal(t, "", bootstrapPeers)
}

func TestGetCcqBootstrapPeersFailsForUnsupportedEnvironment(t *testing.T) {
	bootstrapPeers, err := GetCcqBootstrapPeers(common.UnsafeDevNet)
	require.ErrorContains(t, err, "unsupported environment")
	require.Equal(t, "", bootstrapPeers)
}
