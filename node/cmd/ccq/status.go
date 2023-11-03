package ccq

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	promremotew "github.com/certusone/wormhole/node/pkg/telemetry/prom_remote_write"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type statusServer struct {
	logger *zap.Logger
	env    common.Environment
}

func NewStatusServer(addr string, logger *zap.Logger, env common.Environment) *http.Server {
	s := &statusServer{
		logger: logger,
		env:    env,
	}
	r := mux.NewRouter()
	r.HandleFunc("/health", s.handleHealth).Methods("GET")
	r.Handle("/metrics", promhttp.Handler())
	return &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func (s *statusServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("health check")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "ok")
}

func RunPrometheusScraper(ctx context.Context, logger *zap.Logger, info promremotew.PromTelemetryInfo) error {
	promLogger := logger.With(zap.String("component", "prometheus_scraper"))
	errC := make(chan error)
	common.StartRunnable(ctx, errC, false, "prometheus_scraper", func(ctx context.Context) error {
		t := time.NewTicker(15 * time.Second)

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				err := promremotew.ScrapeAndSendLocalMetrics(ctx, info, promLogger)
				if err != nil {
					promLogger.Error("ScrapeAndSendLocalMetrics error", zap.Error(err))
					return err
				}
			}
		}
	})
	return nil
}
