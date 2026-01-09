package narasu

import (
	"context"
	"time"
)

type Store interface {
	IncrKey(ctx context.Context, bucket string, amount int, cur time.Time) (uint64, error)
	GetKeys(ctx context.Context, bucket string, from time.Time, to time.Time) ([]uint64, error)
	Cleanup(ctx context.Context, now time.Time, age time.Duration) error
}
