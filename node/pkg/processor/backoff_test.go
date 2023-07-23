package processor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackoff(t *testing.T) {
	for i := 0; i < 10; i++ {
		assert.Greater(t, FirstRetryMinWait*1*2+time.Second, nextRetryDuration(0))
		assert.Less(t, FirstRetryMinWait*1-time.Second, nextRetryDuration(0))

		assert.Greater(t, FirstRetryMinWait*2*2+time.Second, nextRetryDuration(1))
		assert.Less(t, FirstRetryMinWait*2-time.Second, nextRetryDuration(1))

		assert.Greater(t, FirstRetryMinWait*4*2+time.Second, nextRetryDuration(2))
		assert.Less(t, FirstRetryMinWait*4-time.Second, nextRetryDuration(2))

		assert.Greater(t, FirstRetryMinWait*8*2+time.Second, nextRetryDuration(3))
		assert.Less(t, FirstRetryMinWait*8-time.Second, nextRetryDuration(3))

		assert.Greater(t, FirstRetryMinWait*1024*2+time.Second, nextRetryDuration(10))
		assert.Less(t, FirstRetryMinWait*1024-time.Second, nextRetryDuration(10))
	}
}
