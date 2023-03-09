package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	google_cloud_logging "cloud.google.com/go/logging"
	"github.com/blendle/zapdriver"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"google.golang.org/api/option"
)

const telemetryLogLevel = zap.InfoLevel

type Telemetry struct {
	encoder *guardianTelemetryEncoder
}

type ExternalLogger interface {
	log(time time.Time, message json.RawMessage, level zapcore.Level)
	flush() error
}

type ExternalLoggerGoogleCloud struct {
	*google_cloud_logging.Logger
	labels map[string]string // labels to add to each cloud log
}

func (logger *ExternalLoggerGoogleCloud) log(time time.Time, message json.RawMessage, level zapcore.Level) {
	logger.Log(google_cloud_logging.Entry{
		Timestamp: time,
		Payload:   message,
		Severity:  logLevelSeverity[level],
		Labels:    logger.labels,
	})
}

func (logger *ExternalLoggerGoogleCloud) flush() error {
	return logger.Flush()
}

// guardianTelemetryEncoder is a wrapper around zapcore.jsonEncoder that logs to google cloud logging
type guardianTelemetryEncoder struct {
	zapcore.Encoder // zapcore.jsonEncoder
	logger          ExternalLogger
	skipPrivateLogs bool
}

// Mirrors the conversion done by zapdriver. We need to convert this
// to proto severity for usage with the SDK client library
// (the JSON value encoded by zapdriver is ignored).
var logLevelSeverity = map[zapcore.Level]google_cloud_logging.Severity{
	zapcore.DebugLevel:  google_cloud_logging.Debug,
	zapcore.InfoLevel:   google_cloud_logging.Info,
	zapcore.WarnLevel:   google_cloud_logging.Warning,
	zapcore.ErrorLevel:  google_cloud_logging.Error,
	zapcore.DPanicLevel: google_cloud_logging.Critical,
	zapcore.PanicLevel:  google_cloud_logging.Alert,
	zapcore.FatalLevel:  google_cloud_logging.Emergency,
}

func (enc *guardianTelemetryEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf, err := enc.Encoder.EncodeEntry(entry, fields)
	if err != nil {
		return nil, err
	}

	// Create a copy of buf (zap will reuse the same buffer otherwise)
	bufCopy := make([]byte, len(buf.Bytes()))
	copy(bufCopy, buf.Bytes())

	// if skipPrivateLogs==true, then private logs don't go to telemetry
	if enc.skipPrivateLogs {
		if bytes.Contains(bufCopy, []byte("\"_privateLogEntry\":true")) {
			// early return because this is a private entry and it should not go to telemetry
			return buf, nil
		}
	}

	// Write raw message to telemetry logger
	enc.logger.log(entry.Time, json.RawMessage(bufCopy), entry.Level)

	return buf, nil
}

// Clone() clones the encoder. This function is not used by the Guardian itself, but it is used by zapcore.
// Without this implementation, a guardianTelemetryEncoder could get silently converted into the underlying zapcore.Encoder at some point, leading to missing telemetry logs.
func (enc *guardianTelemetryEncoder) Clone() zapcore.Encoder {
	return &guardianTelemetryEncoder{
		Encoder:         enc.Encoder.Clone(),
		logger:          enc.logger,
		skipPrivateLogs: enc.skipPrivateLogs,
	}
}

func NewExternalLogger(skipPrivateLogs bool, externalLogger ExternalLogger) (*Telemetry, error) {
	return &Telemetry{
		encoder: &guardianTelemetryEncoder{
			Encoder:         zapcore.NewJSONEncoder(zapdriver.NewProductionEncoderConfig()),
			logger:          externalLogger,
			skipPrivateLogs: skipPrivateLogs,
		},
	}, nil
}

// New creates a new Telemetry logger with Google Cloud Logging
// skipPrivateLogs: if set to `true`, logs with the field zap.Bool("_privateLogEntry", true) will not be logged by telemetry.
func New(ctx context.Context, project string, serviceAccountJSON []byte, skipPrivateLogs bool, labels map[string]string) (*Telemetry, error) {
	gc, err := google_cloud_logging.NewClient(ctx, project, option.WithCredentialsJSON(serviceAccountJSON))
	if err != nil {
		return nil, fmt.Errorf("unable to create logging client: %v", err)
	}

	gc.OnError = func(err error) {
		fmt.Printf("telemetry: logging client error: %v\n", err)
	}

	return &Telemetry{
		encoder: &guardianTelemetryEncoder{
			Encoder:         zapcore.NewJSONEncoder(zapdriver.NewProductionEncoderConfig()),
			logger:          &ExternalLoggerGoogleCloud{Logger: gc.Logger("wormhole"), labels: labels},
			skipPrivateLogs: skipPrivateLogs,
		},
	}, nil
}

func (s *Telemetry) WrapLogger(logger *zap.Logger) *zap.Logger {
	tc := zapcore.NewCore(
		s.encoder,
		zapcore.AddSync(io.Discard),
		telemetryLogLevel,
	)

	return logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, tc)
	}))
}

func (s *Telemetry) Close() error {
	return s.encoder.logger.flush()
}
