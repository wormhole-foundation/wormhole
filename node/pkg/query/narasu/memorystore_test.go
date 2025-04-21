package narasu_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/query/narasu"
	"github.com/stretchr/testify/require"
)

var bucketNames []string

func init() {
	for i := range 32 {
		bucketNames = append(bucketNames, fmt.Sprintf("test-%d", i))
	}
}

func BenchmarkMemoryStore(b *testing.B) {
	ctx := context.Background()
	start := time.Unix(1000, 0)
	end := start.Add(time.Second * 59)
	bucket := "test"

	b.Run("incr+clean", func(b *testing.B) {
		c := narasu.NewMemoryStore(time.Second)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			c.IncrKey(ctx, bucketNames[0], 1, start.Add(time.Second*time.Duration(i)))
			c.Cleanup(ctx, start.Add(time.Second*time.Duration(i)), 60*time.Second)
		}
	})
	b.Run("incr+getrange", func(b *testing.B) {
		c := narasu.NewMemoryStore(time.Second)
		for i := range 60 {
			c.IncrKey(ctx, bucket, 1, start.Add(time.Second*time.Duration(i)))
		}
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			c.GetKeys(ctx, bucket, start, end)
		}
	})

}

func TestMemoryStore(t *testing.T) {
	ctx := context.Background()
	interval := time.Minute
	c := narasu.NewMemoryStore(interval)
	start := time.Unix(0, 0)
	end := start.Add(interval * 14)

	bucket := "test"
	var val int
	var err error
	val, err = c.IncrKey(ctx, bucket, 1, start)
	require.NoError(t, err)
	require.Equal(t, 1, val)

	val, err = c.IncrKey(ctx, bucket, 1, start.Add(interval))
	require.NoError(t, err)
	require.Equal(t, 1, val)

	val, err = c.IncrKey(ctx, bucket, 1, start.Add(time.Duration(float64(interval)*0.5)))
	require.NoError(t, err)
	require.Equal(t, 2, val)

	val, err = c.IncrKey(ctx, bucket, 1, start.Add(interval*2))
	require.NoError(t, err)
	require.Equal(t, 1, val)

	val, err = c.IncrKey(ctx, bucket, 1, start.Add(interval*10))
	require.NoError(t, err)
	require.Equal(t, 1, val)

	val, err = c.IncrKey(ctx, bucket, 1, start.Add(interval*12))
	require.NoError(t, err)
	require.Equal(t, 1, val)

	xs, err := c.GetKeys(ctx, bucket, start, end)
	require.NoError(t, err)
	require.ElementsMatch(t, []int{
		1, 2, 1, 1, 1}, xs)

	err = c.Cleanup(ctx, end, 8*interval)
	require.NoError(t, err)

	xs, err = c.GetKeys(ctx, bucket, start, end)
	require.NoError(t, err)
	require.ElementsMatch(t, []int{
		1, 1,
	}, xs)
}
