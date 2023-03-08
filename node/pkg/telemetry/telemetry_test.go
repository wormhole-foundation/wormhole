package telemetry

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestTelemetryWithPrivate(t *testing.T) {
	// setup
	logger, _ := zap.NewDevelopment()
	var mockEventCounter atomic.Int64

	externalLogger := &ExternalLoggerMock{mockEventCounter: &mockEventCounter}
	tm, err := New(true, externalLogger)
	if err != nil {
		logger.Fatal("Failed to initialize telemetry", zap.Error(err))
	}
	defer tm.Close()
	logger = tm.WrapLogger(logger)

	// test a single private log entry
	logger.Log(zap.InfoLevel, "Single private log", zap.Bool("_privateLogEntry", true))

	// test a private logger
	loggerPrivate := logger.With(zap.Bool("_privateLogEntry", true))
	loggerPrivate.Log(zap.InfoLevel, "Private logger message 1")
	loggerPrivate.Log(zap.InfoLevel, "Private logger message 2")
	assert.Equal(t, int64(0), mockEventCounter.Load())

	// test logging in a child logger
	logger2 := logger.With(zap.String("child", "logger"))
	logger2.Log(zap.InfoLevel, "hi")
	assert.Equal(t, int64(1), mockEventCounter.Load())
}

func TestTelemetryWithOutPrivate(t *testing.T) {
	// setup
	logger, _ := zap.NewDevelopment()
	var mockEventCounter atomic.Int64

	externalLogger := &ExternalLoggerMock{mockEventCounter: &mockEventCounter}
	tm, err := New(false, externalLogger)
	if err != nil {
		logger.Fatal("Failed to initialize telemetry", zap.Error(err))
	}
	defer tm.Close()
	logger = tm.WrapLogger(logger)

	// test a single private log entry
	logger.Log(zap.InfoLevel, "Single private log", zap.Bool("_privateLogEntry", true))

	// test a private logger
	loggerPrivate := logger.With(zap.Bool("_privateLogEntry", true))
	loggerPrivate.Log(zap.InfoLevel, "Private logger message 1")
	loggerPrivate.Log(zap.InfoLevel, "Private logger message 2")
	assert.Equal(t, int64(3), mockEventCounter.Load())

	// test logging in a child logger
	logger2 := logger.With(zap.String("child", "logger"))
	logger2.Log(zap.InfoLevel, "hi")
	assert.Equal(t, int64(4), mockEventCounter.Load())
}
