package ccq

import (
	"fmt"
	"net/http"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
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
