package queryratelimit

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type Action struct {
	Time     time.Time      `json:"time"`
	Key      common.Address `json:"key"`
	Networks map[string]int `json:"networks"`
}

type Policy struct {
	Limits Limits `json:"limits"`
}

type Limits struct {
	Networks map[string]Rule `json:"networks"`
}

type Rule struct {
	MaxPerSecond int `json:"max_per_second"`
	MaxPerMinute int `json:"max_per_minute"`
}
