package unsafe

import (
	"context"

	"codeql/evmrequiresuccessfulreceipt/gethtypes"
	"codeql/evmrequiresuccessfulreceipt/node/pkg/common"
)

type Connector struct{}

type Watcher struct {
	ethConn  Connector
	contract string
}

func (c Connector) TransactionReceipt(ctx context.Context, tx gethtypes.Hash) (*gethtypes.Receipt, error) {
	return &gethtypes.Receipt{}, nil
}

func (c Connector) ParseLogMessagePublished(log gethtypes.Log) (*common.MessagePublication, error) {
	return &common.MessagePublication{}, nil
}

func (w *Watcher) verifyAndPublish(msg *common.MessagePublication, ctx context.Context, txHash gethtypes.Hash, receipt *gethtypes.Receipt) error {
	return nil
}

func MessageEventsForTransaction(ctx context.Context, ethConn Connector, contract string, tx gethtypes.Hash) (*gethtypes.Receipt, []*common.MessagePublication, error) {
	receipt, err := ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return nil, nil, err
	}
	msgs := make([]*common.MessagePublication, 0, len(receipt.Logs))
	for _, log := range receipt.Logs {
		msg, err := ethConn.ParseLogMessagePublished(*log)
		if err != nil {
			return nil, nil, err
		}
		msgs = append(msgs, msg)
	}
	return receipt, msgs, nil
}

func positiveUnsafeSameNamedHelper(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, msgs, err := MessageEventsForTransaction(ctx, w.ethConn, w.contract, tx)
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		if err := w.verifyAndPublish(msg, ctx, tx, receipt); err != nil {
			return err
		}
	}
	return nil
}
