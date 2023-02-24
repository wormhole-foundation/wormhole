// On some chains we are unable to get blocks by transaction hash using the go-ethereum library
// because it fails with "transaction type not supported". However, calling the underlying
// eth_getBlockByHash directly works. The sole function of this connector is to implement
// TimeOfBlockByHash using the raw connection.

package connectors

import (
	"context"
	"fmt"
	"strconv"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

type ConnectorWithGetTimeOfBlock struct {
	Connector
}

func NewConnectorWithGetTimeOfBlock(ctx context.Context, baseConnector Connector) (*ConnectorWithGetTimeOfBlock, error) {
	connector := &ConnectorWithGetTimeOfBlock{Connector: baseConnector}
	return connector, nil
}

func (a *ConnectorWithGetTimeOfBlock) TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error) {
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
