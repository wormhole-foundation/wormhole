package accountant

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCommon "github.com/ethereum/go-ethereum/common"

	tmAbci "github.com/tendermint/tendermint/abci/types"
	tmHttp "github.com/tendermint/tendermint/rpc/client/http"
	tmCoreTypes "github.com/tendermint/tendermint/rpc/core/types"
	tmTypes "github.com/tendermint/tendermint/types"

	"go.uber.org/zap"
)

// watcher reads transaction events from the smart contract and publishes them.
func (acct *Accountant) watcher(ctx context.Context) error {
	errC := make(chan error)

	acct.logger.Info("acctwatch: creating watcher", zap.String("url", acct.wsUrl), zap.String("contract", acct.contract))
	tmConn, err := tmHttp.New(acct.wsUrl, "/websocket")
	if err != nil {
		connectionErrors.Inc()
		return fmt.Errorf("failed to establish tendermint connection: %w", err)
	}

	if err := tmConn.Start(); err != nil {
		connectionErrors.Inc()
		return fmt.Errorf("failed to start tendermint connection: %w", err)
	}
	defer func() {
		if err := tmConn.Stop(); err != nil {
			connectionErrors.Inc()
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
		return fmt.Errorf("failed to subscribe to accountant events: %w", err)
	}
	defer func() {
		if err := tmConn.UnsubscribeAll(ctx, "guardiand"); err != nil {
			acct.logger.Error("acctwatch: failed to unsubscribe from events", zap.Error(err))
		}
	}()

	go acct.handleEvents(ctx, events, errC)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

// handleEvents handles events from the tendermint client library.
func (acct *Accountant) handleEvents(ctx context.Context, evts <-chan tmCoreTypes.ResultEvent, errC chan error) {
	defer close(errC)

	for {
		select {
		case <-ctx.Done():
			return
		case e := <-evts:
			tx, ok := e.Data.(tmTypes.EventDataTx)
			if !ok {
				acct.logger.Error("unknown data from event subscription", zap.Stringer("e.Data", reflect.TypeOf(e.Data)), zap.Any("event", e))
				continue
			}

			for _, event := range tx.Result.Events {
				if event.Type == "wasm-Observation" {
					evt, err := parseEvent[WasmObservation](acct.logger, event, "wasm-Observation", acct.contract)
					if err != nil {
						acct.logger.Error("failed to parse wasm transfer event", zap.Error(err), zap.Stringer("e.Data", reflect.TypeOf(e.Data)), zap.Any("event", event))
						continue
					}

					eventsReceived.Inc()
					acct.processPendingTransfer(evt)
				} else if event.Type == "wasm-ObservationError" {
					evt, err := parseEvent[WasmObservationError](acct.logger, event, "wasm-ObservationError", acct.contract)
					if err != nil {
						acct.logger.Error("failed to parse wasm observation error event", zap.Error(err), zap.Stringer("e.Data", reflect.TypeOf(e.Data)), zap.Any("event", event))
						continue
					}

					errorEventsReceived.Inc()
					acct.handleTransferError(evt.Key.String(), evt.Error, "transfer error event received")
				} else {
					acct.logger.Debug("ignoring uninteresting event", zap.String("eventType", event.Type))
				}
			}
		}
	}
}

type (
	// WasmObservation represents a transfer event from the smart contract.
	WasmObservation Observation

	// WasmObservationError represents an error event from the smart contract.
	WasmObservationError struct {
		Key   TransferKey `json:"key"`
		Error string      `json:"error"`
	}
)

func parseEvent[T any](logger *zap.Logger, event tmAbci.Event, name string, contractAddress string) (*T, error) {
	attrs := make(map[string]json.RawMessage)
	for _, attr := range event.Attributes {
		if string(attr.Key) == "_contract_address" {
			if string(attr.Value) != contractAddress {
				return nil, fmt.Errorf("%s event from unexpected contract: %s", name, string(attr.Value))
			}
		} else {
			logger.Debug("event attribute", zap.String("event", name), zap.String("key", string(attr.Key)), zap.String("value", string(attr.Value)))
			attrs[string(attr.Key)] = attr.Value
		}
	}

	attrBytes, err := json.Marshal(attrs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal %s event attributes: %w", name, err)
	}

	evt := new(T)
	if err := json.Unmarshal(attrBytes, evt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s event: %w", name, err)
	}

	return evt, nil
}

// processPendingTransfer takes a WasmObservation event, determines if we are expecting it, and if so, publishes it.
func (acct *Accountant) processPendingTransfer(xfer *WasmObservation) {
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()

	acct.logger.Info("acctwatch: transfer event detected",
		zap.String("tx_hash", hex.EncodeToString(xfer.TxHash)),
		zap.Uint32("timestamp", xfer.Timestamp),
		zap.Uint32("nonce", xfer.Nonce),
		zap.Stringer("emitter_chain", vaa.ChainID(xfer.EmitterChain)),
		zap.Stringer("emitter_address", xfer.EmitterAddress),
		zap.Uint64("sequence", xfer.Sequence),
		zap.Uint8("consistency_level", xfer.ConsistencyLevel),
		zap.String("payload", hex.EncodeToString(xfer.Payload)),
	)

	msg := &common.MessagePublication{
		TxHash:           ethCommon.BytesToHash(xfer.TxHash),
		Timestamp:        time.Unix(int64(xfer.Timestamp), 0),
		Nonce:            xfer.Nonce,
		Sequence:         xfer.Sequence,
		EmitterChain:     vaa.ChainID(xfer.EmitterChain),
		EmitterAddress:   xfer.EmitterAddress,
		Payload:          xfer.Payload,
		ConsistencyLevel: xfer.ConsistencyLevel,
	}

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

			acct.deletePendingTransferAlreadyLocked(msgId)
			return
		}
		acct.logger.Info("acctwatch: pending transfer has been approved", zap.String("msgId", msgId))
		acct.publishTransferAlreadyLocked(pe)
		transfersApproved.Inc()
	} else {
		// TODO: We could issue a reobservation request here since it looks like other guardians have seen this transfer but we haven't.
		acct.logger.Info("acctwatch: unknown transfer has been approved, ignoring it", zap.String("msgId", msgId))
	}
}
