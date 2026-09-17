package tools

import "github.com/wormhole-foundation/wormhole/codeqltest/messagepublication/node/pkg/common"

func DeprecatedHelpersOutsideNodeAreOutOfScope(msg *common.MessagePublication, buf []byte) {
	_, _ = msg.Marshal()
	_, _ = common.UnmarshalMessagePublication(buf)
}
