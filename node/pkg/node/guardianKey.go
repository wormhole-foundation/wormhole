package node

import (
	"crypto/ecdsa"

	eth_common "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func GuardianKeyToAddress(publicKey ecdsa.PublicKey) eth_common.Address {
	return ethcrypto.PubkeyToAddress(publicKey)
}
