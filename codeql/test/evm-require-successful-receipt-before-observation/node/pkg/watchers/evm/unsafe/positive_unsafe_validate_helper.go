package unsafe

import (
	"context"
	"fmt"

	"codeql/evmrequiresuccessfulreceipt/gethtypes"
	"codeql/evmrequiresuccessfulreceipt/node/pkg/common"
)

// validateTransactionReceipt outside the pinned by_transaction.go path must not be
// trusted: this lookalike drops the status check, so a nil return proves nothing.

func validateTransactionReceipt(receipt *gethtypes.Receipt, err error) error {
	if receipt == nil || err != nil {
		return fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	return nil
}

// MessageEventsForTransactionReobservation outside the pinned path, without any
// receipt-status validation, must not inherit the wrapper summary.

func MessageEventsForTransactionReobservation(ctx context.Context, ethConn Connector, contract string, tx gethtypes.Hash) (*gethtypes.Receipt, uint64, []*common.MessagePublication, error) {
	receipt, err := ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return nil, 0, nil, err
	}
	msgs := []*common.MessagePublication{}
	return receipt, 1, msgs, nil
}

func positiveUnsafeValidateHelperNameTrust(w *Watcher, ctx context.Context, msg *common.MessagePublication, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if valErr := validateTransactionReceipt(receipt, err); valErr != nil {
		return valErr
	}
	return w.verifyAndPublish(msg, ctx, tx, receipt)
}

func positiveUnsafeReobservationWrapperNameTrust(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, _, msgs, err := MessageEventsForTransactionReobservation(ctx, w.ethConn, w.contract, tx)
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
