package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/db"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	solanaRPC  = flag.String("solanaRPC", "http://localhost:8899", "Solana RPC address")
	adminRPC   = flag.String("adminRPC", "/run/guardiand/admin.socket", "Admin RPC address")
	solanaAddr = flag.String("solanaProgram", "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth", "Solana program address")
)

const (
	postMessageInstructionID = 0x01
)

func getAdminClient(ctx context.Context, addr string) (*grpc.ClientConn, error, nodev1.NodePrivilegedServiceClient) {
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("unix:///%s", addr), grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("failed to connect to %s: %v", addr, err)
	}

	c := nodev1.NewNodePrivilegedServiceClient(conn)
	return conn, err, c
}

func main() {
	flag.Parse()

	ctx := context.Background()
	sr := rpc.New(*solanaRPC)

	conn, err, admin := getAdminClient(ctx, *adminRPC)
	defer conn.Close()
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}

	for _, emitter := range sdk.KnownEmitters {
		if emitter.ChainID != vaa.ChainIDSolana {
			continue
		}

		log.Printf("Requesting missing messages for %s", emitter)

		msg := nodev1.FindMissingMessagesRequest{
			EmitterChain:   uint32(vaa.ChainIDSolana),
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
			fmt.Println(id)
			vId, err := db.VaaIDFromString(id)
			if err != nil {
				log.Fatalf("failed to parse VAAID: %v", err)
			}
			msgs[i] = vId
		}

		if len(msgs) == 0 {
			log.Printf("No missing messages found for %s", emitter)
			continue
		}

		lowest := msgs[0].Sequence
		highest := msgs[len(msgs)-1].Sequence

		log.Printf("Found %d missing messages for %s: %d - %d", len(msgs), emitter, lowest, highest)

		limiter := rate.NewLimiter(rate.Every(100*time.Millisecond), 10)

		var before solana.Signature

		decoded, err := hex.DecodeString(emitter.Emitter)
		if err != nil {
			log.Fatalf("Failed to decode emitter address: %v", err)
		}

		addr := solana.PublicKeyFromBytes(decoded)

		hc := http.Client{Timeout: 10 * time.Second}

		log.Printf("Starting repair for %s (%s)", emitter, addr)

		for {
			err := limiter.Wait(ctx)
			if err != nil {
				log.Fatal(err)
			}

			limit := 100

			txs, err := sr.GetSignaturesForAddressWithOpts(ctx, addr, &rpc.GetSignaturesForAddressOpts{
				Limit:  &limit,
				Before: before,
			})
			if err != nil {
				log.Fatalf("GetConfirmedSignaturesForAddress2 %s: %v", emitter, err)
			}

			if len(txs) == 0 {
				log.Printf("fetched all txes for %s (%s)", emitter, addr)
				break
			}

			var lastSeq, firstSeq uint64

			var last solana.Signature
			for i := 0; lastSeq == 0; i-- {
				log.Printf("lastSeq offset: %d", i)
				last = txs[len(txs)-1+i].Signature
				_, lastSeq, err = fetchTxSeq(ctx, sr, last)
				if err != nil {
					log.Fatalf("fetch last tx seq: %v", err)
				}
			}

			for i := 0; firstSeq == 0; i++ {
				log.Printf("firstSeq offset: %d", i)
				_, firstSeq, err = fetchTxSeq(ctx, sr, txs[i].Signature)
				if err != nil {
					log.Fatalf("fetch first tx seq: %v", err)
				}
			}

			log.Printf("fetched %d transactions, from %s (%d) to %s (%d)",
				len(txs), txs[0].Signature, firstSeq, last, lastSeq)

			if highest < lastSeq {
				log.Printf("skipping (%d < %d)", highest, lastSeq)
				goto skip
			}
			if lowest > firstSeq {
				log.Printf("done (%d < %d)", lowest, lastSeq)
				break
			}
			for _, p := range msgs {
				if p.Sequence > lastSeq && p.Sequence < firstSeq {
					offset := firstSeq - p.Sequence - 10
					log.Printf("repairing: %d (offset %d)", p.Sequence, offset)

					var tx *rpc.GetTransactionResult
					var nseq uint64
					var err error

					for {
						if offset >= uint64(len(txs)) {
							log.Fatalf("out of range at offset %d", offset)
						}
						tx, nseq, err = fetchTxSeq(ctx, sr, txs[offset].Signature)
						if err != nil {
							log.Fatalf("failed to fetch %s at offset %d: %v", txs[offset].Signature, offset, err)
						}
						if tx == nil {
							offset += 1
							log.Printf("not a Wormhole tx, offset +1")
							time.Sleep(1 * time.Second)
							continue
						}
						if nseq != p.Sequence {
							offset += 1
							log.Printf("%d != %d, delta +%d, offset +%d", nseq, p.Sequence, nseq-p.Sequence, offset)
							time.Sleep(1 * time.Second)
							continue
						} else {
							break
						}
					}

					acc, err := process(tx)
					if err != nil {
						log.Fatalf("process: %v", err)
					}

					log.Printf("found account %v (%s)", acc, hex.EncodeToString(acc[:]))

					_, err = admin.SendObservationRequest(ctx, &nodev1.SendObservationRequestRequest{
						ObservationRequest: &gossipv1.ObservationRequest{
							ChainId: uint32(vaa.ChainIDSolana),
							TxHash:  acc[:],
						}})
					if err != nil {
						log.Fatalf("SendObservationRequest: %v", err)
					}

					for {
						log.Printf("verifying %d", p.Sequence)
						req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(
							"%s/v1/signed_vaa/%d/%s/%d",
							sdk.PublicRPCEndpoints[0],
							vaa.ChainIDSolana,
							hex.EncodeToString(addr[:]),
							p.Sequence), nil)
						if err != nil {
							panic(err)
						}
						resp, err := hc.Do(req)
						if err != nil {
							log.Fatalf("verify: %v", err)
						}

						defer resp.Body.Close()

						if resp.StatusCode != http.StatusOK {
							log.Printf("status %d, retrying", resp.StatusCode)
							time.Sleep(5 * time.Second)
							continue
						} else {
							log.Printf("success %d", p.Sequence)
							break
						}
					}
				}
			}
		skip:
			before = last
		}
	}
}

func fetchTxSeq(ctx context.Context, c *rpc.Client, sig solana.Signature) (*rpc.GetTransactionResult, uint64, error) {
	maxSupportedTransactionVersion := uint64(0)
	params := rpc.GetTransactionOpts{
		Encoding:                       solana.EncodingBase64,
		Commitment:                     rpc.CommitmentConfirmed,
		MaxSupportedTransactionVersion: &maxSupportedTransactionVersion,
	}
	out, err := c.GetTransaction(ctx, sig, &params)
	if err != nil {
		return nil, 0, fmt.Errorf("GetConfirmedTransaction: %v", err)
	}
	for _, msg := range out.Meta.LogMessages {
		if strings.HasPrefix(msg, "Program log: Sequence:") {
			seq := msg[23:]
			seqInt, err := strconv.Atoi(seq)
			if err != nil {
				log.Printf("failed to parse seq %s: %v", seq, err)
				continue
			}
			return out, uint64(seqInt), nil // #nosec G115 -- The sequence number cannot exceed a uint64
		}
	}
	return nil, 0, nil
}

func process(out *rpc.GetTransactionResult) (*solana.PublicKey, error) {
	program, err := solana.PublicKeyFromBase58(*solanaAddr)
	if err != nil {
		log.Fatalf("Invalid program address: %v", err)
		return nil, err
	}

	tx, err := out.Transaction.GetTransaction()
	if err != nil {
		log.Fatalf("Failed to unmarshal transaction: %v", err)
		return nil, err
	}

	signature := tx.Signatures[0]
	var programIndex uint16
	for n, key := range tx.Message.AccountKeys {
		if key.Equals(program) {
			programIndex = uint16(n) // #nosec G115 -- The solana runtime can only support 64 accounts per transaction max
		}
	}
	if programIndex == 0 {
		return nil, nil
	}

	log.Printf("found Wormhole tx in %s", signature)

	txs := make([]solana.CompiledInstruction, 0, len(tx.Message.Instructions))
	txs = append(txs, tx.Message.Instructions...)
	for _, inner := range out.Meta.InnerInstructions {
		txs = append(txs, inner.Instructions...)
	}

	for _, inst := range txs {
		if inst.ProgramIDIndex != programIndex {
			continue
		}
		if inst.Data[0] != postMessageInstructionID {
			continue
		}
		acc := tx.Message.AccountKeys[inst.Accounts[1]]
		return &acc, nil
	}

	return nil, nil
}
