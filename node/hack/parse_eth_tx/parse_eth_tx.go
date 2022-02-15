package main

import (
	"context"
	"flag"
	"github.com/certusone/wormhole/node/pkg/ethereum"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
)

var (
	flagEthRPC       = flag.String("ethRPC", "http://localhost:8545", "Ethereum JSON-RPC endpoint")
	flagContractAddr = flag.String("contractAddr", "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B", "Ethereum contract address")
	flagTx           = flag.String("tx", "", "Transaction to parse")
)

func main() {
	flag.Parse()
	if *flagTx == "" {
		log.Fatal("No transaction specified")
	}

	ctx := context.Background()

	c, err := ethclient.DialContext(ctx, *flagEthRPC)
	if err != nil {
		log.Fatal(err)
	}

	contractAddr := common.HexToAddress(*flagContractAddr)
	transactionHash := common.HexToHash(*flagTx)

	block, msgs, err := ethereum.MessageEventsForTransaction(ctx, c, contractAddr, vaa.ChainIDEthereum, transactionHash)
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
