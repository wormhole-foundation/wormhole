package terra

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"

	eth_common "github.com/ethereum/go-ethereum/common"

	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/readiness"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

type (
	// BridgeWatcher is responsible for looking over Terra blockchain and reporting new transactions to the bridge
	BridgeWatcher struct {
		urlWS  string
		urlLCD string
		bridge string

		lockChan chan *common.ChainLock
		setChan  chan *common.GuardianSet
	}
)

type clientRequest struct {
	JSONRPC string `json:"jsonrpc"`
	// A String containing the name of the method to be invoked.
	Method string `json:"method"`
	// Object to pass as request parameter to the method.
	Params [1]string `json:"params"`
	// The request id. This can be of any type. It is used to match the
	// response with the request that it is replying to.
	ID uint64 `json:"id"`
}

// NewTerraBridgeWatcher creates a new terra bridge watcher
func NewTerraBridgeWatcher(urlWS string, urlLCD string, bridge string, lockEvents chan *common.ChainLock, setEvents chan *common.GuardianSet) *BridgeWatcher {
	return &BridgeWatcher{urlWS: urlWS, urlLCD: urlLCD, bridge: bridge, lockChan: lockEvents, setChan: setEvents}
}

// Run is the main Terra Bridge run cycle
func (e *BridgeWatcher) Run(ctx context.Context) error {
	errC := make(chan error)
	logger := supervisor.Logger(ctx)

	logger.Info("connecting to websocket", zap.String("url", e.urlWS))

	c, _, err := websocket.DefaultDialer.DialContext(ctx, e.urlWS, nil)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	defer c.Close()

	// Subscribe to smart contract transactions
	params := [...]string{fmt.Sprintf("tm.event='Tx' AND execute_contract.contract_address='%s'", e.bridge)}
	command := &clientRequest{
		JSONRPC: "2.0",
		Method:  "subscribe",
		Params:  params,
		ID:      1,
	}
	err = c.WriteJSON(command)
	if err != nil {
		return fmt.Errorf("websocket subscription failed: %w", err)
	}

	// Wait for the success response
	_, _, err = c.ReadMessage()
	if err != nil {
		return fmt.Errorf("event subscription failed: %w", err)
	}
	logger.Info("subscribed to new transaction events")

	readiness.SetReady(common.ReadinessTerraSyncing)

	go func() {
		defer close(errC)

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				logger.Error("error reading channel", zap.Error(err))
				errC <- err
				return
			}

			// Received a message from the blockchain
			json := string(message)
			targetChain := gjson.Get(json, "result.events.from_contract\\.locked\\.target_chain.0")
			tokenChain := gjson.Get(json, "result.events.from_contract\\.locked\\.token_chain.0")
			tokenDecimals := gjson.Get(json, "result.events.from_contract\\.locked\\.token_decimals.0")
			token := gjson.Get(json, "result.events.from_contract\\.locked\\.token.0")
			sender := gjson.Get(json, "result.events.from_contract\\.locked\\.sender.0")
			recipient := gjson.Get(json, "result.events.from_contract\\.locked\\.recipient.0")
			amount := gjson.Get(json, "result.events.from_contract\\.locked\\.amount.0")
			nonce := gjson.Get(json, "result.events.from_contract\\.locked\\.nonce.0")
			txHash := gjson.Get(json, "result.events.tx\\.hash.0")
			blockTime := gjson.Get(json, "result.events.from_contract\\.locked\\.block_time.0")

			if targetChain.Exists() && tokenChain.Exists() && tokenDecimals.Exists() && token.Exists() && sender.Exists() &&
				recipient.Exists() && amount.Exists() && amount.Exists() && nonce.Exists() && txHash.Exists() && blockTime.Exists() {

				logger.Info("token lock detected on Terra",
					zap.String("txHash", txHash.String()),
					zap.String("targetChain", targetChain.String()),
					zap.String("tokenChain", tokenChain.String()),
					zap.String("tokenDecimals", tokenDecimals.String()),
					zap.String("token", token.String()),
					zap.String("sender", sender.String()),
					zap.String("recipient", recipient.String()),
					zap.String("amount", amount.String()),
					zap.String("nonce", nonce.String()),
					zap.String("blockTime", blockTime.String()),
				)

				senderAddress, err := StringToAddress(sender.String())
				if err != nil {
					logger.Error("cannot decode hex", zap.String("value", sender.String()))
					continue
				}
				recipientAddress, err := StringToAddress(recipient.String())
				if err != nil {
					logger.Error("cannot decode hex", zap.String("value", recipient.String()))
					continue
				}
				tokenAddress, err := StringToAddress(token.String())
				if err != nil {
					logger.Error("cannot decode hex", zap.String("value", token.String()))
					continue
				}
				txHashValue, err := StringToHash(txHash.String())
				if err != nil {
					logger.Error("cannot decode hex", zap.String("value", txHash.String()))
					continue
				}
				lock := &common.ChainLock{
					TxHash:        txHashValue,
					Timestamp:     time.Unix(blockTime.Int(), 0),
					Nonce:         uint32(nonce.Uint()),
					SourceAddress: senderAddress,
					TargetAddress: recipientAddress,
					SourceChain:   vaa.ChainIDTerra,
					TargetChain:   vaa.ChainID(uint8(targetChain.Uint())),
					TokenChain:    vaa.ChainID(uint8(tokenChain.Uint())),
					TokenAddress:  tokenAddress,
					TokenDecimals: uint8(tokenDecimals.Uint()),
					Amount:        new(big.Int).SetUint64(amount.Uint()),
				}
				e.lockChan <- lock
			}

			// Query and report guardian set status
			requestURL := fmt.Sprintf("%s/wasm/contracts/%s/store?query_msg={\"guardian_set_info\":{}}", e.urlLCD, e.bridge)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
			if err != nil {
				logger.Error("query guardian set request error", zap.Error(err))
				errC <- err
				return
			}

			client := &http.Client{
				Timeout: time.Second * 15,
			}
			resp, err := client.Do(req)
			if err != nil {
				logger.Error("query guardian set response error", zap.Error(err))
				errC <- err
				return
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logger.Error("query guardian set error", zap.Error(err))
				errC <- err
				resp.Body.Close()
				return
			}

			json = string(body)
			guardianSetIndex := gjson.Get(json, "result.guardian_set_index")
			addresses := gjson.Get(json, "result.addresses.#.bytes")

			logger.Debug("current guardian set on Terra",
				zap.Any("guardianSetIndex", guardianSetIndex),
				zap.Any("addresses", addresses))

			resp.Body.Close()

			// We do not send guardian changes to the processor - ETH guardians are the source of truth.
		}
	}()

	select {
	case <-ctx.Done():
		err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			logger.Error("error on closing socket ", zap.Error(err))
		}
		return ctx.Err()
	case err := <-errC:
		return err
	}
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
