package accounting

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/prometheus/client_golang/prometheus"

	eth_common "github.com/ethereum/go-ethereum/common"

	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

var (
	connectionErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_accounting_connection_errors_total",
			Help: "Total number of connection errors on accounting",
		})
	messagesConfirmed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_accounting_messages_confirmed_total",
			Help: "Total number of verified messages found on accounting",
		})
	queryLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name: "wormhole_accounting_query_latency",
			Help: "Latency histogram for RPC calls on accounting",
		})
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

// watcher reads transaction events from the smart contract and publishes them.
func (acct *Accounting) watcher(ctx context.Context) error {
	errC := make(chan error)

	acct.logger.Info("acct: connecting to websocket", zap.String("url", acct.wsUrl))

	c, _, err := websocket.DefaultDialer.DialContext(ctx, acct.wsUrl, nil)
	if err != nil {
		connectionErrors.Inc()
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	defer c.Close()

	// Subscribe to events from the smart contract.
	params := [...]string{fmt.Sprintf("execute._contract_address='%s'", acct.contract)}
	//params := [...]string{fmt.Sprintf("tm.event='Tx' AND execute._contract_address='%s'", acct.contract)}
	command := &clientRequest{
		JSONRPC: "2.0",
		Method:  "subscribe",
		Params:  params,
		ID:      1,
	}
	err = c.WriteJSON(command)
	if err != nil {
		connectionErrors.Inc()
		return fmt.Errorf("websocket subscription failed: %w", err)
	}

	// Wait for the success response.
	_, _, err = c.ReadMessage()
	if err != nil {
		connectionErrors.Inc()
		return fmt.Errorf("event subscription failed: %w", err)
	}
	acct.logger.Info("acct: successfully subscribed to events")

	go func() {
		defer close(errC)

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				connectionErrors.Inc()
				acct.logger.Error("acct: error reading watcher channel", zap.Error(err))
				errC <- err
				return
			}

			// Received a message from the smart contract.
			json := string(message)

			txHashRaw := gjson.Get(json, "result.events.tx\\.hash.0")
			if !txHashRaw.Exists() {
				acct.logger.Warn("message does not have tx hash", zap.String("payload", json))
				continue
			}
			txHash := txHashRaw.String()

			events := gjson.Get(json, "result.data.value.TxResult.result.events")
			if !events.Exists() {
				acct.logger.Warn("message has no events", zap.String("payload", json))
				continue
			}

			pendingTransfers := acct.EventsToTransfers(txHash, events.Array())

			acct.mutex.Lock()
			for _, pk := range pendingTransfers {
				pe, exists := acct.pendingTransfers[*pk]
				if exists {
					acct.logger.Info("acct: pending transfer has been approved", zap.Stringer("emitterChainId", pk.emitterChainId), zap.Stringer("txHash", pk.txHash))
					acct.publishTransfer(pe)
					messagesConfirmed.Inc()
				} else {
					acct.logger.Info("acct: unknown transfer has been approved, ignoring it", zap.Stringer("emitterChainId", pk.emitterChainId), zap.Stringer("txHash", pk.txHash))
				}
			}
			acct.mutex.Unlock()
		}
	}()

	select {
	case <-ctx.Done():
		err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			acct.logger.Error("acct: error closing watcher socket ", zap.Error(err))
		}
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

// TODO: Need to see what events from the contract really look like, and implement this properly.
// TODO: Need to handle errors like CommitTransferError (and any others).
func (acct *Accounting) EventsToTransfers(txHash string, events []gjson.Result) []*pendingKey {
	pendingTransfers := make([]*pendingKey, 0, len(events))
	for _, event := range events {
		// TODO This parsing code was lifted from the cosmwasm watcher. If it works here, we should factor it out and share it.
		if !event.IsObject() {
			acct.logger.Warn("event is invalid", zap.String("tx_hash", txHash), zap.String("event", event.String()))
			continue
		}
		eventType := gjson.Get(event.String(), "type")
		if eventType.String() != "wasm" {
			continue
		}

		attributes := gjson.Get(event.String(), "attributes")
		if !attributes.Exists() {
			acct.logger.Warn("message event has no attributes", zap.String("tx_hash", txHash), zap.String("event", event.String()))
			continue
		}
		mappedAttributes := map[string]string{}
		for _, attribute := range attributes.Array() {
			if !attribute.IsObject() {
				acct.logger.Warn("event attribute is invalid", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}
			keyBase := gjson.Get(attribute.String(), "key")
			if !keyBase.Exists() {
				acct.logger.Warn("event attribute does not have key", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}
			valueBase := gjson.Get(attribute.String(), "value")
			if !valueBase.Exists() {
				acct.logger.Warn("event attribute does not have value", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}

			key, err := base64.StdEncoding.DecodeString(keyBase.String())
			if err != nil {
				acct.logger.Warn("event key attribute is invalid", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()))
				continue
			}
			value, err := base64.StdEncoding.DecodeString(valueBase.String())
			if err != nil {
				acct.logger.Warn("event value attribute is invalid", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
				continue
			}

			if _, ok := mappedAttributes[string(key)]; ok {
				acct.logger.Debug("duplicate key in events", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
				continue
			}

			mappedAttributes[string(key)] = string(value)
		}

		contractAddress, ok := mappedAttributes["_contract_address"]
		if !ok {
			acct.logger.Warn("wasm event without contract address field set", zap.String("event", event.String()))
			continue
		}

		// This event is not from the accounting contract.
		if contractAddress != acct.contract {
			continue
		}

		/*
			This is what a transfer event looks like. I'm not sure what the tags will actually be though. . .
			Ok(Some(
				Event::new("Transfer")
					.add_attribute("emitter_chain", o.key.emitter_chain().to_string())
					.add_attribute("emitter_address", o.key.emitter_address().to_string())
					.add_attribute("sequence", o.key.sequence().to_string())
					.add_attribute("nonce", o.nonce.to_string())
					.add_attribute("tx_hash", o.tx_hash.to_base64())
					.add_attribute("payload", o.payload.to_base64()),
			))
		*/

		emitterChainStr, ok := mappedAttributes["Transfer.emitter_chain"]
		if !ok {
			acct.logger.Error("acct: transfer event does not have the emitter_chain field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		emitterAddrStr, ok := mappedAttributes["Transfer.emitter_address"]
		if !ok {
			acct.logger.Error("acct: transfer event does not have the emitter_address field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		sequenceStr, ok := mappedAttributes["Transfer.sequence"]
		if !ok {
			acct.logger.Error("acct: transfer event does not have the sequence field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		nonceStr, ok := mappedAttributes["Transfer.nonce"]
		if !ok {
			acct.logger.Error("acct: transfer event does not have the nonce field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		xferTxHashStr, ok := mappedAttributes["Transfer.tx_hash"]
		if !ok {
			acct.logger.Error("acct: transfer event does not have the tx_hash field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}
		payloadStr, ok := mappedAttributes["Transfer.payload"]
		if !ok {
			acct.logger.Error("acct: transfer event does not have the payload field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}

		acct.logger.Info("acct: transfer event detected on cosmwasm",
			zap.String("emitter_chain", emitterChainStr),
			zap.String("emitter_address", emitterAddrStr),
			zap.String("sequence", sequenceStr),
			zap.String("nonce", nonceStr),
			zap.String("tx_hash", xferTxHashStr),
			zap.String("payload", payloadStr),
		)

		emitterChainInt, err := strconv.ParseUint(emitterChainStr, 10, 16)
		if err != nil {
			acct.logger.Error("acct: emitter_chain in transfer cannot be parsed as int", zap.String("tx_hash", txHash), zap.String("value", emitterChainStr))
			continue
		}
		emitterChainId := vaa.ChainID(emitterChainInt)

		xferTxHash, err := StringToHash(xferTxHashStr)
		if err != nil {
			acct.logger.Error("acct: tx_hash in transfer cannot decode tx hash hex", zap.String("tx_hash", txHash), zap.String("value", xferTxHashStr))
			continue
		}

		pendingTransfer := &pendingKey{emitterChainId: emitterChainId, txHash: xferTxHash}
		pendingTransfers = append(pendingTransfers, pendingTransfer)
	}

	return pendingTransfers
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
