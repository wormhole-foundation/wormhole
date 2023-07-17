package processor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackoff(t *testing.T) {
	for i := 0; i < 10; i++ {
		assert.Greater(t, firstRetryMinWait*1*2+time.Second, nextRetryDuration(0))
		assert.Less(t, firstRetryMinWait*1-time.Second, nextRetryDuration(0))

		assert.Greater(t, firstRetryMinWait*2*2+time.Second, nextRetryDuration(1))
		assert.Less(t, firstRetryMinWait*2-time.Second, nextRetryDuration(1))

		assert.Greater(t, firstRetryMinWait*4*2+time.Second, nextRetryDuration(2))
		assert.Less(t, firstRetryMinWait*4-time.Second, nextRetryDuration(2))

		assert.Greater(t, firstRetryMinWait*8*2+time.Second, nextRetryDuration(3))
		assert.Less(t, firstRetryMinWait*8-time.Second, nextRetryDuration(3))

		assert.Greater(t, firstRetryMinWait*1024*2+time.Second, nextRetryDuration(10))
		assert.Less(t, firstRetryMinWait*1024-time.Second, nextRetryDuration(10))
	}
}
