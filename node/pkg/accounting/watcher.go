package accounting

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	eth_common "github.com/ethereum/go-ethereum/common"

	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/supervisor"
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

	acct.logger.Info("acctwatch: creating watcher", zap.String("url", acct.wsUrl), zap.String("contract", acct.contract))

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

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	go func() {
		defer close(errC)

		for {
			acct.logger.Info("acctwatch: tick")
			_, message, err := c.ReadMessage()
			if err != nil {
				connectionErrors.Inc()
				acct.logger.Error("acctwatch: error reading watcher channel", zap.Error(err))
				time.Sleep(100 * time.Millisecond)
				acct.logger.Info("acctwatch: posting error", zap.Error(err))
				time.Sleep(100 * time.Millisecond)
				errC <- err
				acct.logger.Info("acctwatch: posted error", zap.Error(err))
				return
			}

			// Received a message from the smart contract.
			acct.logger.Info("acctwatch: tock")
			json := string(message)

			txHashRaw := gjson.Get(json, "result.events.tx\\.hash.0")
			if !txHashRaw.Exists() {
				acct.logger.Warn("acctwatch: message does not contain tx hash", zap.String("payload", json))
				continue
			}
			txHash := txHashRaw.String()

			events := gjson.Get(json, "result.data.value.TxResult.result.events")
			if !events.Exists() {
				acct.logger.Warn("acctwatch: message has no events", zap.String("payload", json))
				continue
			}

			pendingTransfers := acct.parseEvents(txHash, events.Array())

			acct.mutex.Lock()
			for _, msg := range pendingTransfers {
				msgId := msg.MessageIDString()
				pe, exists := acct.pendingTransfers[msgId]
				if exists {
					digest := msg.CreateDigest()
					if pe.digest != digest {
						digestMismatches.Inc()
						acct.logger.Error("acctwatch: digest mismatch, dropping transfer",
							zap.String("msgID", msgId),
							zap.String("oldDigest", pe.digest),
							zap.String("newDigest", digest),
						)

						acct.deletePendingTransfer(msgId)
						continue
					}
					acct.logger.Info("acctwatch: pending transfer has been approved", zap.String("msgId", msgId))
					acct.publishTransfer(pe)
					transfersApproved.Inc()
				} else {
					acct.logger.Info("acctwatch: unknown transfer has been approved, ignoring it", zap.String("msgId", msgId))
				}
			}
			acct.mutex.Unlock()
		}

		acct.logger.Error("acctwatch: exiting go func")
	}()

	select {
	case <-ctx.Done():
		err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			acct.logger.Error("acctwatch: error closing watcher socket", zap.Error(err))
		}
		acct.logger.Info("acctwatch: exiting watcher 1")
		return ctx.Err()
	case err := <-errC:
		acct.logger.Error("acctwatch: watcher encountered an error", zap.Error(err))
		acct.logger.Info("acctwatch: exiting watcher 2")
		return err
	}

	acct.logger.Info("acctwatch: exiting watcher 3")
	return nil
}

// TODO: Need to handle errors like CommitTransferError (and any others).
func (acct *Accounting) parseEvents(txHash string, events []gjson.Result) []*common.MessagePublication {
	results := make([]*common.MessagePublication, 0, len(events))
	for _, event := range events {
		if !event.IsObject() {
			acct.logger.Warn("acctwatch: event is invalid", zap.String("tx_hash", txHash), zap.String("event", event.String()))
			continue
		}
		eventType := gjson.Get(event.String(), "type")
		// TODO When development is complete, uncomment this. We just want to log everything for now.
		// if eventType.String() != "wasm-Transfer" {
		// 	acct.logger.Info("acctwatch: debug: ignoring event", zap.String("eventType", eventType.String()), zap.String("event", event.String()))
		// 	continue
		// }

		attributes := gjson.Get(event.String(), "attributes")
		if !attributes.Exists() {
			acct.logger.Warn("acctwatch: message event has no attributes", zap.String("tx_hash", txHash), zap.String("event", event.String()))
			continue
		}
		mappedAttributes := map[string]string{}
		for _, attribute := range attributes.Array() {
			if !attribute.IsObject() {
				acct.logger.Warn("acctwatch: event attribute is invalid", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}

			keyBase := gjson.Get(attribute.String(), "key")
			if !keyBase.Exists() {
				acct.logger.Warn("acctwatch: event attribute does not contain key", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}

			valueBase := gjson.Get(attribute.String(), "value")
			if !valueBase.Exists() {
				acct.logger.Warn("acctwatch: event attribute does not contain value", zap.String("tx_hash", txHash), zap.String("attribute", attribute.String()))
				continue
			}

			key, err := base64.StdEncoding.DecodeString(keyBase.String())
			if err != nil {
				acct.logger.Warn("acctwatch: event key attribute is invalid", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()))
				continue
			}

			value, err := base64.StdEncoding.DecodeString(valueBase.String())
			if err != nil {
				acct.logger.Warn("acctwatch: event value attribute is invalid", zap.String("tx_hash", txHash), zap.String("key", keyBase.String()), zap.String("value", valueBase.String()))
				continue
			}

			if _, ok := mappedAttributes[string(key)]; ok {
				acct.logger.Debug("acctwatch: duplicate key in events",
					zap.String("eventType", eventType.String()),
					zap.String("tx_hash", txHash),
					zap.String("key", keyBase.String()),
					zap.String("value", valueBase.String()),
				)
				continue
			}

			acct.logger.Info("acctwatch: debug: parsing event", zap.String("eventType", eventType.String()), zap.String("key", string(key)), zap.String("value", string(value)), zap.String("valueBase", valueBase.String()))
			mappedAttributes[string(key)] = string(value)
		}

		// TODO When we get rid of the above todo, we can delete this.
		if eventType.String() != "wasm-Transfer" {
			continue
		}

		contractAddress, ok := mappedAttributes["_contract_address"]
		if !ok {
			acct.logger.Warn("acctwatch: wasm event without contract address field set", zap.String("event", event.String()))
			continue
		}

		// This event is not from the accounting contract.
		if contractAddress != acct.contract {
			acct.logger.Info("acctwatch: debug: ignoring event for different contract", zap.String("contractAddress", contractAddress), zap.String("expected", acct.contract))
			continue
		}

		// TODO Is it intentional that we have to double base64 decode tx_hash and payload?

		//////////////////// TxHash
		xferTxHashStr, ok := mappedAttributes["tx_hash"]
		if !ok {
			acct.logger.Error("acctwatch: transfer event does not contain the tx_hash field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}

		xferTxHashBytes, err := base64.StdEncoding.DecodeString(xferTxHashStr)
		if err != nil {
			acct.logger.Warn("acctwatch: tx_hash is not in base64", zap.String("tx_hash", txHash), zap.String("tx_hash", xferTxHashStr))
		}
		xferTxHashStr = hex.EncodeToString(xferTxHashBytes)

		xferTxHash, err := StringToHash(xferTxHashStr)
		if err != nil {
			acct.logger.Error("acctwatch: tx_hash in transfer cannot decode tx hash hex", zap.String("tx_hash", txHash), zap.String("value", xferTxHashStr))
			continue
		}

		//////////////////// Timestamp
		timestampStr, ok := mappedAttributes["timestamp"]
		if !ok {
			acct.logger.Error("acctwatch: transfer event does not contain the timestamp field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}

		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			acct.logger.Error("acctwatch: failed to parse the timestamp field", zap.String("tx_hash", txHash), zap.String("timestampStr", timestampStr), zap.Error(err))
			continue
		}

		//////////////////// Nonce
		nonceStr, ok := mappedAttributes["nonce"]
		if !ok {
			acct.logger.Error("acctwatch: transfer event does not contain the nonce field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}

		nonce, err := strconv.ParseUint(nonceStr, 10, 64)
		if err != nil {
			acct.logger.Error("acctwatch: failed to parse the nonce field", zap.String("tx_hash", txHash), zap.String("nonceStr", nonceStr), zap.Error(err))
			continue
		}

		//////////////////// Sequence
		sequenceStr, ok := mappedAttributes["sequence"]
		if !ok {
			acct.logger.Error("acctwatch: transfer event does not contain the sequence field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}

		sequence, err := strconv.ParseUint(sequenceStr, 10, 64)
		if err != nil {
			acct.logger.Error("acctwatch: failed to parse the sequence field", zap.String("tx_hash", txHash), zap.String("sequenceStr", sequenceStr), zap.Error(err))
			continue
		}

		//////////////////// EmitterChain
		emitterChainStr, ok := mappedAttributes["emitter_chain"]
		if !ok {
			acct.logger.Error("acctwatch: transfer event does not contain the emitter_chain field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}

		emitterChainInt, err := strconv.ParseUint(emitterChainStr, 10, 16)
		if err != nil {
			acct.logger.Error("acctwatch: failed to parse the emitter_chain field", zap.String("tx_hash", txHash), zap.String("value", emitterChainStr))
			continue
		}
		emitterChain := vaa.ChainID(emitterChainInt)

		//////////////////// EmitterChain
		emitterAddressStr, ok := mappedAttributes["emitter_address"]
		if !ok {
			acct.logger.Error("acctwatch: transfer event does not contain the emitter_address field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}

		emitterAddress, err := vaa.StringToAddress(emitterAddressStr)
		if err != nil {
			acct.logger.Error("acctwatch: failed to parse the emitter_address field", zap.String("tx_hash", txHash), zap.String("value", emitterAddressStr))
			continue
		}

		//////////////////// Payload
		payloadStr, ok := mappedAttributes["payload"]
		if !ok {
			acct.logger.Error("acctwatch: transfer event does not contain the payload field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}

		payload, err := base64.StdEncoding.DecodeString(payloadStr)
		if err != nil {
			acct.logger.Warn("acctwatch: payload is not in base64", zap.String("tx_hash", txHash), zap.String("payload", payloadStr))
		}

		//////////////////// ConsistencyLevel
		consistencyLevelStr, ok := mappedAttributes["consistency_level"]
		if !ok {
			acct.logger.Error("acctwatch: transfer event does not contain the consistency_level field", zap.String("tx_hash", txHash), zap.String("attributes", attributes.String()))
			continue
		}

		consistencyLevel64, err := strconv.ParseUint(consistencyLevelStr, 10, 8)
		if err != nil {
			acct.logger.Error("acctwatch: failed to parse the consistency_level field", zap.String("tx_hash", txHash), zap.String("consistencyLevelStr", consistencyLevelStr), zap.Error(err))
			continue
		}
		consistencyLevel := uint8(consistencyLevel64)

		acct.logger.Info("acctwatch: transfer event detected",
			zap.String("tx_hash", xferTxHashStr),
			zap.Int64("timestamp", timestamp),
			zap.Uint64("nonce", nonce),
			zap.Stringer("emitter_chain", emitterChain),
			zap.Stringer("emitter_address", emitterAddress),
			zap.Uint64("sequence", sequence),
			zap.Uint8("consistency_level", consistencyLevel),
			zap.String("payload", hex.EncodeToString(payload)),
		)

		evt := &common.MessagePublication{
			TxHash:           xferTxHash,
			Timestamp:        time.Unix(timestamp, 0),
			Nonce:            uint32(nonce),
			Sequence:         sequence,
			EmitterChain:     emitterChain,
			EmitterAddress:   emitterAddress,
			Payload:          payload,
			ConsistencyLevel: consistencyLevel,
		}

		results = append(results, evt)
		eventsReceived.Inc()
	}

	return results
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
