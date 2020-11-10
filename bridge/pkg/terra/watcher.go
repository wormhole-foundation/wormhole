package terra

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/url"
	"time"

	eth_common "github.com/ethereum/go-ethereum/common"

	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

type (
	// BridgeWatcher is responsible for looking over Terra blockchain and reporting new transactions to the bridge
	BridgeWatcher struct {
		url    string
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
func NewTerraBridgeWatcher(url string, bridge string, lockEvents chan *common.ChainLock, setEvents chan *common.GuardianSet) *BridgeWatcher {
	return &BridgeWatcher{url: url, bridge: bridge, lockChan: lockEvents, setChan: setEvents}
}

// Run is the main Terra Bridge run cycle
func (e *BridgeWatcher) Run(ctx context.Context) error {
	errC := make(chan error)
	logger := supervisor.Logger(ctx)

	u, err := url.Parse(e.url)
	if err != nil {
		return fmt.Errorf("parsing terrad url failed: %w", err)
	}

	logger.Info("connecting to ", zap.Any("url", u))

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("socket dial failed: %w", err)
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
	c.WriteJSON(command)

	// Wait for the success response
	_, _, err = c.ReadMessage()
	if err != nil {
		return fmt.Errorf("event subsciption failed: %w", err)
	}
	logger.Info("Subscribed to new transaction events")

	go func() {
		defer close(errC)

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				logger.Error("error reading channel: ", zap.Any("error", err))
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

			if targetChain.Exists() && tokenChain.Exists() && tokenDecimals.Exists() && token.Exists() &&
				sender.Exists() && recipient.Exists() && amount.Exists() && amount.Exists() && nonce.Exists() && txHash.Exists() {

				logger.Info("Token lock detected on Terra")
				logger.Info("\ttxHash: ", zap.Any("txHash", txHash))
				logger.Info("\ttargetChain: ", zap.Any("targetChain", targetChain))
				logger.Info("\ttokenChain: ", zap.Any("tokenChain", tokenChain))
				logger.Info("\ttokenDecimals: ", zap.Any("tokenDecimals", tokenDecimals))
				logger.Info("\ttoken: ", zap.Any("token", token))
				logger.Info("\tsender: ", zap.Any("sender", sender))
				logger.Info("\trecipient: ", zap.Any("recipient", recipient))
				logger.Info("\tamount: ", zap.Any("amount", amount))
				logger.Info("\tnonce: ", zap.Any("nonce", nonce))

				senderAddress, err := StringToAddress(sender.String())
				if err != nil {
					logger.Error("cannod decode hex ", zap.Any("value", sender.String()))
					continue
				}
				recipientAddress, err := StringToAddress(recipient.String())
				if err != nil {
					logger.Error("cannod decode hex ", zap.Any("value", recipient.String()))
					continue
				}
				tokenAddress, err := StringToAddress(token.String())
				if err != nil {
					logger.Error("cannod decode hex ", zap.Any("value", token.String()))
					continue
				}
				txHashValue, err := StringToHash(txHash.String())
				if err != nil {
					logger.Error("cannod decode hex ", zap.Any("value", txHash.String()))
					continue
				}
				lock := &common.ChainLock{
					TxHash:        txHashValue,
					Timestamp:     time.Now(), // No timestamp available, consider adding it into transaction logs or request additionally from the blockchain
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
		}
	}()

	select {
	case <-ctx.Done():
		err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			logger.Error("error on closing socket ", zap.Any("error", err))
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
