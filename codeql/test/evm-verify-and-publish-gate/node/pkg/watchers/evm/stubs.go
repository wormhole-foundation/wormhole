package evm

import (
	"context"

	"github.com/certusone/wormhole/node/pkg/common"
)

type Hash [32]byte

type Receipt struct{}

type Watcher struct {
	msgC chan<- *common.MessagePublication
}

func NewWatcher(msgC chan<- *common.MessagePublication) *Watcher {
	return &Watcher{msgC: msgC}
}

func (w *Watcher) verifyAndPublish(msg *common.MessagePublication, ctx context.Context, txHash Hash, receipt *Receipt) error {
	w.msgC <- msg
	return nil
}

func makeMessage() *common.MessagePublication {
	return &common.MessagePublication{}
}
