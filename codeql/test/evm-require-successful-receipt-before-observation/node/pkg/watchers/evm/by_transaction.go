package evm

import (
	"context"
	"fmt"

	"codeql/evmrequiresuccessfulreceipt/gethtypes"
	"codeql/evmrequiresuccessfulreceipt/node/pkg/common"
)

// Mirrors the production by_transaction.go delegation shape: a shared
// validateTransactionReceipt helper plus a thin reobservation wrapper around the
// shared messageEventsForTransaction implementation. The parse inside the shared
// implementation must be suppressed by the guarded validateTransactionReceipt call.

func validateTransactionReceipt(receipt *gethtypes.Receipt, err error) error {
	if receipt == nil || err != nil {
		return fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	if receipt.Status != gethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("non-success transaction status: %d", receipt.Status)
	}
	return nil
}

func MessageEventsForTransactionReobservation(ctx context.Context, ethConn Connector, contract string, tx gethtypes.Hash) (*gethtypes.Receipt, uint64, []*common.MessagePublication, error) {
	return messageEventsForTransaction(ctx, ethConn, contract, tx)
}

func messageEventsForTransaction(ctx context.Context, ethConn Connector, contract string, tx gethtypes.Hash) (*gethtypes.Receipt, uint64, []*common.MessagePublication, error) {
	receipt, err := ethConn.TransactionReceipt(ctx, tx)
	if valErr := validateTransactionReceipt(receipt, err); valErr != nil {
		return nil, 0, nil, valErr
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
