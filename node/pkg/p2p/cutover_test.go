package p2p

import (
	"testing"
	"time"

	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestVerifyCutOverTime(t *testing.T) {
	if cutOverTimeStr != "" {
		_, err := time.Parse(cutOverFmtStr, cutOverTimeStr)
		require.NoError(t, err)
	}
}

const oldBootstrapPeers = "/dns4/guardian-0.guardian/udp/8999/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw,/dns4/guardian-0.guardian/udp/8999/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jx"
const newBootstrapPeers = "/dns4/guardian-0.guardian/udp/8999/quic-v1/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw,/dns4/guardian-0.guardian/udp/8999/quic-v1/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jx"

const oldCcqBootstrapPeers = "/dns4/guardian-0.guardian/udp/8999/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw,/dns4/guardian-0.guardian/udp/8999/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jx"
const newCcqBootstrapPeers = "/dns4/guardian-0.guardian/udp/8999/quic-v1/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw,/dns4/guardian-0.guardian/udp/8999/quic-v1/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jx"

var oldComponents *Components = &Components{
	P2PIDInHeartbeat: true,
	ListeningAddressesPatterns: []string{
		// Listen on QUIC only.
		// https://github.com/libp2p/go-libp2p/issues/688
		"/ip4/0.0.0.0/udp/%d/quic",
		"/ip6/::/udp/%d/quic",
	},
	Port:                       DefaultPort,
	ConnMgr:                    oldConnMgr,
	ProtectedHostByGuardianKey: make(map[eth_common.Address]peer.ID),
	SignedHeartbeatLogLevel:    zapcore.DebugLevel,
}

var newComponents *Components = &Components{
	P2PIDInHeartbeat: true,
	ListeningAddressesPatterns: []string{
		// Listen on QUIC only.
		// https://github.com/libp2p/go-libp2p/issues/688
		"/ip4/0.0.0.0/udp/%d/quic-v1",
		"/ip6/::/udp/%d/quic-v1",
	},
	Port:                       DefaultPort,
	ConnMgr:                    oldConnMgr,
	ProtectedHostByGuardianKey: make(map[eth_common.Address]peer.ID),
	SignedHeartbeatLogLevel:    zapcore.DebugLevel,
}

var oldConnMgr *connmgr.BasicConnMgr = func() *connmgr.BasicConnMgr {
	cm, err := DefaultConnectionManager()
	if err != nil {
		panic("failed to create connection manager")
	}
	return cm
}()

func (components *Components) copy() *Components {
	laps := make([]string, len(components.ListeningAddressesPatterns))
	copy(laps, components.ListeningAddressesPatterns)
	phbgk := map[eth_common.Address]peer.ID{}
	for k, v := range components.ProtectedHostByGuardianKey {
		phbgk[k] = v
	}
	return &Components{
		P2PIDInHeartbeat:           components.P2PIDInHeartbeat,
		ListeningAddressesPatterns: laps,
		Port:                       components.Port,
		ConnMgr:                    components.ConnMgr,
		ProtectedHostByGuardianKey: phbgk,
		SignedHeartbeatLogLevel:    components.SignedHeartbeatLogLevel,
	}
}

func TestCutOverDisabled(t *testing.T) {
	logger := zap.NewNop()

	cutOverTimeStr := ""
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T18:19:00-0000")
	require.NoError(t, err)

	updComponents := oldComponents.copy()
	require.Equal(t, oldComponents, updComponents)

	updBootstrapPeers, updCcqBootstrapPeers, delay, err := checkForCutOverImpl(logger, oldBootstrapPeers, oldCcqBootstrapPeers, updComponents, cutOverTimeStr, now)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), delay)
	assert.Equal(t, oldBootstrapPeers, updBootstrapPeers)
	assert.Equal(t, oldCcqBootstrapPeers, updCcqBootstrapPeers)
	assert.Equal(t, oldComponents, updComponents)
}

func TestCutOverCcqDisabled(t *testing.T) {
	logger := zap.NewNop()

	cutOverTimeStr := "2023-10-06T18:18:00-0000"
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T18:19:00-0000")
	require.NoError(t, err)

	updComponents := oldComponents.copy()
	require.Equal(t, oldComponents, updComponents)

	updBootstrapPeers, updCcqBootstrapPeers, delay, err := checkForCutOverImpl(logger, oldBootstrapPeers, "", updComponents, cutOverTimeStr, now)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), delay)
	assert.Equal(t, newBootstrapPeers, updBootstrapPeers)
	assert.Equal(t, "", updCcqBootstrapPeers)
	assert.Equal(t, newComponents, updComponents)
}

func TestCutOverUnexpectedBootstrapPeers(t *testing.T) {
	logger := zap.NewNop()
	badBootstrapPeers := "HelloWorld"

	cutOverTimeStr := "2023-10-06T18:18:00-0000"
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T18:19:00-0000")
	require.NoError(t, err)

	updComponents := oldComponents.copy()

	_, _, _, err = checkForCutOverImpl(logger, badBootstrapPeers, oldCcqBootstrapPeers, updComponents, cutOverTimeStr, now)
	require.EqualError(t, err, `unexpected format of bootstrap peers: unexpected format, does not contain "quic"`)
}

func TestCutOverUnexpectedCcqBootstrapPeers(t *testing.T) {
	logger := zap.NewNop()
	badCcqBootstrapPeers := "HelloWorld"

	cutOverTimeStr := "2023-10-06T18:18:00-0000"
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T18:19:00-0000")
	require.NoError(t, err)

	updComponents := oldComponents.copy()

	_, _, _, err = checkForCutOverImpl(logger, oldBootstrapPeers, badCcqBootstrapPeers, updComponents, cutOverTimeStr, now)
	require.EqualError(t, err, `unexpected format of ccq bootstrap peers: unexpected format, does not contain "quic"`)
}

func TestCutOverInvalidTime(t *testing.T) {
	logger := zap.NewNop()

	cutOverTimeStr := "Hello World"
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T18:19:00-0000")
	require.NoError(t, err)

	updComponents := oldComponents.copy()

	_, _, _, err = checkForCutOverImpl(logger, oldBootstrapPeers, oldCcqBootstrapPeers, updComponents, cutOverTimeStr, now)
	require.EqualError(t, err, `failed to parse cut over time: parsing time "Hello World" as "2006-01-02T15:04:05-0700": cannot parse "Hello World" as "2006"`)
}

func TestCutOverAlreadyHappened(t *testing.T) {
	logger := zap.NewNop()

	cutOverTimeStr := "2023-10-06T18:18:00-0000"
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T18:19:00-0000")
	require.NoError(t, err)

	updComponents := oldComponents.copy()

	updBootstrapPeers, updCcqBootstrapPeers, delay, err := checkForCutOverImpl(logger, oldBootstrapPeers, oldCcqBootstrapPeers, updComponents, cutOverTimeStr, now)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), delay)
	assert.Equal(t, newBootstrapPeers, updBootstrapPeers)
	assert.Equal(t, newCcqBootstrapPeers, updCcqBootstrapPeers)
	assert.Equal(t, newComponents, updComponents)
}

func TestCutOverBootstrapPeersAlreadyUpdated(t *testing.T) {
	logger := zap.NewNop()

	cutOverTimeStr := "2023-10-06T18:18:00-0000"
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T18:19:00-0000")
	require.NoError(t, err)

	updComponents := newComponents.copy()

	updBootstrapPeers, updCcqBootstrapPeers, delay, err := checkForCutOverImpl(logger, newBootstrapPeers, newCcqBootstrapPeers, updComponents, cutOverTimeStr, now)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), delay)
	assert.Equal(t, newBootstrapPeers, updBootstrapPeers)
	assert.Equal(t, newCcqBootstrapPeers, updCcqBootstrapPeers)
	assert.Equal(t, newComponents, updComponents)
}

func TestCutOverDelayRequired(t *testing.T) {
	logger := zap.NewNop()

	cutOverTimeStr := "2023-10-06T18:18:00-0000"
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T17:18:00-0000")
	require.NoError(t, err)

	updComponents := oldComponents.copy()

	updBootstrapPeers, updCcqBootstrapPeers, delay, err := checkForCutOverImpl(logger, oldBootstrapPeers, oldCcqBootstrapPeers, updComponents, cutOverTimeStr, now)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(60*time.Minute), delay)
	assert.Equal(t, oldBootstrapPeers, updBootstrapPeers)
	assert.Equal(t, oldCcqBootstrapPeers, updCcqBootstrapPeers)
	assert.Equal(t, oldComponents, updComponents)
}

func TestCutOverBootstrapPeersUpdatedButComponentsNot(t *testing.T) {
	logger := zap.NewNop()

	cutOverTimeStr := "2023-10-06T18:18:00-0000"
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T18:19:00-0000")
	require.NoError(t, err)

	updComponents := newComponents.copy()
	updComponents.ListeningAddressesPatterns = make([]string, len(oldComponents.ListeningAddressesPatterns))
	copy(updComponents.ListeningAddressesPatterns, oldComponents.ListeningAddressesPatterns)

	_, _, _, err = checkForCutOverImpl(logger, newBootstrapPeers, newCcqBootstrapPeers, updComponents, cutOverTimeStr, now)
	require.EqualError(t, err, "bootstrapPeers has been updated to quic-v1, but components.ListeningAddressesPatterns has not: /ip4/0.0.0.0/udp/%d/quic")
}

func TestCutOverBootstrapPeersUpdatedButCcqNot(t *testing.T) {
	logger := zap.NewNop()

	cutOverTimeStr := "2023-10-06T18:18:00-0000"
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T18:19:00-0000")
	require.NoError(t, err)

	updComponents := newComponents.copy()

	_, _, _, err = checkForCutOverImpl(logger, newBootstrapPeers, oldCcqBootstrapPeers, updComponents, cutOverTimeStr, now)
	require.EqualError(t, err, "quic version mismatch between bootstrap peers and ccq bootstrap peers")
}
