package evm

import (
	"context"

	"github.com/certusone/wormhole/node/pkg/common"
)

func negativeApprovedHelperCall(w *Watcher, msg *common.MessagePublication) error {
	return w.verifyAndPublish(msg, context.Background(), Hash{}, &Receipt{})
}

func negativeConstructorWiring(msgC chan<- *common.MessagePublication) *Watcher {
	return NewWatcher(msgC)
}

func negativeUnrelatedChannel(msg *common.MessagePublication) {
	internal := make(chan *common.MessagePublication, 1)
	internal <- msg
}

func negativeMessageOnlyHelper() *common.MessagePublication {
	return makeMessage()
}

func negativeChannelRead(in <-chan *common.MessagePublication) *common.MessagePublication {
	return <-in
}

func negativeSendBeforeProtectedAssignment(w *Watcher, msg *common.MessagePublication) {
	var out chan<- *common.MessagePublication = make(chan *common.MessagePublication, 1)
	out <- msg
	out = w.msgC
}

func negativeSendAfterUnrelatedReassignment(w *Watcher, msg *common.MessagePublication) {
	out := w.msgC
	out = make(chan *common.MessagePublication, 1)
	out <- msg
}

func negativeUnsupportedClosureCapture(w *Watcher, msg *common.MessagePublication) {
	out := w.msgC
	publish := func() {
		out <- msg
	}
	publish()
}

func negativeUnsupportedClosureOverwriteBeforeCall(w *Watcher, msg *common.MessagePublication) {
	out := w.msgC
	publish := func() {
		out <- msg
	}
	out = make(chan *common.MessagePublication, 1)
	publish()
}

type wrapperPublisher struct {
	out chan<- *common.MessagePublication
}

func newWrapperPublisher(w *Watcher) wrapperPublisher {
	return wrapperPublisher{out: w.msgC}
}

func (p wrapperPublisher) Publish(msg *common.MessagePublication) {
	p.out <- msg
}

func negativeUnsupportedWrapperBoundary(w *Watcher, msg *common.MessagePublication) {
	publisher := newWrapperPublisher(w)
	publisher.Publish(msg)
}
