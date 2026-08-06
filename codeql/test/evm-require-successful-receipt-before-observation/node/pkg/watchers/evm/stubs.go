package evm

import (
	"context"
	"fmt"

	"codeql/evmrequiresuccessfulreceipt/gethtypes"
	"codeql/evmrequiresuccessfulreceipt/node/pkg/common"
)

type Connector struct{}

type Watcher struct {
	ethConn  Connector
	contract string
}

type Event struct {
	Raw struct {
		TxHash gethtypes.Hash
	}
}

func (c Connector) TransactionReceipt(ctx context.Context, tx gethtypes.Hash) (*gethtypes.Receipt, error) {
	return &gethtypes.Receipt{}, nil
}

func (c Connector) ParseLogMessagePublished(log gethtypes.Log) (Event, error) {
	return Event{}, nil
}

func (w *Watcher) verifyAndPublish(msg *common.MessagePublication, ctx context.Context, txHash gethtypes.Hash, receipt *gethtypes.Receipt) error {
	return nil
}

func MessageEventsForTransaction(ctx context.Context, ethConn Connector, contract string, tx gethtypes.Hash) (*gethtypes.Receipt, uint64, []*common.MessagePublication, error) {
	receipt, err := ethConn.TransactionReceipt(ctx, tx)
	if receipt == nil || err != nil {
		return nil, 0, nil, fmt.Errorf("receipt: %w", err)
	}
	if receipt.Status != gethtypes.ReceiptStatusSuccessful {
		return nil, 0, nil, fmt.Errorf("failed receipt")
	}
	msgs := make([]*common.MessagePublication, 0, len(receipt.Logs))
	for _, log := range receipt.Logs {
		if log == nil {
			continue
		}
		event, err := ethConn.ParseLogMessagePublished(*log)
		if err != nil {
			return nil, 0, nil, err
		}
		_ = event
		msgs = append(msgs, &common.MessagePublication{})
	}
	return receipt, 1, msgs, nil
}
