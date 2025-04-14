package processor

import (
	mathrand "math/rand"
	"time"
)

func nextRetryDuration(ctr uint) time.Duration {
	m := 1 << ctr
	wait := FirstRetryMinWait * time.Duration(m)
	// #nosec G404 we don't need cryptographic randomness here.
	jitter := time.Duration(mathrand.Int63n(int64(wait)))
	return wait + jitter
}
