package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	google_cloud_logging "cloud.google.com/go/logging"
	"github.com/blendle/zapdriver"
	"go.uber.org/zap/zapcore"
	"google.golang.org/api/option"
)

// ExternalLoggerGoogleCloud implements ExternalLogger for the Google GCP cloud logging.
type ExternalLoggerGoogleCloud struct {
	*google_cloud_logging.Logger
	labels map[string]string // labels to add to each cloud log
}

func (logger *ExternalLoggerGoogleCloud) log(time time.Time, message json.RawMessage, level zapcore.Level) {
	entry := google_cloud_logging.Entry{
		Timestamp: time,
		Payload:   message,
		Severity:  googleLogLevelSeverity[level],
		Labels:    logger.labels,
	}
	// call google cloud logger
	logger.Log(entry)
}

func (logger *ExternalLoggerGoogleCloud) close() error {
	return logger.Flush()
}

// Mirrors the conversion done by zapdriver. We need to convert this
// to proto severity for usage with the SDK client library
// (the JSON value encoded by zapdriver is ignored).
var googleLogLevelSeverity = map[zapcore.Level]google_cloud_logging.Severity{
	zapcore.DebugLevel:  google_cloud_logging.Debug,
	zapcore.InfoLevel:   google_cloud_logging.Info,
	zapcore.WarnLevel:   google_cloud_logging.Warning,
	zapcore.ErrorLevel:  google_cloud_logging.Error,
	zapcore.DPanicLevel: google_cloud_logging.Critical,
	zapcore.PanicLevel:  google_cloud_logging.Alert,
	zapcore.FatalLevel:  google_cloud_logging.Emergency,
}

// NewGoogleCloudLogger creates a new Telemetry logger with Google Cloud Logging
// skipPrivateLogs: if set to `true`, logs with the field zap.Bool("_privateLogEntry", true) will not be logged by telemetry.
func NewGoogleCloudLogger(ctx context.Context, project string, skipPrivateLogs bool, labels map[string]string, opts ...option.ClientOption) (*Telemetry, error) {
	gc, err := google_cloud_logging.NewClient(ctx, project, opts...)
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
