package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/ethereum/abi"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/certusone/wormhole/node/pkg/vaa"
	abi2 "github.com/ethereum/go-ethereum/accounts/abi"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
)

var etherscanAPIMap = map[vaa.ChainID]string{
	vaa.ChainIDEthereum:  "https://api.etherscan.io/api",
	vaa.ChainIDBSC:       "https://api.bscscan.com/api",
	vaa.ChainIDAvalanche: "https://api.snowtrace.io/api",
	vaa.ChainIDPolygon:   "https://api.polygonscan.com/api",
	vaa.ChainIDOasis:     "https://explorer.emerald.oasis.dev/api",
}

var coreContractMap = map[vaa.ChainID]string{
	vaa.ChainIDEthereum:  "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B",
	vaa.ChainIDBSC:       "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B",
	vaa.ChainIDAvalanche: "0x54a8e5f9c4CbA08F9943965859F6c34eAF03E26c",
	vaa.ChainIDPolygon:   "0x7A4B5a56256163F07b2C80A7cA55aBE66c4ec4d7",
	vaa.ChainIDOasis:     "0xfe8cd454b4a1ca468b57d79c0cc77ef5b6f64585", // <- converted to all lower case for easy compares
}

var (
	adminRPC     = flag.String("adminRPC", "/run/guardiand/admin.socket", "Admin RPC address")
	etherscanKey = flag.String("etherscanKey", "", "Etherscan API Key")
	chain        = flag.String("chain", "ethereum", "Eth Chain name")
	dryRun       = flag.Bool("dryRun", true, "Dry run")
	step         = flag.Uint64("step", 10000, "Step")
)

var (
	tokenLockupTopic = eth_common.HexToHash("0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2")
)

func getAdminClient(ctx context.Context, addr string) (*grpc.ClientConn, error, nodev1.NodePrivilegedServiceClient) {
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("unix:///%s", addr), grpc.WithInsecure())

	if err != nil {
		log.Fatalf("failed to connect to %s: %v", addr, err)
	}

	c := nodev1.NewNodePrivilegedServiceClient(conn)
	return conn, err, c
}

type logEntry struct {
	// 0x98f3c9e6e3face36baad05fe09d375ef1464288b
	Address string `json:"address"`
	// [
	//  "0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2",
	//  "0x0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585"
	// ]
	Topics []string `json:"topics"`
	// Hex-encoded log data
	Data string `json:"data"`
	// 0xcaebbf
	BlockNumber string `json:"blockNumber"`
	// 0x614fd32b
	TimeStamp string `json:"timeStamp"`
	// 0x960778c48
	GasPrice string `json:"gasPrice"`
	// 0x139d5
	GasUsed string `json:"gasUsed"`
	// 0x18d
	LogIndex string `json:"logIndex"`
	// 0xcc5d73aea74ffe6c8e5e9c212da7eb3ea334f41ac3fd600a9979de727535c849
	TransactionHash string `json:"transactionHash"`
	// 0x117
	TransactionIndex string `json:"transactionIndex"`
}

type logResponse struct {
	// "1" if ok, "0" if error
	Status string `json:"status"`
	// "OK" if ok, "NOTOK" otherwise
	Message string `json:"message"`
	// String when status is "0", result type otherwise.
	Result json.RawMessage `json:"result"`
}

func getCurrentHeight(chainId vaa.ChainID, ctx context.Context, c *http.Client, api, key string) (uint64, error) {
	var req *http.Request;
	var err error;
	if (chainId == vaa.ChainIDOasis) {
		req, err = http.NewRequest("GET", fmt.Sprintf("%s?module=block&action=eth_block_number", api), nil)
	} else {
		req, err = http.NewRequest("GET", fmt.Sprintf("%s?module=proxy&action=eth_blockNumber&apikey=%s", api, key), nil)
	}
	if err != nil {
		panic(err)
	}

	resp, err := c.Do(req.WithContext(ctx))
	if err != nil {
		return 0, fmt.Errorf("failed to get current height: %w", err)
	}

	defer resp.Body.Close()

	var r struct {
		Result string `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return hexutil.DecodeUint64(r.Result)
}

func getLogs(chainId vaa.ChainID, ctx context.Context, c *http.Client, api, key, contract, topic0 string, from, to string) ([]*logEntry, error) {
	var req *http.Request;
	var err error;
	if (chainId == vaa.ChainIDOasis) {
		// This is the Oasis leg
		req, err = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(
			"%s?module=logs&action=getLogs&fromBlock=%s&toBlock=%s&topic0=%s",
			api, from, to, topic0), nil)
	} else {
		req, err = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(
			"%s?module=logs&action=getLogs&fromBlock=%s&toBlock=%s&address=%s&topic0=%s&apikey=%s",
			api, from, to, contract, topic0, key), nil)
	}
	if err != nil {
		panic(err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	defer resp.Body.Close()

	var r logResponse

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if r.Status != "1" && r.Message != "No records found" {
		var e string
		_ = json.Unmarshal(r.Result, &e)
		return nil, fmt.Errorf("failed to get logs (%s): %s", r.Message, e)
	}

	var logs []*logEntry
	if err := json.Unmarshal(r.Result, &logs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal log entry: %w", err)
	}

	if (chainId == vaa.ChainIDOasis) {
		// Because of a bug in BlockScout we need to check the address 
		// in the log to see if it is the Oasis core bridge
		var filtered []*logEntry
		for _, logLine := range logs {
			// Check value of address in log
			if (logLine.Address == contract) {
				filtered = append(filtered, logLine)
			}
		}
		logs = filtered;
	}

	return logs, nil
}

func main() {
	flag.Parse()
	chainID, err := vaa.ChainIDFromString(*chain)
	if err != nil {
		log.Fatalf("Invalid chain: %v", err)
	}

	if *etherscanKey == "" {
		if (chainID != vaa.ChainIDOasis) {
			log.Fatal("Etherscan API Key is required")
		}
	}

	etherscanAPI, ok := etherscanAPIMap[chainID]
	if !ok {
		log.Fatalf("Unsupported chain: %v", err)
	}

	coreContract, ok := coreContractMap[chainID]
	if !ok {
		panic("no core contract")
	}

	ctx := context.Background()

	currentHeight, err := getCurrentHeight(chainID, ctx, http.DefaultClient, etherscanAPI, *etherscanKey)
	if err != nil {
		log.Fatalf("Failed to get current height: %v", err)
	}

	log.Printf("Current height: %d", currentHeight)

	missingMessages := make(map[eth_common.Address]map[uint64]bool)

	conn, err, admin := getAdminClient(ctx, *adminRPC)
	defer conn.Close()
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}

	for _, emitter := range common.KnownEmitters {
		if emitter.ChainID != chainID {
			continue
		}

		contract := eth_common.HexToAddress(emitter.Emitter)

		log.Printf("Requesting missing messages for %s (%v)", emitter.Emitter, contract)

		msg := nodev1.FindMissingMessagesRequest{
			EmitterChain:   uint32(chainID),
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
			log.Printf("No missing messages found for %s", emitter.Emitter)
			continue
		}

		lowest := msgs[0].Sequence
		highest := msgs[len(msgs)-1].Sequence

		log.Printf("Found %d missing messages for %s: %d – %d", len(msgs), emitter.Emitter, lowest, highest)

		if _, ok := missingMessages[contract]; !ok {
			missingMessages[contract] = make(map[uint64]bool)
		}
		for _, msg := range msgs {
			missingMessages[contract][msg.Sequence] = true
		}
	}

	// Press enter to continue if not in dryRun mode
	if !*dryRun {
		fmt.Println("Press enter to continue")
		fmt.Scanln()
	}

	log.Printf("finding sequences")

	limiter := rate.NewLimiter(rate.Every(1*time.Second), 1)

	c := &http.Client{Timeout: 5 * time.Second}

	ethAbi, err := abi2.JSON(strings.NewReader(abi.AbiABI))
	if err != nil {
		log.Fatalf("failed to parse Eth ABI: %v", err)
	}

	var lastHeight uint64
	step := *step
	for {
		if err := limiter.Wait(ctx); err != nil {
			log.Fatalf("failed to wait: %v", err)
		}

		var from, to string
		if lastHeight == 0 {
			from = strconv.Itoa(int(currentHeight - step))
			to = "latest"
			lastHeight = currentHeight
		} else {
			from = strconv.Itoa(int(lastHeight - step))
			to = strconv.Itoa(int(lastHeight))
		}
		lastHeight -= step

		log.Printf("Requesting logs from block %s to %s", from, to)

		logs, err := getLogs(chainID, ctx, c, etherscanAPI, *etherscanKey, coreContract, tokenLockupTopic.Hex(), from, to)
		if err != nil {
			log.Fatalf("failed to get logs: %v", err)
		}

		if len(logs) == 0 {
			log.Printf("No logs found")
			continue
		}

		firstBlock, err := hexutil.DecodeUint64(logs[0].BlockNumber)
		if err != nil {
			log.Fatalf("failed to decode block number: %v", err)
		}
		lastBlock, err := hexutil.DecodeUint64(logs[len(logs)-1].BlockNumber)
		if err != nil {
			log.Fatalf("failed to decode block number: %v", err)
		}

		log.Printf("Got %d logs (first block: %d, last block: %d)",
			len(logs), firstBlock, lastBlock)

		if len(logs) >= 1000 {
			// Bail if we exceeded the maximum number of logs returns in single API call -
			// we might have skipped some and would have to make another call to get the rest.
			//
			// This is a one-off script, so we just set an appropriate interval and bail
			// if we ever hit this.
			log.Fatalf("Range exhausted - %d logs found", len(logs))
		}

		var min, max uint64
		for _, l := range logs {
			if eth_common.HexToHash(l.Topics[0]) != tokenLockupTopic {
				continue
			}

			b, err := hexutil.Decode(l.Data)
			if err != nil {
				log.Fatalf("failed to decode log data for %s: %v", l.TransactionHash, err)
			}

			var seq uint64
			if m, err := ethAbi.Unpack("LogMessagePublished", b); err != nil {
				log.Fatalf("failed to unpack log data for %s: %v", l.TransactionHash, err)
			} else {
				seq = m[0].(uint64)
			}

			if seq < min || min == 0 {
				min = seq
			}
			if seq > max {
				max = seq
			}

			emitter := eth_common.HexToAddress(l.Topics[1])
			tx := eth_common.HexToHash(l.TransactionHash)

			if _, ok := missingMessages[emitter]; !ok {
				continue
			}
			if !missingMessages[emitter][seq] {
				continue
			}

			log.Printf("Found missing message %d for %s in tx %s", seq, emitter, tx.Hex())
			delete(missingMessages[emitter], seq)

			if *dryRun {
				continue
			}

			log.Printf("Requesting re-observation for %s", tx.Hex())

			_, err = admin.SendObservationRequest(ctx, &nodev1.SendObservationRequestRequest{
				ObservationRequest: &gossipv1.ObservationRequest{
					ChainId: uint32(chainID),
					TxHash:  tx.Bytes(),
				}})
			if err != nil {
				log.Fatalf("SendObservationRequest: %v", err)
			}

			for {
				log.Printf("verifying %d", seq)
				req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(
					"%s/v1/signed_vaa/%d/%s/%d",
					common.PublicRPCEndpoints[0],
					chainID,
					hex.EncodeToString(eth_common.LeftPadBytes(emitter.Bytes(), 32)),
					seq), nil)
				if err != nil {
					panic(err)
				}
				resp, err := c.Do(req)
				if err != nil {
					log.Fatalf("verify: %v", err)
				}

				if resp.StatusCode != http.StatusOK {
					log.Printf("status %d, retrying", resp.StatusCode)
					time.Sleep(5 * time.Second)
					continue
				} else {
					log.Printf("success %d", seq)
					break
				}
			}
		}

		log.Printf("Seq: %d - %d", min, max)

		var total int
		for em, entries := range missingMessages {
			total += len(entries)
			log.Printf("%d missing messages for %s left", len(entries), em.Hex())
		}
		if total == 0 {
			log.Printf("No missing messages left")
			break
		}
	}
}
