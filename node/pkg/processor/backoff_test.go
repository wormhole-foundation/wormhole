package processor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackoff(t *testing.T) {
	assert.Equal(t, FirstRetryMinWait, nextRetryDuration(0))
	assert.Equal(t, FirstRetryMinWait*2, nextRetryDuration(1))
	assert.Equal(t, FirstRetryMinWait*4, nextRetryDuration(2))

	for i := 0; i < 10; i++ {
		assert.Greater(t, time.Now().Add(FirstRetryMinWait*2+time.Second), initRetryTime())
		assert.Less(t, time.Now().Add(FirstRetryMinWait-time.Second), initRetryTime())
	}
}
