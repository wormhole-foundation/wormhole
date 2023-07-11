// We are using promtail client version v2.8.2:
// go get github.com/grafana/loki/clients/pkg/promtail/client@9f809eda70babaf583bdf6bf335a28038f286618

package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/blendle/zapdriver"
	"go.uber.org/zap/zapcore"
	"google.golang.org/api/option"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/loki/clients/pkg/promtail/api"
	"github.com/grafana/loki/clients/pkg/promtail/client"
	"github.com/grafana/loki/pkg/logproto"
	lokiflag "github.com/grafana/loki/pkg/util/flagext"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
)

// ExternalLoggerLoki implements ExternalLogger for the Grafana Loki cloud logging.
type ExternalLoggerLoki struct {
	// c is the promtail client.
	c client.Client

	// labels is the set of labels to be added to each log entry, based on the severity (since severity is one of the labels).
	labels map[zapcore.Level]model.LabelSet
}

func (logger *ExternalLoggerLoki) log(time time.Time, message json.RawMessage, level zapcore.Level) {
	lokiLabels := logger.labels[level]

	bytes, err := message.MarshalJSON()
	if err != nil {
		return
	}
	entry := api.Entry{
		Entry: logproto.Entry{
			Timestamp: time,
			Line:      string(bytes),
		},

		Labels: lokiLabels,
	}

	logger.c.Chan() <- entry
}

func (logger *ExternalLoggerLoki) flush() error {
	// flush() is only called from the Telemetry.Close() function, which is only called from the Guardian.Close() function.
	logger.c.Stop()
	return nil
}

// NewLokiCloudLogger creates a new Telemetry logger using Grafana Loki Cloud Logging.
// skipPrivateLogs: if set to `true`, logs with the field zap.Bool("_privateLogEntry", true) will not be logged by telemetry.
func NewLokiCloudLogger(ctx context.Context, url string, project string, skipPrivateLogs bool, labels map[string]string, opts ...option.ClientOption) (*Telemetry, error) {
	reg := prometheus.NewRegistry()
	m := client.NewMetrics(reg)

	serverURL := flagext.URLValue{}
	err := serverURL.Set(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Loki client url: %v", err)
	}

	cfg := client.Config{
		URL:                    serverURL,
		BatchWait:              100 * time.Millisecond,
		BatchSize:              10,
		DropRateLimitedBatches: true,
		Client:                 config.HTTPClientConfig{},
		BackoffConfig:          backoff.Config{MinBackoff: 1 * time.Millisecond, MaxBackoff: 2 * time.Millisecond, MaxRetries: 3},
		ExternalLabels:         lokiflag.LabelSet{},
		Timeout:                1 * time.Second,
		// TenantID: We are not using the tenantID.
	}

	clientMaxLineSize := 1024
	clientMaxLineSizeTruncate := true

	c, err := client.New(m, cfg, 0, clientMaxLineSize, clientMaxLineSizeTruncate, log.NewNopLogger())
	if err != nil {
		return nil, fmt.Errorf("failed to create Loki client: %v", err)
	}

	// Since severity is one of the labels, create a label set for each severity, to avoid copying the labels map for each log entry.
	lokiLabels := make(map[zapcore.Level]model.LabelSet)
	for level := zapcore.DebugLevel; level <= zapcore.FatalLevel; level++ {
		levLabels := model.LabelSet{}
		for k, v := range labels {
			levLabels[model.LabelName(k)] = model.LabelValue(v)
		}
		levLabels[model.LabelName("severity")] = model.LabelValue(level.CapitalString())
		levLabels[model.LabelName("projectId")] = model.LabelValue(project)
		lokiLabels[level] = levLabels
	}

	return &Telemetry{
		encoder: &guardianTelemetryEncoder{
			Encoder:         zapcore.NewJSONEncoder(zapdriver.NewProductionEncoderConfig()),
			logger:          &ExternalLoggerLoki{c: c, labels: lokiLabels},
			skipPrivateLogs: skipPrivateLogs,
		},
	}, nil
}
