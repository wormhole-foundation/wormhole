package telemetry

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/blendle/zapdriver"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

const telemetryLogLevel = zap.InfoLevel

type Telemetry struct {
	encoder *guardianTelemetryEncoder
}

type ExternalLogger interface {
	log(time time.Time, message json.RawMessage, level zapcore.Level)
	close()
}

// guardianTelemetryEncoder is a wrapper around zapcore.jsonEncoder that logs to cloud based logging
type guardianTelemetryEncoder struct {
	zapcore.Encoder // zapcore.jsonEncoder
	logger          ExternalLogger
	skipPrivateLogs bool
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

func (s *Telemetry) Close() {
	s.encoder.logger.close()
}
