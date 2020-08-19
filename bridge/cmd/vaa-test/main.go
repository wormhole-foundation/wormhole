package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/certusone/wormhole/bridge/pkg/devnet"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

func main() {
	addr := devnet.GanacheClientDefaultAccountAddress
	addrP := common.LeftPadBytes(addr[:], 32)
	addrTarget := vaa.Address{}
	copy(addrTarget[:], addrP)

	tAddr := common.HexToAddress("0x0000000000000000000000009561c133dd8580860b6b7e504bc5aa500f0f06a7")
	tAddrP := common.LeftPadBytes(tAddr[:], 32)
	tAddrTarget := vaa.Address{}
	copy(tAddrTarget[:], tAddrP)
	v := &vaa.VAA{
		Version:          1,
		GuardianSetIndex: 2,
		Timestamp:        time.Unix(4000, 0),
		Payload: &vaa.BodyTransfer{
			Nonce:         56,
			SourceChain:   1,
			TargetChain:   2,
			SourceAddress: vaa.Address{2, 1, 4},
			TargetAddress: addrTarget,
			Asset: &vaa.AssetMeta{
				Chain:   vaa.ChainIDSolana,
				Address: tAddrTarget,
			},
			Amount: big.NewInt(1000000000000000000),
		},
	}

	r := rand.New(rand.NewSource(555))
	key, err := ecdsa.GenerateKey(crypto.S256(), r)
	if err != nil {
		panic(err)
	}
	key2, err := ecdsa.GenerateKey(crypto.S256(), r)
	if err != nil {
		panic(err)
	}
	key3, err := ecdsa.GenerateKey(crypto.S256(), r)
	if err != nil {
		panic(err)
	}
	key4, err := ecdsa.GenerateKey(crypto.S256(), r)
	if err != nil {
		panic(err)
	}
	key5, err := ecdsa.GenerateKey(crypto.S256(), r)
	if err != nil {
		panic(err)
	}
	key6, err := ecdsa.GenerateKey(crypto.S256(), r)
	if err != nil {
		panic(err)
	}

	//v = &vaa.VAA{
	//	Version:          1,
	//	GuardianSetIndex: 1,
	//	Timestamp:        time.Unix(5000, 0),
	//	Payload: &vaa.BodyGuardianSetUpdate{
	//		Keys:     []common.Address{
	//			crypto.PubkeyToAddress(key.PublicKey),
	//			crypto.PubkeyToAddress(key2.PublicKey),
	//			crypto.PubkeyToAddress(key3.PublicKey),
	//			crypto.PubkeyToAddress(key4.PublicKey),
	//			crypto.PubkeyToAddress(key5.PublicKey),
	//			crypto.PubkeyToAddress(key6.PublicKey),
	//		},
	//		NewIndex: 2,
	//	},
	//}

	v.AddSignature(key, 0)
	v.AddSignature(key2, 1)
	v.AddSignature(key3, 2)
	v.AddSignature(key5, 4)
	v.AddSignature(key6, 5)
	sigAddr := crypto.PubkeyToAddress(key.PublicKey)
	println(sigAddr.String())
	println(crypto.PubkeyToAddress(key2.PublicKey).String())
	println(crypto.PubkeyToAddress(key3.PublicKey).String())
	println(crypto.PubkeyToAddress(key4.PublicKey).String())
	println(crypto.PubkeyToAddress(key5.PublicKey).String())
	println(crypto.PubkeyToAddress(key6.PublicKey).String())

	vData, err := v.Marshal()
	if err != nil {
		panic(err)
	}

	println(hex.EncodeToString(vData))
}
