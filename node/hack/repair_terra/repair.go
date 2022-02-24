package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"go.uber.org/zap"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/certusone/wormhole/node/pkg/terra"
	"github.com/certusone/wormhole/node/pkg/vaa"

	"github.com/tidwall/gjson"
	"google.golang.org/grpc"
)

var (
	adminRPC   = flag.String("adminRPC", "/run/guardiand/admin.socket", "Admin RPC address")
	terraAddr  = flag.String("terraProgram", "terra1dq03ugtd40zu9hcgdzrsq6z2z4hwhc9tqk2uy5", "Terra program address")
	dryRun     = flag.Bool("dryRun", true, "Dry run")
	TerraEmitter = struct {
		ChainID vaa.ChainID
		Emitter string
	}{vaa.ChainIDTerra, "0000000000000000000000007cf7b764e38a0a5e967972c1df77d432510564e2"}
)

func getAdminClient(ctx context.Context, addr string) (*grpc.ClientConn, error, nodev1.NodePrivilegedServiceClient) {
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("unix:///%s", addr), grpc.WithInsecure())

	if err != nil {
		log.Fatalf("failed to connect to %s: %v", addr, err)
	}

	c := nodev1.NewNodePrivilegedServiceClient(conn)
	return conn, err, c
}

func getSequenceForTxhash(txhash string) (uint64, error) {
	client := &http.Client{ 
		Timeout: time.Second * 5, 
	}
	url := fmt.Sprintf("https://fcd.terra.dev/cosmos/tx/v1beta1/txs/%s", txhash)
	resp, err := client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to get message: %w", err)
	}
	defer resp.Body.Close()
	txBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read message: %w", err)
	}

	txJSON := string(txBody)
	if !gjson.Valid(txJSON) {
		return 0, fmt.Errorf("invalid JSON response")
	}
	txHashRaw := gjson.Get(txJSON, "tx_response.txhash")
	if !txHashRaw.Exists() {
		return 0, fmt.Errorf("terra tx does not have tx hash")
	}
	txHash := txHashRaw.String()

	events := gjson.Get(txJSON, "tx_response.events")
	if !events.Exists() {
		return 0, fmt.Errorf("terra tx has no events")
	}
	msgs := EventsToMessagePublications(*terraAddr, txHash, events.Array())
	// Should only ever be 1 message. Stole the above function from watcher.go
	if len(msgs) != 1 {
		return 0, fmt.Errorf("EventsToMessagePublications returned %d msgs", len(msgs))
	}
	return msgs[0].Sequence, nil
}

// This was stolen from pkg/terra/watcher.go
func EventsToMessagePublications(contract string, txHash string, events []gjson.Result) []*common.MessagePublication {
	msgs := make([]*common.MessagePublication, 0, len(events))
	for _, event := range events {
		if !event.IsObject() {
			log.Println("terra event is invalid", zap.String("tx_hash", txHash), zap.String("event", event.String()))
			continue
		}
		eventType := gjson.Get(event.String(), "type")
		if eventType.String() != "wasm" {
			continue
		}

		attributes := gjson.Get(event.String(), "attributes")
		if !attributes.Exists() {
			log.Println("terra message event has no attributes", zap.String("tx_hash", txHash), zap.String("event", event.String()))
			continue
		}
		mappedAttributes := map[string]string{}
		for _, attribute := range attributes.Array() {
			if !attribute.IsObject() {
				log.Println("terra event attribute is invalid", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}
			keyBase := gjson.Get(attribute.String(), "key")
			if !keyBase.Exists() {
				log.Println("terra event attribute does not have key", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}
			valueBase := gjson.Get(attribute.String(), "value")
			if !valueBase.Exists() {
				log.Println("terra event attribute does not have value", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}

			key, err := base64.StdEncoding.DecodeString(keyBase.String())
			if err != nil {
				log.Println("terra event key attribute is invalid", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()))
				continue
			}
			value, err := base64.StdEncoding.DecodeString(valueBase.String())
			if err != nil {
				log.Println("terra event value attribute is invalid", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
				continue
			}

			if _, ok := mappedAttributes[string(key)]; ok {
				log.Println("duplicate key in events", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
				continue
			}

			mappedAttributes[string(key)] = string(value)
		}

		contractAddress, ok := mappedAttributes["contract_address"]
		if !ok {
			log.Println("terra wasm event without contract address field set", zap.String("event", event.String()))
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

		senderAddress, err := terra.StringToAddress(sender)
		if err != nil {
			log.Println("cannot decode emitter hex", zap.String("tx_hash", txHash), zap.String("value", sender))
			continue
		}
		txHashValue, err := terra.StringToHash(txHash)
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
			TxHash:           txHashValue,
			Timestamp:        time.Unix(blockTimeInt, 0),
			Nonce:            uint32(nonceInt),
			Sequence:         sequenceInt,
			EmitterChain:     vaa.ChainIDTerra,
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

	ctx := context.Background()

	missingMessages := make(map[uint64]bool)

	conn, err, admin := getAdminClient(ctx, *adminRPC)
	defer conn.Close()
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}

	log.Printf("Requesting missing messages for %s", TerraEmitter.Emitter)

	msg := nodev1.FindMissingMessagesRequest{
		EmitterChain:   uint32(vaa.ChainIDTerra),
		EmitterAddress: TerraEmitter.Emitter,
		RpcBackfill:    true,
		BackfillNodes:  common.PublicRPCEndpoints,
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
		log.Printf("No missing messages found for %s", TerraEmitter)
		return
	}

	lowest := msgs[0].Sequence
	highest := msgs[len(msgs)-1].Sequence

	log.Printf("Found %d missing messages for %s: %d - %d", len(msgs), TerraEmitter, lowest, highest)

	for _, msg := range msgs {
		missingMessages[msg.Sequence] = true
	}

	log.Printf("Starting search for missing sequence numbers...")
	offset := 0
	var firstTime bool = true
	for (offset > 0) || firstTime {
		firstTime = false
		client := &http.Client{ 
			Timeout: time.Second * 5, 
		}
		resp, err := client.Get(fmt.Sprintf("https://fcd.terra.dev/v1/txs?offset=%d&limit=100&account=%s", offset, *terraAddr))
		if err != nil {
			log.Fatalf("failed to get log: %v", err)
			continue
		}
		defer resp.Body.Close()

		blocksBody, err := ioutil.ReadAll(resp.Body)
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
		offset = int(next.Uint())
		// Get the transactions.  Should be 100 of them
		txs := gjson.Get(blockJSON, "txs")
		for _, tx := range txs.Array() {
			if !tx.IsObject() {
				log.Fatalln("Bad Object")
				continue
			}
			txhash := gjson.Get(tx.String(), "txhash")
			// Get sequence number for tx
			seq, err := getSequenceForTxhash(txhash.String())
			if err != nil {
				log.Fatalln("Failed getting sequence number", err)
				continue
			}
			// Check to see if this is a missing sequence number
			if !missingMessages[seq] {
				continue;
			}
			log.Println("txhash", txhash.String(), "sequence number", seq)
			// send observation request to guardian
			if *dryRun {
				log.Println("Would have sent txhash", txhash, "to the guardian to re-observe")
			} else {
				txHashAsByteArray, err := terra.StringToHash(txhash.String())
				if err != nil {
					log.Fatalln("Couldn't decode the txhash", txhash)
				} else {
					_, err = admin.SendObservationRequest(ctx, &nodev1.SendObservationRequestRequest{
						ObservationRequest: &gossipv1.ObservationRequest{
							ChainId: uint32(vaa.ChainIDTerra),
							TxHash:  txHashAsByteArray.Bytes(),
						}})
					if err != nil {
						log.Fatalf("SendObservationRequest: %v", err)
					}
				}
			}
			if (seq <= uint64(lowest)) {
				// We are done
				log.Println("Finished!")
				return
			}
		}
	}
}
