package common

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const myDelay = time.Millisecond * 100
const myMaxSize = 2
const myQueueSize = myMaxSize * 10

func TestReadFromChannelWithTimeout_NoData(t *testing.T) {
	ctx := context.Background()
	myChan := make(chan int, myQueueSize)

	// No data should timeout.
	timeout, cancel := context.WithTimeout(ctx, myDelay)
	defer cancel()
	observations, err := ReadFromChannelWithTimeout[int](timeout, myChan, myMaxSize)
	assert.Equal(t, err, context.DeadlineExceeded)
	assert.Equal(t, 0, len(observations))
}

func TestReadFromChannelWithTimeout_SomeData(t *testing.T) {
	ctx := context.Background()
	myChan := make(chan int, myQueueSize)
	myChan <- 1

	// Some data but not enough to fill a message should timeout and return the data.
	timeout, cancel := context.WithTimeout(ctx, myDelay)
	defer cancel()
	observations, err := ReadFromChannelWithTimeout[int](timeout, myChan, myMaxSize)
	assert.Equal(t, err, context.DeadlineExceeded)
	require.Equal(t, 1, len(observations))
	assert.Equal(t, 1, observations[0])
}

func TestReadFromChannelWithTimeout_JustEnoughData(t *testing.T) {
	ctx := context.Background()
	myChan := make(chan int, myQueueSize)
	myChan <- 1
	myChan <- 2

	// Just enough data should return the data and no error.
	timeout, cancel := context.WithTimeout(ctx, myDelay)
	defer cancel()
	observations, err := ReadFromChannelWithTimeout[int](timeout, myChan, myMaxSize)
	assert.NoError(t, err)
	require.Equal(t, 2, len(observations))
	assert.Equal(t, 1, observations[0])
	assert.Equal(t, 2, observations[1])
}

func TestReadFromChannelWithTimeout_TooMuchData(t *testing.T) {
	ctx := context.Background()
	myChan := make(chan int, myQueueSize)
	myChan <- 1
	myChan <- 2
	myChan <- 3

	// If there is more data than will fit, it should immediately return a full message, then timeout and return the remainder.
	timeout, cancel := context.WithTimeout(ctx, myDelay)
	defer cancel()
	observations, err := ReadFromChannelWithTimeout[int](timeout, myChan, myMaxSize)
	assert.NoError(t, err)
	require.Equal(t, 2, len(observations))
	assert.Equal(t, 1, observations[0])
	assert.Equal(t, 2, observations[1])

	timeout2, cancel2 := context.WithTimeout(ctx, myDelay)
	defer cancel2()
	observations, err = ReadFromChannelWithTimeout[int](timeout2, myChan, myMaxSize)
	assert.Equal(t, err, context.DeadlineExceeded)
	require.Equal(t, 1, len(observations))
	assert.Equal(t, 3, observations[0])
}
