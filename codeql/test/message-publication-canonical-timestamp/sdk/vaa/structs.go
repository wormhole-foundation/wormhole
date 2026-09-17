package vaa

import (
	"errors"
	"math"
	"time"
)

func TimeFromUnix(timestamp uint64) (time.Time, error) {
	if timestamp > math.MaxUint32 {
		return time.Time{}, errors.New("timestamp exceeds VAA wire precision")
	}
	return time.Unix(int64(timestamp), 0), nil
}
