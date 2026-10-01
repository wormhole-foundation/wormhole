package evm

import "github.com/certusone/wormhole/node/pkg/common"

func TestDirectSendFixtureIsExcluded(t interface{ Fatal(args ...any) }) {
	msgC := make(chan *common.MessagePublication, 1)
	w := NewWatcher(msgC)
	w.msgC <- &common.MessagePublication{}
	<-msgC
}
