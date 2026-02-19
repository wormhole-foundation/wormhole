package common

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

func ListenSysExit(logger *zap.Logger, ctxCancel context.CancelFunc) {
	// Handle SIGTERM, SIGINT
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigterm
		logger.Info("Received sigterm. exiting.")
		ctxCancel()
	}()
}
