package queryratelimit

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type Action struct {
	Time  time.Time      `json:"time"`
	Key   common.Address `json:"key"`
	Types map[uint8]int  `json:"networks"`
}

type Policy struct {
	Limits Limits `json:"limits"`
}

type Limits struct {
	Types map[uint8]Rule `json:"types"`
}

type Rule struct {
	MaxPerSecond int `json:"max_per_second"`
	MaxPerMinute int `json:"max_per_minute"`
}
