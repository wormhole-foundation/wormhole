package narasu

import (
	"context"
	"sync"
	"time"

	"github.com/tidwall/btree"
)

type MemoryStore struct {
	tree     *btree.Map[int, *memoryStoreItem]
	mapPool  sync.Pool
	interval time.Duration
}

type memoryStoreItem struct {
	m map[string]int
}

func (m *memoryStoreItem) Clear() {
	for k := range m.m {
		delete(m.m, k)
	}
}

func NewMemoryStore(interval time.Duration) *MemoryStore {
	c := &MemoryStore{
		tree:     btree.NewMap[int, *memoryStoreItem](8),
		interval: interval,
		mapPool: sync.Pool{
			New: func() any {
				return &memoryStoreItem{
					m: make(map[string]int),
				}
			},
		},
	}
	return c
}

func (s *MemoryStore) Close() error {
	return nil
}

func (s *MemoryStore) getMap() *memoryStoreItem {
	v := s.mapPool.Get()
	if v == nil {
		return &memoryStoreItem{
			m: make(map[string]int),
		}
	}
	item := v.(*memoryStoreItem)
	item.Clear()
	return item
}

func (s *MemoryStore) putMap(m *memoryStoreItem) {
	s.mapPool.Put(m)
}

func (s *MemoryStore) time(cur time.Time) int {
	return int(cur.Truncate(s.interval).Unix())
}

func (s *MemoryStore) IncrKey(ctx context.Context, bucket string, amount int, cur time.Time) (int, error) {
	now := s.time(cur)
	val, ok := s.tree.Get(now)
	if !ok {
		n := s.getMap()
		s.tree.Set(now, n)
		val = n
	}
	if _, ok := val.m[bucket]; !ok {
		val.m[bucket] = amount
	} else {
		val.m[bucket] = val.m[bucket] + amount
	}
	return val.m[bucket], nil
}

func (s *MemoryStore) GetKeys(ctx context.Context, bucket string, from time.Time, to time.Time) ([]int, error) {
	out := make([]int, 0)
	toseconds := s.time(to)
	fromseconds := s.time(from)
	s.tree.Ascend(fromseconds, func(key int, val *memoryStoreItem) bool {
		if key > toseconds {
			return false
		}
		if count, ok := val.m[bucket]; ok {
			out = append(out, count)
		}
		return true
	})
	return out, nil
}

func (s *MemoryStore) Cleanup(ctx context.Context, now time.Time, age time.Duration) error {
	var expired []int
	nowseconds := int(now.Unix())
	ageSeconds := int(age.Seconds())
	func() {
		s.tree.Ascend(0, func(key int, val *memoryStoreItem) bool {
			// extract the timestamp from the key timestamp:bucket
			ts := key
			if nowseconds-ts >= ageSeconds {
				expired = append(expired, key)
				return true
			}
			return false
		})
	}()
	for _, key := range expired {
		item, ok := s.tree.Delete(key)
		if ok {
			s.putMap(item)
		}
	}
	return nil
}
