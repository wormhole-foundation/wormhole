package evm

import (
	"context"
	"fmt"

	"codeql/evmrequiresuccessfulreceipt/gethtypes"
	"codeql/evmrequiresuccessfulreceipt/node/pkg/common"
)

func negativeCheckedBeforePublish(w *Watcher, ctx context.Context, msg *common.MessagePublication, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if receipt == nil || err != nil {
		return err
	}
	if receipt.Status != gethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("failed")
	}
	return w.verifyAndPublish(msg, ctx, tx, receipt)
}

func negativeSuccessBranch(w *Watcher, ctx context.Context, msg *common.MessagePublication, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	if receipt.Status == gethtypes.ReceiptStatusSuccessful {
		return w.verifyAndPublish(msg, ctx, tx, receipt)
	}
	return fmt.Errorf("failed")
}

func negativeMessageEventsReceiptBoundary(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, _, msgs, err := MessageEventsForTransaction(ctx, w.ethConn, w.contract, tx)
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

func negativeMessageEventsRenamedTuple(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	r, _, publications, helperErr := MessageEventsForTransaction(ctx, w.ethConn, w.contract, tx)
	if helperErr != nil {
		return helperErr
	}
	for _, publication := range publications {
		if err := w.verifyAndPublish(publication, ctx, tx, r); err != nil {
			return err
		}
	}
	return nil
}

func negativeMessageEventsPredeclaredTuple(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	var receipt *gethtypes.Receipt
	var msgs []*common.MessagePublication
	var helperErr error

	receipt, _, msgs, helperErr = MessageEventsForTransaction(ctx, w.ethConn, w.contract, tx)
	if helperErr != nil {
		return helperErr
	}
	for _, msg := range msgs {
		if err := w.verifyAndPublish(msg, ctx, tx, receipt); err != nil {
			return err
		}
	}
	return nil
}

func negativeParseReceiptLogsAfterStatus(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	if receipt.Status != gethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("failed")
	}
	for _, log := range receipt.Logs {
		if log == nil {
			continue
		}
		_, err := w.ethConn.ParseLogMessagePublished(*log)
		if err != nil {
			return err
		}
	}
	return nil
}

func negativeParseLogsAliasAfterStatus(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	if receipt.Status != gethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("failed")
	}
	logs := receipt.Logs
	for _, log := range logs {
		_, err := w.ethConn.ParseLogMessagePublished(*log)
		if err != nil {
			return err
		}
	}
	return nil
}

func negativeParseIndexedLogsAfterStatus(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	if receipt.Status != gethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("failed")
	}
	for i := range receipt.Logs {
		_, err := w.ethConn.ParseLogMessagePublished(*receipt.Logs[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func negativeParseLocalIndexedLogsAfterStatus(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	if receipt.Status != gethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("failed")
	}
	logs := receipt.Logs
	for i := range logs {
		_, err := w.ethConn.ParseLogMessagePublished(*logs[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func negativeHelperParameterParseAfterStatus(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	if receipt.Status != gethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("failed")
	}
	for _, log := range receipt.Logs {
		if err := parseOneLog(w.ethConn, *log); err != nil {
			return err
		}
	}
	return nil
}
