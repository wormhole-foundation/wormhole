package evm

import (
	"context"

	"codeql/evmfinalityrelease/gethtypes"
	"codeql/evmfinalityrelease/node/pkg/common"
)

const (
	ConsistencyLevelPublishImmediately uint8 = 200
	ConsistencyLevelSafe               uint8 = 201
	ConsistencyLevelFinalized          uint8 = 202
	Latest                             uint8 = 1
	Safe                               uint8 = 2
	Finalized                          uint8 = 3
)

type Connector struct{}

type Watcher struct {
	ethConn Connector
	pending map[pendingKey]*pendingMessage
}

type pendingKey struct {
	TxHash    gethtypes.Hash
	BlockHash gethtypes.Hash
}

type pendingMessage struct {
	message          *common.MessagePublication
	height           uint64
	additionalBlocks uint64
	effectiveCL      uint8
}

type NewBlock struct {
	Number   uint64
	Finality uint8
}

func consistencyLevelMatches(blockCL uint8, msgCL uint8) bool { return blockCL == msgCL }

func currentBlockConsistencyLevel(ev NewBlock) uint8 {
	if ev.Finality == Safe {
		return ConsistencyLevelSafe
	}
	return ConsistencyLevelFinalized
}

func (c Connector) TransactionReceipt(ctx context.Context, tx gethtypes.Hash) (*gethtypes.Receipt, error) {
	return &gethtypes.Receipt{}, nil
}

func (w *Watcher) verifyAndPublish(msg *common.MessagePublication, ctx context.Context, txHash gethtypes.Hash, receipt *gethtypes.Receipt) error {
	return nil
}
