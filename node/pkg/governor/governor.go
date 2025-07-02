package governor

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

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	guardianDB "github.com/certusone/wormhole/node/pkg/db"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"go.uber.org/zap"
)

const (
	transferComplete = true
	transferEnqueued = false
)

const maxEnqueuedTime = time.Hour * 24

type (
	// Layout of the config data for each token
	TokenConfigEntry struct {
		Chain       uint16
		Addr        string
		Symbol      string
		CoinGeckoId string
		Decimals    int64
		Price       float64
	}

	// Layout of the config data for each chain
	ChainConfigEntry struct {
		EmitterChainID     vaa.ChainID
		DailyLimit         uint64
		BigTransactionSize uint64
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
		flowCancels    bool
	}

	// Payload for each enqueued transfer
	pendingEntry struct {
		token  *tokenEntry // Store a reference to the token so we can get the current price to compute the value each interval.
		amount *big.Int
		hash   string
		dbData guardianDB.PendingTransfer // This info gets persisted in the DB.
	}

	// Used in flow cancel calculations. Wraps a database Transfer. Also contains a signed amount field in order to
	// hold negative values. This field will be used in flow cancel calculations to reduce the Governor usage for a
	// supported token.
	transfer struct {
		dbTransfer *guardianDB.Transfer
		// Corresponds to dbTransfers.Value field, but may be negative
		value int64
	}

	// Payload of the map of chains being monitored. Contains transfer data for both emitted and received transfers.
	// `transfers` with positive Value represent outgoing transfers from the emitterChainId. Transfers with negative
	// Value represent incoming transfers of Assets that can Flow Cancel.
	chainEntry struct {
		emitterChainId          vaa.ChainID
		emitterAddr             vaa.Address
		dailyLimit              uint64
		bigTransactionSize      uint64
		checkForBigTransactions bool

		transfers []transfer
		pending   []*pendingEntry
	}

	// Represents a pair of Governed chains. Ordering is arbitrary.
	corridor struct {
		first  vaa.ChainID
		second vaa.ChainID
	}
)

// valid checks whether a corridor is valid. A corridor is invalid if both chain IDs are equal.
func (p *corridor) valid() bool {
	return p.first != p.second && p.first != vaa.ChainIDUnset && p.second != vaa.ChainIDUnset
}

// equals checks whether two corrdidors are equal. This method exists to demonstrate that the ordering of the
// corridor's elements doesn't matter. It also makes it easier to check whether two chains are 'connected' by a corridor
// without needing to sort or manipulate the elements.
func (p *corridor) equals(p2 *corridor) bool {
	if !p.valid() || !p2.valid() {
		// We want to make invalid corridors unusable, so make them fail the equality check.
		// This is a protective measure in case a developer tries to do some logic on invalid corridors
		// and forgets to check valid() first.
		return false
	}
	if p.first == p2.first && p.second == p2.second {
		return true
	}
	// Ordering doesn't matter
	if p.first == p2.second && p2.first == p.second {
		return true
	}
	return false
}

// newTransferFromDbTransfer converts a db.Transfer into a transfer. It
// performs a bounds check on dbTransfer.Value to ensure it can fit into int64.
// This should always be the case for normal operations as dbTransfer.Value
// represents the USD value of a transfer, which should be less than the max int64 value.
func newTransferFromDbTransfer(dbTransfer *guardianDB.Transfer) (tx transfer, err error) {
	if dbTransfer.Value > math.MaxInt64 {
		return tx, fmt.Errorf("value for db.Transfer exceeds MaxInt64: %d", dbTransfer.Value)
	}
	return transfer{dbTransfer, int64(dbTransfer.Value)}, nil
}

// addFlowCancelTransfer appends a transfer to a ChainEntry's transfers property.
// SECURITY: The calling code is responsible for ensuring that the asset within the transfer is a flow-cancelling asset.
// SECURITY: The calling code is responsible for ensuring that the transfer's source and destination has a matching flow cancel corridor.
// SECURITY: This method performs validation to ensure that the Flow Cancel transfer is valid. This is important to
// ensure that the Governor usage cannot be lowered due to malicious or invalid transfers.
// - the Value must be negative (in order to ultimately reduce the Governor usage of the target chain)
// - the TargetChain must match the chain ID of the Chain Entry
func (ce *chainEntry) addFlowCancelTransfer(transfer transfer) error {
	value := transfer.value
	targetChain := transfer.dbTransfer.TargetChain
	if value > 0 {
		return fmt.Errorf("flow cancel transfer Value must be negative. Value: %d", value)
	}
	if transfer.dbTransfer.Value > math.MaxInt64 {
		return fmt.Errorf("value for transfer.dbTransfer exceeds MaxInt64: %d", transfer.dbTransfer.Value)
	}
	//#nosec G115: Type conversion is safe here because of the MaxInt64 bounds check above
	if value != -int64(transfer.dbTransfer.Value) {
		return fmt.Errorf("transfer is invalid: transfer.value %d must equal the inverse of transfer.dbTransfer.Value %d", value, transfer.dbTransfer.Value)
	}
	if targetChain != ce.emitterChainId {
		return fmt.Errorf("flow cancel transfer TargetChain %s does not match this chainEntry %s", targetChain, ce.emitterChainId)
	}

	ce.transfers = append(ce.transfers, transfer)
	return nil
}

// addFlowCancelTransferFromDbTransfer is a wrapper for addFlowCancelTransfer that works with a `db.Transfer` argument
// instead of a `transfer` argument.
// Validation of transfer data is performed by other methods: see addFlowCancelTransfer, newTransferFromDbTransfer.
func (ce *chainEntry) addFlowCancelTransferFromDbTransfer(dbTransfer *guardianDB.Transfer) error {
	transfer, err := newTransferFromDbTransfer(dbTransfer)
	if err != nil {
		return err
	}
	err = ce.addFlowCancelTransfer(transfer.inverse())
	if err != nil {
		return err
	}
	return nil
}

// inverse takes a transfer and returns a copy of that transfer with the
// additive inverse of its Value property (i.e. flip the sign).
func (t *transfer) inverse() transfer {
	return transfer{t.dbTransfer, -t.value}
}

func (ce *chainEntry) isBigTransfer(value uint64) bool {
	return value >= ce.bigTransactionSize && ce.checkForBigTransactions
}

type ChainGovernor struct {
	db                  guardianDB.GovernorDB // protected by `mutex`
	logger              *zap.Logger
	mutex               sync.Mutex
	tokens              map[tokenKey]*tokenEntry    // protected by `mutex`
	tokensByCoinGeckoId map[string][]*tokenEntry    // protected by `mutex`
	chains              map[vaa.ChainID]*chainEntry // protected by `mutex`
	// We maintain a sorted slice of governed chainIds so we can iterate over maps in a deterministic way
	// This slice should be sorted in ascending order by (Wormhole) Chain ID.
	chainIds              []vaa.ChainID
	msgsSeen              map[string]bool              // protected by `mutex` // Key is hash, payload is consts transferComplete and transferEnqueued.
	msgsToPublish         []*common.MessagePublication // protected by `mutex`
	dayLengthInMinutes    int
	coinGeckoQueries      []string
	env                   common.Environment
	nextStatusPublishTime time.Time
	nextConfigPublishTime time.Time
	statusPublishCounter  int64
	configPublishCounter  int64
	flowCancelEnabled     bool
	coinGeckoApiKey       string
	// Pairs of chains for which flow canceling is enabled. Note that an asset may be flow canceling even if
	// it was minted on a chain that is not configured to be an 'end' of any of the corridors.
	flowCancelCorridors []corridor
}

func NewChainGovernor(
	logger *zap.Logger,
	db guardianDB.GovernorDB,
	env common.Environment,
	flowCancelEnabled bool,
	coinGeckoApiKey string,
) *ChainGovernor {
	return &ChainGovernor{
		db:                  db,
		logger:              logger.With(zap.String("component", "cgov")),
		tokens:              make(map[tokenKey]*tokenEntry),
		tokensByCoinGeckoId: make(map[string][]*tokenEntry),
		chains:              make(map[vaa.ChainID]*chainEntry),
		msgsSeen:            make(map[string]bool),
		env:                 env,
		flowCancelEnabled:   flowCancelEnabled,
		coinGeckoApiKey:     coinGeckoApiKey,
	}
}

func (gov *ChainGovernor) Run(ctx context.Context) error {
	gov.logger.Info("starting chain governor")

	if err := gov.initConfig(); err != nil {
		return err
	}

	if gov.env != common.GoTest {
		if err := gov.loadFromDB(); err != nil {
			return err
		}

		if err := gov.initCoinGecko(ctx, true); err != nil {
			return err
		}
	}

	return nil
}

func (gov *ChainGovernor) IsFlowCancelEnabled() bool {
	return gov.flowCancelEnabled
}

func (gov *ChainGovernor) initConfig() error {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	gov.dayLengthInMinutes = 24 * 60
	configChains := ChainList()
	configTokens := TokenList()
	flowCancelTokens := []TokenConfigEntry{}
	flowCancelCorridors := []corridor{}

	if gov.env == common.UnsafeDevNet {
		configTokens, flowCancelTokens, configChains, flowCancelCorridors = gov.initDevnetConfig()
	} else if gov.env == common.TestNet {
		configTokens, flowCancelTokens, configChains, flowCancelCorridors = gov.initTestnetConfig()
	} else {
		// mainnet, unit tests, or accountant-mock
		if gov.flowCancelEnabled {
			flowCancelTokens = FlowCancelTokenList()
			flowCancelCorridors = FlowCancelCorridors()
		}
	}

	// We're done with this value for the rest of this function, so write it to the governor struct now
	if gov.flowCancelEnabled {
		gov.flowCancelCorridors = flowCancelCorridors
	}

	for _, ct := range configTokens {
		addr, err := vaa.StringToAddress(ct.Addr)
		if err != nil {
			return fmt.Errorf("invalid address: %s", ct.Addr)
		}

		cfgPrice := big.NewFloat(ct.Price)
		initialPrice := new(big.Float)
		initialPrice.Set(cfgPrice)

		// Transfers have a maximum of eight decimal places.
		dec := ct.Decimals
		if dec > 8 {
			dec = 8
		}

		decimalsFloat := big.NewFloat(math.Pow(10.0, float64(dec)))
		decimals, _ := decimalsFloat.Int(nil)

		// Some Solana tokens don't have the symbol set. In that case, use the chain and token address as the symbol.
		symbol := ct.Symbol
		if symbol == "" {
			symbol = fmt.Sprintf("%d:%s", ct.Chain, ct.Addr)
		}

		key := tokenKey{chain: vaa.ChainID(ct.Chain), addr: addr}
		te := &tokenEntry{
			cfgPrice:    cfgPrice,
			price:       initialPrice,
			decimals:    decimals,
			symbol:      symbol,
			coinGeckoId: ct.CoinGeckoId,
			token:       key,
		}
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

		if gov.env != common.GoTest {
			gov.logger.Info("will monitor token:", zap.Stringer("chain", key.chain),
				zap.Stringer("addr", key.addr),
				zap.String("symbol", te.symbol),
				zap.String("coinGeckoId", te.coinGeckoId),
				zap.String("price", te.price.String()),
				zap.Int64("decimals", dec),
				zap.Int64("origDecimals", ct.Decimals),
			)
		}
	}

	// If flow cancelling is enabled, enable the `flowCancels` field for the Governed assets that
	// correspond to the entries in the Flow Cancel Tokens List
	if gov.flowCancelEnabled {
		for _, flowCancelConfigEntry := range flowCancelTokens {
			addr, err := vaa.StringToAddress(flowCancelConfigEntry.Addr)
			if err != nil {
				return err
			}
			key := tokenKey{chain: vaa.ChainID(flowCancelConfigEntry.Chain), addr: addr}

			// Only add flow cancelling for tokens that are already configured for rate-limiting.
			if _, ok := gov.tokens[key]; ok {
				gov.tokens[key].flowCancels = true
			} else {
				gov.logger.Debug("token present in flow cancel list but absent from main token list:",
					zap.Stringer("chain", key.chain),
					zap.Stringer("addr", key.addr),
					zap.String("symbol", flowCancelConfigEntry.Symbol),
					zap.String("coinGeckoId", flowCancelConfigEntry.CoinGeckoId),
				)
			}
		}
	}

	if len(gov.tokens) == 0 {
		return fmt.Errorf("no tokens are configured")
	}

	emitterMap := &sdk.KnownTokenbridgeEmitters
	if gov.env == common.TestNet {
		emitterMap = &sdk.KnownTestnetTokenbridgeEmitters
	} else if gov.env == common.UnsafeDevNet {
		emitterMap = &sdk.KnownDevnetTokenbridgeEmitters
	}

	for _, cc := range configChains {
		var emitterAddr vaa.Address
		var err error

		emitterAddrBytes, exists := (*emitterMap)[cc.EmitterChainID]
		if !exists {
			return fmt.Errorf("failed to look up token bridge emitter address for chain: %v", cc.EmitterChainID)
		}

		emitterAddr, err = vaa.BytesToAddress(emitterAddrBytes)
		if err != nil {
			return fmt.Errorf("failed to convert emitter address for chain: %v", cc.EmitterChainID)
		}

		ce := &chainEntry{
			emitterChainId:          cc.EmitterChainID,
			emitterAddr:             emitterAddr,
			dailyLimit:              cc.DailyLimit,
			bigTransactionSize:      cc.BigTransactionSize,
			checkForBigTransactions: cc.BigTransactionSize != 0,
		}

		if gov.env != common.GoTest {
			gov.logger.Info("will monitor chain:", zap.Stringer("emitterChainId", cc.EmitterChainID),
				zap.Stringer("emitterAddr", ce.emitterAddr),
				zap.String("dailyLimit", fmt.Sprint(ce.dailyLimit)),
				zap.Uint64("bigTransactionSize", ce.bigTransactionSize),
				zap.Bool("checkForBigTransactions", ce.checkForBigTransactions),
			)
		}

		gov.chains[cc.EmitterChainID] = ce
	}

	if len(gov.chains) == 0 {
		return fmt.Errorf("no chains are configured")
	}

	// Populate a sorted list of chain IDs so that we can iterate over maps in a determinstic way.
	// https://go.dev/blog/maps, "Iteration order" section
	governedChainIds := make([]vaa.ChainID, len(gov.chains))
	i := 0
	for id := range gov.chains {
		// updating the slice in place here to satisfy prealloc lint. In theory this should be more performant
		governedChainIds[i] = id
		i++
	}
	// Custom sorting for the vaa.ChainID type
	sort.Slice(governedChainIds, func(i, j int) bool {
		return governedChainIds[i] < governedChainIds[j]
	})

	gov.chainIds = governedChainIds

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

// ProcessMsgForTime handles an incoming message (transfer) and registers it in the chain entries for the Governor.
// Returns true if:
// - the message is not governed
// - the transfer is complete and has already been observed
// - the transfer does not trigger any error conditions (happy path)
// Validation:
// - ensure MessagePublication is not nil
// - check that the MessagePublication is governed
// - check that the message is not a duplicate of one we've seen before.
func (gov *ChainGovernor) ProcessMsgForTime(msg *common.MessagePublication, now time.Time) (bool, error) {
	if msg == nil {
		return false, fmt.Errorf("msg is nil")
	}

	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	msgIsGoverned, emitterChainEntry, token, payload, err := gov.parseMsgAlreadyLocked(msg)

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
				zap.String("txID", msg.TxIDString()),
			)
			return false, nil
		}

		gov.logger.Info("allowing duplicate vaa to be published again, but not adding it to the notional value",
			zap.String("msgID", msg.MessageIDString()),
			zap.String("hash", hash),
			zap.String("txID", msg.TxIDString()),
		)
		return true, nil
	}

	// Get all outgoing transfers for `emitterChainEntry` that happened within the last 24 hours
	startTime := now.Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))
	prevTotalValue, err := gov.TrimAndSumValueForChain(emitterChainEntry, startTime)
	if err != nil {
		gov.logger.Error("Error when attempting to trim and sum transfers",
			zap.String("msgID", msg.MessageIDString()),
			zap.String("hash", hash),
			zap.String("txID", msg.TxIDString()),
			zap.Error(err),
		)
		return false, err
	}

	// Compute the notional USD value of the transfers
	value, err := computeValue(payload.Amount, token)
	if err != nil {
		gov.logger.Error("failed to compute value of transfer",
			zap.String("msgID", msg.MessageIDString()),
			zap.String("hash", hash),
			zap.String("txID", msg.TxIDString()),
			zap.Error(err),
		)
		return false, err
	}

	newTotalValue := prevTotalValue + value
	if newTotalValue < prevTotalValue {
		gov.logger.Error("total value has overflowed",
			zap.String("msgID", msg.MessageIDString()),
			zap.String("hash", hash),
			zap.String("txID", msg.TxIDString()),
			zap.Uint64("prevTotalValue", prevTotalValue),
			zap.Uint64("newTotalValue", newTotalValue),
		)
		return false, fmt.Errorf("total value has overflowed")
	}

	enqueueIt := false
	var releaseTime time.Time
	if emitterChainEntry.isBigTransfer(value) {
		enqueueIt = true
		releaseTime = now.Add(maxEnqueuedTime)
		gov.logger.Error("enqueuing vaa because it is a big transaction",
			zap.Uint64("value", value),
			zap.Uint64("prevTotalValue", prevTotalValue),
			zap.Uint64("newTotalValue", newTotalValue),
			zap.String("msgID", msg.MessageIDString()),
			zap.Stringer("releaseTime", releaseTime),
			zap.Uint64("bigTransactionSize", emitterChainEntry.bigTransactionSize),
			zap.String("hash", hash),
			zap.String("txID", msg.TxIDString()),
		)
	} else if newTotalValue > emitterChainEntry.dailyLimit {
		enqueueIt = true
		releaseTime = now.Add(maxEnqueuedTime)
		gov.logger.Error("enqueuing vaa because it would exceed the daily limit",
			zap.Uint64("value", value),
			zap.Uint64("prevTotalValue", prevTotalValue),
			zap.Uint64("newTotalValue", newTotalValue),
			zap.Stringer("releaseTime", releaseTime),
			zap.String("msgID", msg.MessageIDString()),
			zap.String("hash", hash),
			zap.String("txID", msg.TxIDString()),
		)
	}

	if enqueueIt {
		dbData := guardianDB.PendingTransfer{ReleaseTime: releaseTime, Msg: *msg}
		err = gov.db.StorePendingMsg(&dbData)
		if err != nil {
			gov.logger.Error("failed to store pending vaa",
				zap.String("msgID", msg.MessageIDString()),
				zap.String("hash", hash),
				zap.String("txID", msg.TxIDString()),
				zap.Error(err),
			)
			return false, err
		}

		emitterChainEntry.pending = append(
			emitterChainEntry.pending,
			&pendingEntry{token: token, amount: payload.Amount, hash: hash, dbData: dbData},
		)
		gov.msgsSeen[hash] = transferEnqueued
		return false, nil
	}

	gov.logger.Info("posting vaa",
		zap.Uint64("value", value),
		zap.Uint64("prevTotalValue", prevTotalValue),
		zap.Uint64("newTotalValue", newTotalValue),
		zap.String("msgID", msg.MessageIDString()),
		zap.String("hash", hash),
		zap.String("txID", msg.TxIDString()),
	)

	dbTransfer := guardianDB.Transfer{
		Timestamp:      now,
		Value:          value,
		OriginChain:    token.token.chain,
		OriginAddress:  token.token.addr,
		EmitterChain:   msg.EmitterChain,
		EmitterAddress: msg.EmitterAddress,
		TargetChain:    payload.TargetChain,
		TargetAddress:  payload.TargetAddress,
		MsgID:          msg.MessageIDString(),
		Hash:           hash,
	}

	// Check that the transfer is valid before writing to the database.
	transfer, err := newTransferFromDbTransfer(&dbTransfer)
	if err != nil {
		return false, err
	}

	err = gov.db.StoreTransfer(&dbTransfer)
	if err != nil {
		gov.logger.Error("failed to store transfer",
			zap.String("msgID", msg.MessageIDString()),
			zap.String("hash", hash), zap.Error(err),
			zap.String("txID", msg.TxIDString()),
		)
		return false, err
	}

	// Update the chainEntries. For the emitter chain, add the transfer so that it can be factored into calculating
	// the usage of this chain the next time that the Governor processes a transfer.
	// For the destination chain entry, add the inverse of this transfer.
	// e.g. A transfer of USDC originally minted on Solana is sent from Ethereum to Sui.
	// - This increases the Governor usage of Ethereum by the `transfer.Value` amount.
	// - If the USDC version of Solana is flow cancelled, we also want to decrease the Governor usage for Sui.
	// - We do this by adding an 'inverse' transfer to Sui's chainEntry that contains a negative `transfer.Value`.
	// - This will cause the summed value of Sui to decrease.
	emitterChainEntry.transfers = append(emitterChainEntry.transfers, transfer)

	if gov.flowCancelEnabled {
		_, err = gov.tryAddFlowCancelTransfer(&transfer)
		if err != nil {
			// Don't interrupt the control flow when a flow cancel fails. Instead, fail open and allow
			// the transfers to be processed normally. The only consequence is that the outbound limit
			// for the emitter won't be reduced.
			gov.logger.Warn("Error when attempting to add a flow cancel transfer",
				zap.Error(err),
			)
		}
	}

	gov.msgsSeen[hash] = transferComplete
	return true, nil
}

// corridorCanFlowCancel checks whether the corridor passed as an argument is present in the list of flow-cancel enabled
// corridors. This method returns false for all values if flow canceling is disabled.
func (gov *ChainGovernor) corridorCanFlowCancel(corridor *corridor) bool {
	if !gov.flowCancelEnabled {
		return false
	}
	for _, configuredcorridor := range gov.flowCancelCorridors {
		// Note that equals() also checks that both corridors are valid.
		if corridor.equals(&configuredcorridor) {
			return true
		}
	}
	return false
}

// IsGovernedMsg determines if the message applies to the governor. It grabs the lock.
func (gov *ChainGovernor) IsGovernedMsg(msg *common.MessagePublication) (msgIsGoverned bool, err error) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()
	msgIsGoverned, _, _, _, err = gov.parseMsgAlreadyLocked(msg)
	return
}

// parseMsgAlreadyLocked determines if the message applies to the governor and also returns data useful to the governor. It assumes the caller holds the lock.
func (gov *ChainGovernor) parseMsgAlreadyLocked(
	msg *common.MessagePublication,
) (bool, *chainEntry, *tokenEntry, *vaa.TransferPayloadHdr, error) {
	// If we don't care about this chain, the VAA can be published.
	ce, exists := gov.chains[msg.EmitterChain]
	if !exists {
		if msg.EmitterChain != vaa.ChainIDPythNet {
			gov.logger.Info(
				"ignoring vaa because the emitter chain is not configured",
				zap.String("msgID", msg.MessageIDString()),
			)
		}
		return false, nil, nil, nil, nil
	}

	// If we don't care about this emitter, the VAA can be published.
	if msg.EmitterAddress != ce.emitterAddr {
		gov.logger.Info(
			"ignoring vaa because the emitter address is not configured",
			zap.String("msgID", msg.MessageIDString()),
		)
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

// CheckPending is a wrapper method for CheckPendingForTime. It is called by the processor with the purpose of releasing
// queued transfers.
func (gov *ChainGovernor) CheckPending() ([]*common.MessagePublication, error) {
	return gov.CheckPendingForTime(time.Now())
}

// CheckPendingForTime checks whether a pending message is ready to be released, and if so, modifies the chain entry's `pending` and `transfers` slices by
// moving a `dbTransfer` element from `pending` to `transfers`. Returns a slice of Messages that will be published.
// A transfer is ready to be released when one of the following conditions holds:
//   - The 'release time' duration has passed since `now` (i.e. the transfer has been queued for 24 hours, regardless of
//     the Governor's current capacity)
//   - Within the release time duration, other transfers have been processed and have freed up outbound Governor capacity.
//     This happens either because other transfers get released after 24 hours or because incoming transfers of
//     flow-cancelling assets have freed up outbound capacity.
//
// WARNING: When this function returns an error, it propagates to the `processor` which in turn interprets this as a
// signal to RESTART THE PROCESSOR. Therefore, errors returned by this function effectively act as panics.
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

	// Iterate deterministically by accessing keys from this slice instead of the chainEntry map directly
	for _, chainId := range gov.chainIds {
		ce, ok := gov.chains[chainId]
		if !ok {
			gov.logger.Error("chainId not found in gov.chains", zap.Stringer("chainId", chainId))
		}
		// Keep going as long as we find something that will fit.
		for {
			foundOne := false
			prevTotalValue, err := gov.TrimAndSumValueForChain(ce, startTime)
			if err != nil {
				gov.logger.Error("error when attempting to trim and sum transfers", zap.Error(err))
				gov.logger.Error("refusing to release transfers for this chain until the sum can be correctly calculated",
					zap.Stringer("chainId", chainId),
					zap.Uint64("prevTotalValue", prevTotalValue),
					zap.Error(err))
				gov.msgsToPublish = msgsToPublish
				// Skip further processing for this chain entry
				break
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
						zap.String("msgID", pe.dbData.Msg.MessageIDString()),
						zap.String("flowCancels", strconv.FormatBool(pe.token.flowCancels)))
				}

				payload, err := vaa.DecodeTransferPayloadHdr(pe.dbData.Msg.Payload)
				if err != nil {
					gov.logger.Error("failed to decode payload for pending VAA, dropping it",
						zap.String("msgID", pe.dbData.Msg.MessageIDString()),
						zap.String("hash", pe.hash),
						zap.Error(err),
					)
					delete(gov.msgsSeen, pe.hash) // Rest of the clean up happens below.
				} else {
					// If we get here, publish it and move it from the pending list to the
					// transfers list. Also add a flow-cancel transfer to the destination chain
					// if the transfer is sending a flow-canceling asset.
					msgsToPublish = append(msgsToPublish, &pe.dbData.Msg)

					if countsTowardsTransfers {
						dbTransfer := guardianDB.Transfer{
							Timestamp:      now,
							Value:          value,
							OriginChain:    pe.token.token.chain,
							OriginAddress:  pe.token.token.addr,
							EmitterChain:   pe.dbData.Msg.EmitterChain,
							EmitterAddress: pe.dbData.Msg.EmitterAddress,
							TargetChain:    payload.TargetChain,
							TargetAddress:  payload.TargetAddress,
							MsgID:          pe.dbData.Msg.MessageIDString(),
							Hash:           pe.hash,
						}

						transfer, err := newTransferFromDbTransfer(&dbTransfer)
						if err != nil {
							// Should never occur unless dbTransfer.Value overflows MaxInt64
							gov.logger.Error("could not convert dbTransfer to transfer",
								zap.String("msgID", dbTransfer.MsgID),
								zap.String("hash", pe.hash),
								zap.Error(err),
							)
							// This causes the processor to die. We don't want to process transfers that
							// have USD value in excess of MaxInt64 under any circumstances.
							// This check should occur before the call to the database so
							// that we don't store a problematic transfer.
							return nil, err
						}

						if err := gov.db.StoreTransfer(&dbTransfer); err != nil {
							// This causes the processor to die. We can't tolerate DB connection
							// errors.
							return nil, err
						}

						ce.transfers = append(ce.transfers, transfer)

						gov.msgsSeen[pe.hash] = transferComplete
						if gov.flowCancelEnabled {
							// The point here is to add a flow-cancelling transfer to the governor's pending queue if applicable.
							// If the function returns (false, nil), just ignore it.
							// Note that the inverse, flow-cancelling transfers are not stored in the database; they only
							// exist in memory. When the Guardian is restarted, the flow cancelling transfers
							// will be reconstructed manually.
							_, err := gov.tryAddFlowCancelTransfer(&transfer)
							if err != nil {
								gov.logger.Error("Error when attempting to add a flow cancel transfer",
									zap.Error(err),
								)
							}
						}

					} else {
						delete(gov.msgsSeen, pe.hash)
					}
				}

				// Cleanup step: remove the pending message from the database now that it has been processed.
				// This must happen for all non-error (panic) scenarios.
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

// tryAddFlowCancelTransfer adds an inverse transfer to the destination chain entry. This occurs only if the transfer's asset
// is allow-listed, and if the source and destination chain are connected via an enabled "corridor" (allow-list of chain ID pairs).
// A return value of false means that the transfer will not do flow cancelling, i.e. it will not reduce the destination chain's
// Governor usage.
func (gov *ChainGovernor) tryAddFlowCancelTransfer(transfer *transfer) (bool, error) {
	originChain := transfer.dbTransfer.OriginChain
	originAddr := transfer.dbTransfer.OriginAddress
	hash := transfer.dbTransfer.Hash
	target := transfer.dbTransfer.TargetChain
	emitter := transfer.dbTransfer.EmitterChain

	corridor := &corridor{emitter, target}

	// Check whether the source and destination chain are in the allow-list for flow cancelling.
	if !gov.corridorCanFlowCancel(corridor) {
		return false, nil
	}

	key := tokenKey{originChain, originAddr}
	tokenEntry := gov.tokens[key]
	if tokenEntry == nil {
		// Weird but not critical.
		gov.logger.Warn("not adding flow cancel transfer because token is not governed",
			zap.String("OriginChain", originChain.String()),
			zap.String("tokenEntry", originAddr.String()),
			zap.String("msgID", transfer.dbTransfer.MsgID),
			zap.String("hash", hash),
		)
		return false, nil
	}
	if !tokenEntry.flowCancels {
		// Nothing to do in this case
		return false, nil
	}

	// Ensure destination exists
	destinationChainEntry, ok := gov.chains[transfer.dbTransfer.TargetChain]
	if !ok {
		// Weird that TargetChain does not exist but not relevant for the flow cancel feature. We just
		// fail closed here: do not add the flow cancelling transfer.
		gov.logger.Warn("tried to cancel flow but there is no chain entry for the target chain",
			zap.String("msgID", transfer.dbTransfer.MsgID),
			zap.String("hash", hash),
			zap.Stringer("target chain", transfer.dbTransfer.TargetChain),
		)
		return false, nil
	}

	// Add the 'inverse' transfer to the destination chain entry.
	if err := destinationChainEntry.addFlowCancelTransfer(transfer.inverse()); err != nil {
		gov.logger.Warn("could not add flow canceling transfer to destination chain",
			zap.String("msgID", transfer.dbTransfer.MsgID),
			zap.String("hash", transfer.dbTransfer.Hash),
			zap.Error(err),
		)
		return false, err
	}

	return true, nil
}

// TrimAndSumValueForChain calculates the `sum` of `Transfer`s for a given chain `chainEntry`. In effect, it represents a
// chain's "Governor Usage" for a given 24 hour period.
// This sum may be reduced by the sum of 'flow cancelling' transfers: that is, transfers of an allow-listed token
// that have the `emitter` as their destination chain.
// The resulting `sum` return value therefore represents the net flow across a chain when taking flow-cancelling transfers
// into account. Therefore, this value should never be less than 0 and should never exceed the "Governor limit" for the chain.
// As a side-effect, this function modifies the parameter `chainEntry`, updating its `transfers` field so that it only includes
// filtered `Transfer`s (i.e. outgoing `Transfer`s newer than `startTime`).
// Returns an error if the sum cannot be calculated. The transfers field will still be updated in this case. When
// an error condition occurs, this function returns the chain's `dailyLimit` as the sum. This should result in the
// chain appearing at maximum capacity from the perspective of the Governor, and therefore cause new transfers to be
// queued until space opens up.
// SECURITY Invariant: The `sum` return value should never be less than 0
func (gov *ChainGovernor) TrimAndSumValueForChain(chainEntry *chainEntry, startTime time.Time) (sum uint64, err error) {
	if chainEntry == nil {
		// We don't expect this to happen but this prevents a nil pointer deference
		return 0, errors.New("TrimAndSumValeForChain parameter chainEntry must not be nil")
	}
	// Sum the value of all transfers for this chain. This sum can be negative if flow-cancelling is enabled
	// and the incoming value of flow-cancelling assets exceeds the summed value of all outgoing assets.
	var sumValue int64
	sumValue, chainEntry.transfers, err = gov.TrimAndSumValue(chainEntry.transfers, startTime)
	if err != nil {
		// Return the daily limit as the sum so that any further transfers will be queued.
		return chainEntry.dailyLimit, err
	}

	// Return 0 even if the sum is negative.
	if sumValue <= 0 {
		return 0, nil
	}

	return uint64(sumValue), nil
}

// TrimAndSumValue iterates over a slice of transfer structs. It filters out transfers that have Timestamp values that
// are earlier than the parameter `startTime`. The function then iterates over the remaining transfers, sums their Value,
// and returns the sum and the filtered transfers.
// As a side-effect, this function deletes transfers from the database if their Timestamp is before `startTime`.
// The `transfers` slice must be sorted by Timestamp. We expect this to be the case as transfers are added to the
// Governor in chronological order as they arrive. Note that `Timestamp` is created by the Governor; it is not read
// from the actual on-chain transaction.
func (gov *ChainGovernor) TrimAndSumValue(transfers []transfer, startTime time.Time) (int64, []transfer, error) {
	if len(transfers) == 0 {
		return 0, transfers, nil
	}

	var trimIdx int = -1
	var sum int64

	for idx, t := range transfers {
		if t.dbTransfer.Timestamp.Before(startTime) {
			trimIdx = idx
		} else {
			checkedSum, err := CheckedAddInt64(sum, t.value)
			if err != nil {
				// We have to stop and return an error here (rather than saturate, for example). The
				// transfers are not sorted by value so we can't make any guarantee on the final value
				// if we hit the upper or lower bound. We don't expect this to happen in any case
				// because we don't expect this number to ever overflow, as it would represent
				// $184467440737095516.15 USD moving between two chains in a 24h period.
				return 0, transfers, err
			}
			sum = checkedSum
		}
	}

	if trimIdx >= 0 {
		for idx := 0; idx <= trimIdx; idx++ {
			dbTransfer := transfers[idx].dbTransfer
			if err := gov.db.DeleteTransfer(dbTransfer); err != nil {
				return 0, transfers, err
			}

			delete(gov.msgsSeen, dbTransfer.Hash)
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

// CheckedAddUint64 adds two uint64 values with overflow checks
func CheckedAddUint64(x uint64, y uint64) (uint64, error) {
	if x == 0 {
		return y, nil
	}
	if y == 0 {
		return x, nil
	}

	sum := x + y

	if sum < x || sum < y {
		return 0, fmt.Errorf("integer overflow when adding %d and %d", x, y)
	}

	return sum, nil
}

// CheckedAddInt64 adds two uint64 values with overflow checks. Returns an error if the calculation would
// overflow or underflow. In this case, the returned value is 0.
func CheckedAddInt64(x int64, y int64) (int64, error) {
	if x == 0 {
		return y, nil
	}
	if y == 0 {
		return x, nil
	}

	sum := x + y

	// Both terms positive - overflow check
	if x > 0 && y > 0 {
		if sum < x || sum < y {
			return 0, fmt.Errorf("integer overflow when adding %d and %d", x, y)
		}
	}

	// Both terms negative - underflow check
	if x < 0 && y < 0 {
		if sum > x || sum > y {
			return 0, fmt.Errorf("integer underflow when adding %d and %d", x, y)
		}
	}
	return x + y, nil
}
