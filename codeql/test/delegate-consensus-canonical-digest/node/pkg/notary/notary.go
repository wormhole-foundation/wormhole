package notary

import (
	whcommon "github.com/wormhole-foundation/wormhole/node/pkg/common"
	whcrypto "github.com/wormhole-foundation/wormhole/node/pkg/crypto"
)

func serializationOutsideProcessor(mp *whcommon.MessagePublication) string {
	buf, _ := mp.MarshalBinary()
	return whcrypto.Keccak256Hash(buf).Hex()
}
