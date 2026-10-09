package evm

import (
	"context"

	"codeql/evmrequiresuccessfulreceipt/gethtypes"
	"codeql/evmrequiresuccessfulreceipt/node/pkg/common"
)

func TestExcludedVerifyAndPublish(t interface{ Fatal(...interface{}) }) {
	w := &Watcher{}
	_ = w.verifyAndPublish(&common.MessagePublication{}, context.Background(), gethtypes.Hash{}, &gethtypes.Receipt{})
}
