// The purpose of the Chain Governor is to limit the notional TVL that can leave a chain in a single day.
// It works by tracking transfers (types one and three) for a configured set of tokens from a configured set of emitters (chains).
//
// To compute the notional value of a transfer, the governor uses the amount from the transfer multiplied by the maximum of
// a hard coded price and the latest price pulled from CoinkGecko (every five minutes). Once a transfer is published,
// its value (as factored into the daily total) is fixed. However the value of pending transfers is computed using the latest price each interval.
//
// The governor maintains a rolling 24 hour window of transfers that have been received from a configured chain (emitter)
// and compares that value to the configured limit for that chain. If a new transfer would exceed the limit, it is enqueued
// until it can be published without exceeding the limit. Even if the governor has an enqueued transfer, it will still allow
// additional transfers that do not exceed the threshold.
//
// The chain governor checks for pending transfers each minute to see if any can be published yet. It will publish any that can be published
// without exceeding the daily limit, even if one in front of it in the queue is too big.
//
// All completed transfers from the last 24 hours and all pending transfers are stored in the Badger DB, and reloaded on start up.
//
// The chain governor supports admin client commands as documented in governor_cmd.go.
//
// The set of tokens to be monitored is specified in tokens.go, which can be auto generated using the tool in node/hack/governor. See the README there.
//
// The set of chains to be monitored is specified in chains.go, which can be edited by hand.
//
// To enable the chain governor, you must specified the --chainGovernorEnabled guardiand command line argument.

package governor

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"go.uber.org/zap"
)

const (
	MainNetMode = 1
	TestNetMode = 2
	DevNetMode  = 3
	GoTestMode  = 4

	transferComplete = true
	transferEnqueued = false
)

// WARNING: Change me in ./node/db as well
const maxEnqueuedTime = time.Hour * 24

type (
	// Layout of the config data for each token
	tokenConfigEntry struct {
		chain       uint16
		addr        string
		symbol      string
		coinGeckoId string
		decimals    int64
		price       float64
	}

	// Layout of the config data for each chain
	chainConfigEntry struct {
		emitterChainID     vaa.ChainID
		dailyLimit         uint64
		bigTransactionSize uint64
	}

	// Key to the map of the tokens being monitored
	tokenKey struct {
		chain vaa.ChainID
		addr  vaa.Address
	}

	// Payload of the map of the tokens being monitored
	tokenEntry struct {
		price          *big.Float
		decimals       *big.Int
		symbol         string
		coinGeckoId    string
		token          tokenKey
		cfgPrice       *big.Float
		coinGeckoPrice *big.Float
		priceTime      time.Time
	}

	// Payload for each enqueued transfer
	pendingEntry struct {
		token  *tokenEntry // Store a reference to the token so we can get the current price to compute the value each interval.
		amount *big.Int
		hash   string
		dbData db.PendingTransfer // This info gets persisted in the DB.
	}

	// Payload of the map of chains being monitored
	chainEntry struct {
		emitterChainId          vaa.ChainID
		emitterAddr             vaa.Address
		dailyLimit              uint64
		bigTransactionSize      uint64
		checkForBigTransactions bool

		transfers []*db.Transfer
		pending   []*pendingEntry
	}
)

func (ce *chainEntry) isBigTransfer(value uint64) bool {
	return value >= ce.bigTransactionSize && ce.checkForBigTransactions
}

type ChainGovernor struct {
	db                    db.GovernorDB // protected by `mutex`
	logger                *zap.Logger
	mutex                 sync.Mutex
	tokens                map[tokenKey]*tokenEntry     // protected by `mutex`
	tokensByCoinGeckoId   map[string][]*tokenEntry     // protected by `mutex`
	chains                map[vaa.ChainID]*chainEntry  // protected by `mutex`
	msgsSeen              map[string]bool              // protected by `mutex` // Key is hash, payload is consts transferComplete and transferEnqueued.
	msgsToPublish         []*common.MessagePublication // protected by `mutex`
	dayLengthInMinutes    int
	coinGeckoQueries      []string
	env                   int
	nextStatusPublishTime time.Time
	nextConfigPublishTime time.Time
	statusPublishCounter  int64
	configPublishCounter  int64
}

func NewChainGovernor(
	logger *zap.Logger,
	db db.GovernorDB,
	env int,
) *ChainGovernor {
	return &ChainGovernor{
		db:                  db,
		logger:              logger.With(zap.String("component", "cgov")),
		tokens:              make(map[tokenKey]*tokenEntry),
		tokensByCoinGeckoId: make(map[string][]*tokenEntry),
		chains:              make(map[vaa.ChainID]*chainEntry),
		msgsSeen:            make(map[string]bool),
		env:                 env,
	}
}

func (gov *ChainGovernor) Run(ctx context.Context) error {
	gov.logger.Info("starting chain governor")

	if err := gov.initConfig(); err != nil {
		return err
	}

	if gov.env != GoTestMode {
		if err := gov.loadFromDB(); err != nil {
			return err
		}

		if err := gov.initCoinGecko(ctx, true); err != nil {
			return err
		}
	}

	return nil
}

func (gov *ChainGovernor) initConfig() error {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	gov.dayLengthInMinutes = 24 * 60
	configTokens := tokenList()
	configChains := chainList()

	if gov.env == DevNetMode {
		configTokens, configChains = gov.initDevnetConfig()
	} else if gov.env == TestNetMode {
		configTokens, configChains = gov.initTestnetConfig()
	}

	for _, ct := range configTokens {
		addr, err := vaa.StringToAddress(ct.addr)
		if err != nil {
			return fmt.Errorf("invalid address: %s", ct.addr)
		}

		cfgPrice := big.NewFloat(ct.price)
		initialPrice := new(big.Float)
		initialPrice.Set(cfgPrice)

		// Transfers have a maximum of eight decimal places.
		dec := ct.decimals
		if dec > 8 {
			dec = 8
		}

		decimalsFloat := big.NewFloat(math.Pow(10.0, float64(dec)))
		decimals, _ := decimalsFloat.Int(nil)

		// Some Solana tokens don't have the symbol set. In that case, use the chain and token address as the symbol.
		symbol := ct.symbol
		if symbol == "" {
			symbol = fmt.Sprintf("%d:%s", ct.chain, ct.addr)
		}

		key := tokenKey{chain: vaa.ChainID(ct.chain), addr: addr}
		te := &tokenEntry{cfgPrice: cfgPrice, price: initialPrice, decimals: decimals, symbol: symbol, coinGeckoId: ct.coinGeckoId, token: key}
		te.updatePrice()

		gov.tokens[key] = te

		// Multiple tokens can share a CoinGecko price, so we keep an array of tokens per CoinGecko ID.
		cge, cgExists := gov.tokensByCoinGeckoId[te.coinGeckoId]
		if !cgExists {
			gov.tokensByCoinGeckoId[te.coinGeckoId] = []*tokenEntry{te}
		} else {
			cge = append(cge, te)
			gov.tokensByCoinGeckoId[te.coinGeckoId] = cge
		}

		gov.logger.Info("will monitor token:", zap.Stringer("chain", key.chain),
			zap.Stringer("addr", key.addr),
			zap.String("symbol", te.symbol),
			zap.String("coinGeckoId", te.coinGeckoId),
			zap.String("price", te.price.String()),
			zap.Int64("decimals", dec),
			zap.Int64("origDecimals", ct.decimals),
		)
	}

	if len(gov.tokens) == 0 {
		return fmt.Errorf("no tokens are configured")
	}

	emitterMap := &sdk.KnownTokenbridgeEmitters
	if gov.env == TestNetMode {
		emitterMap = &sdk.KnownTestnetTokenbridgeEmitters
	} else if gov.env == DevNetMode {
		emitterMap = &sdk.KnownDevnetTokenbridgeEmitters
	}

	for _, cc := range configChains {
		var emitterAddr vaa.Address
		var err error

		emitterAddrBytes, exists := (*emitterMap)[cc.emitterChainID]
		if !exists {
			return fmt.Errorf("failed to look up token bridge emitter address for chain: %v", cc.emitterChainID)
		}

		emitterAddr, err = vaa.BytesToAddress(emitterAddrBytes)
		if err != nil {
			return fmt.Errorf("failed to convert emitter address for chain: %v", cc.emitterChainID)
		}

		ce := &chainEntry{
			emitterChainId:          cc.emitterChainID,
			emitterAddr:             emitterAddr,
			dailyLimit:              cc.dailyLimit,
			bigTransactionSize:      cc.bigTransactionSize,
			checkForBigTransactions: cc.bigTransactionSize != 0,
		}

		gov.logger.Info("will monitor chain:", zap.Stringer("emitterChainId", cc.emitterChainID),
			zap.Stringer("emitterAddr", ce.emitterAddr),
			zap.String("dailyLimit", fmt.Sprint(ce.dailyLimit)),
			zap.Uint64("bigTransactionSize", ce.bigTransactionSize),
			zap.Bool("checkForBigTransactions", ce.checkForBigTransactions),
		)

		gov.chains[cc.emitterChainID] = ce
	}

	if len(gov.chains) == 0 {
		return fmt.Errorf("no chains are configured")
	}

	return nil
}

// Returns true if the message can be published, false if it has been added to the pending list.
func (gov *ChainGovernor) ProcessMsg(msg *common.MessagePublication) bool {
	publish, err := gov.ProcessMsgForTime(msg, time.Now())
	if err != nil {
		gov.logger.Error("failed to process VAA: %v", zap.Error(err))
		return false
	}

	return publish
}

func (gov *ChainGovernor) ProcessMsgForTime(msg *common.MessagePublication, now time.Time) (bool, error) {
	if msg == nil {
		return false, fmt.Errorf("msg is nil")
	}

	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	msgIsGoverned, ce, token, payload, err := gov.parseMsgAlreadyLocked(msg)
	if err != nil {
		return false, err
	}

	if !msgIsGoverned {
		return true, nil
	}

	hash := gov.HashFromMsg(msg)
	xferComplete, alreadySeen := gov.msgsSeen[hash]
	if alreadySeen {
		if !xferComplete {
			gov.logger.Info("ignoring duplicate vaa because it is enqueued",
				zap.String("msgID", msg.MessageIDString()),
				zap.String("hash", hash),
				zap.Stringer("txHash", msg.TxHash),
			)
			return false, nil
		}

		gov.logger.Info("allowing duplicate vaa to be published again, but not adding it to the notional value",
			zap.String("msgID", msg.MessageIDString()),
			zap.String("hash", hash),
			zap.Stringer("txHash", msg.TxHash),
		)
		return true, nil
	}

	startTime := now.Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))
	prevTotalValue, err := gov.TrimAndSumValueForChain(ce, startTime)
	if err != nil {
		gov.logger.Error("failed to trim transfers",
			zap.String("msgID", msg.MessageIDString()),
			zap.String("hash", hash),
			zap.Stringer("txHash", msg.TxHash),
			zap.Error(err),
		)
		return false, err
	}

	value, err := computeValue(payload.Amount, token)
	if err != nil {
		gov.logger.Error("failed to compute value of transfer",
			zap.String("msgID", msg.MessageIDString()),
			zap.String("hash", hash),
			zap.Stringer("txHash", msg.TxHash),
			zap.Error(err),
		)
		return false, err
	}

	newTotalValue := prevTotalValue + value
	if newTotalValue < prevTotalValue {
		gov.logger.Error("total value has overflowed",
			zap.String("msgID", msg.MessageIDString()),
			zap.String("hash", hash),
			zap.Stringer("txHash", msg.TxHash),
			zap.Uint64("prevTotalValue", prevTotalValue),
			zap.Uint64("newTotalValue", newTotalValue),
		)
		return false, fmt.Errorf("total value has overflowed")
	}

	enqueueIt := false
	var releaseTime time.Time
	if ce.isBigTransfer(value) {
		enqueueIt = true
		releaseTime = now.Add(maxEnqueuedTime)
		gov.logger.Error("enqueuing vaa because it is a big transaction",
			zap.Uint64("value", value),
			zap.Uint64("prevTotalValue", prevTotalValue),
			zap.Uint64("newTotalValue", newTotalValue),
			zap.String("msgID", msg.MessageIDString()),
			zap.Stringer("releaseTime", releaseTime),
			zap.Uint64("bigTransactionSize", ce.bigTransactionSize),
			zap.String("hash", hash),
			zap.Stringer("txHash", msg.TxHash),
		)
	} else if newTotalValue > ce.dailyLimit {
		enqueueIt = true
		releaseTime = now.Add(maxEnqueuedTime)
		gov.logger.Error("enqueuing vaa because it would exceed the daily limit",
			zap.Uint64("value", value),
			zap.Uint64("prevTotalValue", prevTotalValue),
			zap.Uint64("newTotalValue", newTotalValue),
			zap.Stringer("releaseTime", releaseTime),
			zap.String("msgID", msg.MessageIDString()),
			zap.String("hash", hash),
			zap.Stringer("txHash", msg.TxHash),
		)
	}

	if enqueueIt {
		dbData := db.PendingTransfer{ReleaseTime: releaseTime, Msg: *msg}
		gov.logger.Info("writing pending transfer to database", zap.String("msgId", msg.MessageIDString()))
		err = gov.db.StorePendingMsg(&dbData)
		if err != nil {
			gov.logger.Error("failed to store pending vaa",
				zap.String("msgID", msg.MessageIDString()),
				zap.String("hash", hash),
				zap.Stringer("txHash", msg.TxHash),
				zap.Error(err),
			)
			return false, err
		}
		gov.logger.Info("wrote pending transfer to database", zap.String("msgId", msg.MessageIDString()))

		ce.pending = append(ce.pending, &pendingEntry{token: token, amount: payload.Amount, hash: hash, dbData: dbData})
		gov.msgsSeen[hash] = transferEnqueued
		return false, nil
	}

	gov.logger.Info("posting vaa",
		zap.Uint64("value", value),
		zap.Uint64("prevTotalValue", prevTotalValue),
		zap.Uint64("newTotalValue", newTotalValue),
		zap.String("msgID", msg.MessageIDString()),
		zap.String("hash", hash),
		zap.Stringer("txHash", msg.TxHash),
	)

	xfer := db.Transfer{Timestamp: now,
		Value:          value,
		OriginChain:    token.token.chain,
		OriginAddress:  token.token.addr,
		EmitterChain:   msg.EmitterChain,
		EmitterAddress: msg.EmitterAddress,
		MsgID:          msg.MessageIDString(),
		Hash:           hash,
	}
	err = gov.db.StoreTransfer(&xfer)
	if err != nil {
		gov.logger.Error("failed to store transfer",
			zap.String("msgID", msg.MessageIDString()),
			zap.String("hash", hash), zap.Error(err),
			zap.Stringer("txHash", msg.TxHash),
		)
		return false, err
	}

	ce.transfers = append(ce.transfers, &xfer)
	gov.msgsSeen[hash] = transferComplete
	return true, nil
}

// IsGovernedMsg determines if the message applies to the governor. It grabs the lock.
func (gov *ChainGovernor) IsGovernedMsg(msg *common.MessagePublication) (msgIsGoverned bool, err error) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()
	msgIsGoverned, _, _, _, err = gov.parseMsgAlreadyLocked(msg)
	return
}

// parseMsgAlreadyLocked determines if the message applies to the governor and also returns data useful to the governor. It assumes the caller holds the lock.
func (gov *ChainGovernor) parseMsgAlreadyLocked(msg *common.MessagePublication) (bool, *chainEntry, *tokenEntry, *vaa.TransferPayloadHdr, error) {
	// If we don't care about this chain, the VAA can be published.
	ce, exists := gov.chains[msg.EmitterChain]
	if !exists {
		if msg.EmitterChain != vaa.ChainIDPythNet {
			gov.logger.Info("ignoring vaa because the emitter chain is not configured", zap.String("msgID", msg.MessageIDString()))
		}
		return false, nil, nil, nil, nil
	}

	// If we don't care about this emitter, the VAA can be published.
	if msg.EmitterAddress != ce.emitterAddr {
		gov.logger.Info("ignoring vaa because the emitter address is not configured", zap.String("msgID", msg.MessageIDString()))
		return false, nil, nil, nil, nil
	}

	// We only care about transfers.
	if !vaa.IsTransfer(msg.Payload) {
		gov.logger.Info("ignoring vaa because it is not a transfer", zap.String("msgID", msg.MessageIDString()))
		return false, nil, nil, nil, nil
	}

	payload, err := vaa.DecodeTransferPayloadHdr(msg.Payload)
	if err != nil {
		gov.logger.Error("failed to decode vaa", zap.String("msgID", msg.MessageIDString()), zap.Error(err))
		return false, nil, nil, nil, err
	}

	// If we don't care about this token, the VAA can be published.
	tk := tokenKey{chain: payload.OriginChain, addr: payload.OriginAddress}
	token, exists := gov.tokens[tk]
	if !exists {
		gov.logger.Info("ignoring vaa because the token is not in the list", zap.String("msgID", msg.MessageIDString()))
		return false, nil, nil, nil, nil
	}

	return true, ce, token, payload, nil
}

func (gov *ChainGovernor) CheckPending() ([]*common.MessagePublication, error) {
	return gov.CheckPendingForTime(time.Now())
}

func (gov *ChainGovernor) CheckPendingForTime(now time.Time) ([]*common.MessagePublication, error) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	// Note: Using Add() with a negative value because Sub() takes a time and returns a duration, which is not what we want.
	startTime := now.Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))

	var msgsToPublish []*common.MessagePublication
	if len(gov.msgsToPublish) != 0 {
		gov.logger.Info("posting released vaas", zap.Int("num", len(gov.msgsToPublish)))
		msgsToPublish = gov.msgsToPublish
		gov.msgsToPublish = nil
	}

	for _, ce := range gov.chains {
		// Keep going as long as we find something that will fit.
		for {
			foundOne := false
			prevTotalValue, err := gov.TrimAndSumValueForChain(ce, startTime)
			if err != nil {
				gov.logger.Error("failed to trim transfers", zap.Error(err))
				gov.msgsToPublish = msgsToPublish
				return nil, err
			}

			// Keep going until we find something that fits or hit the end.
			for idx, pe := range ce.pending {
				value, err := computeValue(pe.amount, pe.token)
				if err != nil {
					gov.logger.Error("failed to compute value for pending vaa",
						zap.Stringer("amount", pe.amount),
						zap.Stringer("price", pe.token.price),
						zap.String("msgID", pe.dbData.Msg.MessageIDString()),
						zap.Error(err),
					)

					gov.msgsToPublish = msgsToPublish
					return nil, err
				}

				countsTowardsTransfers := true
				if ce.isBigTransfer(value) {
					if now.Before(pe.dbData.ReleaseTime) {
						continue // Keep waiting for the timer to expire.
					}

					countsTowardsTransfers = false
					gov.logger.Info("posting pending big vaa because the release time has been reached",
						zap.Stringer("amount", pe.amount),
						zap.Stringer("price", pe.token.price),
						zap.Uint64("value", value),
						zap.Stringer("releaseTime", pe.dbData.ReleaseTime),
						zap.String("msgID", pe.dbData.Msg.MessageIDString()))
				} else if now.After(pe.dbData.ReleaseTime) {
					countsTowardsTransfers = false
					gov.logger.Info("posting pending vaa because the release time has been reached",
						zap.Stringer("amount", pe.amount),
						zap.Stringer("price", pe.token.price),
						zap.Uint64("value", value),
						zap.Stringer("releaseTime", pe.dbData.ReleaseTime),
						zap.String("msgID", pe.dbData.Msg.MessageIDString()))
				} else {
					newTotalValue := prevTotalValue + value
					if newTotalValue < prevTotalValue {
						gov.msgsToPublish = msgsToPublish
						return nil, fmt.Errorf("total value has overflowed")
					}

					if newTotalValue > ce.dailyLimit {
						// This one won't fit. Keep checking other enqueued ones.
						continue
					}

					gov.logger.Info("posting pending vaa",
						zap.Stringer("amount", pe.amount),
						zap.Stringer("price", pe.token.price),
						zap.Uint64("value", value),
						zap.Uint64("prevTotalValue", prevTotalValue),
						zap.Uint64("newTotalValue", newTotalValue),
						zap.String("msgID", pe.dbData.Msg.MessageIDString()))
				}

				// If we get here, publish it and remove it from the pending list.
				msgsToPublish = append(msgsToPublish, &pe.dbData.Msg)

				if countsTowardsTransfers {
					xfer := db.Transfer{Timestamp: now,
						Value:          value,
						OriginChain:    pe.token.token.chain,
						OriginAddress:  pe.token.token.addr,
						EmitterChain:   pe.dbData.Msg.EmitterChain,
						EmitterAddress: pe.dbData.Msg.EmitterAddress,
						MsgID:          pe.dbData.Msg.MessageIDString(),
						Hash:           pe.hash,
					}

					if err := gov.db.StoreTransfer(&xfer); err != nil {
						gov.msgsToPublish = msgsToPublish
						return nil, err
					}

					ce.transfers = append(ce.transfers, &xfer)
					gov.msgsSeen[pe.hash] = transferComplete
				} else {
					delete(gov.msgsSeen, pe.hash)
				}

				if err := gov.db.DeletePendingMsg(&pe.dbData); err != nil {
					gov.msgsToPublish = msgsToPublish
					return nil, err
				}

				ce.pending = append(ce.pending[:idx], ce.pending[idx+1:]...)
				foundOne = true
				break // We messed up our loop indexing, so we have to break out and start over.
			}

			if !foundOne {
				break
			}
		}
	}

	return msgsToPublish, nil
}

func computeValue(amount *big.Int, token *tokenEntry) (uint64, error) {
	amountFloat := new(big.Float)
	amountFloat = amountFloat.SetInt(amount)

	valueFloat := new(big.Float)
	valueFloat = valueFloat.Mul(amountFloat, token.price)

	valueBigInt, _ := valueFloat.Int(nil)
	valueBigInt = valueBigInt.Div(valueBigInt, token.decimals)

	if !valueBigInt.IsUint64() {
		return 0, fmt.Errorf("value is too large to fit in uint64")
	}

	value := valueBigInt.Uint64()

	return value, nil
}

func (gov *ChainGovernor) TrimAndSumValueForChain(ce *chainEntry, startTime time.Time) (sum uint64, err error) {
	sum, ce.transfers, err = gov.TrimAndSumValue(ce.transfers, startTime)
	return sum, err
}

func (gov *ChainGovernor) TrimAndSumValue(transfers []*db.Transfer, startTime time.Time) (uint64, []*db.Transfer, error) {
	if len(transfers) == 0 {
		return 0, transfers, nil
	}

	var trimIdx int = -1
	var sum uint64

	for idx, t := range transfers {
		if t.Timestamp.Before(startTime) {
			trimIdx = idx
		} else {
			sum += t.Value
		}
	}

	if trimIdx >= 0 {
		for idx := 0; idx <= trimIdx; idx++ {
			if err := gov.db.DeleteTransfer(transfers[idx]); err != nil {
				return 0, transfers, err
			}

			delete(gov.msgsSeen, transfers[idx].Hash)
		}

		transfers = transfers[trimIdx+1:]
	}

	return sum, transfers, nil
}

func (tk tokenKey) String() string {
	return tk.chain.String() + ":" + tk.addr.String()
}

func (gov *ChainGovernor) HashFromMsg(msg *common.MessagePublication) string {
	v := msg.CreateVAA(0) // We can pass zero in as the guardian set index because it is not part of the digest.
	digest := v.SigningDigest()
	return hex.EncodeToString(digest.Bytes())
}
