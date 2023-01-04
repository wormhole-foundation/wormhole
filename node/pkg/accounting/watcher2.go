package accounting

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	tmAbci "github.com/tendermint/tendermint/abci/types"

	eth_common "github.com/ethereum/go-ethereum/common"

	tmHttp "github.com/tendermint/tendermint/rpc/client/http"
	tmCoreTypes "github.com/tendermint/tendermint/rpc/core/types"
	tmTypes "github.com/tendermint/tendermint/types"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/common"
)

// watcher reads transaction events from the smart contract and publishes them.
func (acct *Accounting) watcher2(ctx context.Context) error {
	errC := make(chan error)

	acct.logger.Info("acctwatch: creating watcher", zap.String("url", acct.wsUrl), zap.String("contract", acct.contract))
	tmConn, err := tmHttp.New(acct.wsUrl, "/websocket")
	if err != nil {
		return fmt.Errorf("failed to establish tendermint connection: %w", err)
	}

	if err := tmConn.Start(); err != nil {
		return fmt.Errorf("failed to start tendermint connection: %w", err)
	}
	defer func() {
		if err := tmConn.Stop(); err != nil {
			acct.logger.Error("acctwatch: failed to stop tendermint connection", zap.Error(err))
		}
	}()

	query := fmt.Sprintf("execute._contract_address='%s'", acct.contract)
	events, err := tmConn.Subscribe(
		ctx,
		"guardiand",
		query,
		64, // channel capacity
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to accounting events: %w", err)
	}
	defer func() {
		if err := tmConn.UnsubscribeAll(ctx, "guardiand"); err != nil {
			acct.logger.Error("acctwatch: failed to unsubscribe for events", zap.Error(err))
		}
	}()

	go acct.handleEvents2(ctx, events, errC)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

func (acct *Accounting) handleEvents2(ctx context.Context, evts <-chan tmCoreTypes.ResultEvent, errC chan error) {
	defer close(errC)

	for {
		select {
		case <-ctx.Done():
			return
		case e := <-evts:
			tx, ok := e.Data.(tmTypes.EventDataTx)
			if !ok {
				acct.logger.Error("acctwatcher: unknown data from event subscription", zap.Stringer("e.Data", reflect.TypeOf(e.Data)), zap.Any("event", e))
				continue
			}

			for _, event := range tx.Result.Events {
				xfer, err := parseWasmTransfer(acct.logger, event)
				if err != nil {
					acct.logger.Error("acctwatcher: failed to parse wasm event", zap.Error(err), zap.Stringer("e.Data", reflect.TypeOf(e.Data)), zap.Any("event", event))
					continue
				}
				if xfer != nil {
					acct.logger.Debug("acctwatcher: received a transfer event")
					eventsReceived.Inc()
					acct.processPendingTransfer(xfer)
				} else {
					acct.logger.Debug("acctwatcher: ignoring non-transfer event", zap.String("eventType", event.Type))
				}
			}
		}
	}
}

/*
2023-01-04T19:27:46.647Z	DEBUG	guardian-0	acctwatch, attribute	{"key": "_contract_address", "value": "wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh"}
2023-01-04T19:27:46.648Z	DEBUG	guardian-0	acctwatch, attribute	{"key": "tx_hash", "value": "guolNsXRZxgwy0kSD5RHnjS1RZao3TafvCZmZnp2X0s="}
2023-01-04T19:27:46.648Z	DEBUG	guardian-0	acctwatch, attribute	{"key": "timestamp", "value": "1672860466"}
2023-01-04T19:27:46.648Z	DEBUG	guardian-0	acctwatch, attribute	{"key": "nonce", "value": "0"}
2023-01-04T19:27:46.648Z	DEBUG	guardian-0	acctwatch, attribute	{"key": "emitter_chain", "value": "2"}
2023-01-04T19:27:46.648Z	DEBUG	guardian-0	acctwatch, attribute	{"key": "emitter_address", "value": "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"}
2023-01-04T19:27:46.648Z	DEBUG	guardian-0	acctwatch, attribute	{"key": "sequence", "value": "1672860466"}
2023-01-04T19:27:46.648Z	DEBUG	guardian-0	acctwatch, attribute	{"key": "consistency_level", "value": "15"}
2023-01-04T19:27:46.648Z	DEBUG	guardian-0	acctwatch, attribute	{"key": "payload", "value": "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA3gtrOnZAAAAAAAAAAAAAAAAAAALYvmvwuqdOCpBwFmecrpGQ6A3QoAAgAAAAAAAAAAAAAAAMEIIJg/M0Vs576zoEb1qD+jTwJ9DCAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="}
2023-01-04T19:27:46.649Z	ERROR	guardian-0	acctwatcher: failed to parse wasm event	{"error": "failed to marshal event attributes: json: error calling MarshalJSON for type json.RawMessage: invalid character 'w' looking for beginning of value", "e.Data": "types.EventDataTx", "event": {"type":"wasm-Transfer","attributes":[{"key":"X2NvbnRyYWN0X2FkZHJlc3M=","value":"d29ybWhvbGUxNDY2bmYzenV4cHlhOHE5ZW14dWtkN3ZmdGFmNmg0cHNyMGEwN3NybDV6dzc0emg4NHlqcTRseWptaA==","index":true},{"key":"dHhfaGFzaA==","value":"Z3VvbE5zWFJaeGd3eTBrU0Q1UkhualMxUlphbzNUYWZ2Q1ptWm5wMlgwcz0=","index":true},{"key":"dGltZXN0YW1w","value":"MTY3Mjg2MDQ2Ng==","index":true},{"key":"bm9uY2U=","value":"MA==","index":true},{"key":"ZW1pdHRlcl9jaGFpbg==","value":"Mg==","index":true},{"key":"ZW1pdHRlcl9hZGRyZXNz","value":"MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDI5MGZiMTY3MjA4YWY0NTViYjEzNzc4MDE2M2I3YjdhOWExMGMxNg==","index":true},{"key":"c2VxdWVuY2U=","value":"MTY3Mjg2MDQ2Ng==","index":true},{"key":"Y29uc2lzdGVuY3lfbGV2ZWw=","value":"MTU=","index":true},{"key":"cGF5bG9hZA==","value":"QVFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQTNndHJPblpBQUFBQUFBQUFBQUFBQUFBQUFBTFl2bXZ3dXFkT0NwQndGbWVjcnBHUTZBM1FvQUFnQUFBQUFBQUFBQUFBQUFBTUVJSUpnL00wVnM1NzZ6b0ViMXFEK2pUd0o5RENBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQT09","index":true}]}}
*/

type WasmTransfer struct {
	TxHash           string      `json:"tx_hash"`
	Timestamp        uint32      `json:"timestamp"`
	Nonce            uint32      `json:"nonce"`
	EmitterChain     uint16      `json:"emitter_chain"`
	EmitterAddress   vaa.Address `json:"emitter_address"`
	Sequence         uint64      `json:"sequence"`
	ConsistencyLevel uint8       `json:"consistency_level"`
	Payload          []byte      `json:"payload"`
}

func parseWasmTransfer(logger *zap.Logger, event tmAbci.Event) (*WasmTransfer, error) {
	if event.Type != "wasm-Transfer" {
		return nil, nil // fmt.Errorf("not a transfer event: %s", event.Type)
	}

	// eventStr, err := json.Marshal(event)
	// if err != nil {
	// 	logger.Error("acctwatcher: failed to marshal event", zap.Error(err))
	// } else {
	// 	logger.Debug("acctwatcher", zap.String("eventStr", string(eventStr)))
	// }

	attrs := make(map[string]json.RawMessage)
	for _, attr := range event.Attributes {

		logger.Debug("acctwatcher: attribute", zap.String("key", string(attr.Key)), zap.String("value", string(attr.Value)))
		attrs[string(attr.Key)] = attr.Value
	}

	attrBytes, err := json.Marshal(attrs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event attributes: %w", err)
	}

	evt := new(WasmTransfer)
	if err := json.Unmarshal(attrBytes, evt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal WasmTransfer event: %w", err)
	}

	return evt, nil
}

func (acct *Accounting) processPendingTransfer(xfer *WasmTransfer) {
	txHashBytes, err := base64.StdEncoding.DecodeString(xfer.TxHash)
	if err != nil {
		acct.logger.Error("acctwatch: tx_hash is not in base64", zap.String("tx_hash", xfer.TxHash))
		return
	}
	txHashStr := hex.EncodeToString(txHashBytes)

	txHash, err := StringToHash2(txHashStr)
	if err != nil {
		acct.logger.Error("acctwatch: tx_hash in transfer cannot decode tx hash hex", zap.String("value", xfer.TxHash))
		return
	}

	acct.logger.Info("acctwatch: transfer event detected",
		zap.Stringer("tx_hash", txHash),
		zap.Uint32("timestamp", xfer.Timestamp),
		zap.Uint32("nonce", xfer.Nonce),
		zap.Stringer("emitter_chain", vaa.ChainID(xfer.EmitterChain)),
		zap.Stringer("emitter_address", xfer.EmitterAddress),
		zap.Uint64("sequence", xfer.Sequence),
		zap.Uint8("consistency_level", xfer.ConsistencyLevel),
		zap.String("payload", hex.EncodeToString(xfer.Payload)),
	)

	msg := &common.MessagePublication{
		TxHash:           txHash,
		Timestamp:        time.Unix(int64(xfer.Timestamp), 0),
		Nonce:            xfer.Nonce,
		Sequence:         xfer.Sequence,
		EmitterChain:     vaa.ChainID(xfer.EmitterChain),
		EmitterAddress:   xfer.EmitterAddress,
		Payload:          xfer.Payload,
		ConsistencyLevel: xfer.ConsistencyLevel,
	}

	msgId := msg.MessageIDString()

	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()

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
			return
		}
		acct.logger.Info("acctwatch: pending transfer has been approved", zap.String("msgId", msgId))
		acct.publishTransfer(pe)
		transfersApproved.Inc()
	} else {
		acct.logger.Info("acctwatch: unknown transfer has been approved, ignoring it", zap.String("msgId", msgId))
	}
}

// StringToHash2 convert string into transaction hash
func StringToHash2(value string) (eth_common.Hash, error) {
	var hash eth_common.Hash
	res, err := hex.DecodeString(value)
	if err != nil {
		return hash, err
	}
	copy(hash[:], res)
	return hash, nil
}
