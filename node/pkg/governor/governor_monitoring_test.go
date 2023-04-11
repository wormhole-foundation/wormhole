package governor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/test-go/testify/require"
	"go.uber.org/zap"
)

func TestIsVAAEnqueuedNilMessageID(t *testing.T) {
	logger, _ := zap.NewProduction()
	gov := NewChainGovernor(logger, nil, GoTestMode)
	enqueued, err := gov.IsVAAEnqueued(nil)
	require.EqualError(t, err, "no message ID specified")
	assert.Equal(t, false, enqueued)
}
