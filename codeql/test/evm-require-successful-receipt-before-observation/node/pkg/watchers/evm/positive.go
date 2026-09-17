package evm

import (
	"context"
	"fmt"

	"codeql/evmrequiresuccessfulreceipt/gethtypes"
	"codeql/evmrequiresuccessfulreceipt/node/pkg/common"
)

func positiveHistoricalInstantPublication(w *Watcher, ctx context.Context, msg *common.MessagePublication, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if receipt == nil || err != nil {
		return err
	}
	return w.verifyAndPublish(msg, ctx, tx, receipt)
}

func positiveDifferentReceiptChecked(w *Watcher, ctx context.Context, msg *common.MessagePublication, tx gethtypes.Hash) error {
	checked, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	if checked.Status != gethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("failed")
	}
	other, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	return w.verifyAndPublish(msg, ctx, tx, other)
}

func positivePostCheck(w *Watcher, ctx context.Context, msg *common.MessagePublication, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	pubErr := w.verifyAndPublish(msg, ctx, tx, receipt)
	if receipt.Status != gethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("failed")
	}
	return pubErr
}

func positiveReassignedAfterCheck(w *Watcher, ctx context.Context, msg *common.MessagePublication, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	if receipt.Status != gethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("failed")
	}
	receipt, err = w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	return w.verifyAndPublish(msg, ctx, tx, receipt)
}

func positiveNilReceiptOnly(w *Watcher, ctx context.Context, msg *common.MessagePublication, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if receipt == nil || err != nil {
		return err
	}
	return w.verifyAndPublish(msg, ctx, tx, receipt)
}

func positiveParseReceiptLogsWithoutStatus(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
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
	if receipt.Status != gethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("failed")
	}
	return nil
}

func positiveParseLogsAliasWithoutStatus(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
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

func positiveParseIndexedLogsWithoutStatus(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
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

func positiveParseDirectIndexedLogsWithoutStatus(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	for i := range receipt.Logs {
		_, err := w.ethConn.ParseLogMessagePublished(*receipt.Logs[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func parseOneLog(c Connector, log gethtypes.Log) error {
	_, err := c.ParseLogMessagePublished(log)
	return err
}

func positiveHelperParameterParseWithoutStatus(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	for _, log := range receipt.Logs {
		if err := parseOneLog(w.ethConn, *log); err != nil {
			return err
		}
	}
	return nil
}

func positiveMsgDerivedFromUncheckedReceipt(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	for _, log := range receipt.Logs {
		event, err := w.ethConn.ParseLogMessagePublished(*log)
		if err != nil {
			return err
		}
		msg := &common.MessagePublication{}
		_ = event
		return w.verifyAndPublish(msg, ctx, tx, receipt)
	}
	return nil
}

func positiveCheckedLogsButDifferentPublishReceipt(w *Watcher, ctx context.Context, msg *common.MessagePublication, tx gethtypes.Hash) error {
	checked, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	if checked.Status != gethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("failed")
	}
	logs := checked.Logs
	for i := range logs {
		_, err := w.ethConn.ParseLogMessagePublished(*logs[i])
		if err != nil {
			return err
		}
	}
	other, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	return w.verifyAndPublish(msg, ctx, tx, other)
}

func positiveLogsAliasFromReassignedReceipt(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, err := w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	logs := receipt.Logs
	receipt, err = w.ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return err
	}
	if receipt.Status != gethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("failed")
	}
	for _, log := range logs {
		if _, err := w.ethConn.ParseLogMessagePublished(*log); err != nil {
			return err
		}
	}
	return nil
}

func positiveMessageEventsUncheckedError(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, _, msgs, _ := MessageEventsForTransaction(ctx, w.ethConn, w.contract, tx)
	for _, msg := range msgs {
		if err := w.verifyAndPublish(msg, ctx, tx, receipt); err != nil {
			return err
		}
	}
	return nil
}

func positiveMessageEventsErrorOverwrittenBeforeCheck(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, _, msgs, err := MessageEventsForTransaction(ctx, w.ethConn, w.contract, tx)
	err = nil
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

func positiveMessageEventsMixedTuples(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, _, _, receiptErr := MessageEventsForTransaction(ctx, w.ethConn, w.contract, tx)
	if receiptErr != nil {
		return receiptErr
	}
	_, _, msgs, messagesErr := MessageEventsForTransaction(ctx, w.ethConn, w.contract, tx)
	if messagesErr != nil {
		return messagesErr
	}
	for _, msg := range msgs {
		if err := w.verifyAndPublish(msg, ctx, tx, receipt); err != nil {
			return err
		}
	}
	return nil
}

func positiveMessageEventsReceiptReassigned(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, _, msgs, err := MessageEventsForTransaction(ctx, w.ethConn, w.contract, tx)
	if err != nil {
		return err
	}
	receipt, err = w.ethConn.TransactionReceipt(ctx, tx)
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

func positiveMessageEventsMessagesReassigned(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, _, msgs, err := MessageEventsForTransaction(ctx, w.ethConn, w.contract, tx)
	if err != nil {
		return err
	}
	msgs = []*common.MessagePublication{{}}
	for _, msg := range msgs {
		if err := w.verifyAndPublish(msg, ctx, tx, receipt); err != nil {
			return err
		}
	}
	return nil
}

func positiveMessageEventsRangeValueReassigned(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, _, msgs, err := MessageEventsForTransaction(ctx, w.ethConn, w.contract, tx)
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		msg = &common.MessagePublication{}
		if err := w.verifyAndPublish(msg, ctx, tx, receipt); err != nil {
			return err
		}
	}
	return nil
}

func positiveMessageEventsSliceElementReassigned(w *Watcher, ctx context.Context, tx gethtypes.Hash) error {
	receipt, _, msgs, err := MessageEventsForTransaction(ctx, w.ethConn, w.contract, tx)
	if err != nil {
		return err
	}
	msgs[0] = &common.MessagePublication{}
	if err := w.verifyAndPublish(msgs[0], ctx, tx, receipt); err != nil {
		return err
	}
	return nil
}
