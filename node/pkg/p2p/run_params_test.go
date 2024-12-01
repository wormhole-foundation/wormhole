package p2p

import (
	"context"
	"testing"

	"github.com/certusone/wormhole/node/pkg/accountant"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	p2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const bootstrapPeers = "/dns4/guardian-0.guardian/udp/8999/quic/p2p/12D3KooWHHzSeKaY8xuZVzkLbKFfvNgPPeKhFBGrMbNzbm5akpqu"
const networkId = "/wormhole/dev"
const nodeName = "guardian-0"

func TestRunParamsBootstrapPeersRequired(t *testing.T) {
	priv, _, err := p2pcrypto.GenerateKeyPair(p2pcrypto.Ed25519, -1)
	require.NoError(t, err)
	gst := common.NewGuardianSetState(nil)
	_, rootCtxCancel := context.WithCancel(context.Background())
	defer rootCtxCancel()

	params, err := NewRunParams(
		"", // bootstrapPeers,
		networkId,
		priv,
		gst,
		rootCtxCancel,
	)
	require.ErrorContains(t, err, "bootstrapPeers may not be nil")
	require.Nil(t, params)
}

func TestRunParamsNetworkIdRequired(t *testing.T) {
	priv, _, err := p2pcrypto.GenerateKeyPair(p2pcrypto.Ed25519, -1)
	require.NoError(t, err)
	gst := common.NewGuardianSetState(nil)
	_, rootCtxCancel := context.WithCancel(context.Background())
	defer rootCtxCancel()

	params, err := NewRunParams(
		bootstrapPeers,
		"", // networkId,
		priv,
		gst,
		rootCtxCancel,
	)
	require.ErrorContains(t, err, "networkID may not be nil")
	require.Nil(t, params)
}

func TestRunParamsPrivRequired(t *testing.T) {
	gst := common.NewGuardianSetState(nil)
	_, rootCtxCancel := context.WithCancel(context.Background())
	defer rootCtxCancel()

	params, err := NewRunParams(
		bootstrapPeers,
		networkId,
		nil, // priv,
		gst,
		rootCtxCancel,
	)
	require.ErrorContains(t, err, "priv may not be nil")
	require.Nil(t, params)
}

func TestRunParamsGstRequired(t *testing.T) {
	priv, _, err := p2pcrypto.GenerateKeyPair(p2pcrypto.Ed25519, -1)
	require.NoError(t, err)
	_, rootCtxCancel := context.WithCancel(context.Background())
	defer rootCtxCancel()

	params, err := NewRunParams(
		bootstrapPeers,
		networkId,
		priv,
		nil, // gst,
		rootCtxCancel,
	)
	require.ErrorContains(t, err, "gst may not be nil")
	require.Nil(t, params)
}

func TestRunParamsRootCtxCancelRequired(t *testing.T) {
	priv, _, err := p2pcrypto.GenerateKeyPair(p2pcrypto.Ed25519, -1)
	require.NoError(t, err)
	gst := common.NewGuardianSetState(nil)
	_, rootCtxCancel := context.WithCancel(context.Background())
	defer rootCtxCancel()

	params, err := NewRunParams(
		bootstrapPeers,
		networkId,
		priv,
		gst,
		nil, // rootCtxCancel,
	)
	require.ErrorContains(t, err, "rootCtxCancel may not be nil")
	require.Nil(t, params)
}

func TestRunParamsWithDisableHeartbeatVerify(t *testing.T) {
	priv, _, err := p2pcrypto.GenerateKeyPair(p2pcrypto.Ed25519, -1)
	require.NoError(t, err)
	gst := common.NewGuardianSetState(nil)
	_, rootCtxCancel := context.WithCancel(context.Background())
	defer rootCtxCancel()

	params, err := NewRunParams(
		bootstrapPeers,
		networkId,
		priv,
		gst,
		rootCtxCancel,
		WithDisableHeartbeatVerify(true),
	)

	require.NoError(t, err)
	require.NotNil(t, params)
	assert.True(t, params.disableHeartbeatVerify)
}

func TestRunParamsWithGuardianOptions(t *testing.T) {
	priv, _, err := p2pcrypto.GenerateKeyPair(p2pcrypto.Ed25519, -1)
	require.NoError(t, err)
	gst := common.NewGuardianSetState(nil)
	_, rootCtxCancel := context.WithCancel(context.Background())
	defer rootCtxCancel()

	guardianSigner, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(nil)
	require.NoError(t, err)
	require.NotNil(t, guardianSigner)

	obsvC := make(chan<- *common.MsgWithTimeStamp[gossipv1.SignedObservation], 42)
	batchObsvC := make(chan<- *common.MsgWithTimeStamp[gossipv1.SignedObservationBatch], 42)
	signedInC := make(chan<- *gossipv1.SignedVAAWithQuorum, 42)
	obsvReqC := make(chan<- *gossipv1.ObservationRequest, 42)
	gossipControlSendC := make(chan []byte, 42)
	gossipAttestationSendC := make(chan []byte, 42)
	gossipVaaSendC := make(chan []byte, 42)
	obsvReqSendC := make(<-chan *gossipv1.ObservationRequest, 42)

	acct := &accountant.Accountant{}
	gov := &governor.ChainGovernor{}
	disableHeartbeatVerify := false
	components := &Components{}
	ibcFeaturesFunc := func() string { return "Hello, World!" }
	gatewayRelayerEnabled := true

	ccqEnabled := true
	signedQueryReqC := make(chan<- *gossipv1.SignedQueryRequest, 42)
	queryResponseReadC := make(<-chan *query.QueryResponsePublication, 42)
	ccqBootstrapPeers := "some bootstrap string"
	ccqPort := uint(4242)
	ccqAllowedPeers := "some allowed peers"

	params, err := NewRunParams(
		bootstrapPeers,
		networkId,
		priv,
		gst,
		rootCtxCancel,
		WithGuardianOptions(
			nodeName,
			guardianSigner,
			obsvC,
			batchObsvC,
			signedInC,
			obsvReqC,
			gossipControlSendC,
			gossipAttestationSendC,
			gossipVaaSendC,
			obsvReqSendC,
			acct,
			gov,
			disableHeartbeatVerify,
			components,
			ibcFeaturesFunc,
			gatewayRelayerEnabled,
			ccqEnabled,
			signedQueryReqC,
			queryResponseReadC,
			ccqBootstrapPeers,
			ccqPort,
			ccqAllowedPeers,
		),
	)

	require.NoError(t, err)
	require.NotNil(t, params)
	assert.Equal(t, nodeName, params.nodeName)
	assert.Equal(t, obsvC, params.obsvRecvC)
	assert.Equal(t, signedInC, params.signedIncomingVaaRecvC)
	assert.Equal(t, obsvReqC, params.obsvReqRecvC)
	assert.Equal(t, gossipControlSendC, params.gossipControlSendC)
	assert.Equal(t, gossipAttestationSendC, params.gossipAttestationSendC)
	assert.Equal(t, gossipVaaSendC, params.gossipVaaSendC)
	assert.Equal(t, obsvReqSendC, params.obsvReqSendC)
	assert.Equal(t, acct, params.acct)
	assert.Equal(t, gov, params.gov)
	assert.Equal(t, components, params.components)
	assert.NotNil(t, params.ibcFeaturesFunc) // Can't compare function pointers, so just verify it's set.
	assert.True(t, params.gatewayRelayerEnabled)
	assert.True(t, params.ccqEnabled)
	assert.Equal(t, signedQueryReqC, params.signedQueryReqC)
	assert.Equal(t, queryResponseReadC, params.queryResponseReadC)
	assert.Equal(t, ccqBootstrapPeers, params.ccqBootstrapPeers)
	assert.Equal(t, ccqPort, params.ccqPort)
	assert.Equal(t, ccqAllowedPeers, params.ccqAllowedPeers)
}
