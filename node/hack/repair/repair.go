package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
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
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("unix:///%s", addr), grpc.WithInsecure())

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

	for _, emitter := range common.KnownEmitters {
		if emitter.ChainID != vaa.ChainIDSolana {
			continue
		}

		log.Printf("Requesting missing messages for %s", emitter)

		msg := nodev1.FindMissingMessagesRequest{
			EmitterChain:   uint32(vaa.ChainIDSolana),
			EmitterAddress: emitter.Emitter,
			RpcBackfill:    true,
			BackfillNodes:  common.PublicRPCEndpoints,
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

			last := txs[len(txs)-1].Signature

			_, lastSeq, err := fetchTxSeq(ctx, sr, last)
			if err != nil {
				log.Fatalf("fetch last tx seq: %v", err)
			}

			_, firstSeq, err := fetchTxSeq(ctx, sr, txs[0].Signature)
			if err != nil {
				log.Fatalf("fetch first tx seq: %v", err)
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
					offset := firstSeq - p.Sequence
					log.Printf("repairing: %d (offset %d)", p.Sequence, offset)

					var tx *rpc.TransactionWithMeta
					var nseq uint64
					var err error

					for {
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
							log.Printf("%d != %d, offset +%d", nseq, p.Sequence, nseq-p.Sequence)
							offset += nseq - p.Sequence
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
							common.PublicRPCEndpoints[0],
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

func fetchTxSeq(ctx context.Context, c *rpc.Client, sig solana.Signature) (*rpc.TransactionWithMeta, uint64, error) {
	out, err := c.GetConfirmedTransaction(ctx, sig)
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
			return out, uint64(seqInt), nil
		}
	}
	return nil, 0, nil
}

func process(tx *rpc.TransactionWithMeta) (*solana.PublicKey, error) {
	program, err := solana.PublicKeyFromBase58(*solanaAddr)
	if err != nil {
		log.Fatalf("Invalid program address: %v", err)
	}

	signature := tx.Transaction.Signatures[0]
	var programIndex uint16
	for n, key := range tx.Transaction.Message.AccountKeys {
		if key.Equals(program) {
			programIndex = uint16(n)
		}
	}
	if programIndex == 0 {
		return nil, nil
	}

	log.Printf("found Wormhole tx in %s", signature)

	txs := make([]solana.CompiledInstruction, 0, len(tx.Transaction.Message.Instructions))
	txs = append(txs, tx.Transaction.Message.Instructions...)
	for _, inner := range tx.Meta.InnerInstructions {
		txs = append(txs, inner.Instructions...)
	}

	for _, inst := range txs {
		if inst.ProgramIDIndex != programIndex {
			continue
		}
		if inst.Data[0] != postMessageInstructionID {
			continue
		}
		acc := tx.Transaction.Message.AccountKeys[inst.Accounts[1]]
		return &acc, nil
	}

	return nil, nil
}
