package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"cloud.google.com/go/logging"
	"github.com/blendle/zapdriver"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"google.golang.org/api/option"
)

type Telemetry struct {
	encoder            *guardianTelemetryEncoder
	serviceAccountJSON []byte
}

// guardianTelemetryEncoder is a wrapper around zapcore.jsonEncoder that logs to google cloud logging
type guardianTelemetryEncoder struct {
	zapcore.Encoder                   // zapcore.jsonEncoder
	logger          *logging.Logger   // Google Cloud logger
	labels          map[string]string // labels to add to each cloud log
}

// Mirrors the conversion done by zapdriver. We need to convert this
// to proto severity for usage with the SDK client library
// (the JSON value encoded by zapdriver is ignored).
var logLevelSeverity = map[zapcore.Level]logging.Severity{
	zapcore.DebugLevel:  logging.Debug,
	zapcore.InfoLevel:   logging.Info,
	zapcore.WarnLevel:   logging.Warning,
	zapcore.ErrorLevel:  logging.Error,
	zapcore.DPanicLevel: logging.Critical,
	zapcore.PanicLevel:  logging.Alert,
	zapcore.FatalLevel:  logging.Emergency,
}

func (enc *guardianTelemetryEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf, err := enc.Encoder.EncodeEntry(entry, fields)
	if err != nil {
		return nil, err
	}

	// Create a copy of buf (zap will reuse the same buffer otherwise)
	bufCopy := make([]byte, len(buf.Bytes()))
	copy(bufCopy, buf.Bytes())

	// Convert the zapcore.Level to a logging.Severity
	severity := logLevelSeverity[entry.Level]

	// Write raw message to log
	enc.logger.Log(logging.Entry{
		Timestamp: entry.Time,
		Payload:   json.RawMessage(bufCopy),
		Severity:  severity,
		Labels:    enc.labels,
	})

	return buf, nil
}

func (enc *guardianTelemetryEncoder) Clone() zapcore.Encoder {
	return &guardianTelemetryEncoder{
		Encoder: enc.Encoder.Clone(),
		labels:  enc.labels,
	}
}

func New(ctx context.Context, project string, serviceAccountJSON []byte, labels map[string]string) (*Telemetry, error) {
	gc, err := logging.NewClient(ctx, project, option.WithCredentialsJSON(serviceAccountJSON))
	if err != nil {
		return nil, fmt.Errorf("unable to create logging client: %v", err)
	}

	gc.OnError = func(err error) {
		fmt.Printf("telemetry: logging client error: %v\n", err)
	}

	return &Telemetry{
		serviceAccountJSON: serviceAccountJSON,
		encoder: &guardianTelemetryEncoder{
			Encoder: zapcore.NewJSONEncoder(zapdriver.NewProductionEncoderConfig()),
			logger:  gc.Logger("wormhole"),
			labels:  labels,
		},
	}, nil
}

func (s *Telemetry) WrapLogger(logger *zap.Logger) *zap.Logger {
	tc := zapcore.NewCore(
		s.encoder,
		zapcore.AddSync(io.Discard),
		zap.InfoLevel,
	)

	return logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, tc)
	}))
}

func (s *Telemetry) Close() error {
	return s.encoder.logger.Flush()
}
