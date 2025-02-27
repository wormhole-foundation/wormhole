package promremotew

import (
	"bytes"
	"context"
	"net/http"

	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type PromTelemetryInfo struct {
	PromRemoteURL string
	Labels        map[string]string
}

func scrapeLocalMetricsViaGatherer() (map[string]*dto.MetricFamily, error) {
	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return nil, err
	}
	// Here we have an array of metrics.
	mapByName := map[string]*dto.MetricFamily{}
	for _, met := range metrics {
		name := met.GetName()
		if _, ok := mapByName[name]; !ok {
			mapByName[name] = met
		}
	}
	return mapByName, nil
}

func ScrapeAndSendLocalMetrics(ctx context.Context, info PromTelemetryInfo, logger *zap.Logger) error {
	metrics, err := scrapeLocalMetricsViaGatherer()
	if err != nil {
		logger.Error("Could not scrape local metrics", zap.Error(err))
		return err
	}
	// At this point we have a map of metrics by name.
	// Need to convert to write request.
	writeRequest, err := MetricFamiliesToWriteRequest(metrics, info.Labels)
	if err != nil {
		logger.Error("Could not create write request", zap.Error(err))
		return err
	}

	raw, err := proto.Marshal(writeRequest)
	if err != nil {
		logger.Error("Could not marshal write request", zap.Error(err))
		return err
	}
	oSnap := snappy.Encode(nil, raw)
	bodyReader := bytes.NewReader(oSnap)

	// Create the http request
	// requestURL := fmt.Sprintf("https://%s:%s@%s", info.PromRemoteUser, info.PromRemoteKey, info.PromRemoteURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, info.PromRemoteURL, bodyReader)
	if err != nil {
		logger.Error("Could not create request", zap.Error(err))
		return err
	}
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("User-Agent", "Guardian")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error("Error creating http request", zap.Error(err))
		return err
	}

	defer res.Body.Close()

	logger.Debug("Grafana result", zap.Int("status code", res.StatusCode))
	if res.StatusCode != 200 && res.StatusCode != 204 {
		logger.Error("Grafana returned a status code other than 200 or 204", zap.Int("status code", res.StatusCode))
		return err
	}
	return nil
}
