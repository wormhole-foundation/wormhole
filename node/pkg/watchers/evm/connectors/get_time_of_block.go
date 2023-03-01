package connectors

import (
	"context"
	"fmt"
	"strconv"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

// ConnectorWithGetTimeOfBlockOverride is used to override the implementation of TimeOfBlockByHash() as defined in
// the go-ethereum library because on some chains it fails with "transaction type not supported". Calling the underlying
// eth_getBlockByHash directly works, so the sole function of this connector is to implement TimeOfBlockByHash() using the raw connection.
type ConnectorWithGetTimeOfBlockOverride struct {
	Connector
}

func NewConnectorWithGetTimeOfBlockOverride(ctx context.Context, baseConnector Connector) (*ConnectorWithGetTimeOfBlockOverride, error) {
	connector := &ConnectorWithGetTimeOfBlockOverride{Connector: baseConnector}
	return connector, nil
}

func (a *ConnectorWithGetTimeOfBlockOverride) TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error) {
	type Marshaller struct {
		Time string `json:"timestamp"        gencodec:"required"`
	}

	var m *Marshaller
	err := a.RawCallContext(ctx, &m, "eth_getBlockByHash", hash, false)
	if err != nil {
		return 0, fmt.Errorf("failed to get block %s: %w", hash.String(), err)
	}

	num, err := strconv.ParseUint(m.Time[2:], 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse time %s: %w", m.Time, err)
	}

	return num, nil
}
