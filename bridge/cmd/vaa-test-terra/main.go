// vaa-test generates VAA test fixtures used by the ETH devnet tests
package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/certusone/wormhole/bridge/pkg/ethereum"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

type signerInfo struct {
	signer *ecdsa.PrivateKey
	index  int
}

var i = 0

var defaultTargetAddress = vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0}

func main() {

	keys := generateKeys(6)
	for i, key := range keys {
		fmt.Printf("Key [%d]: %s\n", i, crypto.PubkeyToAddress(key.PublicKey).String())
	}

	// 0 - Valid transfer, single signer
	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(2000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         56,
			SourceChain:   1,
			TargetChain:   3,
			SourceAddress: vaa.Address{2, 1, 4},
			TargetAddress: defaultTargetAddress,
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDSolana,
				Address:  hexToAddress("0x347ef34687bdc9f189e87a9200658d9c40e9988"),
				Decimals: 8,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}, []*signerInfo{{keys[0], 0}})

	// 1 - 2 signers, invalid order
	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(2000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         56,
			SourceChain:   1,
			TargetChain:   3,
			SourceAddress: vaa.Address{2, 1, 4},
			TargetAddress: defaultTargetAddress,
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDEthereum,
				Address:  hexToAddress("0xd833215cbcc3f914bd1c9ece3ee7bf8b14f841bb"),
				Decimals: 8,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}, []*signerInfo{{keys[1], 1}, {keys[0], 0}})

	// 2 - Valid transfer, 2 signers
	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(2000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         56,
			SourceChain:   1,
			TargetChain:   3,
			SourceAddress: vaa.Address{2, 1, 4},
			TargetAddress: defaultTargetAddress,
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDEthereum,
				Address:  hexToAddress("0xd833215cbcc3f914bd1c9ece3ee7bf8b14f841bb"),
				Decimals: 8,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}, []*signerInfo{{keys[0], 0}, {keys[1], 1}})

	// 3 - Valid transfer, 3 signers
	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(2000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         56,
			SourceChain:   1,
			TargetChain:   3,
			SourceAddress: vaa.Address{2, 1, 4},
			TargetAddress: defaultTargetAddress,
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDEthereum,
				Address:  hexToAddress("0xd833215cbcc3f914bd1c9ece3ee7bf8b14f841bb"),
				Decimals: 8,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}, []*signerInfo{{keys[0], 0}, {keys[1], 1}, {keys[2], 2}})

	// 4 - Invalid signature, single signer
	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(2000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         56,
			SourceChain:   1,
			TargetChain:   3,
			SourceAddress: vaa.Address{2, 1, 4},
			TargetAddress: defaultTargetAddress,
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDSolana,
				Address:  hexToAddress("0x347ef34687bdc9f189e87a9200658d9c40e9988"),
				Decimals: 8,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}, []*signerInfo{{keys[1], 0}})

	// 5 - Valid guardian set change
	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(2000, 0),
		Payload: &vaa.BodyGuardianSetUpdate{
			Keys: []common.Address{
				crypto.PubkeyToAddress(keys[1].PublicKey),
			},
			NewIndex: 1,
		},
	}, []*signerInfo{{keys[0], 0}})

	// 6 - Change from set 0 to set 1, single guardian with key#2 (zero-based)
	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(2000, 0),
		Payload: &vaa.BodyGuardianSetUpdate{
			Keys: []common.Address{
				crypto.PubkeyToAddress(keys[2].PublicKey),
			},
			NewIndex: 1,
		},
	}, []*signerInfo{{keys[0], 0}})

	// 7 - Guardian set index jump
	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(2000, 0),
		Payload: &vaa.BodyGuardianSetUpdate{
			Keys: []common.Address{
				crypto.PubkeyToAddress(keys[2].PublicKey),
			},
			NewIndex: 2,
		},
	}, []*signerInfo{{keys[0], 0}})

	// 8 - Invalid target address format
	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(2000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         56,
			SourceChain:   1,
			TargetChain:   3,
			SourceAddress: vaa.Address{2, 1, 4},
			TargetAddress: hexToAddress("0x347ef34687bdc9f189e87a9200658d9c40e9988"),
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDSolana,
				Address:  hexToAddress("0x347ef34687bdc9f189e87a9200658d9c40e9988"),
				Decimals: 8,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}, []*signerInfo{{keys[0], 0}})

	// 9 - Amount too high (max u128 + 1)
	amount, ok := new(big.Int).SetString("100000000000000000000000000000000", 16)
	if !ok {
		panic("Cannot convert big amount")
	}
	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(2000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         56,
			SourceChain:   1,
			TargetChain:   3,
			SourceAddress: vaa.Address{2, 1, 4},
			TargetAddress: defaultTargetAddress,
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDSolana,
				Address:  hexToAddress("0x347ef34687bdc9f189e87a9200658d9c40e9988"),
				Decimals: 8,
			},
			Amount: amount,
		},
	}, []*signerInfo{{keys[0], 0}})

	// 10 - Same source and target
	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(1000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         56,
			SourceChain:   3,
			TargetChain:   3,
			SourceAddress: vaa.Address{2, 1, 4},
			TargetAddress: defaultTargetAddress,
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDSolana,
				Address:  hexToAddress("0x347ef34687bdc9f189e87a9200658d9c40e9988"),
				Decimals: 8,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}, []*signerInfo{{keys[0], 0}})

	// 11 - Wrong target chain
	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(1000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         56,
			SourceChain:   1,
			TargetChain:   2,
			SourceAddress: vaa.Address{2, 1, 4},
			TargetAddress: defaultTargetAddress,
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDSolana,
				Address:  hexToAddress("0x347ef34687bdc9f189e87a9200658d9c40e9988"),
				Decimals: 8,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}, []*signerInfo{{keys[0], 0}})

	// 12 - Change guardian set to 6 addresses
	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(4000, 0),
		Payload: &vaa.BodyGuardianSetUpdate{
			Keys: []common.Address{
				crypto.PubkeyToAddress(keys[0].PublicKey),
				crypto.PubkeyToAddress(keys[1].PublicKey),
				crypto.PubkeyToAddress(keys[2].PublicKey),
				crypto.PubkeyToAddress(keys[3].PublicKey),
				crypto.PubkeyToAddress(keys[4].PublicKey),
				crypto.PubkeyToAddress(keys[5].PublicKey),
			},
			NewIndex: 1,
		},
	}, []*signerInfo{{keys[0], 0}})

	// 13 - Valid transfer, partial quorum
	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 1,
		Timestamp:        time.Unix(4000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         57,
			SourceChain:   1,
			TargetChain:   3,
			SourceAddress: vaa.Address{2, 1, 5},
			TargetAddress: defaultTargetAddress,
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDSolana,
				Address:  hexToAddress("0x347ef34687bdc9f189e87a9200658d9c40e9988"),
				Decimals: 8,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}, []*signerInfo{{keys[0], 0}, {keys[1], 1}, {keys[3], 3}, {keys[4], 4}, {keys[5], 5}})
}

func signAndPrintVAA(vaa *vaa.VAA, signers []*signerInfo) {
	for _, signer := range signers {
		vaa.AddSignature(signer.signer, uint8(signer.index))
	}
	vData, err := vaa.Marshal()
	if err != nil {
		panic(err)
	}
	println(i, hex.EncodeToString(vData))
	i++
}

func generateKeys(n int) (keys []*ecdsa.PrivateKey) {
	r := rand.New(rand.NewSource(555))

	for i := 0; i < n; i++ {
		key, err := ecdsa.GenerateKey(crypto.S256(), r)
		if err != nil {
			panic(err)
		}

		keys = append(keys, key)
	}

	return
}

func hexToAddress(hex string) vaa.Address {
	hexAddr := common.HexToAddress(hex)
	return ethereum.PadAddress(hexAddr)
}
