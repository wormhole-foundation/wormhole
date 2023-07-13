package processor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackoff(t *testing.T) {
	assert.Equal(t, firstRetryMinWait, nextRetryDuration(0))
	assert.Equal(t, firstRetryMinWait*2, nextRetryDuration(1))
	assert.Equal(t, firstRetryMinWait*4, nextRetryDuration(2))

	for i := 0; i < 10; i++ {
		assert.Greater(t, time.Now().Add(firstRetryMinWait*2+time.Second), initRetryTime())
		assert.Less(t, time.Now().Add(firstRetryMinWait-time.Second), initRetryTime())
	}
}
