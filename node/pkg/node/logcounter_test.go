package node

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLogSizeCounterCountsAndResetsBytes(t *testing.T) {
	t.Parallel()

	counter := NewLogSizeCounter(zapcore.WarnLevel)
	n, err := counter.Write([]byte("abc"))
	require.NoError(t, err)
	require.Equal(t, 3, n)
	require.NoError(t, counter.Sync())
	require.EqualValues(t, 3, counter.Reset())
	require.EqualValues(t, 0, counter.Reset())

	logger := zap.New(counter.Core())
	logger.Info("ignored")
	require.EqualValues(t, 0, counter.Reset())
	logger.Warn("counted")
	require.Greater(t, counter.Reset(), uint64(0))
}
