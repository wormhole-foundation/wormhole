package governor

import (
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestIsVAAEnqueuedNilMessageID(t *testing.T) {
	logger, _ := zap.NewProduction()
	gov := NewChainGovernor(logger, nil, common.GoTest)
	enqueued, err := gov.IsVAAEnqueued(nil)
	require.EqualError(t, err, "no message ID specified")
	assert.Equal(t, false, enqueued)
}
