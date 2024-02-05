package ccq

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	promremotew "github.com/certusone/wormhole/node/pkg/telemetry/prom_remote_write"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type statusServer struct {
	logger        *zap.Logger
	env           common.Environment
	httpServer    *http.Server
	healthEnabled atomic.Bool
}

func NewStatusServer(addr string, logger *zap.Logger, env common.Environment) *statusServer {
	s := &statusServer{
		logger: logger,
		env:    env,
	}
	s.healthEnabled.Store(true)
	r := mux.NewRouter()
	r.HandleFunc("/health", s.handleHealth).Methods("GET")
	r.Handle("/metrics", promhttp.Handler())
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

func (s *statusServer) disableHealth() {
	s.healthEnabled.Store(false)
}

func (s *statusServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !s.healthEnabled.Load() {
		s.logger.Info("ignoring health check")
		http.Error(w, "shutting down", http.StatusServiceUnavailable)
		return
	}
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
					promLogger.Error("ScrapeAndSendLocalMetrics encountered an error, will try again next interval", zap.Error(err))
				}
			}
		}
	})
	return nil
}
