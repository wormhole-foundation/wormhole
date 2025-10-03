package tvm

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"go.uber.org/zap"
)

type TxSubscriber struct {
	tonClient    *ton.APIClient
	addr         *address.Address
	lt           uint64
	tonConfigURL string
	outChan      chan *tlb.Transaction
	logger       *zap.Logger
}

func NewTxSubscriber(
	addr *address.Address,
	lt uint64,
	tonConfigURL string,
	outChan chan *tlb.Transaction,
	logger *zap.Logger,
) (*TxSubscriber, error) {
	pool := liteclient.NewConnectionPool()

	err := pool.AddConnectionsFromConfigUrl(context.Background(), tonConfigURL)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from `%s`: %v", tonConfigURL, err)
	}

	api := ton.NewAPIClient(pool).WithRetry(5)
	return &TxSubscriber{
		tonClient:    api.(*ton.APIClient),
		addr:         addr,
		lt:           lt,
		tonConfigURL: tonConfigURL,
		outChan:      outChan,
		logger:       logger,
	}, nil
}

func (ts *TxSubscriber) Work(ctx context.Context) (err error) {
	ts.logger.Info("Start listening to txs",
		zap.String("chainID", ts.addr.String()),
		zap.String("component", "TxSubscriber"),
		zap.String("addr", ts.addr.String()),
		zap.Uint64("start_tx_lt", ts.lt),
	)

	defer ts.logFinishWork(err)

	go ts.tonClient.SubscribeOnTransactions(ctx, ts.addr, ts.lt, ts.outChan)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

const (
	OpcodeMessagePublished = 0xee3a207e
)

// event::message_published#ee3a207e sender:MsgAddressInt sequence:uint64 nonce:uint32 payload:^Cell consistency_level:uint8
type MessagePublishedEvent struct {
	TransactionID    []byte
	OPCode           uint32
	EmitterAddress   *address.Address
	Sequence         uint64
	Nonce            uint32
	Payload          *cell.Cell
	ConsistencyLevel uint8
}

func (w *Watcher) GetCoreAccountLastLT(ctx context.Context) (uint64, error) {
	block, err := w.Subscriber.tonClient.CurrentMasterchainInfo(ctx)
	if err != nil {
		return 0, fmt.Errorf("CurrentMasterchainInfo: %w", err)
	}

	acc, err := w.Subscriber.tonClient.GetAccount(ctx, block, w.contractAddress)
	if err != nil {
		return 0, fmt.Errorf("w.tonClient.GetAccount: %w", err)
	}

	return acc.LastTxLT, nil
}

func (w *Watcher) GetLastMasterchainBlockSeqno(ctx context.Context) (uint32, error) {
	block, err := w.Subscriber.tonClient.CurrentMasterchainInfo(ctx)
	if err != nil {
		return 0, fmt.Errorf("CurrentMasterchainInfo: %w", err)
	}

	return block.SeqNo, nil
}

func (w *Watcher) inspectBody(logger *zap.Logger, tx *tlb.Transaction, isReobservation bool) error {
	messagePublishEvents, err := w.findMessagePublishedEvents(tx)
	if err != nil {
		logger.Error("failed to unmarshal message publish events", zap.Error(err))
		p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDTON, 1)
		return fmt.Errorf("findMessagePublishedEvent: %w", err)
	}

	if messagePublishEvents != nil {
		for _, messagePublishEvent := range messagePublishEvents {
			emitterAddress, err := vaa.StringToAddress(hex.EncodeToString(messagePublishEvent.EmitterAddress.Data()))
			if err != nil {
				return fmt.Errorf("vaa.StringToAddress(messagePublishEvent.EmitterAddress): %w", err)
			}

			observation := &common.MessagePublication{
				TxID:             messagePublishEvent.TransactionID,
				Timestamp:        time.Unix(int64(tx.Now), 0),
				Nonce:            messagePublishEvent.Nonce,
				Sequence:         messagePublishEvent.Sequence,
				EmitterChain:     w.chainID,
				EmitterAddress:   emitterAddress,
				Payload:          []byte(messagePublishEvent.Payload.String()),
				ConsistencyLevel: messagePublishEvent.ConsistencyLevel,
				IsReobservation:  isReobservation,
			}

			// messagesConfirmed.Inc()
			if isReobservation {
				watchers.ReobservationsByChain.WithLabelValues("ton", "std").Inc()
			}

			logger.Info("TON MESSAGE OBSERVED",
				zap.String("txHash", observation.TxIDString()),
				zap.Time("timestamp", observation.Timestamp),
				zap.Uint32("nonce", observation.Nonce),
				zap.Uint64("sequence", observation.Sequence),
				zap.Stringer("emitter_chain", observation.EmitterChain),
				zap.Stringer("emitter_address", observation.EmitterAddress),
				zap.String("payload_hex", hex.EncodeToString(observation.Payload)),
				zap.Uint8("consistencyLevel", observation.ConsistencyLevel),
				zap.Bool("is_reobservation", isReobservation),
			)

			w.msgChan <- observation //nolint:channelcheck // The channel to the processor is buffered and shared across chains, if it backs up we should stop processing new observations
		}
	}

	return nil
}

func (w *Watcher) findMessagePublishedEvents(tx *tlb.Transaction) ([]*MessagePublishedEvent, error) {
	//ignore such cases
	if tx == nil || tx.IO.Out == nil {
		return nil, nil
	}

	messages, err := tx.IO.Out.ToSlice()
	if err != nil {
		return nil, fmt.Errorf("tx.IO.Out.ToSlice: %w", err)
	}

	//ignore such cases
	if len(messages) == 0 {
		return nil, nil
	}
	externalMessages := make([]*MessagePublishedEvent, 0)

	fmt.Println(len(messages))
	for _, msg := range messages {
		extMsg := msg.AsExternalOut()
		if extMsg != nil {
			msgBody := extMsg.Payload().BeginParse()
			opcode, err := msgBody.LoadUInt(32)
			if err != nil {
				continue
			}
			if opcode != OpcodeMessagePublished {
				continue
			}

			emitterAddress, err := msgBody.LoadAddr()
			if err != nil {
				return nil, fmt.Errorf("failed to load emitter address")
			}

			if emitterAddress == nil {
				return nil, fmt.Errorf("emitter address is nil")
			}

			sequence, err := msgBody.LoadUInt(64)
			if err != nil {
				return nil, fmt.Errorf("failed to load sequence")
			}

			nonce, err := msgBody.LoadUInt(32)
			if err != nil {
				return nil, fmt.Errorf("failed to load nonce")
			}

			payload, err := msgBody.LoadRefCell()
			if err != nil {
				return nil, fmt.Errorf("failed to load payload")
			}

			consistencyLevel, err := msgBody.LoadUInt(8)
			if err != nil {
				return nil, fmt.Errorf("failed to load consistency_level")
			}

			externalMessages = append(externalMessages, &MessagePublishedEvent{
				TransactionID:    extMsg.Payload().Hash(),
				OPCode:           uint32(opcode),
				EmitterAddress:   emitterAddress,
				Sequence:         sequence,
				Nonce:            uint32(nonce),
				Payload:          payload,
				ConsistencyLevel: uint8(consistencyLevel),
			})
		}
	}

	return externalMessages, nil
}

func (w *Watcher) GetTransactionByReobserveRequest(ctx context.Context, txHash []byte) (*tlb.Transaction, error) {
	tx, err := w.Subscriber.tonClient.FindLastTransactionByOutMsgHash(ctx, w.contractAddress, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to find last transaction by out message hash: %w", err)
	}

	return tx, nil
}

func (ts *TxSubscriber) logFinishWork(err error) {
	if err != nil {
		ts.logger.Error("Finished listening to txs with error",
			zap.String("chainID", ts.addr.String()),
			zap.String("component", "TxSubscriber"),
			zap.String("addr", ts.addr.String()),
			zap.Error(err),
		)
	} else {
		ts.logger.Info("Finished listening to txs",
			zap.String("chainID", ts.addr.String()),
			zap.String("component", "TxSubscriber"),
			zap.String("addr", ts.addr.String()),
		)
	}
}
