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

	"github.com/certusone/wormhole/bridge/pkg/devnet"
	"github.com/certusone/wormhole/bridge/pkg/ethereum"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

type signerInfo struct {
	signer *ecdsa.PrivateKey
	index  int
}

var i = 0

func main() {

	keys := generateKeys(6)
	for i, key := range keys {
		fmt.Printf("Key [%d]: %s\n", i, crypto.PubkeyToAddress(key.PublicKey).String())
	}

	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(2000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         56,
			SourceChain:   1,
			TargetChain:   2,
			SourceAddress: vaa.Address{2, 1, 4},
			TargetAddress: ethereum.PadAddress(devnet.GanacheClientDefaultAccountAddress),
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDSolana,
				Address:  hexToAddress("0x347ef34687bdc9f189e87a9200658d9c40e9988"),
				Decimals: 8,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}, []*signerInfo{{keys[0], 0}})

	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(2000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         56,
			SourceChain:   1,
			TargetChain:   2,
			SourceAddress: vaa.Address{2, 1, 4},
			TargetAddress: ethereum.PadAddress(devnet.GanacheClientDefaultAccountAddress),
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDEthereum,
				Address:  hexToAddress("0xd833215cbcc3f914bd1c9ece3ee7bf8b14f841bb"),
				Decimals: 8,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}, []*signerInfo{{keys[0], 0}})

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

	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(1000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         56,
			SourceChain:   1,
			TargetChain:   2,
			SourceAddress: vaa.Address{2, 1, 4},
			TargetAddress: ethereum.PadAddress(devnet.GanacheClientDefaultAccountAddress),
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDSolana,
				Address:  hexToAddress("0x347ef34687bdc9f189e87a9200658d9c40e9988"),
				Decimals: 8,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}, []*signerInfo{{keys[0], 0}})

	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 0,
		Timestamp:        time.Unix(2000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         56,
			SourceChain:   1,
			TargetChain:   2,
			SourceAddress: vaa.Address{2, 1, 5},
			TargetAddress: ethereum.PadAddress(devnet.GanacheClientDefaultAccountAddress),
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDSolana,
				Address:  hexToAddress("0x347ef34687bdc9f189e87a9200658d9c40e9988"),
				Decimals: 8,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}, []*signerInfo{{keys[0], 0}})

	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 1,
		Timestamp:        time.Unix(2000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         56,
			SourceChain:   1,
			TargetChain:   2,
			SourceAddress: vaa.Address{2, 1, 5},
			TargetAddress: ethereum.PadAddress(devnet.GanacheClientDefaultAccountAddress),
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDSolana,
				Address:  hexToAddress("0x347ef34687bdc9f189e87a9200658d9c40e9988"),
				Decimals: 8,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}, []*signerInfo{{keys[1], 0}})

	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 1,
		Timestamp:        time.Unix(2000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         57,
			SourceChain:   1,
			TargetChain:   2,
			SourceAddress: vaa.Address{2, 1, 5},
			TargetAddress: ethereum.PadAddress(devnet.GanacheClientDefaultAccountAddress),
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDSolana,
				Address:  hexToAddress("0x347ef34687bdc9f189e87a9200658d9c40e9988"),
				Decimals: 8,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}, []*signerInfo{{keys[0], 0}})

	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 1,
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
			NewIndex: 2,
		},
	}, []*signerInfo{{keys[1], 0}})

	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 2,
		Timestamp:        time.Unix(4000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         57,
			SourceChain:   1,
			TargetChain:   2,
			SourceAddress: vaa.Address{2, 1, 5},
			TargetAddress: ethereum.PadAddress(devnet.GanacheClientDefaultAccountAddress),
			Asset: &vaa.AssetMeta{
				Chain:    vaa.ChainIDSolana,
				Address:  hexToAddress("0x347ef34687bdc9f189e87a9200658d9c40e9988"),
				Decimals: 8,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}, []*signerInfo{{keys[0], 0}, {keys[1], 1}, {keys[2], 2}})

	signAndPrintVAA(&vaa.VAA{
		Version:          1,
		GuardianSetIndex: 2,
		Timestamp:        time.Unix(4000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         57,
			SourceChain:   1,
			TargetChain:   2,
			SourceAddress: vaa.Address{2, 1, 5},
			TargetAddress: ethereum.PadAddress(devnet.GanacheClientDefaultAccountAddress),
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
