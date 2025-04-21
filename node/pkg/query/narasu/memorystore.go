package narasu

import (
	"context"
	"sync"
	"time"

	"github.com/google/btree"
)

// MemoryStore is an in-memory, time-bucketed counter store.
type MemoryStore struct {
	mu       sync.RWMutex
	tree     *btree.BTreeG[*memoryStoreItem]
	interval time.Duration
}

type memoryStoreItem struct {
	ts int // normalized timestamp (bucket key)
	m  map[string]uint64
}

// NewMemoryStore creates a new MemoryStore with the given time bucket interval.
func NewMemoryStore(interval time.Duration) *MemoryStore {
	return &MemoryStore{
		tree: btree.NewG(8, func(a, b *memoryStoreItem) bool {
			return a.ts < b.ts
		}),
		interval: interval,
	}
}

// Close implements a no-op closer for interface compatibility.
func (s *MemoryStore) Close() error {
	return nil
}

func (s *MemoryStore) newItem(ts int) *memoryStoreItem {
	return &memoryStoreItem{
		ts: ts,
		m:  make(map[string]uint64),
	}
}

// normalizeTime converts a time into a bucket timestamp.
func (s *MemoryStore) normalizeTime(t time.Time) int {
	return int(t.Truncate(s.interval).Unix())
}

// IncrKey increments a bucketed counter.
// Context is intentionally ignored
func (s *MemoryStore) IncrKey(
	_ context.Context,
	bucket string,
	amount int,
	at time.Time,
) (uint64, error) {
	ts := s.normalizeTime(at)

	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.tree.Get(&memoryStoreItem{ts: ts})
	if !ok {
		item = s.newItem(ts)
		s.tree.ReplaceOrInsert(item)
	}

	item.m[bucket] += uint64(amount) // #nosec G115 -- amount is always non-negative in practice
	return item.m[bucket], nil
}

// GetKeys returns all counter values for a bucket in the given time range.
func (s *MemoryStore) GetKeys(
	ctx context.Context,
	bucket string,
	from time.Time,
	to time.Time,
) ([]uint64, error) {
	fromTS := s.normalizeTime(from)
	toTS := s.normalizeTime(to)

	out := make([]uint64, 0)
	var ctxErr error

	s.mu.RLock()
	defer s.mu.RUnlock()

	s.tree.AscendRange(
		&memoryStoreItem{ts: fromTS},
		&memoryStoreItem{ts: toTS},
		func(item *memoryStoreItem) bool {
			select {
			case <-ctx.Done():
				ctxErr = ctx.Err()
				return false
			default:
			}

			if v, ok := item.m[bucket]; ok {
				out = append(out, v)
			}
			return true
		},
	)

	if ctxErr != nil {
		return nil, ctxErr
	}

	return out, nil
}

// Cleanup removes entries older than the given age.
// Cleanup is best-effort and respects context cancellation.
func (s *MemoryStore) Cleanup(
	ctx context.Context,
	now time.Time,
	age time.Duration,
) error {
	nowTS := s.normalizeTime(now)
	expireBefore := nowTS - int(age.Seconds())

	var expired []int

	// Scan phase
	s.mu.RLock()
	s.tree.Ascend(func(item *memoryStoreItem) bool {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		if item.ts <= expireBefore {
			expired = append(expired, item.ts)
			return true
		}
		return false
	})
	s.mu.RUnlock()

	// Delete phase
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ts := range expired {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		s.tree.Delete(&memoryStoreItem{ts: ts})
	}

	return nil
}
