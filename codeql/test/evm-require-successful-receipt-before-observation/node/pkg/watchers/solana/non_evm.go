package solana

import (
	"context"

	"codeql/evmrequiresuccessfulreceipt/gethtypes"
	"codeql/evmrequiresuccessfulreceipt/node/pkg/common"
)

type Watcher struct{}

func (w *Watcher) verifyAndPublish(msg *common.MessagePublication, ctx context.Context, txHash gethtypes.Hash, receipt *gethtypes.Receipt) error {
	return nil
}

func nonEvm(w *Watcher, ctx context.Context, msg *common.MessagePublication) error {
	return w.verifyAndPublish(msg, ctx, gethtypes.Hash{}, &gethtypes.Receipt{})
}
