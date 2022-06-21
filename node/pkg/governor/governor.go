// The purpose of the Chain Governor is to limit the notional TVL that can leave a chain in a single day.
// It works by tracking transfers (types one and three) for a configured set of tokens from a configured set of emitters (chains).
//
// To compute the notional value of a transfer, the governor uses the amount from the transfer multiplied by the maximum of
// a hard coded price and the latest price pulled from CoinkGecko (every five minutes). Once a transfer is published,
// its value (as factored into the daily total) is fixed. However the value of pending transfers is computed using the latest price each interval.
//
// The governor maintains a rolling 24 hour window of transfers that have been received from a configured chain (emitter)
// and compares that value to the configured limit for that chain. If a new transfer would exceed the limit, then it is
// enqueued until it can be published without exceeding the limit. Once the governor has an enqueued transfer, all subsequent
// transfers are enqueued after it, even if their value would not exceed the threshold.
//
// The chain governor checks for pending transfers each minute to see if any can be published yet.
//
// All completed transfers from the last 24 hours and all pending transfers are stored in the Badger DB, and reloaded on start up.
//
// The chain governor supports the following admin client commands:
//   - cgov-status - displays the status of the chain governor to the log file.
//   - cgov-drop-pending-vaa [VAA_ID] - removes the specified transfer from the pending list and discards it.
//   - cgov-release-pending-vaa [VAA_ID] - removes the specified transfer from the pending list and publishes it, without regard to the threshold.
//
// The VAA_ID is of the form "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/3", which is "emitter chain / emitter address / sequence number".
//
// The set of tokens to be monitored is specified in tokens.go, which can be auto generated using the tool in node/hack/governor. See the README there.
//
// The set of chains to be monitored is specified in chains.go, which can be edited by hand.
//
// To enable the chain governor, you must specified the --chainGovernorEnabled guardiand command line argument.

package governor

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/vaa"

	"go.uber.org/zap"
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
		emitterAddr    string
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
	testMode            bool
}

func NewChainGovernor(
	logger *zap.Logger,
	db *db.Database,
) *ChainGovernor {
	return newChainGovernor(logger, db, false)
}

func NewChainGovernorForTest() *ChainGovernor {
	return newChainGovernor(nil, nil, true)
}

func newChainGovernor(logger *zap.Logger, db *db.Database, testMode bool) *ChainGovernor {
	return &ChainGovernor{
		db:                  db,
		logger:              logger,
		tokens:              make(map[tokenKey]*tokenEntry),
		tokensByCoinGeckoId: make(map[string]*tokenEntry),
		chains:              make(map[vaa.ChainID]*chainEntry),
		testMode:            testMode,
	}
}

func (gov *ChainGovernor) Run(ctx context.Context) error {
	if gov.logger != nil {
		gov.logger.Info("cgov: starting chain governor")
	}

	if err := gov.initConfig(); err != nil {
		return err
	}

	if !gov.testMode {
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

	//////////////////////// Start of block to comment out before commit
	// gov.dayLengthInMinutes = 5
	// configTokens = []tokenConfigEntry{
	// 	tokenConfigEntry{chain: 2, addr: "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E", symbol: "WETH", coinGeckoId: "weth", decimals: 18, price: 1774.62},
	// }

	// // This is the data for each chain being monitored. Note that the emitter address is the token bridge.
	// configChains = []chainConfigEntry{
	// 	chainConfigEntry{emitterChainID: vaa.ChainIDEthereum, emitterAddr: "0x0290fb167208af455bb137780163b7b7a9a10c16", dailyLimit: 100000},
	// }
	//////////////////////// End of block to comment out before commit

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

	for _, cc := range configChains {
		emitterAddr, err := vaa.StringToAddress(cc.emitterAddr)
		if err != nil {
			return fmt.Errorf("invalid emitter address: %s", cc.emitterAddr)
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

// This is so we can have consistent config data for unit tests.
func (gov *ChainGovernor) InitConfigForTest(
	emitterChainID vaa.ChainID,
	emitterAddr vaa.Address,
	dailyLimit uint64,
	tokenChainID vaa.ChainID,
	tokenAddr vaa.Address,
	tokenSymbol string,
	tokenPrice float64,
	tokenDecimals int64,
) {
	gov.chains[emitterChainID] = &chainEntry{emitterChainId: emitterChainID, emitterAddr: emitterAddr, dailyLimit: dailyLimit}

	price := big.NewFloat(tokenPrice)
	decimalsFloat := big.NewFloat(math.Pow(10.0, float64(tokenDecimals)))
	decimals, _ := decimalsFloat.Int(nil)
	key := tokenKey{chain: tokenChainID, addr: tokenAddr}
	gov.tokens[key] = &tokenEntry{price: price, decimals: decimals, symbol: tokenSymbol, token: key}
}

func (gov *ChainGovernor) loadFromDB() error {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()
	return gov.loadFromDBAlreadyLocked()
}

func (gov *ChainGovernor) loadFromDBAlreadyLocked() error {
	xfers, pending, err := gov.db.GetChainGovernorData(gov.logger)
	if err != nil {
		gov.logger.Error("cgov: failed to reload transactions from db", zap.Error(err))
		return err
	}

	now := time.Now()
	if len(pending) != 0 {
		sort.SliceStable(pending, func(i, j int) bool {
			return pending[i].Timestamp.Before(pending[j].Timestamp)
		})

		for _, k := range pending {
			gov.reloadPendingTransfer(k, now)
		}
	}

	if len(xfers) != 0 {
		sort.SliceStable(xfers, func(i, j int) bool {
			return xfers[i].Timestamp.Before(xfers[j].Timestamp)
		})

		startTime := now.Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))
		for _, t := range xfers {
			if startTime.Before(t.Timestamp) {
				gov.reloadTransfer(t, now, startTime)
			} else {
				if err := gov.db.DeleteTransfer(t); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (gov *ChainGovernor) reloadPendingTransfer(k *common.MessagePublication, now time.Time) {
	ce, exists := gov.chains[k.EmitterChain]
	if !exists {
		gov.logger.Error("cgov: reloaded pending transfer for unsupported chain, dropping it",
			zap.String("MsgID", k.MessageIDString()),
			zap.Stringer("TxHash", k.TxHash),
			zap.Stringer("Timestamp", k.Timestamp),
			zap.Uint32("Nonce", k.Nonce),
			zap.Uint64("Sequence", k.Sequence),
			zap.Uint8("ConsistencyLevel", k.ConsistencyLevel),
			zap.Stringer("EmitterChain", k.EmitterChain),
			zap.Stringer("EmitterAddress", k.EmitterAddress),
		)
		return
	}

	if k.EmitterAddress != ce.emitterAddr {
		gov.logger.Error("cgov: reloaded pending transfer for unsupported emitter address, dropping it",
			zap.String("MsgID", k.MessageIDString()),
			zap.Stringer("TxHash", k.TxHash),
			zap.Stringer("Timestamp", k.Timestamp),
			zap.Uint32("Nonce", k.Nonce),
			zap.Uint64("Sequence", k.Sequence),
			zap.Uint8("ConsistencyLevel", k.ConsistencyLevel),
			zap.Stringer("EmitterChain", k.EmitterChain),
			zap.Stringer("EmitterAddress", k.EmitterAddress),
		)
		return
	}

	payload, err := vaa.DecodeTransferPayloadHdr(k.Payload)
	if err != nil {
		gov.logger.Error("cgov: failed to parse payload for reloaded pending transfer, dropping it",
			zap.String("MsgID", k.MessageIDString()),
			zap.Stringer("TxHash", k.TxHash),
			zap.Stringer("Timestamp", k.Timestamp),
			zap.Uint32("Nonce", k.Nonce),
			zap.Uint64("Sequence", k.Sequence),
			zap.Uint8("ConsistencyLevel", k.ConsistencyLevel),
			zap.Stringer("EmitterChain", k.EmitterChain),
			zap.Stringer("EmitterAddress", k.EmitterAddress),
			zap.Stringer("tokenChain", payload.OriginChain),
			zap.Stringer("tokenAddress", payload.OriginAddress),
			zap.Error(err),
		)
		return
	}

	tk := tokenKey{chain: payload.OriginChain, addr: payload.OriginAddress}
	token, exists := gov.tokens[tk]
	if !exists {
		gov.logger.Error("cgov: reloaded pending transfer for unsupported token, dropping it",
			zap.String("MsgID", k.MessageIDString()),
			zap.Stringer("TxHash", k.TxHash),
			zap.Stringer("Timestamp", k.Timestamp),
			zap.Uint32("Nonce", k.Nonce),
			zap.Uint64("Sequence", k.Sequence),
			zap.Uint8("ConsistencyLevel", k.ConsistencyLevel),
			zap.Stringer("EmitterChain", k.EmitterChain),
			zap.Stringer("EmitterAddress", k.EmitterAddress),
			zap.Stringer("tokenChain", payload.OriginChain),
			zap.Stringer("tokenAddress", payload.OriginAddress),
		)
		return
	}

	gov.logger.Info("cgov: reloaded pending transfer",
		zap.String("MsgID", k.MessageIDString()),
		zap.Stringer("TxHash", k.TxHash),
		zap.Stringer("Timestamp", k.Timestamp),
		zap.Uint32("Nonce", k.Nonce),
		zap.Uint64("Sequence", k.Sequence),
		zap.Uint8("ConsistencyLevel", k.ConsistencyLevel),
		zap.Stringer("EmitterChain", k.EmitterChain),
		zap.Stringer("EmitterAddress", k.EmitterAddress),
		zap.Stringer("Amount", payload.Amount),
	)

	ce.pending = append(ce.pending, pendingEntry{timeStamp: now, token: token, amount: payload.Amount, msg: k})
}

func (gov *ChainGovernor) reloadTransfer(t *db.Transfer, now time.Time, startTime time.Time) {
	ce, exists := gov.chains[t.EmitterChain]
	if !exists {
		gov.logger.Error("cgov: reloaded transfer for unsupported chain, dropping it",
			zap.Stringer("Timestamp", t.Timestamp),
			zap.Uint64("Value", t.Value),
			zap.Stringer("EmitterChain", t.EmitterChain),
			zap.Stringer("EmitterAddress", t.EmitterAddress),
			zap.String("MsgID", t.MsgID),
		)
		return
	}

	if t.EmitterAddress != ce.emitterAddr {
		gov.logger.Error("cgov: reloaded transfer for unsupported emitter address, dropping it",
			zap.Stringer("Timestamp", t.Timestamp),
			zap.Uint64("Value", t.Value),
			zap.Stringer("OriginChain", t.OriginChain),
			zap.Stringer("OriginAddress", t.OriginAddress),
			zap.String("MsgID", t.MsgID),
		)
		return
	}

	tk := tokenKey{chain: t.OriginChain, addr: t.OriginAddress}
	_, exists = gov.tokens[tk]
	if !exists {
		gov.logger.Error("cgov: reloaded transfer for unsupported token, dropping it",
			zap.Stringer("Timestamp", t.Timestamp),
			zap.Uint64("Value", t.Value),
			zap.Stringer("OriginChain", t.OriginChain),
			zap.Stringer("OriginAddress", t.OriginAddress),
			zap.String("MsgID", t.MsgID),
		)
		return
	}

	gov.logger.Info("cgov: reloaded transfer",
		zap.Stringer("Timestamp", t.Timestamp),
		zap.Uint64("Value", t.Value),
		zap.Stringer("OriginChain", t.OriginChain),
		zap.Stringer("OriginAddress", t.OriginAddress),
		zap.String("MsgID", t.MsgID),
	)

	ce.transfers = append(ce.transfers, *t)
}

func (gov *ChainGovernor) initCoinGecko(ctx context.Context) error {
	//https://api.coingecko.com/api/v3/simple/price?ids=gemma-extending-tech,bitcoin,weth&vs_currencies=usd
	str := "https://api.coingecko.com/api/v3/simple/price?ids="
	first := true
	for coinGeckoId := range gov.tokensByCoinGeckoId {
		if first {
			first = false
		} else {
			str += ","
		}

		str += coinGeckoId
	}

	str += "&vs_currencies=usd"
	gov.coinGeckoQuery = str

	if first {
		if gov.logger != nil {
			gov.logger.Info("cgov: did not find any securities, nothing to do!")
		}

		return nil
	}

	if gov.logger != nil {
		gov.logger.Info("cgov: coingecko query: ", zap.String("query", str))
	}

	timer := time.NewTimer(time.Millisecond) // Start immediately.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				gov.queryCoinGecko()
				timer = time.NewTimer(time.Duration(5) * time.Minute)
			}
		}
	}()

	return nil
}

// This does not return an error. Instead, it just logs the error and we will try again five minutes later.
func (gov *ChainGovernor) queryCoinGecko() {
	gov.logger.Info("cgov: querying coin gecko")
	response, err := http.Get(gov.coinGeckoQuery)
	if err != nil {
		gov.logger.Error("cgov: failed to query coin gecko", zap.String("query", gov.coinGeckoQuery), zap.Error(err))
		return
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		gov.logger.Error("cgov: failed to parse coin gecko response", zap.Error(err))
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(responseData, &result); err != nil {
		gov.logger.Error("cgov: failed to unmarshal coin gecko json", zap.Error(err))
		return
	}

	now := time.Now()
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	for coinGeckoId, data := range result {
		te, exists := gov.tokensByCoinGeckoId[coinGeckoId]
		if exists {
			price := data.(map[string]interface{})["usd"].(float64)
			te.coinGeckoPrice = big.NewFloat(price)
			te.updatePrice()
			te.priceTime = now

			gov.logger.Info("cgov: updated price",
				zap.String("symbol", te.symbol),
				zap.String("coinGeckoId",
					te.coinGeckoId),
				zap.Stringer("price", te.price),
				zap.Stringer("cfgPrice", te.cfgPrice),
				zap.Stringer("coinGeckoPrice", te.coinGeckoPrice),
			)
		}
	}
}

// Returns true if the message can be published, false if it has been added to the pending list.
func (gov *ChainGovernor) ProcessMsg(k *common.MessagePublication) bool {
	publish, err := gov.ProcessMsgForTime(k, time.Now())
	if err != nil {
		if gov.logger != nil {
			gov.logger.Error("cgov: failed to process VAA: %v", zap.Error(err))
		}
		return false
	}

	return publish
}

func (gov *ChainGovernor) ProcessMsgForTime(k *common.MessagePublication, now time.Time) (bool, error) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	ce, exists := gov.chains[k.EmitterChain]

	// If we don't care about this chain, the VAA can be published.
	if !exists {
		return true, nil
	}

	// If we don't care about this emitter, the VAA can be published.
	if k.EmitterAddress != ce.emitterAddr {
		return true, nil
	}

	// We only care about transfers.
	if !vaa.IsTransfer(k.Payload) {
		if gov.logger != nil {
			gov.logger.Info("cgov: ignoring VAA for uninteresting payload type", zap.String("msgID", k.MessageIDString()), zap.Uint8("payload_type", k.Payload[0]))
		}
		return true, nil
	}

	payload, err := vaa.DecodeTransferPayloadHdr(k.Payload)
	if err != nil {
		return true, err
	}

	// If we don't care about this token, the VAA can be published.
	tk := tokenKey{chain: payload.OriginChain, addr: payload.OriginAddress}
	token, exists := gov.tokens[tk]
	if !exists {
		if gov.logger != nil {
			gov.logger.Info("cgov: ignoring VAA for uninteresting token", zap.String("msgID", k.MessageIDString()), zap.Stringer("token", tk))
		}
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

	if (newTotalValue > ce.dailyLimit) || (len(ce.pending) != 0) {
		if gov.logger != nil {
			gov.logger.Error("cgov: enqueuing vaa because it would exceed the daily limit",
				zap.Uint64("value", value),
				zap.Uint64("prevTotalValue", prevTotalValue),
				zap.Uint64("newTotalValue", newTotalValue),
				zap.String("msgID", k.MessageIDString()))
		}

		ce.pending = append(ce.pending, pendingEntry{timeStamp: now, token: token, amount: payload.Amount, msg: k})
		if gov.db != nil {
			err = gov.db.StorePendingMsg(k)
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
			zap.String("msgID", k.MessageIDString()))
	}

	xfer := db.Transfer{Timestamp: now, Value: value, OriginChain: token.token.chain, OriginAddress: token.token.addr, EmitterChain: k.EmitterChain, EmitterAddress: k.EmitterAddress, MsgID: k.MessageIDString()}
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
	return gov.CheckPendingForTime(time.Now(), true)
}

func (gov *ChainGovernor) CheckPendingForTime(now time.Time, publish bool) ([]*common.MessagePublication, error) {
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
		for len(ce.pending) != 0 {
			pe := &ce.pending[0]
			prevTotalValue, err := ce.TrimAndSumValue(startTime, gov.db)
			if err != nil {
				if gov.logger != nil {
					gov.logger.Error("cgov: failed to trim transfers", zap.Error(err))
				}

				return msgsToPublish, err
			}

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
				break
			}

			if publish {
				if gov.logger != nil {
					gov.logger.Info("cgov: posting pending vaa",
						zap.Stringer("amount", pe.amount),
						zap.Stringer("price", pe.token.price),
						zap.Uint64("value", value),
						zap.Uint64("prevTotalValue", prevTotalValue),
						zap.Uint64("newTotalValue", newTotalValue),
						zap.String("msgID", pe.msg.MessageIDString()))
				}
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
			ce.pending = ce.pending[1:]
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
func SumValue(transfers []db.Transfer, startTime time.Time) uint64 {
	if len(transfers) == 0 {
		return 0
	}

	var sum uint64

	for _, t := range transfers {
		if !t.Timestamp.Before(startTime) {
			sum += t.Value
		}
	}

	return sum
}

func (gov *ChainGovernor) Status() string {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	startTime := time.Now().Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))
	for _, ce := range gov.chains {
		valueTrans := SumValue(ce.transfers, startTime)
		s := fmt.Sprintf("cgov: chain: %v, dailyLimit: %v, total: %v, numPending: %v", ce.emitterChainId, ce.dailyLimit, valueTrans, len(ce.pending))
		gov.logger.Info(s)
		if len(ce.pending) != 0 {
			for idx, pe := range ce.pending {
				value, _ := computeValue(pe.amount, pe.token)
				s := fmt.Sprintf("   cgov: chain: %v, pending[%v], value: %v, vaa: %v, time: %v", ce.emitterChainId, idx, value,
					pe.msg.MessageIDString(), pe.timeStamp.String())
				gov.logger.Info(s)
			}
		}
	}

	return "grep the log for \"cgov:\" for status"
}

func (gov *ChainGovernor) Reload() (string, error) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	if gov.db == nil {
		return "", fmt.Errorf("unable to reload because the database is not initialized")
	}

	for _, ce := range gov.chains {
		ce.transfers = nil
		ce.pending = nil
	}

	if err := gov.loadFromDBAlreadyLocked(); err != nil {
		gov.logger.Error("cgov: failed to load from the database", zap.Error(err))
		return "", err
	}

	return "chain governor has been reset and reloaded", nil
}

func (gov *ChainGovernor) DropPendingVAA(vaaId string) (string, error) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	for _, ce := range gov.chains {
		for idx, pe := range ce.pending {
			if pe.msg.MessageIDString() == vaaId {
				value, _ := computeValue(pe.amount, pe.token)
				gov.logger.Info("cgov: dropping pending vaa",
					zap.String("msgId", pe.msg.MessageIDString()),
					zap.Uint64("value", value),
					zap.Stringer("timeStamp", pe.timeStamp),
				)
				ce.pending = append(ce.pending[:idx], ce.pending[idx+1:]...)
				return "vaa has been dropped from the pending list", nil
			}
		}
	}

	return "", fmt.Errorf("vaa not found in the pending list")
}

func (gov *ChainGovernor) ReleasePendingVAA(vaaId string) (string, error) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	for _, ce := range gov.chains {
		for idx, pe := range ce.pending {
			if pe.msg.MessageIDString() == vaaId {
				value, _ := computeValue(pe.amount, pe.token)
				gov.logger.Info("cgov: releasing pending vaa, should be published soon",
					zap.String("msgId", pe.msg.MessageIDString()),
					zap.Uint64("value", value),
					zap.Stringer("timeStamp", pe.timeStamp),
				)

				gov.msgsToPublish = append(gov.msgsToPublish, pe.msg)
				ce.pending = append(ce.pending[:idx], ce.pending[idx+1:]...)
				return "pending vaa has been released and will be published soon", nil
			}
		}
	}

	return "", fmt.Errorf("vaa not found in the pending list")
}

func (gov *ChainGovernor) GetStatsForChain(chainID vaa.ChainID) (numTrans int, valueTrans uint64, numPending int, valuePending uint64) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	ce, exists := gov.chains[chainID]
	if !exists {
		return
	}

	numTrans = len(ce.transfers)
	for _, te := range ce.transfers {
		valueTrans += te.Value
	}

	numPending = len(ce.pending)
	for _, pe := range ce.pending {
		value, _ := computeValue(pe.amount, pe.token)
		valuePending += value
	}

	return
}

func (gov *ChainGovernor) GetStatsForAllChains() (numTrans int, valueTrans uint64, numPending int, valuePending uint64) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	for _, ce := range gov.chains {
		numTrans += len(ce.transfers)
		for _, te := range ce.transfers {
			valueTrans += te.Value
		}

		numPending += len(ce.pending)
		for _, pe := range ce.pending {
			value, _ := computeValue(pe.amount, pe.token)
			valuePending += value
		}
	}

	return
}

func (gov *ChainGovernor) SetDayLengthInMinutes(min int) {
	gov.dayLengthInMinutes = min
}

func (gov *ChainGovernor) SetTokenPriceForTesting(tokenChainID vaa.ChainID, tokenAddrStr string, price float64) error {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	tokenAddr, err := vaa.StringToAddress(tokenAddrStr)
	if err != nil {
		return err
	}

	token, exists := gov.tokens[tokenKey{chain: tokenChainID, addr: tokenAddr}]
	if !exists {
		return fmt.Errorf("token does not exist")
	}

	token.price = big.NewFloat(price)
	return nil
}

// We should use the max(coinGeckoPrice, configuredPrice) as our price for computing notional value.
func (te tokenEntry) updatePrice() {
	if (te.coinGeckoPrice == nil) || (te.coinGeckoPrice.Cmp(te.cfgPrice) < 0) {
		te.price.Set(te.cfgPrice)
	} else {
		te.price.Set(te.coinGeckoPrice)
	}
}

func (tk tokenKey) String() string {
	return tk.chain.String() + ":" + tk.addr.String()
}
