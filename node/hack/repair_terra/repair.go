//nolint:noctx // this is a hack
package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/wormhole-foundation/wormhole/sdk"
	"go.uber.org/zap"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/certusone/wormhole/node/pkg/watchers/cosmwasm"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/tidwall/gjson"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var fcdMap = map[vaa.ChainID]string{
	vaa.ChainIDTerra:  "https://fcd.terra.dev",
	vaa.ChainIDTerra2: "https://phoenix-fcd.terra.dev",
	vaa.ChainIDXpla:   "https://dimension-fcd.xpla.dev",
}

var coreContractMap = map[vaa.ChainID]string{
	vaa.ChainIDTerra:  "terra1dq03ugtd40zu9hcgdzrsq6z2z4hwhc9tqk2uy5",
	vaa.ChainIDTerra2: "terra12mrnzvhx3rpej6843uge2yyfppfyd3u9c3uq223q8sl48huz9juqffcnhp",
	vaa.ChainIDXpla:   "xpla1jn8qmdda5m6f6fqu9qv46rt7ajhklg40ukpqchkejcvy8x7w26cqxamv3w",
}

var emitterMap = map[vaa.ChainID]string{
	vaa.ChainIDTerra:  "0000000000000000000000007cf7b764e38a0a5e967972c1df77d432510564e2",
	vaa.ChainIDTerra2: "a463ad028fb79679cfc8ce1efba35ac0e77b35080a1abe9bebe83461f176b0a3",
	vaa.ChainIDXpla:   "8f9cf727175353b17a5f574270e370776123d90fd74956ae4277962b4fdee24c",
}

type Emitter struct {
	ChainID vaa.ChainID
	Emitter string
}

var (
	adminRPC  = flag.String("adminRPC", "/run/guardiand/admin.socket", "Admin RPC address")
	chain     = flag.String("chain", "terra", "CosmWasm Chain name")
	dryRun    = flag.Bool("dryRun", true, "Dry run")
	sleepTime = flag.Int("sleepTime", 1, "Time to sleep between http requests")
)

func getAdminClient(ctx context.Context, addr string) (*grpc.ClientConn, error, nodev1.NodePrivilegedServiceClient) {
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("unix:///%s", addr), grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("failed to connect to %s: %v", addr, err)
	}

	c := nodev1.NewNodePrivilegedServiceClient(conn)
	return conn, err, c
}

func getSequencesForTxhash(txhash string, fcd string, contractAddressLogKey string, coreContract string, emitter Emitter, chainID vaa.ChainID) ([]uint64, error) {
	client := &http.Client{
		Timeout: time.Second * 5,
	}
	url := fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", fcd, txhash)
	resp, err := client.Get(url)
	if err != nil {
		return []uint64{}, fmt.Errorf("failed to get message: %w", err)
	}
	defer resp.Body.Close()
	txBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return []uint64{}, fmt.Errorf("failed to read message: %w", err)
	}

	txJSON := string(txBody)
	if !gjson.Valid(txJSON) {
		return []uint64{}, fmt.Errorf("invalid JSON response")
	}
	txHashRaw := gjson.Get(txJSON, "tx_response.txhash")
	if !txHashRaw.Exists() {
		return []uint64{}, fmt.Errorf("cosmwasm tx does not have tx hash")
	}
	txHash := txHashRaw.String()

	events := gjson.Get(txJSON, "tx_response.events")
	if !events.Exists() {
		return []uint64{}, fmt.Errorf("cosmwasm tx has no events")
	}
	msgs := EventsToMessagePublications(coreContract, txHash, events.Array(), chainID, contractAddressLogKey)
	// Should only ever be 1 message. Stole the above function from watcher.go
	var sequences = []uint64{}
	for _, msg := range msgs {
		tokenBridgeEmitter, err := vaa.StringToAddress(emitter.Emitter)
		if err != nil {
			log.Fatalf("Emitter address is not valid: %s", emitter.Emitter)
		}
		if msg.EmitterAddress == tokenBridgeEmitter {
			sequences = append(sequences, msg.Sequence)
		}
	}
	return sequences, nil
}

// This was stolen from pkg/cosmwasm/watcher.go
func EventsToMessagePublications(contract string, txHash string, events []gjson.Result, chainID vaa.ChainID, contractAddressLogKey string) []*common.MessagePublication {
	msgs := make([]*common.MessagePublication, 0, len(events))
	for _, event := range events {
		if !event.IsObject() {
			log.Println("event is invalid", zap.String("tx_hash", txHash), zap.String("event", event.String()))
			continue
		}
		eventType := gjson.Get(event.String(), "type")
		if eventType.String() != "wasm" {
			continue
		}

		attributes := gjson.Get(event.String(), "attributes")
		if !attributes.Exists() {
			log.Println("message event has no attributes", zap.String("tx_hash", txHash), zap.String("event", event.String()))
			continue
		}
		mappedAttributes := map[string]string{}
		for _, attribute := range attributes.Array() {
			if !attribute.IsObject() {
				log.Println("event attribute is invalid", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}
			keyBase := gjson.Get(attribute.String(), "key")
			if !keyBase.Exists() {
				log.Println("event attribute does not have key", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}
			valueBase := gjson.Get(attribute.String(), "value")
			if !valueBase.Exists() {
				log.Println("event attribute does not have value", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}

			key, err := base64.StdEncoding.DecodeString(keyBase.String())
			if err != nil {
				log.Println("event key attribute is invalid", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()))
				continue
			}
			value, err := base64.StdEncoding.DecodeString(valueBase.String())
			if err != nil {
				log.Println("event value attribute is invalid", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
				continue
			}

			if _, ok := mappedAttributes[string(key)]; ok {
				log.Println("duplicate key in events", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
				continue
			}

			mappedAttributes[string(key)] = string(value)
		}

		contractAddress, ok := mappedAttributes[contractAddressLogKey]
		if !ok {
			log.Println("wasm event without contract address field set", zap.String("event", event.String()))
			continue
		}
		// This is not a wormhole message
		if contractAddress != contract {
			continue
		}

		payload, ok := mappedAttributes["message.message"]
		if !ok {
			log.Println("wormhole event does not have a message field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		sender, ok := mappedAttributes["message.sender"]
		if !ok {
			log.Println("wormhole event does not have a sender field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		nonce, ok := mappedAttributes["message.nonce"]
		if !ok {
			log.Println("wormhole event does not have a nonce field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		sequence, ok := mappedAttributes["message.sequence"]
		if !ok {
			log.Println("wormhole event does not have a sequence field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		blockTime, ok := mappedAttributes["message.block_time"]
		if !ok {
			log.Println("wormhole event does not have a block_time field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}

		senderAddress, err := cosmwasm.StringToAddress(sender)
		if err != nil {
			log.Println("cannot decode emitter hex", zap.String("tx_hash", txHash), zap.String("value", sender))
			continue
		}
		txHashValue, err := cosmwasm.StringToHash(txHash)
		if err != nil {
			log.Println("cannot decode tx hash hex", zap.String("tx_hash", txHash), zap.String("value", txHash))
			continue
		}
		payloadValue, err := hex.DecodeString(payload)
		if err != nil {
			log.Println("cannot decode payload", zap.String("tx_hash", txHash), zap.String("value", payload))
			continue
		}

		blockTimeInt, err := strconv.ParseInt(blockTime, 10, 64)
		if err != nil {
			log.Println("blocktime cannot be parsed as int", zap.String("tx_hash", txHash), zap.String("value", blockTime))
			continue
		}
		nonceInt, err := strconv.ParseUint(nonce, 10, 32)
		if err != nil {
			log.Println("nonce cannot be parsed as int", zap.String("tx_hash", txHash), zap.String("value", blockTime))
			continue
		}
		sequenceInt, err := strconv.ParseUint(sequence, 10, 64)
		if err != nil {
			log.Println("sequence cannot be parsed as int", zap.String("tx_hash", txHash), zap.String("value", blockTime))
			continue
		}
		messagePublication := &common.MessagePublication{
			TxID:             txHashValue.Bytes(),
			Timestamp:        time.Unix(blockTimeInt, 0),
			Nonce:            uint32(nonceInt),
			Sequence:         sequenceInt,
			EmitterChain:     chainID,
			EmitterAddress:   senderAddress,
			Payload:          payloadValue,
			ConsistencyLevel: 0, // Instant finality
		}
		msgs = append(msgs, messagePublication)
	}
	return msgs
}

func main() {
	flag.Parse()

	chainID, err := vaa.ChainIDFromString(*chain)
	if err != nil {
		log.Fatalf("Invalid chain: %v", err)
	}

	fcd, ok := fcdMap[chainID]
	if !ok {
		log.Fatal("Unsupported chain: no FCD defined")
	}

	coreContract, ok := coreContractMap[chainID]
	if !ok {
		log.Fatal("Unsupported chain: no core contract defined")
	}

	emitterAddress, ok := emitterMap[chainID]
	if !ok {
		log.Fatal("Unsupported chain: no emitter defined")
	}
	emitter := Emitter{chainID, emitterAddress}

	// CosmWasm 1.0.0
	contractAddressLogKey := "_contract_address"
	if chainID == vaa.ChainIDTerra {
		// CosmWasm <1.0.0
		contractAddressLogKey = "contract_address"
	}

	ctx := context.Background()

	missingMessages := make(map[uint64]bool)

	conn, err, admin := getAdminClient(ctx, *adminRPC)
	defer conn.Close()
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}

	log.Printf("Requesting missing messages for %s", emitter.Emitter)

	msg := nodev1.FindMissingMessagesRequest{
		EmitterChain:   uint32(chainID),
		EmitterAddress: emitter.Emitter,
		RpcBackfill:    true,
		BackfillNodes:  sdk.PublicRPCEndpoints,
	}
	resp, err := admin.FindMissingMessages(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run find FindMissingMessages RPC: %v", err)
	}

	msgs := make([]*db.VAAID, len(resp.MissingMessages))
	for i, id := range resp.MissingMessages {
		vId, err := db.VaaIDFromString(id)
		if err != nil {
			log.Fatalf("failed to parse VAAID: %v", err)
		}
		msgs[i] = vId
	}

	if len(msgs) == 0 {
		log.Printf("No missing messages found for %s", emitter)
		return
	}

	lowest := msgs[0].Sequence
	highest := msgs[len(msgs)-1].Sequence

	log.Printf("Found %d missing messages for %s: %d - %d", len(msgs), emitter, lowest, highest)

	for _, msg := range msgs {
		missingMessages[msg.Sequence] = true
	}

	limiter := rate.NewLimiter(rate.Every(time.Duration(*sleepTime)*time.Second), 1)

	log.Printf("Starting search for missing sequence numbers (sleeping %ds between requests)...", *sleepTime)
	var offset uint64 = 0

	var firstTime bool = true
	for (offset > 0) || firstTime {
		if err := limiter.Wait(ctx); err != nil {
			log.Fatalf("failed to wait: %v", err)
		}

		firstTime = false
		client := &http.Client{
			Timeout: time.Second * 5,
		}
		resp, err := client.Get(fmt.Sprintf("%s/v1/txs?offset=%d&limit=100&account=%s", fcd, offset, coreContract))
		if err != nil {
			log.Fatalf("failed to get log: %v", err)
			continue
		}
		defer resp.Body.Close()

		blocksBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("failed to read log: %v", err)
			continue
		}

		blockJSON := string(blocksBody)
		if !gjson.Valid(blockJSON) {
			log.Println("invalid JSON response")
			continue
		}
		next := gjson.Get(blockJSON, "next")
		log.Println("next block", next.Int())
		offset = next.Uint()
		// Get the transactions.  Should be 100 of them
		txs := gjson.Get(blockJSON, "txs")
		for _, tx := range txs.Array() {
			if !tx.IsObject() {
				log.Fatalln("Bad Object")
				continue
			}
			txhash := gjson.Get(tx.String(), "txhash")
			// Get sequence number for tx
			seqs, err := getSequencesForTxhash(txhash.String(), fcd, contractAddressLogKey, coreContract, emitter, chainID)
			if err != nil {
				log.Fatalln("Failed getting sequence number", err)
				continue
			}
			for _, seq := range seqs {
				// Check to see if this is a missing sequence number
				if !missingMessages[seq] {
					continue
				}
				log.Println("txhash", txhash.String(), "sequence number", seq)
				// send observation request to guardian
				if *dryRun {
					log.Println("Would have sent txhash", txhash, "to the guardian to re-observe")
				} else {
					txHashAsByteArray, err := cosmwasm.StringToHash(txhash.String())
					if err != nil {
						log.Fatalln("Couldn't decode the txhash", txhash)
					} else {
						_, err = admin.SendObservationRequest(ctx, &nodev1.SendObservationRequestRequest{
							ObservationRequest: &gossipv1.ObservationRequest{
								ChainId: uint32(chainID),
								TxHash:  txHashAsByteArray.Bytes(),
							}})
						if err != nil {
							log.Fatalf("SendObservationRequest: %v", err)
						}
					}
				}
				if seq <= lowest {
					// We are done
					log.Println("Finished!")
					return
				}
			}
		}
	}
}
