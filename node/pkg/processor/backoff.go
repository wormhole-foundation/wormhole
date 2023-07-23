package processor

import (
	mathrand "math/rand"
	"time"
)

func nextRetryDuration(ctr uint) time.Duration {
	m := 1 << ctr
	wait := FirstRetryMinWait * time.Duration(m)
	jitter := time.Duration(mathrand.Int63n(int64(wait))) // nolint:gosec
	return wait + jitter
}
