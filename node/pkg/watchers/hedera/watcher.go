package hedera

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	eth_abi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/hashgraph/hedera-sdk-go/v2"

	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	eth_common "github.com/ethereum/go-ethereum/common"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

var (
	messagesObserved = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_hedera_messages_observed_total",
			Help: "Total number of Hedera messages observed (pre-confirmation)",
		}, []string{"hedera_network"})
	messagesOrphaned = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_hedera_messages_orphaned_total",
			Help: "Total number of Hedera messages dropped (orphaned)",
		}, []string{"hedera_network", "reason"})
	messagesConfirmed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_hedera_messages_confirmed_total",
			Help: "Total number of Hedera messages verified (post-confirmation)",
		}, []string{"hedera_network"})
	currentHeight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_hedera_current_height",
			Help: "Current Hedera block height",
		}, []string{"hedera_network"})
	queryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "wormhole_hedera_query_latency",
			Help: "Latency histogram for Hedera calls (note that most interactions are streaming queries, NOT calls, and we cannot measure latency for those",
		}, []string{"hedera_network", "operation"})
)

// HCS Mirror Node gRPC Endpoints (urlWS):
// PREVIEWNET: hcs.previewnet.mirrornode.hedera.com:5600
// TESTNET: hcs.testnet.mirrornode.hedera.com:5600
// MAINNET: mainnet-public.mirrornode.hedera.com:443

// urlLCD endpoints
// MAINNET BASEURL
// https://mainnet-public.mirrornode.hedera.com/
// TESTNET BASEURL
// https://testnet.mirrornode.hedera.com/
// PREVIEWNET BASEURL
// https://previewnet.mirrornode.hedera.com/

// {
//   "anonymous": false,
//   "inputs": [
//     {
//       "indexed": true,
//       "internalType": "address",
//       "name": "sender",
//       "type": "address"
//     },
//     {
//       "indexed": false,
//       "internalType": "uint64",
//       "name": "sequence",
//       "type": "uint64"
//     },
//     {
//       "indexed": false,
//       "internalType": "uint32",
//       "name": "nonce",
//       "type": "uint32"
//     },
//     {
//       "indexed": false,
//       "internalType": "bytes",
//       "name": "payload",
//       "type": "bytes"
//     },
//     {
//       "indexed": false,
//       "internalType": "uint8",
//       "name": "consistencyLevel",
//       "type": "uint8"
//     }
//   ],
//   "name": "LogMessagePublished",
//   "type": "event"
// },

type (
	// Watcher is responsible for looking over the hedera blockchain and reporting new transactions to the contract
	Watcher struct {
		urlRest  string // REST URL
		contract string // topic

		msgChan chan *common.MessagePublication

		// Incoming re-observation requests from the network. Pre-filtered to only
		// include requests for our chainID.
		obsvReqC chan *gossipv1.ObservationRequest

		// Readiness component
		readiness readiness.Component
		// VAA ChainID of the network we're connecting to.
		chainID vaa.ChainID
	}
)

// NewWatcher creates a new hedera watcher
func NewWatcher(
	urlRest string,
	contract string,
	lockEvents chan *common.MessagePublication,
	obsvReqC chan *gossipv1.ObservationRequest,
	readiness readiness.Component,
	chainID vaa.ChainID) *Watcher {

	// The contract is in solidity format.  Needs to be converted to Hedera format.
	solToAcct, err := hedera.AccountIDFromSolidityAddress(contract)
	if err != nil {
		fmt.Println("Failed to convert sol to acct")
	} else {
		fmt.Println("solToAcct", solToAcct)
	}

	return &Watcher{
		urlRest:   urlRest,
		contract:  contract,
		msgChan:   lockEvents,
		obsvReqC:  obsvReqC,
		readiness: readiness,
		chainID:   chainID,
	}
}

const TOPIC_LOG_MSG = "0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2"

func (watcher *Watcher) Run(ctx context.Context) error {
	networkName := vaa.ChainID(watcher.chainID).String()

	p2p.DefaultRegistry.SetNetworkStats(watcher.chainID, &gossipv1.Heartbeat_Network{
		ContractAddress: watcher.contract,
	})

	errC := make(chan error)
	logger := supervisor.Logger(ctx)

	readiness.SetReady(watcher.readiness)

	// This function/thread queries and reports the current block height every interval
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		client := &http.Client{
			Timeout: time.Second * 5,
		}

		// Do not add a leading slash
		latestBlockURL := "api/v1/blocks"

		for {
			<-ticker.C
			// Query and report latest block
			logger.Info("Checking the following", zap.String("urlRest", watcher.urlRest), zap.String("latestBlockURL", latestBlockURL))
			latency := time.Now()
			resp, err := client.Get(fmt.Sprintf("%s/%s", watcher.urlRest, latestBlockURL))
			if err != nil {
				logger.Error("query latest block response error", zap.String("network", networkName), zap.Error(err))
				continue
			}
			blocksBody, err := io.ReadAll(resp.Body)
			if err != nil {
				logger.Error("query latest block response read error", zap.String("network", networkName), zap.Error(err))
				errC <- err
				resp.Body.Close()
				// When this thread goes away due to error, shouldn't the other threads go away, too?
				break
			}
			resp.Body.Close()

			// Update the prometheus metrics with how long the http request took to the rest api
			queryLatency.WithLabelValues(networkName, "block_latest").Observe(time.Since(latency).Seconds())

			blockJSON := string(blocksBody)
			latestBlock := gjson.Get(blockJSON, "blocks.0.number")
			if !latestBlock.Exists() {
				logger.Error("Failed to query for the latest block.  Could not parse json.")
				continue
			}
			logger.Info("current height", zap.String("network", networkName), zap.Int64("block", latestBlock.Int()))
			currentHeight.WithLabelValues(networkName).Set(float64(latestBlock.Int()))
			p2p.DefaultRegistry.SetNetworkStats(watcher.chainID, &gossipv1.Heartbeat_Network{
				Height:          latestBlock.Int(),
				ContractAddress: watcher.contract,
			})
		}
	}()

	// This function is for reobservations
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case r := <-watcher.obsvReqC:
				if vaa.ChainID(r.ChainId) != watcher.chainID {
					panic("invalid chain ID")
				}

				tx := hex.EncodeToString(r.TxHash)

				logger.Info("received observation request", zap.String("network", networkName), zap.String("tx_hash", tx))

				client := &http.Client{
					Timeout: time.Second * 5,
				}

				// Query for tx by hash
				hashString := "api/v1/contracts/results/" + tx
				latency := time.Now()
				resp, err := client.Get(fmt.Sprintf("%s/%s", watcher.urlRest, hashString))
				if err != nil {
					logger.Error("query tx response error", zap.String("network", networkName), zap.Error(err))
					continue
				}
				txBody, err := io.ReadAll(resp.Body)
				if err != nil {
					logger.Error("query tx response read error", zap.String("network", networkName), zap.Error(err))
					resp.Body.Close()
					continue
				}
				resp.Body.Close()

				// Update the prometheus metrics with how long the http request took to the rest api
				queryLatency.WithLabelValues(networkName, "block_latest").Observe(time.Since(latency).Seconds())

				txJSON := string(txBody)
				logger.Info("RO: txJSON", zap.String("txJSON", txJSON))

				events, err := TxMessageToEvent(logger, txJSON)

				for _, ev := range events {
					watcher.msgChan <- ev
					messagesConfirmed.WithLabelValues(networkName).Inc()
				}
			}
		}
	}()

	// This function is for normal watcher operations
	go func() {
		defer close(errC)
		t := time.NewTicker(5 * time.Second)
		client := &http.Client{
			Timeout: time.Second * 5,
		}
		// var beginningTS = time.Now().String()
		var beginningTS = "1234567890.000000400"

		for {
			<-t.C
			logString := "api/v1/contracts/" + watcher.contract + "/results/logs?order=asc&timestamp=gt%3A" + beginningTS
			logger.Info("current logString", zap.String("string", logString))
			latency := time.Now()
			// Query and report height and set currentSlotHeight
			resp, err := client.Get(fmt.Sprintf("%s/%s", watcher.urlRest, logString))
			if err != nil {
				logger.Error("query latest logs response error", zap.String("network", networkName), zap.Error(err))
				continue
			}
			logsBody, err := io.ReadAll(resp.Body)
			if err != nil {
				logger.Error("query latest logs response read error", zap.String("network", networkName), zap.Error(err))
				errC <- err
				resp.Body.Close()
				continue
			}
			resp.Body.Close()

			// Update the prometheus metrics with how long the http request took to the rest api
			queryLatency.WithLabelValues(networkName, "block_latest").Observe(time.Since(latency).Seconds())

			var events []*common.MessagePublication
			logJSON := string(logsBody)
			// logger.Info("logJSON", zap.String("logJSON", logJSON))
			logs := gjson.Get(logJSON, "logs")
			logger.Info("after gjson.get", zap.Stringer("logs", logs))
			// Don't need to check each topic here.
			// topic[0] = topic of interest, maybe
			// topic[1] = emitter address
			logs.ForEach(func(logKey, logValue gjson.Result) bool {
				logger.Info("YIKES..............", zap.Stringer("topics", gjson.Get(logValue.String(), "topics")))
				topic := gjson.Get(logValue.String(), "topics.0")
				if !topic.Exists() {
					return true // continue ForEach loop
				}
				if topic.String() == TOPIC_LOG_MSG {
					event, err := LogMessageToEvent(logger, logValue.String())
					// Check for event being nil
					if err == nil {
						events = append(events, event)
						blockNum := gjson.Get(logValue.String(), "block_number")
						messagesObserved.WithLabelValues(networkName).Inc()
						logger.Info("Found True Log Msg", zap.Stringer("block Number", blockNum))
					}
				}

				// update timestamp
				timeStampBase := gjson.Get(logValue.String(), "timestamp")
				if !timeStampBase.Exists() {
					logger.Error("Message has no timestamp field")
					return true // continue ForEach() loop
				}
				beginningTS = timeStampBase.String()
				logger.Info("Updating beginningTS", zap.String("beginningTS", beginningTS))
				return true // continue ForEach() loop
			})

			for _, ev := range events {
				watcher.msgChan <- ev
				messagesConfirmed.WithLabelValues(networkName).Inc()
			}
		}
	}()

	select {
	case <-ctx.Done():
		// err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		// if err != nil {
		// 	logger.Error("error on closing socket ", zap.String("network", networkName), zap.Error(err))
		// }
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

// LogMessageToEvent takes a log message like this as a string:
// {\"address\":\"0x0000000000000000000000000000000002dc28a4\",
// \"bloom\":\"0x00000000040100000000000000000000000000000000000000000000080000000010000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000008\",
// \"contract_id\":\"0.0.47982756\",
// \"data\":\"0x00000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000006b68000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000064020000000000000000000000000000000000000000000000000000000002df637d001b025553444200000000000000000000000000000000000000000000000000000000555344204261720000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\",
// \"index\":0,
// \"topics\":[
//
//	\"0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2\",
//	\"0x0000000000000000000000000000000000000000000000000000000002dc2a0e\"],
//
// \"block_hash\":\"0x406dc87975ce5a82336bdba2632fc0f1e7176ce5781b4f6687953895e198382859e796f0b2557cbbdd6dcf68ad98f76e\",
// \"block_number\":24656989,
// \"root_contract_id\":\"0.0.47983118\",
// \"timestamp\":\"1662651989.200248270\",
// \"transaction_hash\":\"0xa26603e164b550ae964dea97dc496a3d63268ed5df538dac5323a32c5be46519\",
// \"transaction_index\":5},
// and converts it to a MessagePublication
func LogMessageToEvent(logger *zap.Logger, logMsg string) (*common.MessagePublication, error) {

	ethAbi, err := eth_abi.JSON(strings.NewReader(ethabi.AbiABI))
	if err != nil {
		logger.Fatal("failed to parse Eth ABI", zap.Error(err))
		return nil, err
	}

	// Check existence of required values
	txHashBase := gjson.Get(logMsg, "transaction_hash")
	if !txHashBase.Exists() {
		return nil, errors.New("Message has no transaction hash")
	}
	txhash := txHashBase.String()
	logDataBase := gjson.Get(logMsg, "data")
	if !logDataBase.Exists() {
		return nil, fmt.Errorf("Message has no data field for txhash %s", txhash)
	}
	timeStampBase := gjson.Get(logMsg, "timestamp")
	if !timeStampBase.Exists() {
		return nil, fmt.Errorf("Message has no timestamp field for txhash %s", txhash)
	}
	emitterBase := gjson.Get(logMsg, "topics.1")
	if !emitterBase.Exists() {
		return nil, fmt.Errorf("Message has no topics array for txhash %s", txhash)
	}
	emitter := emitterBase.String()
	logger.Info("Found emitter", zap.String("hedera emitter", emitter))
	emitterAddr, err := vaa.StringToAddress(emitter)
	if err != nil {
		logger.Fatal("failed to unpack emitter address", zap.Error(err))
		return nil, fmt.Errorf("Emitter field could not be converted for txhash %s, value[%s]", txhash)
	}

	txHash := eth_common.HexToHash(txhash)

	// Get the other values from the Data value
	logDataString := logDataBase.String()
	logDataString = logDataString[2:] // remove the leading 0x
	logDataBytes, err := hex.DecodeString(logDataString)
	if err != nil {
		logger.Fatal("failed to unpack log data", zap.Error(err))
		return nil, fmt.Errorf("Data field could not be converted for txhash %s, value[%s]", txhash)
	}

	unpackedMsg, err := ethAbi.Unpack("LogMessagePublished", logDataBytes)
	if err != nil {
		logger.Fatal("failed to unpack log data", zap.Error(err))
		return nil, fmt.Errorf("Log Data field could not be unpacked for txhash %s", txhash)
	}
	// Make sure unpackedMsg has enough elements
	if len(unpackedMsg) < 4 {
		logger.Fatal("LogMessagePublished does not have enough elements", zap.Int("number of log elements", len(unpackedMsg)))
		return nil, fmt.Errorf("LogMessagePublish does not have enough elements for txhash %s", txhash)
	}

	// // AbiLogMessagePublished represents a LogMessagePublished event raised by the Abi contract.
	// type AbiLogMessagePublished struct {
	// 	Sender           common.Address <- in topics[1]
	// 	Sequence         uint64
	// 	Nonce            uint32
	// 	Payload          []byte
	// 	ConsistencyLevel uint8
	// 	Raw              types.Log // Blockchain specific contextual infos
	// }

	seq := unpackedMsg[0].(uint64)
	nonce := unpackedMsg[1].(uint32)
	payload := unpackedMsg[2].([]byte)
	cLevel := unpackedMsg[3].(uint8)
	logger.Info("unpackedMsg",
		zap.Int("Length of unpackedMsg", len(unpackedMsg)),
		zap.Uint64("Sequence", seq),
		zap.Uint32("Nonce", nonce),
		zap.Uint8("consistencyLevel", cLevel))

	// Convert timestamp
	unixTS, err := TimeStringToUnixTime(timeStampBase.String())
	if err != nil {
		logger.Error("Failed to convert timestamp to Unix time", zap.Error(err))
		return nil, fmt.Errorf("Failed to convert timestamp for txhash %s", txhash)
	}

	messagePublication := &common.MessagePublication{
		TxHash:           txHash,            // In log
		Timestamp:        unixTS,            // In log
		Nonce:            nonce,             // In log data
		Sequence:         seq,               // In log data
		EmitterChain:     vaa.ChainIDHedera, // constant
		EmitterAddress:   emitterAddr,       // In log
		Payload:          payload,           // In log data
		ConsistencyLevel: cLevel,            // In log data
	}

	logger.Info("messagePublication",
		zap.Stringer("txHash", txHash),
		zap.Stringer("timestamp", unixTS),
		zap.Uint32("nonce", nonce),
		zap.Uint64("sequence", seq),
		zap.Stringer("EmitterAddr", emitterAddr),
		zap.Uint8("ConsistencyLevel", cLevel),
	)

	return messagePublication, nil
}

// TxMessageToEvent takes a JSON structured "contracts" output and converts it into an event
func TxMessageToEvent(logger *zap.Logger, msg string) ([]*common.MessagePublication, error) {

	ethAbi, err := eth_abi.JSON(strings.NewReader(ethabi.AbiABI))
	if err != nil {
		logger.Fatal("failed to parse Eth ABI", zap.Error(err))
		return nil, err
	}

	// Check existence of required values in outer scope
	txHashBase := gjson.Get(msg, "hash")
	if !txHashBase.Exists() {
		return nil, errors.New("Message has no transaction hash")
	}
	txhash := txHashBase.String()
	txHash := eth_common.HexToHash(txhash)

	timeStampBase := gjson.Get(msg, "timestamp")
	if !timeStampBase.Exists() {
		return nil, fmt.Errorf("Message has no timestamp field for txhash %s", txhash)
	}

	// Convert timestamp
	unixTS, err := TimeStringToUnixTime(timeStampBase.String())
	if err != nil {
		logger.Error("Failed to convert timestamp to Unix time", zap.Error(err))
		return nil, fmt.Errorf("Failed to convert timestamp for txhash %s", txhash)
	}

	var events []*common.MessagePublication

	// Get the logs array and process it
	logs := gjson.Get(msg, "logs")
	logger.Info("RO: after gjson.get", zap.Stringer("logs", logs))
	logs.ForEach(func(logKey, logValue gjson.Result) bool {
		logger.Info("RO: YIKES..............", zap.Stringer("topics", gjson.Get(logValue.String(), "topics")))
		logString := logValue.String()
		topics := gjson.Get(logString, "topics")
		topics.ForEach(func(topicKey, topicValue gjson.Result) bool {
			if topicValue.String() == TOPIC_LOG_MSG {
				logger.Info("TxMessageToEvent:  Found topic")

				// data is inside the logs object
				logDataBase := gjson.Get(logString, "data")
				if !logDataBase.Exists() {
					logger.Error("Message has no data field", zap.String("txhash", txhash))
					return false
				}

				emitterBase := gjson.Get(logString, "topics.1")
				if !emitterBase.Exists() {
					logger.Error("Message has no address field", zap.String("txhash", txhash))
					return false
				}
				emitter := emitterBase.String()
				logger.Info("Found emitter", zap.String("hedera emitter", emitter))
				emitterAddr, err := vaa.StringToAddress(emitter)
				if err != nil {
					logger.Fatal("failed to unpack emitter address", zap.Error(err))
					return false
				}

				// Get the other values from the Data value
				logDataString := logDataBase.String()
				logDataString = logDataString[2:] // remove the leading 0x
				logDataBytes, err := hex.DecodeString(logDataString)
				if err != nil {
					logger.Fatal("failed to unpack log data", zap.Error(err))
					return false
				}

				unpackedMsg, err := ethAbi.Unpack("LogMessagePublished", logDataBytes)
				if err != nil {
					logger.Fatal("failed to unpack log data", zap.Error(err))
					return false
				}

				seq := unpackedMsg[0].(uint64)
				nonce := unpackedMsg[1].(uint32)
				payload := unpackedMsg[2].([]byte)
				cLevel := unpackedMsg[3].(uint8)
				logger.Info("unpackedMsg",
					zap.Int("Length of unpackedMsg", len(unpackedMsg)),
					zap.Uint64("Sequence", seq),
					zap.Uint32("Nonce", nonce),
					zap.Uint8("consistencyLevel", cLevel))

				messagePublication := &common.MessagePublication{
					TxHash:           txHash,            // In log
					Timestamp:        unixTS,            // In log
					Nonce:            nonce,             // In log data
					Sequence:         seq,               // In log data
					EmitterChain:     vaa.ChainIDHedera, // Constant
					EmitterAddress:   emitterAddr,       // In log
					Payload:          payload,           // In log data
					ConsistencyLevel: cLevel,            // In log data
				}

				logger.Info("messagePublication",
					zap.Stringer("txHash", txHash),
					zap.Stringer("timestamp", unixTS),
					zap.Uint32("nonce", nonce),
					zap.Uint64("sequence", seq),
					zap.Stringer("EmitterAddr", emitterAddr),
					zap.Uint8("ConsistenceLevel", cLevel),
				)

				events = append(events, messagePublication)
				blockNum := gjson.Get(msg, "block_number")
				logger.Info("RO: Found True Log Msg", zap.Stringer("block Number", blockNum))
				return false // break out of the ForEach() loop
			}
			return true // continue inner ForEach() loop
		})
		return true // continue outer ForEach() loop
	})

	return events, nil
}

// StringToAddress convert string into address
func StringToAddress(value string) (vaa.Address, error) {
	var address vaa.Address
	res, err := hex.DecodeString(value)
	if err != nil {
		return address, err
	}
	copy(address[:], res)
	return address, nil
}

// StringToHash convert string into transaction hash
func StringToHash(value string) (eth_common.Hash, error) {
	var hash eth_common.Hash
	res, err := hex.DecodeString(value)
	if err != nil {
		return hash, err
	}
	copy(hash[:], res)
	return hash, nil
}

func TimeStringToUnixTime(value string) (time.Time, error) {
	timeVals := strings.Split(value, ".")
	timeValSeconds, err := strconv.ParseInt(timeVals[0], 10, 64)
	if err != nil {
		return time.Now(), fmt.Errorf("Failed to convert time seconds %s", timeVals[0])
	}
	timeValNS, err := strconv.ParseInt(timeVals[1], 10, 64)
	if err != nil {
		return time.Now(), fmt.Errorf("Failed to convert time nanoseconds %s", timeVals[1])
	}
	unixTS := time.Unix(timeValSeconds, timeValNS)
	return unixTS, nil
}
