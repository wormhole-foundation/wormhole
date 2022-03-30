package reporter

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
)

func TestEventListener(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	subs_expect := map[int]*lifecycleEventChannels{}

	attestationEventReporter := EventListener(logger)
	assert.Equal(t, subs_expect, attestationEventReporter.subs)
}

func TestGetUniqueClientId(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	attestationEventReporter := EventListener(logger)
	assert.Equal(t, 498081, attestationEventReporter.getUniqueClientId())
}

func TestSubscribe(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	attestationEventReporter := EventListener(logger)
	activeSubscription := attestationEventReporter.Subscribe()
	assert.Equal(t, 727887, activeSubscription.ClientId)
}
