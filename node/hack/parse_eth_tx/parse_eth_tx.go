// To compile:
//   go build --ldflags '-extldflags "-Wl,--allow-multiple-definition"' -o parse_eth_tx
// Usage:
//   ./parse_eth_tx -chainID=14 -ethRPC=wss://alfajores-forno.celo-testnet.org/ws -contractAddr=0x88505117CA88e7dd2eC6EA1E13f0948db2D50D56 -tx=0x20a1e7e491dd82b6b33db0820e88a96b58bac28d65770ea73af80e457745aab1

package main

import (
	"context"
	"flag"
	"log"

	"github.com/certusone/wormhole/node/pkg/celo"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/ethereum"
	"github.com/certusone/wormhole/node/pkg/vaa"
	ethCommon "github.com/ethereum/go-ethereum/common"
)

var (
	flagChainID      = flag.Int("chainID", 2, "Wormhole chain ID")
	flagEthRPC       = flag.String("ethRPC", "http://localhost:8545", "Ethereum JSON-RPC endpoint")
	flagContractAddr = flag.String("contractAddr", "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B", "Ethereum contract address")
	flagTx           = flag.String("tx", "", "Transaction to parse")
)

func main() {
	flag.Parse()
	if *flagTx == "" {
		log.Fatal("No transaction specified")
	}

	chainID := vaa.ChainID(*flagChainID)

	ctx := context.Background()

	var ethIntf common.Ethish
	if chainID == vaa.ChainIDCelo {
		ethIntf = &celo.CeloImpl{NetworkName: "celo"}
	} else {
		ethIntf = &ethereum.EthImpl{NetworkName: "eth"}
	}

	err := ethIntf.DialContext(ctx, *flagEthRPC)
	if err != nil {
		log.Fatal(err)
	}

	contractAddr := ethCommon.HexToAddress(*flagContractAddr)
	transactionHash := ethCommon.HexToHash(*flagTx)

	err = ethIntf.NewAbiFilterer(contractAddr)
	if err != nil {
		log.Fatal(err)
	}

	block, msgs, err := ethereum.MessageEventsForTransaction(ctx, ethIntf, contractAddr, chainID, transactionHash)
	if err != nil {
		log.Fatal(err)
	}

	if len(msgs) == 0 {
		log.Fatal("No messages found")
	}

	for _, k := range msgs {
		v := &vaa.VAA{
			Version:          vaa.SupportedVAAVersion,
			GuardianSetIndex: 1,
			Signatures:       nil,
			Timestamp:        k.Timestamp,
			Nonce:            k.Nonce,
			EmitterChain:     k.EmitterChain,
			EmitterAddress:   k.EmitterAddress,
			Payload:          k.Payload,
			Sequence:         k.Sequence,
			ConsistencyLevel: k.ConsistencyLevel,
		}

		log.Println("------------------------------------------------------")
		log.Printf("Block: %d", block)
		log.Printf("Message ID: %s", v.MessageID())
		log.Printf("Digest: %s", v.HexDigest())
		log.Printf("VAA: %+v", v)
	}
}
