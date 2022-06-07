package governor

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/vaa"

	"go.uber.org/zap"
)

type (
	// Layout of the config data for each token
	tokenConfigEntry struct {
		chain    vaa.ChainID
		addr     string
		symbol   string
		decimals int64
		price    float64
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
		price    *big.Float
		decimals *big.Int
		symbol   string
		token    tokenKey
	}

	pendingEntry struct {
		timeStamp time.Time
		token     *tokenEntry
		value     uint64
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
	db                 *db.Database
	logger             *zap.Logger
	lockC              chan *common.MessagePublication
	mutex              sync.Mutex
	tokens             map[tokenKey]*tokenEntry
	chains             map[vaa.ChainID]*chainEntry
	dayLengthInMinutes int
}

func NewChainGovernor(
	ctx context.Context,
	db *db.Database,
) *ChainGovernor {
	return newChainGovernor(db, supervisor.Logger(ctx))
}

func NewChainGovernorForTest() *ChainGovernor {
	return newChainGovernor(nil, nil)
}

func newChainGovernor(db *db.Database, logger *zap.Logger) *ChainGovernor {
	return &ChainGovernor{
		db:     db,
		logger: logger,
		tokens: make(map[tokenKey]*tokenEntry),
		chains: make(map[vaa.ChainID]*chainEntry),
	}
}

func (gov *ChainGovernor) Run(ctx context.Context) error {
	if gov.logger != nil {
		gov.logger.Info("governor: starting chain governor")
	}

	if err := gov.initConfig(); err != nil {
		return err
	}

	if gov.db != nil {
		if err := gov.loadFromDB(); err != nil {
			return err
		}
	}

	return nil
}

func (gov *ChainGovernor) initConfig() error {
	if gov.logger != nil {
		gov.logger.Info("governor: initConfig")
	}

	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	// This is the data for each token being monitored.
	/* mainnet stuff
	gov.dayLengthInMinutes = 24 * 60
	var configTokens = []tokenConfigEntry{
		tokenConfigEntry{chain: vaa.ChainIDEthereum, addr: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", symbol: "WETH", decimals: 18, price: 1774.62},
		tokenConfigEntry{chain: vaa.ChainIDEthereum, addr: "0x50d1c9771902476076ecfc8b2a83ad6b9355a4c9", symbol: "FTX Token", decimals: 18, price: 26.93},
		tokenConfigEntry{chain: vaa.ChainIDEthereum, addr: "0xe831f96a7a1dce1aa2eb760b1e296c6a74caa9d5", symbol: "NEXM Token", decimals: 18, price: 0.4231},
		tokenConfigEntry{chain: vaa.ChainIDEthereum, addr: "0x707f9118e33a9b8998bea41dd0d46f38bb963fc8", symbol: "bETH Token", decimals: 18, price: 1718.53},
		tokenConfigEntry{chain: vaa.ChainIDEthereum, addr: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", symbol: "USDC Token", decimals: 18, price: 1.00},
	}

	// This is the data for each chain being monitored. Note that the emitter address is the token bridge.
	var configChains = []chainConfigEntry{
		chainConfigEntry{emitterChainID: vaa.ChainIDEthereum, emitterAddr: "0x3ee18B2214AFF97000D974cf647E7C347E8fa585", dailyLimit: 1000000},
		chainConfigEntry{emitterChainID: vaa.ChainIDPolygon, emitterAddr: "0x5a58505a96D1dbf8dF91cB21B54419FC36e93fdE", dailyLimit: 1000000},
	}
	*/
	/* devnet stuff */
	gov.dayLengthInMinutes = 5
	var configTokens = []tokenConfigEntry{
		tokenConfigEntry{chain: vaa.ChainIDEthereum, addr: "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E", symbol: "WETH", decimals: 8, price: 1774.62},
	}

	// This is the data for each chain being monitored. Note that the emitter address is the token bridge.
	var configChains = []chainConfigEntry{
		chainConfigEntry{emitterChainID: vaa.ChainIDEthereum, emitterAddr: "0x0290fb167208af455bb137780163b7b7a9a10c16", dailyLimit: 100000},
	}
	/**/

	for _, ct := range configTokens {
		addr, err := vaa.StringToAddress(ct.addr)
		if err != nil {
			return fmt.Errorf("invalid address: %s", ct.addr)
		}

		price := big.NewFloat(ct.price)

		decimalsFloat := big.NewFloat(math.Pow(10.0, float64(ct.decimals)))
		decimals, _ := decimalsFloat.Int(nil)

		key := tokenKey{chain: ct.chain, addr: addr}
		te := &tokenEntry{price: price, decimals: decimals, symbol: ct.symbol, token: key}

		if gov.logger != nil {
			gov.logger.Info("governor: will monitor token:", zap.Stringer("chain", key.chain),
				zap.Stringer("addr", key.addr),
				zap.String("symbol", te.symbol),
				zap.String("price", te.price.String()),
				zap.Int64("decimals", ct.decimals))
		}

		gov.tokens[key] = te
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
			gov.logger.Info("governor: will monitor chain:", zap.Stringer("emitterChainId", cc.emitterChainID),
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

// TODO: Implement this.
func (gov *ChainGovernor) loadFromDB() error {
	gov.logger.Info("governor: loadFromDB")
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	// Scan the DB.
	// - If the key starts with "PENDING", the payload will be a message. Unmarshall it, create a VAA, and add it to the pending list.
	// - If the key does not start with "PENDING", it's a VAA. If it is less than 24 hours old, add it to the transfers list.

	xfers, pending, err := gov.db.GetChainGovernorData(gov.logger)
	if err != nil {
		gov.logger.Error("governor: failed to read pending transactions from db", zap.Error(err))
	} else {
		if len(pending) != 0 {
			sort.SliceStable(pending, func(i, j int )bool{
				return pending[i].Timestamp.Before(pending[j].Timestamp)
			})

			for _, k := range pending {
				ce, exists := gov.chains[k.EmitterChain]

				// If we don't care about this chain, the VAA can be published.
				if !exists {
					continue
				}
			
				// If we don't care about this emitter, the VAA can be published.
				if k.EmitterAddress != ce.emitterAddr {
					continue
				}

				gov.logger.Info("governor: pending transfer",
					zap.Stringer("TxHash", k.TxHash),
					zap.Stringer("Timestamp", k.Timestamp),
					zap.Uint32("Nonce", k.Nonce),
					zap.Uint64("Sequence", k.Sequence),
					zap.Uint8("ConsistencyLevel", k.ConsistencyLevel),
					zap.Stringer("EmitterChain", k.EmitterChain),
					zap.Stringer("EmitterAddress", k.EmitterAddress),
				)
			}
		}

		if len(xfers) != 0 {
			sort.SliceStable(xfers, func(i, j int )bool{
				return xfers[i].Timestamp.Before(xfers[j].Timestamp)
			})

			startTime := time.Now().Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))
			for _, t := range xfers {
				if startTime.Before(t.Timestamp) {
					gov.logger.Info("governor: transfer",
						zap.Stringer("Timestamp", t.Timestamp),
						zap.Uint64("Value", t.Value),
						zap.Stringer("TokenChainID", t.TokenChainID),
						zap.Stringer("TokenAddress", t.TokenAddress),
						zap.String("MsgID", t.MsgID),
					)
				}
			}
		}
	}

	return nil
}

// Returns true if the message can be published, false if it has been added to the pending list.
func (gov *ChainGovernor) ProcessMsg(k *common.MessagePublication) bool {
	publish, err := gov.ProcessMsgForTime(k, time.Now())
	if err != nil {
		if gov.logger != nil {
			gov.logger.Error("governor: failed to process VAA: %v", zap.Error(err))
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
			gov.logger.Info("governor: ignoring VAA for uninteresting payload type", zap.String("msgID", k.MessageIDString()), zap.Uint8("payload_type", k.Payload[0]))
		}
		return true, nil
	}

	payload, err := vaa.DecodeTransferPayloadHdr(k.Payload)
	if err != nil {
		return true, err
	}

	// If we don't care about this token, the VAA can be published.
	tk := tokenKey{chain: payload.TokenChainID, addr: payload.TokenAddress}
	token, exists := gov.tokens[tk]
	if !exists {
		if gov.logger != nil {
			gov.logger.Info("governor: ignoring VAA for uninteresting token", zap.String("msgID", k.MessageIDString()), zap.Stringer("token", tk))
		}
		return true, nil
	}

	startTime := now.Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))
	prevTotalValue := ce.TrimAndSumValue(startTime, gov.db)
	prevTotalValue += SumPendingValue(ce.pending)

	amountFloat := new(big.Float)
	amountFloat = amountFloat.SetInt(payload.Amount)

	valueFloat := new(big.Float)
	valueFloat = valueFloat.Mul(amountFloat, token.price)

	valueBigInt, _ := valueFloat.Int(nil)
	valueBigInt = valueBigInt.Div(valueBigInt, token.decimals)

	if !valueBigInt.IsUint64() {
		return false, fmt.Errorf("value is too large to fit in uint64")
	}

	value := valueBigInt.Uint64()
	newTotalValue := prevTotalValue + value

	if newTotalValue > ce.dailyLimit {
		if gov.logger != nil {
			gov.logger.Error("governor: enqueuing vaa because it would exceed the daily limit",
				zap.Uint64("value", value),
				zap.Uint64("prevTotalValue", prevTotalValue),
				zap.Uint64("newTotalValue", newTotalValue),
				zap.String("msgID", k.MessageIDString()))
		}

		ce.pending = append(ce.pending, pendingEntry{timeStamp: now, token: token, value: value, msg: k})
		if gov.db != nil {
			err = gov.db.StorePendingMsg(k)
			if err != nil {
				return false, err
			}

			// TODO: Delete this!
			xfers, pending, err := gov.db.GetChainGovernorData(gov.logger)
			if err != nil {
				gov.logger.Error("governor: failed to read pending transactions from db", zap.Error(err))
			} else {
				for _, k := range pending {
					gov.logger.Info("governor: pending transfer",
						zap.Stringer("TxHash", k.TxHash),
						zap.Stringer("Timestamp", k.Timestamp),
						zap.Uint32("Nonce", k.Nonce),
						zap.Uint64("Sequence", k.Sequence),
						zap.Uint8("ConsistencyLevel", k.ConsistencyLevel),
						zap.Stringer("EmitterChain", k.EmitterChain),
						zap.Stringer("EmitterAddress", k.EmitterAddress),
					)
				}

				for _, t := range xfers {
					gov.logger.Info("governor: transfer",
						zap.Stringer("Timestamp", t.Timestamp),
						zap.Uint64("Value", t.Value),
						zap.Stringer("TokenChainID", t.TokenChainID),
						zap.Stringer("TokenAddress", t.TokenAddress),
						zap.String("MsgID", t.MsgID),
					)
				}
			}
		}

		return false, nil
	}

	if gov.logger != nil {
		gov.logger.Info("governor: posting vaa",
			zap.Uint64("value", value),
			zap.Uint64("prevTotalValue", prevTotalValue),
			zap.Uint64("newTotalValue", newTotalValue),
			zap.String("msgID", k.MessageIDString()))
	}

	xfer := db.Transfer{Timestamp: now, Value: value, TokenChainID: token.token.chain, TokenAddress: token.token.addr, MsgID: k.MessageIDString()}
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
	for _, ce := range gov.chains {
		for len(ce.pending) != 0 {
			pe := &ce.pending[0]
			prevTotalValue := ce.TrimAndSumValue(startTime, gov.db)
			newTotalValue := prevTotalValue + pe.value
			if newTotalValue > ce.dailyLimit {
				break
			}

			if publish {
				if gov.logger != nil {
					gov.logger.Info("governor: posting pending vaa",
						zap.String("value", fmt.Sprint(pe.value)),
						zap.String("prevTotalValue", fmt.Sprint(prevTotalValue)),
						zap.String("newTotalValue", fmt.Sprint(newTotalValue)),
						zap.String("msgID", pe.msg.MessageIDString()))
				}
			}

			msgsToPublish = append(msgsToPublish, pe.msg)

			xfer := db.Transfer{Timestamp: now, Value: pe.value, TokenChainID: pe.token.token.chain, TokenAddress: pe.token.token.addr, MsgID: pe.msg.MessageIDString()}
			ce.transfers = append(ce.transfers, xfer)

			if gov.db != nil {
				err := gov.db.StoreTransfer(&xfer)
				if err != nil {
					return msgsToPublish, err
				}
			}

			if gov.db != nil {
				gov.db.DeletePendingMsg(pe.msg)
			}
			ce.pending = ce.pending[1:]
		}
	}

	return msgsToPublish, nil
}

func (ce *chainEntry) TrimAndSumValue(startTime time.Time, db *db.Database) uint64 {
	var sum uint64
	sum, ce.transfers = TrimAndSumValue(ce.transfers, startTime, db)
	return sum
}

func TrimAndSumValue(transfers []db.Transfer, startTime time.Time, db *db.Database) (uint64, []db.Transfer) {
	if len(transfers) == 0 {
		return 0, transfers
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
				db.DeleteTransfer(&transfers[idx])
			}
		}

		transfers = transfers[trimIdx+1:]
	}

	return sum, transfers
}

func SumPendingValue(pending []pendingEntry) uint64 {
	var sum uint64
	for _, pe := range pending {
		sum += pe.value
	}

	return sum
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
		valuePending += pe.value
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
			valuePending += pe.value
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

func (tk tokenKey) String() string {
	return tk.chain.String() + ":" + tk.addr.String()
}

/*
Questions:
- If we have a big transfer at the front of the pending list, should we queue everything after it, or if a smaller one that would not cause us to exceed the limit comes in,
  should we allow that one? For now, assuming we will block it.
- What should I do with a single transfer that exceeds the daily limit? Just keep it in the queue until they manually release or delete it? Assuming we will queue it up.
- If we are going to be reading prices in real time, is the value of an already processed / pending transfer fixed, or does it change as the prices move? Assuming values should change?
*/
