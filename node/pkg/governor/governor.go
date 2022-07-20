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
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/vaa"

	"go.uber.org/zap"
)

const (
	MainNetMode = 1
	TestNetMode = 2
	DevNetMode  = 3
	GoTestMode  = 4
)

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
		emitterChainID vaa.ChainID
		dailyLimit     uint64
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
		timeStamp time.Time
		token     *tokenEntry
		amount    *big.Int
		msg       *common.MessagePublication
	}

	// Payload of the map of chains being monitored
	chainEntry struct {
		emitterChainId vaa.ChainID
		emitterAddr    vaa.Address
		dailyLimit     uint64
		transfers      []db.Transfer
		pending        []pendingEntry
	}
)

type ChainGovernor struct {
	db                  *db.Database
	logger              *zap.Logger
	mutex               sync.Mutex
	tokens              map[tokenKey]*tokenEntry
	tokensByCoinGeckoId map[string]*tokenEntry
	chains              map[vaa.ChainID]*chainEntry
	msgsToPublish       []*common.MessagePublication
	dayLengthInMinutes  int
	coinGeckoQuery      string
	env                 int
}

func NewChainGovernor(
	logger *zap.Logger,
	db *db.Database,
	env int,
) *ChainGovernor {
	return &ChainGovernor{
		db:                  db,
		logger:              logger,
		tokens:              make(map[tokenKey]*tokenEntry),
		tokensByCoinGeckoId: make(map[string]*tokenEntry),
		chains:              make(map[vaa.ChainID]*chainEntry),
		env:                 env,
	}
}

func (gov *ChainGovernor) Run(ctx context.Context) error {
	if gov.logger != nil {
		gov.logger.Info("cgov: starting chain governor")
	}

	if err := gov.initConfig(); err != nil {
		return err
	}

	if gov.env != GoTestMode {
		if err := gov.loadFromDB(); err != nil {
			return err
		}

		if err := gov.initCoinGecko(ctx); err != nil {
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

		key := tokenKey{chain: vaa.ChainID(ct.chain), addr: addr}
		te := &tokenEntry{cfgPrice: cfgPrice, price: initialPrice, decimals: decimals, symbol: ct.symbol, coinGeckoId: ct.coinGeckoId, token: key}
		te.updatePrice()

		if gov.logger != nil {
			gov.logger.Info("cgov: will monitor token:", zap.Stringer("chain", key.chain),
				zap.Stringer("addr", key.addr),
				zap.String("symbol", te.symbol),
				zap.String("coinGeckoId", te.coinGeckoId),
				zap.String("price", te.price.String()),
				zap.Int64("decimals", dec),
				zap.Int64("origDecimals", ct.decimals),
			)
		}

		gov.tokens[key] = te
		gov.tokensByCoinGeckoId[te.coinGeckoId] = te
	}

	if len(gov.tokens) == 0 {
		return fmt.Errorf("no tokens are configured")
	}

	emitterMap := &common.KnownTokenbridgeEmitters
	if gov.env == TestNetMode {
		emitterMap = &common.KnownTestnetTokenbridgeEmitters
	} else if gov.env == DevNetMode {
		emitterMap = &common.KnownDevnetTokenbridgeEmitters
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

		ce := &chainEntry{emitterChainId: cc.emitterChainID, emitterAddr: emitterAddr, dailyLimit: cc.dailyLimit}

		if gov.logger != nil {
			gov.logger.Info("cgov: will monitor chain:", zap.Stringer("emitterChainId", cc.emitterChainID),
				zap.Stringer("emitterAddr", ce.emitterAddr),
				zap.String("dailyLimit", fmt.Sprint(ce.dailyLimit)))
		}

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
		if gov.logger != nil {
			gov.logger.Error("cgov: failed to process VAA: %v", zap.Error(err))
		}
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

	ce, exists := gov.chains[msg.EmitterChain]

	// If we don't care about this chain, the VAA can be published.
	if !exists {
		return true, nil
	}

	// If we don't care about this emitter, the VAA can be published.
	if msg.EmitterAddress != ce.emitterAddr {
		return true, nil
	}

	// We only care about transfers.
	if !vaa.IsTransfer(msg.Payload) {
		return true, nil
	}

	payload, err := vaa.DecodeTransferPayloadHdr(msg.Payload)
	if err != nil {
		return true, err
	}

	// If we don't care about this token, the VAA can be published.
	tk := tokenKey{chain: payload.OriginChain, addr: payload.OriginAddress}
	token, exists := gov.tokens[tk]
	if !exists {
		return true, nil
	}

	startTime := now.Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))
	prevTotalValue, err := ce.TrimAndSumValue(startTime, gov.db)
	if err != nil {
		if gov.logger != nil {
			gov.logger.Error("cgov: failed to trim transfers", zap.Error(err))
		}

		return false, err
	}

	value, err := computeValue(payload.Amount, token)
	if err != nil {
		return false, err
	}

	newTotalValue := prevTotalValue + value

	if newTotalValue > ce.dailyLimit {
		if gov.logger != nil {
			gov.logger.Error("cgov: enqueuing vaa because it would exceed the daily limit",
				zap.Uint64("value", value),
				zap.Uint64("prevTotalValue", prevTotalValue),
				zap.Uint64("newTotalValue", newTotalValue),
				zap.String("msgID", msg.MessageIDString()))
		}

		ce.pending = append(ce.pending, pendingEntry{timeStamp: now, token: token, amount: payload.Amount, msg: msg})
		if gov.db != nil {
			err = gov.db.StorePendingMsg(msg)
			if err != nil {
				return false, err
			}
		}

		return false, nil
	}

	if gov.logger != nil {
		gov.logger.Info("cgov: posting vaa",
			zap.Uint64("value", value),
			zap.Uint64("prevTotalValue", prevTotalValue),
			zap.Uint64("newTotalValue", newTotalValue),
			zap.String("msgID", msg.MessageIDString()))
	}

	xfer := db.Transfer{Timestamp: now, Value: value, OriginChain: token.token.chain, OriginAddress: token.token.addr, EmitterChain: msg.EmitterChain, EmitterAddress: msg.EmitterAddress, MsgID: msg.MessageIDString()}
	ce.transfers = append(ce.transfers, xfer)
	if gov.db != nil {
		err = gov.db.StoreTransfer(&xfer)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func (gov *ChainGovernor) CheckPending() ([]*common.MessagePublication, error) {
	return gov.CheckPendingForTime(time.Now())
}

func (gov *ChainGovernor) CheckPendingForTime(now time.Time) ([]*common.MessagePublication, error) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	startTime := now.Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))

	var msgsToPublish []*common.MessagePublication
	if len(gov.msgsToPublish) != 0 {
		gov.logger.Info("cgov: posting released vaas", zap.Int("num", len(gov.msgsToPublish)))
		msgsToPublish = gov.msgsToPublish
		gov.msgsToPublish = nil
	}

	for _, ce := range gov.chains {
		// Keep going as long as we find something that will fit.
		for {
			foundOne := false
			prevTotalValue, err := ce.TrimAndSumValue(startTime, gov.db)
			if err != nil {
				if gov.logger != nil {
					gov.logger.Error("cgov: failed to trim transfers", zap.Error(err))
				}

				return msgsToPublish, err
			}

			// Keep going until we find something that fits or hit the end.
			for idx, pe := range ce.pending {
				value, err := computeValue(pe.amount, pe.token)
				if err != nil {
					if gov.logger != nil {
						gov.logger.Error("cgov: failed to compute value for pending vaa",
							zap.Stringer("amount", pe.amount),
							zap.Stringer("price", pe.token.price),
							zap.String("msgID", pe.msg.MessageIDString()),
							zap.Error(err),
						)
					}

					return msgsToPublish, err
				}

				newTotalValue := prevTotalValue + value
				if newTotalValue > ce.dailyLimit {
					// This one won't fit. Keep checking other enqueued ones.
					continue
				}

				// If we get here, we found something that fits. Publish it and remove it from the pending list.
				if gov.logger != nil {
					gov.logger.Info("cgov: posting pending vaa",
						zap.Stringer("amount", pe.amount),
						zap.Stringer("price", pe.token.price),
						zap.Uint64("value", value),
						zap.Uint64("prevTotalValue", prevTotalValue),
						zap.Uint64("newTotalValue", newTotalValue),
						zap.String("msgID", pe.msg.MessageIDString()))
				}

				msgsToPublish = append(msgsToPublish, pe.msg)

				xfer := db.Transfer{Timestamp: now, Value: value, OriginChain: pe.token.token.chain, OriginAddress: pe.token.token.addr, EmitterChain: pe.msg.EmitterChain, EmitterAddress: pe.msg.EmitterAddress, MsgID: pe.msg.MessageIDString()}
				ce.transfers = append(ce.transfers, xfer)

				if gov.db != nil {
					if err := gov.db.StoreTransfer(&xfer); err != nil {
						return msgsToPublish, err
					}
				}

				if gov.db != nil {
					if err := gov.db.DeletePendingMsg(pe.msg); err != nil {
						return msgsToPublish, err
					}
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

func (ce *chainEntry) TrimAndSumValue(startTime time.Time, db *db.Database) (sum uint64, err error) {
	sum, ce.transfers, err = TrimAndSumValue(ce.transfers, startTime, db)
	return sum, err
}

func TrimAndSumValue(transfers []db.Transfer, startTime time.Time, db *db.Database) (uint64, []db.Transfer, error) {
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
		if db != nil {
			for idx := 0; idx <= trimIdx; idx++ {
				if err := db.DeleteTransfer(&transfers[idx]); err != nil {
					return 0, transfers, err
				}
			}
		}

		transfers = transfers[trimIdx+1:]
	}

	return sum, transfers, nil
}

func (tk tokenKey) String() string {
	return tk.chain.String() + ":" + tk.addr.String()
}
