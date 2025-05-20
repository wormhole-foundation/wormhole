//nolint:forcetypeassert //this is a hack
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/watchers/evm"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"

	"github.com/certusone/wormhole/node/pkg/db"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	abi2 "github.com/ethereum/go-ethereum/accounts/abi"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var etherscanAPIMap = map[vaa.ChainID]string{
	vaa.ChainIDEthereum:  "https://api.etherscan.io/api",
	vaa.ChainIDBSC:       "https://api.bscscan.com/api",
	vaa.ChainIDAvalanche: "https://api.snowtrace.io/api",
	vaa.ChainIDPolygon:   "https://api.polygonscan.com/api",
	vaa.ChainIDOasis:     "https://explorer.emerald.oasis.dev/api",
	vaa.ChainIDAurora:    "https://explorer.mainnet.aurora.dev/api",
	vaa.ChainIDFantom:    "https://api.ftmscan.com/api",
	vaa.ChainIDKarura:    "https://blockscout.karura.network/api",
	vaa.ChainIDAcala:     "https://blockscout.acala.network/api",
	// NOTE: Not sure what should be here for Klaytn, since they use: https://scope.klaytn.com/
	vaa.ChainIDCelo:       "https://celoscan.xyz/api",
	vaa.ChainIDMoonbeam:   "https://api-moonbeam.moonscan.io",
	vaa.ChainIDArbitrum:   "https://api.arbiscan.io",
	vaa.ChainIDOptimism:   "https://api-optimistic.etherscan.io",
	vaa.ChainIDBase:       "https://api.basescan.org",
	vaa.ChainIDScroll:     "https://api.scrollscan.com",
	vaa.ChainIDMantle:     "https://api.mantlescan.xyz/",
	vaa.ChainIDBlast:      "https://api.blastscan.io",
	vaa.ChainIDXLayer:     "", // TODO: Does X Layer have an etherscan API endpoint?
	vaa.ChainIDBerachain:  "https://api.berascan.com/",
	vaa.ChainIDSeiEVM:     "", // TODO: Does SeiEVM have an etherscan API endpoint?
	vaa.ChainIDUnichain:   "https://api.uniscan.xyz/",
	vaa.ChainIDWorldchain: "https://api.worldscan.org",
	vaa.ChainIDInk:        "", // TODO: Does Ink have an etherscan API endpoint?
}

var (
	adminRPC     = flag.String("adminRPC", "/run/guardiand/admin.socket", "Admin RPC address")
	etherscanKey = flag.String("etherscanKey", "", "Etherscan API Key")
	chain        = flag.String("chain", "ethereum", "Eth Chain name")
	dryRun       = flag.Bool("dryRun", true, "Dry run")
	step         = flag.Uint64("step", 10000, "Step")
	showError    = flag.Bool("showError", false, "On http error, show the response body")
	sleepTime    = flag.Int("sleepTime", 0, "Time to sleep between loops when getting logs")
)

var (
	tokenLockupTopic = eth_common.HexToHash("0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2")
)

// Add a browser User-Agent to make cloudflare more happy
func addUserAgent(req *http.Request) *http.Request {
	if req == nil {
		return nil
	}
	req.Header.Set(
		"User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36",
	)
	return req
}

func usesBlockscout(chainId vaa.ChainID) bool {
	return chainId == vaa.ChainIDOasis || chainId == vaa.ChainIDAurora || chainId == vaa.ChainIDKarura || chainId == vaa.ChainIDAcala
}

func getAdminClient(ctx context.Context, addr string) (*grpc.ClientConn, error, nodev1.NodePrivilegedServiceClient) {
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("unix:///%s", addr), grpc.WithTransportCredentials(insecure.NewCredentials()))

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

func getCurrentHeight(chainId vaa.ChainID, ctx context.Context, c *http.Client, api, key string, showErr bool) (uint64, error) {
	var req *http.Request
	var err error
	if usesBlockscout(chainId) {
		// This is the BlockScout based explorer leg
		req, err = http.NewRequest("GET", fmt.Sprintf("%s?module=block&action=eth_block_number", api), nil)
	} else {
		req, err = http.NewRequest("GET", fmt.Sprintf("%s?module=proxy&action=eth_blockNumber&apikey=%s", api, key), nil)
	}
	if err != nil {
		panic(err)
	}
	req = addUserAgent(req)

	resp, err := c.Do(req.WithContext(ctx))
	if err != nil {
		return 0, fmt.Errorf("failed to get current height: %w", err)
	}

	defer resp.Body.Close()

	var r struct {
		Result string `json:"result"`
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && showErr {
		fmt.Println(string(body))
	}

	if err := json.Unmarshal(body, &r); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return hexutil.DecodeUint64(r.Result)
}

func getLogs(chainId vaa.ChainID, ctx context.Context, c *http.Client, api, key, contract, topic0 string, from, to string, showErr bool) ([]*logEntry, error) {
	var req *http.Request
	var err error
	if usesBlockscout(chainId) {
		// This is the BlockScout based explorer leg
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
	req = addUserAgent(req)

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	defer resp.Body.Close()

	var r logResponse

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && showErr {
		fmt.Println(string(body))
	}

	if err := json.Unmarshal(body, &r); err != nil {
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

	if usesBlockscout(chainId) {
		// Because of a bug in BlockScout based explorers we need to check the address
		// in the log to see if it is the core bridge
		var filtered []*logEntry
		for _, logLine := range logs {
			// Check value of address in log
			if logLine.Address == contract {
				filtered = append(filtered, logLine)
			}
		}
		logs = filtered
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
		// BlockScout based explorers don't require an ether scan key
		if !usesBlockscout(chainID) {
			log.Fatal("Etherscan API Key is required")
		}
	}

	etherscanAPI, ok := etherscanAPIMap[chainID]
	if !ok {
		log.Fatalf("Unsupported chain: %v", err)
	}

	coreContract, err := evm.GetContractAddrString(common.MainNet, chainID)
	if err != nil {
		panic("no core contract")
	}
	ctx := context.Background()

	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("Error creating http cookiejar: %v", err)
	}
	httpClient := &http.Client{
		Jar: jar,
	}
	currentHeight, err := getCurrentHeight(chainID, ctx, httpClient, etherscanAPI, *etherscanKey, *showError)
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

	// A polygon VAA that was not reobserved before the blocks aged out of guardian rpc nodes
	ignoreAddress, _ := vaa.StringToAddress("0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde")
	polygonIgnoredVaa := db.VAAID{
		Sequence:       6840,
		EmitterChain:   vaa.ChainIDPolygon,
		EmitterAddress: ignoreAddress,
	}

	for _, emitter := range sdk.KnownEmitters {
		if emitter.ChainID != chainID {
			continue
		}

		contract := eth_common.HexToAddress(emitter.Emitter)

		log.Printf("Requesting missing messages for %s (%v)", emitter.Emitter, contract)

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

		msgs := []*db.VAAID{}
		for _, id := range resp.MissingMessages {
			fmt.Println(id)
			vId, err := db.VaaIDFromString(id)
			if err != nil {
				log.Fatalf("failed to parse VAAID: %v", err)
			}
			if *vId == polygonIgnoredVaa {
				log.Printf("Ignored message: %+v", &polygonIgnoredVaa)
				continue
			}
			msgs = append(msgs, vId)
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
		_, err := fmt.Scanln()
		if err != nil {
			log.Printf("Scanln error: %s\n", err)
		}
	}

	log.Printf("finding sequences")

	limiter := rate.NewLimiter(rate.Every(1*time.Second), 1)

	c := &http.Client{
		Jar:     jar,
		Timeout: 5 * time.Second,
	}

	ethAbi, err := abi2.JSON(strings.NewReader(ethabi.AbiABI))
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
			if currentHeight-step > math.MaxInt {
				log.Fatalf("from block overflowed: %v", currentHeight-step)
			}
			from = strconv.Itoa(int(currentHeight - step)) // #nosec G115 -- This is checked above
			to = "latest"
			lastHeight = currentHeight
		} else {
			if lastHeight > math.MaxInt {
				log.Fatalf("from block overflowed: %v", lastHeight)
			}
			from = strconv.Itoa(int(lastHeight - step)) // #nosec G115 -- If the above is safe, this is safe too
			to = strconv.Itoa(int(lastHeight))          // #nosec G115 -- This is checked above
		}
		lastHeight -= step

		log.Printf("Requesting logs from block %s to %s", from, to)

		logs, err := getLogs(chainID, ctx, c, etherscanAPI, *etherscanKey, coreContract, tokenLockupTopic.Hex(), from, to, *showError)
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

		var minimum, maximum uint64
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

			if seq < minimum || minimum == 0 {
				minimum = seq
			}
			if seq > maximum {
				maximum = seq
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

			for i := 0; i < 10; i++ {
				log.Printf("verifying %d", seq)
				req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(
					"%s/v1/signed_vaa/%d/%s/%d",
					sdk.PublicRPCEndpoints[0],
					chainID,
					hex.EncodeToString(eth_common.LeftPadBytes(emitter.Bytes(), 32)),
					seq), nil)
				if err != nil {
					panic(err)
				}
				req = addUserAgent(req)
				resp, err := c.Do(req)
				if err != nil {
					log.Fatalf("verify: %v", err)
				}
				defer resp.Body.Close()

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

		log.Printf("Seq: %d - %d", minimum, maximum)

		var total int
		for em, entries := range missingMessages {
			total += len(entries)
			log.Printf("%d missing messages for %s left", len(entries), em.Hex())
		}
		if total == 0 {
			log.Printf("No missing messages left")
			break
		}
		// Allow sleeping between loops for chains that have aggressive blocking in the explorers
		if sleepTime != nil {
			time.Sleep(time.Duration(*sleepTime) * time.Second)
		}
	}
}
