package testutils

import (
	"testing"
	"time"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func MustTimeFromUnix[N number](t testing.TB, timestamp N) time.Time {
	t.Helper()

	ts, err := vaa.TimeFromUnix(timestamp)
	if err != nil {
		t.Fatalf("invalid Unix timestamp %d: %v", timestamp, err)
	}
	return ts
}
