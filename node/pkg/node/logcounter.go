package node

import (
	"sync/atomic"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogSizeCounter struct {
	level zapcore.Level
	ctr   atomic.Uint64
}

func NewLogSizeCounter(lvl zapcore.Level) *LogSizeCounter {
	return &LogSizeCounter{
		level: lvl,
	}
}

func (lc *LogSizeCounter) Reset() uint64 {
	n := lc.ctr.Load()
	lc.ctr.Store(0)
	return n
}

func (lc *LogSizeCounter) Sync() error { return nil }

func (lc *LogSizeCounter) Write(p []byte) (n int, err error) {
	n = len(p)
	lc.ctr.Add(uint64(n))
	return n, nil
}

func (lc *LogSizeCounter) Core() zapcore.Core {
	var output zapcore.WriteSyncer = lc
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	priority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= lc.level
	})
	return zapcore.NewCore(encoder, output, priority)
}
