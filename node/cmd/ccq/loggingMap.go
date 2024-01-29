package ccq

import (
	"context"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"go.uber.org/zap"
)

// LoggingMap is used to track the requests for which we should log response. It contains a map keyed by the request signature
// where the payload is time the request was received. Requests will be removed from the map after two minutes.
type LoggingMap struct {
	loggingLock sync.Mutex
	loggingMap  map[string]time.Time
}

// NewLoggingMap creates the map used to track requests for which we should log responses.
func NewLoggingMap() *LoggingMap {
	return &LoggingMap{
		loggingMap: make(map[string]time.Time),
	}
}

// Start starts a go routine to clean up requests that have been in the map for two minutes.
func (lm *LoggingMap) Start(ctx context.Context, logger *zap.Logger, errC chan error) {
	common.RunWithScissors(ctx, errC, "logging_cleanup", func(ctx context.Context) error {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				lm.CleanUp(logger)
			}
		}
	})
}

// CleanUp iterates over the map and removes all entries that are more than two minutes old.
func (lm *LoggingMap) CleanUp(logger *zap.Logger) {
	lm.loggingLock.Lock()
	defer lm.loggingLock.Unlock()
	for requestId, cleanUpTime := range lm.loggingMap {
		if time.Now().After(cleanUpTime) {
			delete(lm.loggingMap, requestId)
		}
	}
}

// AddRequest adds a request to the map, giving it an expiration time two minutes into the future.
func (lm *LoggingMap) AddRequest(requestSignature string) {
	lm.loggingLock.Lock()
	defer lm.loggingLock.Unlock()
	lm.loggingMap[requestSignature] = time.Now().Add(2 * time.Minute)
}

// ShouldLogResponse returns true if the request is in the map.
func (lm *LoggingMap) ShouldLogResponse(requestSignature string) bool {
	lm.loggingLock.Lock()
	defer lm.loggingLock.Unlock()
	if _, exists := lm.loggingMap[requestSignature]; exists {
		return true
	}
	return false
}
