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
	skipPrivateLogs bool
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

	// if skipPrivateLogs==true, then private logs don't go to telemetry
	if enc.skipPrivateLogs {
		for _, f := range fields {
			if f.Type == zapcore.BoolType {
				if f.Key == "_privateLogEntry" {
					if f.Integer == 1 {
						// do not forward to telemetry by short-circuiting to the underlying encoder.
						return enc.Encoder.EncodeEntry(entry, fields)
					} else {
						break
					}
				}
			}
		}
	}

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

// Clone() clones the encoder. This function is not used by the Guardian itself, but it is used by zapcore.
// Without this implementation, a guardianTelemetryEncoder could get silently converted into the underlying zapcore.Encoder at some point, leading to missing telemetry logs.
func (enc *guardianTelemetryEncoder) Clone() zapcore.Encoder {
	return &guardianTelemetryEncoder{
		Encoder: enc.Encoder.Clone(),
		labels:  enc.labels,
	}
}

// New creates a new Telemetry logger.
// skipPrivateLogs: if set to `true`, logs with the field zap.Bool("_privateLogEntry", true) will not be logged by telemetry.
func New(ctx context.Context, project string, serviceAccountJSON []byte, skipPrivateLogs bool, labels map[string]string) (*Telemetry, error) {
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
			Encoder:         zapcore.NewJSONEncoder(zapdriver.NewProductionEncoderConfig()),
			logger:          gc.Logger("wormhole"),
			labels:          labels,
			skipPrivateLogs: skipPrivateLogs,
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
