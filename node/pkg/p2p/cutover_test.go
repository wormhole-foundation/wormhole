package p2p

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestVerifyCutOverTime(t *testing.T) {
	if mainnetCutOverTimeStr != "" {
		_, err := time.Parse(cutOverFmtStr, mainnetCutOverTimeStr)
		require.NoError(t, err)
	}
	if testnetCutOverTimeStr != "" {
		_, err := time.Parse(cutOverFmtStr, testnetCutOverTimeStr)
		require.NoError(t, err)
	}
	if devnetCutOverTimeStr != "" {
		_, err := time.Parse(cutOverFmtStr, devnetCutOverTimeStr)
		require.NoError(t, err)
	}
}

const oldBootstrapPeers = "/dns4/guardian-0.guardian/udp/8999/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw,/dns4/guardian-0.guardian/udp/8999/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jx"
const newBootstrapPeers = "/dns4/guardian-0.guardian/udp/8999/quic-v1/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw,/dns4/guardian-0.guardian/udp/8999/quic-v1/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jx"

func TestGetCutOverTimeStr(t *testing.T) {
	assert.Equal(t, mainnetCutOverTimeStr, getCutOverTimeStr("blah/blah/mainnet/blah"))
	assert.Equal(t, testnetCutOverTimeStr, getCutOverTimeStr("blah/blah/testnet/blah"))
	assert.Equal(t, devnetCutOverTimeStr, getCutOverTimeStr("blah/blah/devnet/blah"))
}

func TestValidateBootstrapPeers(t *testing.T) {
	assert.NoError(t, validateBootstrapPeers(oldBootstrapPeers)) // All old quic should work.
	assert.NoError(t, validateBootstrapPeers(newBootstrapPeers)) // All new quic-v1 should work.

	// No quics should fail.
	err := validateBootstrapPeers("HelloWorld")
	assert.EqualError(t, err, `unexpected format, does not contain "/quic/" or  "/quic-v1/"`)

	// A mix of quics should fail.
	err = validateBootstrapPeers("/quic/quic-v1/")
	assert.EqualError(t, err, `unexpected format, contains both "/quic/" or  "/quic-v1/"`)
}

func TestCutOverDisabled(t *testing.T) {
	logger := zap.NewNop()

	cutOverTimeStr := ""
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T18:19:00-0000")
	require.NoError(t, err)

	cuttingOver, delay, err := evaluateCutOverImpl(logger, cutOverTimeStr, now)
	require.NoError(t, err)
	assert.False(t, cuttingOver)
	assert.Equal(t, time.Duration(0), delay)
}

func TestCutOverInvalidTime(t *testing.T) {
	logger := zap.NewNop()

	cutOverTimeStr := "Hello World"
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T18:19:00-0000")
	require.NoError(t, err)

	_, _, err = evaluateCutOverImpl(logger, cutOverTimeStr, now)
	require.EqualError(t, err, `failed to parse cut over time: parsing time "Hello World" as "2006-01-02T15:04:05-0700": cannot parse "Hello World" as "2006"`)
}

func TestCutOverAlreadyHappened(t *testing.T) {
	logger := zap.NewNop()

	cutOverTimeStr := "2023-10-06T18:18:00-0000"
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T18:19:00-0000")
	require.NoError(t, err)

	cuttingOver, delay, err := evaluateCutOverImpl(logger, cutOverTimeStr, now)
	require.NoError(t, err)
	assert.True(t, cuttingOver)
	assert.Equal(t, time.Duration(0), delay)
}

func TestCutOverDelayRequired(t *testing.T) {
	logger := zap.NewNop()

	cutOverTimeStr := "2023-10-06T18:18:00-0000"
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T17:18:00-0000")
	require.NoError(t, err)

	cuttingOver, delay, err := evaluateCutOverImpl(logger, cutOverTimeStr, now)
	require.NoError(t, err)
	assert.False(t, cuttingOver)
	assert.Equal(t, time.Duration(60*time.Minute), delay)
}
