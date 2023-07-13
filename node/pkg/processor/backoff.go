package processor

import (
	mathrand "math/rand"
	"time"
)

func initRetryTime() time.Time {
	// return some time between firstRetryMinWait and firstRetryMinWait*2.
	return time.Now().Add(FirstRetryMinWait).Add(time.Duration(mathrand.Int63n(int64(FirstRetryMinWait)))) // nolint:gosec
}

func nextRetryDuration(ctr uint) time.Duration {
	m := 1 << ctr
	return FirstRetryMinWait * time.Duration(m)
}
