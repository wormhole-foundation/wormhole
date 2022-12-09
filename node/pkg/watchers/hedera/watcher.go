package hedera

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	abi2 "github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"

	eth_common "github.com/ethereum/go-ethereum/common"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
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
	// Watcher is responsible for looking over a cosmwasm blockchain and reporting new transactions to the contract
	Watcher struct {
		// urlWS    string // gRPC websocket URL
		urlLCD   string // REST URL
		contract string // topic

		msgChan chan *common.MessagePublication

		// Incoming re-observation requests from the network. Pre-filtered to only
		// include requests for our chainID.
		obsvReqC chan *gossipv1.ObservationRequest

		// Readiness component
		readiness readiness.Component
		// VAA ChainID of the network we're connecting to.
		chainID vaa.ChainID
		// // Key for contract address in the wasm logs
		// contractAddressFilterKey string
		// // Key for contract address in the wasm logs
		// contractAddressLogKey string

		// URL to get the latest block info from
		latestBlockURL string
	}
)

// NewWatcher creates a new hedera watcher
func NewWatcher(
	// urlWS string,
	urlLCD string,
	contract string,
	lockEvents chan *common.MessagePublication,
	obsvReqC chan *gossipv1.ObservationRequest,
	readiness readiness.Component,
	chainID vaa.ChainID) *Watcher {

	// // CosmWasm 1.0.0
	// contractAddressFilterKey := "execute._contract_address"
	// contractAddressLogKey := "_contract_address"

	// Do not add a leading slash
	latestBlockURL := "api/v1/blocks"

	return &Watcher{
		// urlWS:     urlWS,
		urlLCD:    urlLCD,
		contract:  contract,
		msgChan:   lockEvents,
		obsvReqC:  obsvReqC,
		readiness: readiness,
		chainID:   chainID,
		// contractAddressFilterKey: contractAddressFilterKey,
		// contractAddressLogKey:    contractAddressLogKey,
		latestBlockURL: latestBlockURL,
	}
}

func (e *Watcher) Run(ctx context.Context) error {
	networkName := vaa.ChainID(e.chainID).String()

	p2p.DefaultRegistry.SetNetworkStats(e.chainID, &gossipv1.Heartbeat_Network{
		ContractAddress: e.contract,
	})

	errC := make(chan error)
	logger := supervisor.Logger(ctx)

	// logger.Info("connecting to websocket", zap.String("network", networkName), zap.String("url", e.urlWS))

	// c, _, err := websocket.DefaultDialer.DialContext(ctx, e.urlWS, nil)
	// if err != nil {
	// 	p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
	// 	// connectionErrors.WithLabelValues(networkName, "websocket_dial_error").Inc()
	// 	return fmt.Errorf("websocket dial failed: %w", err)
	// }
	// defer c.Close()

	logger.Info("subscribed to new transaction events", zap.String("network", networkName))

	readiness.SetReady(e.readiness)

	go func() {
		t := time.NewTicker(5 * time.Second)
		client := &http.Client{
			Timeout: time.Second * 5,
		}

		for {
			<-t.C
			// msm := time.Now()
			// Query and report height and set currentSlotHeight
			logger.Info("Checking the following", zap.String("urlLCD", e.urlLCD), zap.String("latestBlockURL", e.latestBlockURL))
			resp, err := client.Get(fmt.Sprintf("%s/%s", e.urlLCD, e.latestBlockURL))
			if err != nil {
				logger.Error("query latest block response error", zap.String("network", networkName), zap.Error(err))
				continue
			}
			blocksBody, err := io.ReadAll(resp.Body)
			if err != nil {
				logger.Error("query latest block response read error", zap.String("network", networkName), zap.Error(err))
				errC <- err
				resp.Body.Close()
				continue
			}
			resp.Body.Close()

			blockJSON := string(blocksBody)
			latestBlock := gjson.Get(blockJSON, "blocks.0.number")
			logger.Info("current height", zap.String("network", networkName), zap.Int64("block", latestBlock.Int()))
			p2p.DefaultRegistry.SetNetworkStats(e.chainID, &gossipv1.Heartbeat_Network{
				Height:          latestBlock.Int(),
				ContractAddress: e.contract,
			})
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case r := <-e.obsvReqC:
				if vaa.ChainID(r.ChainId) != e.chainID {
					panic("invalid chain ID")
				}

				tx := hex.EncodeToString(r.TxHash)

				logger.Info("received observation request", zap.String("network", networkName), zap.String("tx_hash", tx))

				client := &http.Client{
					Timeout: time.Second * 5,
				}

				// Query for tx by hash
				resp, err := client.Get(fmt.Sprintf("%s/api/v1/blocks/%s", e.urlLCD, tx))
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

				txJSON := string(txBody)

				txHashRaw := gjson.Get(txJSON, "block.hash")
				if !txHashRaw.Exists() {
					logger.Error("block does not have a hash", zap.String("network", networkName), zap.String("payload", txJSON))
					continue
				}
				txHash := txHashRaw.String()

				events := gjson.Get(txJSON, "tx_response.events")
				if !events.Exists() {
					logger.Error("tx has no events", zap.String("network", networkName), zap.String("payload", txJSON))
					continue
				}

				// msgs := EventsToMessagePublications(e.contract, txHash, events.Array(), logger, e.chainID, e.contractAddressLogKey)
				msgs := EventsToMessagePublications(e.contract, txHash, events.Array(), logger, e.chainID, "")
				for _, msg := range msgs {
					e.msgChan <- msg
					// messagesConfirmed.WithLabelValues(networkName).Inc()
				}
			}
		}
	}()

	go func() {
		defer close(errC)
		t := time.NewTicker(5 * time.Second)
		client := &http.Client{
			Timeout: time.Second * 5,
		}
		// const contractId = "0.0.47982756"
		// var logString = "api/v1/contracts/" + contractId + "/results/logs?order=asc&timestamp=gte%3A1234567890.000000400"
		var logString = "api/v1/contracts/" + e.contract + "/results/logs?order=asc&timestamp=gte%3A1234567890.000000400"

		const TOPIC_LOG_MSG = "0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2"

		for {
			<-t.C
			// Query and report height and set currentSlotHeight
			resp, err := client.Get(fmt.Sprintf("%s/%s", e.urlLCD, logString))
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

			var events []*common.MessagePublication
			logJSON := string(logsBody)
			// logger.Info("logJSON", zap.String("logJSON", logJSON))
			logs := gjson.Get(logJSON, "logs")
			// logger.Info("after gjson.get", zap.Stringer("logs", logs))
			logs.ForEach(func(logKey, logValue gjson.Result) bool {
				// logger.Info("YIKES..............", zap.Stringer("topics", gjson.Get(value.String(), "topics")))
				topics := gjson.Get(logValue.String(), "topics")
				topics.ForEach(func(topicKey, topicValue gjson.Result) bool {
					if topicValue.String() == TOPIC_LOG_MSG {
						event, err := LogMessageToEvent(ctx, logValue.String())
						// Check for event being nil
						if err == nil {
							events = append(events, event)
							blockNum := gjson.Get(logValue.String(), "block_number")
							logger.Info("Found True Log Msg", zap.Stringer("block Number", blockNum))
						}
						return false
					}
					return true
				})
				return true
			})

			for _, ev := range events {
				e.msgChan <- ev
				// messagesConfirmed.WithLabelValues(networkName).Inc()
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
func LogMessageToEvent(ctx context.Context, logMsg string) (*common.MessagePublication, error) {
	logger := supervisor.Logger(ctx)

	ethAbi, err := abi2.JSON(strings.NewReader(ethabi.AbiABI))
	if err != nil {
		logger.Fatal("failed to parse Eth ABI", zap.Error(err))
		return nil, err
	}

	// Check existence of required values
	txHashBase := gjson.Get(logMsg, "transaction_hash")
	if !txHashBase.Exists() {
		return nil, errors.New("Message has no transaction hash")
	}
	logDataBase := gjson.Get(logMsg, "data")
	if !logDataBase.Exists() {
		return nil, fmt.Errorf("Message has no data field for txhash %s", txHashBase.String())
	}
	timeStampBase := gjson.Get(logMsg, "timestamp")
	if !timeStampBase.Exists() {
		return nil, fmt.Errorf("Message has no timestamp field for txhash %s", txHashBase.String())
	}
	emitterBase := gjson.Get(logMsg, "address")
	if !emitterBase.Exists() {
		return nil, fmt.Errorf("Message has no address field for txhash %s", txHashBase.String())
	}

	txHashString := txHashBase.String()
	txHash := eth_common.HexToHash(txHashString)
	// txHash, err := StringToHash(txHashString)
	// if err != nil {
	// logger.Error("cannot decode txHashHex", zap.String("txHash", txHashString))
	// return nil, fmt.Errorf("Could not decode txhash %s", txHashBase.String())
	// }

	timeStamp := timeStampBase.Int()

	emitter := emitterBase.String()
	emitterAddr, err := vaa.StringToAddress(emitter)
	if err != nil {
		logger.Fatal("failed to unpack emitter address", zap.Error(err))
		return nil, fmt.Errorf("Emitter field could not be converted for txhash %s, value[%s]", txHashBase.String(), emitter)
	}

	// Get the other values from the Data value
	logDataString := logDataBase.String()
	logDataString = logDataString[2:] // remove the leading 0x
	logDataBytes, err := hex.DecodeString(logDataString)
	if err != nil {
		logger.Fatal("failed to unpack log data", zap.Error(err))
		return nil, fmt.Errorf("Data field could not be converted for txhash %s, value[%s]", txHashBase.String(), &logDataString)
	}

	unpackedMsg, err := ethAbi.Unpack("LogMessagePublished", logDataBytes)
	if err != nil {
		logger.Fatal("failed to unpack log data", zap.Error(err))
		return nil, fmt.Errorf("Log Data field could not be unpacked for txhash %s", txHashBase.String())
	}

	seq := unpackedMsg[0].(uint64)
	nonce := unpackedMsg[1].(uint32)
	payload := unpackedMsg[2].([]byte)
	cLevel := unpackedMsg[3].(uint8)
	logger.Info("unpackedMsg", zap.Int("Length of unpackedMsg", len(unpackedMsg)), zap.Uint64("0", seq), zap.Uint32("2", nonce), zap.Uint8("4", cLevel))
	var chainID vaa.ChainID

	messagePublication := &common.MessagePublication{
		TxHash:           txHash,                  // In log
		Timestamp:        time.Unix(timeStamp, 0), // In log
		Nonce:            nonce,                   // In log data
		Sequence:         seq,                     // In log data
		EmitterChain:     chainID,                 // Don't know where to get this
		EmitterAddress:   emitterAddr,             // In log
		Payload:          payload,                 // In log data
		ConsistencyLevel: cLevel,                  // In log data
	}

	logger.Info("messagePublication",
		zap.Stringer("txHash", txHash),
		zap.Int64("timestamp", timeStamp),
		zap.Uint32("nonce", nonce),
		zap.Uint64("sequence", seq),
		zap.Stringer("EmitterAddr", emitterAddr),
		zap.Uint8("ConsistenceLevel", cLevel),
	)

	return messagePublication, nil
}

func EventsToMessagePublications(contract string, txHash string, events []gjson.Result, logger *zap.Logger, chainID vaa.ChainID, contractAddressKey string) []*common.MessagePublication {
	networkName := vaa.ChainID(chainID).String()
	msgs := make([]*common.MessagePublication, 0, len(events))
	for _, event := range events {
		if !event.IsObject() {
			logger.Warn("event is invalid", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("event", event.String()))
			continue
		}
		eventType := gjson.Get(event.String(), "type")
		if eventType.String() != "wasm" {
			continue
		}

		attributes := gjson.Get(event.String(), "attributes")
		if !attributes.Exists() {
			logger.Warn("message event has no attributes", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("event", event.String()))
			continue
		}
		mappedAttributes := map[string]string{}
		for _, attribute := range attributes.Array() {
			if !attribute.IsObject() {
				logger.Warn("event attribute is invalid", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}
			keyBase := gjson.Get(attribute.String(), "key")
			if !keyBase.Exists() {
				logger.Warn("event attribute does not have key", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}
			valueBase := gjson.Get(attribute.String(), "value")
			if !valueBase.Exists() {
				logger.Warn("event attribute does not have value", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}

			key, err := base64.StdEncoding.DecodeString(keyBase.String())
			if err != nil {
				logger.Warn("event key attribute is invalid", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("key", keyBase.String()))
				continue
			}
			value, err := base64.StdEncoding.DecodeString(valueBase.String())
			if err != nil {
				logger.Warn("event value attribute is invalid", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
				continue
			}

			if _, ok := mappedAttributes[string(key)]; ok {
				logger.Debug("duplicate key in events", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
				continue
			}

			mappedAttributes[string(key)] = string(value)
		}

		contractAddress, ok := mappedAttributes[contractAddressKey]
		if !ok {
			logger.Warn("wasm event without contract address field set", zap.String("network", networkName), zap.String("event", event.String()))
			continue
		}
		// This is not a wormhole message
		if contractAddress != contract {
			continue
		}

		payload, ok := mappedAttributes["message.message"]
		if !ok {
			logger.Error("wormhole event does not have a message field", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		sender, ok := mappedAttributes["message.sender"]
		if !ok {
			logger.Error("wormhole event does not have a sender field", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		chainId, ok := mappedAttributes["message.chain_id"]
		if !ok {
			logger.Error("wormhole event does not have a chain_id field", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		nonce, ok := mappedAttributes["message.nonce"]
		if !ok {
			logger.Error("wormhole event does not have a nonce field", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		sequence, ok := mappedAttributes["message.sequence"]
		if !ok {
			logger.Error("wormhole event does not have a sequence field", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		blockTime, ok := mappedAttributes["message.block_time"]
		if !ok {
			logger.Error("wormhole event does not have a block_time field", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}

		logger.Info("new message detected on cosmwasm",
			zap.String("network", networkName),
			zap.String("chainId", chainId),
			zap.String("txHash", txHash),
			zap.String("sender", sender),
			zap.String("nonce", nonce),
			zap.String("sequence", sequence),
			zap.String("blockTime", blockTime),
		)

		senderAddress, err := StringToAddress(sender)
		if err != nil {
			logger.Error("cannot decode emitter hex", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("value", sender))
			continue
		}
		txHashValue, err := StringToHash(txHash)
		if err != nil {
			logger.Error("cannot decode tx hash hex", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("value", txHash))
			continue
		}
		payloadValue, err := hex.DecodeString(payload)
		if err != nil {
			logger.Error("cannot decode payload", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("value", payload))
			continue
		}

		blockTimeInt, err := strconv.ParseInt(blockTime, 10, 64)
		if err != nil {
			logger.Error("blocktime cannot be parsed as int", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("value", blockTime))
			continue
		}
		nonceInt, err := strconv.ParseUint(nonce, 10, 32)
		if err != nil {
			logger.Error("nonce cannot be parsed as int", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("value", blockTime))
			continue
		}
		sequenceInt, err := strconv.ParseUint(sequence, 10, 64)
		if err != nil {
			logger.Error("sequence cannot be parsed as int", zap.String("network", networkName), zap.String("tx_hash", txHash), zap.String("value", blockTime))
			continue
		}
		messagePublication := &common.MessagePublication{
			TxHash:           txHashValue,                // In log
			Timestamp:        time.Unix(blockTimeInt, 0), // In log
			Nonce:            uint32(nonceInt),           // In decoded log event
			Sequence:         sequenceInt,                // In decoded log event
			EmitterChain:     chainID,                    // Don't know where to get this
			EmitterAddress:   senderAddress,              // In decoded log event
			Payload:          payloadValue,               // In decoded log event
			ConsistencyLevel: 0,                          // In decoded log event
		}
		msgs = append(msgs, messagePublication)
	}
	return msgs
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
