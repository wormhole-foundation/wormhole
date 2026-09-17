package solana

import "github.com/certusone/wormhole/node/pkg/common"

type Watcher struct {
	msgC chan<- *common.MessagePublication
}

func NonEvmDirectSend(w *Watcher, msg *common.MessagePublication) {
	w.msgC <- msg
}
