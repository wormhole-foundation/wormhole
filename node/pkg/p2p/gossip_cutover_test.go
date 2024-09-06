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

func TestGetCutOverTimeStr(t *testing.T) {
	assert.Equal(t, mainnetCutOverTimeStr, getCutOverTimeStr("blah/blah/mainnet/blah"))
	assert.Equal(t, testnetCutOverTimeStr, getCutOverTimeStr("blah/blah/testnet/blah"))
	assert.Equal(t, devnetCutOverTimeStr, getCutOverTimeStr("blah/blah/devnet/blah"))
}

func TestCutOverDisabled(t *testing.T) {
	logger := zap.NewNop()

	cutOverTimeStr := ""
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T18:19:00-0000")
	require.NoError(t, err)

	cuttingOver, delay, err := evaluateGossipCutOverImpl(logger, cutOverTimeStr, now)
	require.NoError(t, err)
	assert.False(t, cuttingOver)
	assert.Equal(t, time.Duration(0), delay)
}

func TestCutOverInvalidTime(t *testing.T) {
	logger := zap.NewNop()

	cutOverTimeStr := "Hello World"
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T18:19:00-0000")
	require.NoError(t, err)

	_, _, err = evaluateGossipCutOverImpl(logger, cutOverTimeStr, now)
	require.EqualError(t, err, `failed to parse cut over time: parsing time "Hello World" as "2006-01-02T15:04:05-0700": cannot parse "Hello World" as "2006"`)
}

func TestCutOverAlreadyHappened(t *testing.T) {
	logger := zap.NewNop()

	cutOverTimeStr := "2023-10-06T18:18:00-0000"
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T18:19:00-0000")
	require.NoError(t, err)

	cuttingOver, delay, err := evaluateGossipCutOverImpl(logger, cutOverTimeStr, now)
	require.NoError(t, err)
	assert.True(t, cuttingOver)
	assert.Equal(t, time.Duration(0), delay)
}

func TestCutOverDelayRequired(t *testing.T) {
	logger := zap.NewNop()

	cutOverTimeStr := "2023-10-06T18:18:00-0000"
	now, err := time.Parse(cutOverFmtStr, "2023-10-06T17:18:00-0000")
	require.NoError(t, err)

	cuttingOver, delay, err := evaluateGossipCutOverImpl(logger, cutOverTimeStr, now)
	require.NoError(t, err)
	assert.False(t, cuttingOver)
	assert.Equal(t, time.Duration(60*time.Minute), delay)
}
