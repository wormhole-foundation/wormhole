// To compile:
//   go build -o parse_eth_tx
// Usage:
//   ./parse_eth_tx -chainID=14 -ethRPC=wss://alfajores-forno.celo-testnet.org/ws -contractAddr=0x88505117CA88e7dd2eC6EA1E13f0948db2D50D56 -tx=0x20a1e7e491dd82b6b33db0820e88a96b58bac28d65770ea73af80e457745aab1

package main

import (
	"context"
	"flag"
	"log"
	"math"

	"github.com/certusone/wormhole/node/pkg/watchers/evm"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"go.uber.org/zap"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
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

	if *flagChainID > math.MaxUint16 {
		log.Fatalf("chain id is not a valid uint16: %d", *flagChainID)
	}

	chainID := vaa.ChainID(*flagChainID) // #nosec G115 -- This is validated above

	ctx := context.Background()

	contractAddr := ethCommon.HexToAddress(*flagContractAddr)

	var ethIntf connectors.Connector
	var err error
	ethIntf, err = connectors.NewEthereumBaseConnector(ctx, "", *flagEthRPC, contractAddr, zap.L())
	if err != nil {
		log.Fatalf("dialing eth client failed: %v", err)
	}

	transactionHash := ethCommon.HexToHash(*flagTx)

	_, block, msgs, err := evm.MessageEventsForTransaction(ctx, ethIntf, contractAddr, chainID, transactionHash)
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
