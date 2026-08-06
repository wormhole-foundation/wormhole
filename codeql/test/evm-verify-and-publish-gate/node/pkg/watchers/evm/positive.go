package evm

import "github.com/certusone/wormhole/node/pkg/common"

func positiveDirectSend(w *Watcher, msg *common.MessagePublication) {
	w.msgC <- msg
}

func positiveAliasSend(w *Watcher, msg *common.MessagePublication) {
	out := w.msgC
	out <- msg
}

func positiveThinHelper(w *Watcher, msg *common.MessagePublication) {
	publishDirect(w.msgC, msg)
}

func publishDirect(out chan<- *common.MessagePublication, msg *common.MessagePublication) {
	out <- msg
}

func positiveAliasChain(w *Watcher, msg *common.MessagePublication) {
	first := w.msgC
	second := first
	second <- msg
}

func positiveClosureOverwriteDoesNotAffectOuterAlias(w *Watcher, msg *common.MessagePublication) {
	out := w.msgC
	unused := func() {
		out = make(chan *common.MessagePublication, 1)
	}
	_ = unused
	out <- msg
}
