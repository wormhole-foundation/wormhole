package processor

import (
	whcommon "github.com/wormhole-foundation/wormhole/node/pkg/common"
	whcrypto "github.com/wormhole-foundation/wormhole/node/pkg/crypto"
)

func testHistoricalFixtureIsOutOfScope(p *processor, mp *whcommon.MessagePublication) {
	buf, _ := mp.MarshalBinary()
	hash := whcrypto.Keccak256Hash(buf).Hex()
	_ = p.delegateState.observations[hash]
}
