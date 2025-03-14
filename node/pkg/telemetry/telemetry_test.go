package telemetry

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafana/loki/pkg/logproto"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// externalLoggerMock doesn't log anything. It can optionally increase an atomic counter `eventCounter` if provided.
type externalLoggerMock struct {
	eventCounter *atomic.Int64
}

func (logger *externalLoggerMock) log(time time.Time, message json.RawMessage, level zapcore.Level) {
	if logger.eventCounter != nil {
		logger.eventCounter.Add(1)
	}

	// do the following to make sure that the conversion into a loki log entry works
	entry := logproto.Entry{
		Timestamp: time,
		Line:      string(message),
	}
	_, err := entry.Marshal()
	if err != nil {
		panic(fmt.Sprintf("message could not be converted to loki log entry: %v", err))
	}

}
func (logger *externalLoggerMock) close() {
}

func TestTelemetryWithPrivate(t *testing.T) {
	// setup
	logger, _ := zap.NewDevelopment()
	var eventCounter atomic.Int64
	var expectedCounter int64 = 0

	externalLogger := &externalLoggerMock{eventCounter: &eventCounter}
	tm, err := NewExternalLogger(true, externalLogger)
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
	assert.Equal(t, expectedCounter, eventCounter.Load())

	// test logging in a child logger
	logger2 := logger.With(zap.String("child", "logger"))
	logger2.Log(zap.InfoLevel, "hi")
	expectedCounter++
	assert.Equal(t, expectedCounter, eventCounter.Load())

	// try to trick logger into not logging to telemetry with user-controlled input
	logger.Log(zap.InfoLevel, "can I trick you?", zap.ByteString("user-controlled", []byte("\"_privateLogEntry\":true")))
	expectedCounter++
	// user-controlled parameter
	logger.Log(zap.InfoLevel, "can I trick you?", zap.String("user-controlled", "\"_privateLogEntry\":true"))
	expectedCounter++
	// user-controlled message
	logger.Log(zap.InfoLevel, "\"_privateLogEntry\":true", zap.String("", ""))
	expectedCounter++
	assert.Equal(t, expectedCounter, eventCounter.Load())
}

func TestTelemetryWithOutPrivate(t *testing.T) {
	// setup
	logger, _ := zap.NewDevelopment()
	var eventCounter atomic.Int64

	externalLogger := &externalLoggerMock{eventCounter: &eventCounter}
	tm, err := NewExternalLogger(false, externalLogger)
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
	assert.Equal(t, int64(3), eventCounter.Load())

	// test logging in a child logger
	logger2 := logger.With(zap.String("child", "logger"))
	logger2.Log(zap.InfoLevel, "hi")
	assert.Equal(t, int64(4), eventCounter.Load())
}
