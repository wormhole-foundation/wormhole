package accounting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCommon "github.com/ethereum/go-ethereum/common"

	"go.uber.org/zap"
)

const (
	MainNetMode = 1
	TestNetMode = 2
	DevNetMode  = 3
	GoTestMode  = 4

	// We will retry requests once per minute for up to an hour.
	auditInterval = time.Duration(time.Minute)
	maxRetries    = 60
)

type (
	// tokenBridgeKey is the key to the map of token bridges being monitored
	tokenBridgeKey struct {
		emitterChainId vaa.ChainID
		emitterAddr    vaa.Address
	}

	// tokenBridgeEntry is the payload of the map of the token bridges being monitored
	tokenBridgeEntry struct {
	}

	// pendingKey is the key to the map of pending transfers
	pendingKey struct {
		emitterChainId vaa.ChainID
		txHash         ethCommon.Hash
	}

	// pendingEntry is the payload for each pending transfer
	pendingEntry struct {
		msg        *common.MessagePublication
		updTime    time.Time
		retryCount int
	}
)

// Accounting is the object that manages the interface to the wormchain accounting smart contract.
type Accounting struct {
	logger           *zap.Logger
	db               db.AccountingDB
	contract         string
	wsUrl            string
	lcdUrl           string
	enforceFlag      bool
	msgChan          chan<- *common.MessagePublication
	mutex            sync.Mutex
	tokenBridges     map[tokenBridgeKey]*tokenBridgeEntry
	pendingTransfers map[pendingKey]*pendingEntry
	env              int
}

// NewAccounting creates a new instance of the Accounting object.
func NewAccounting(
	logger *zap.Logger,
	db db.AccountingDB,
	contract string, // the address of the smart contract on wormchain
	wsUrl string, // the URL of the wormchain websocket interface
	lcdUrl string, // the URL of the wormchain LCD interface
	enforceFlag bool, // whether or not accounting should be enforced
	msgChan chan<- *common.MessagePublication, // the channel where transfers received by the accounting runnable should be published
	env int, // Controls the set of token bridges to be monitored
) *Accounting {
	return &Accounting{
		logger:           logger,
		db:               db,
		contract:         contract,
		wsUrl:            wsUrl,
		lcdUrl:           lcdUrl,
		enforceFlag:      enforceFlag,
		msgChan:          msgChan,
		tokenBridges:     make(map[tokenBridgeKey]*tokenBridgeEntry),
		pendingTransfers: make(map[pendingKey]*pendingEntry),
		env:              env,
	}
}

// Run initializes the accounting module and starts the watcher runnable.
func (acct *Accounting) Run(ctx context.Context) error {
	acct.mutex.Lock()
	defer acct.mutex.Unlock()

	emitterMap := &sdk.KnownTokenbridgeEmitters
	if acct.env == TestNetMode {
		emitterMap = &sdk.KnownTestnetTokenbridgeEmitters
	} else if acct.env == DevNetMode || acct.env == GoTestMode {
		emitterMap = &sdk.KnownDevnetTokenbridgeEmitters
	}

	// Build the map of token bridges to be monitored.
	for chainId, emitterAddrBytes := range *emitterMap {
		emitterAddr, err := vaa.BytesToAddress(emitterAddrBytes)
		if err != nil {
			return fmt.Errorf("failed to convert emitter address for chain: %v", chainId)
		}

		tbk := tokenBridgeKey{emitterChainId: chainId, emitterAddr: emitterAddr}
		_, exists := acct.tokenBridges[tbk]
		if exists {
			return fmt.Errorf("detected duplicate token bridge for chain: %v", chainId)
		}

		tbe := &tokenBridgeEntry{}
		acct.tokenBridges[tbk] = tbe
		acct.logger.Info("acct: will monitor token bridge:", zap.Stringer("emitterChainId", tbk.emitterChainId), zap.Stringer("emitterAddr", tbk.emitterAddr))
	}

	// Load any existing pending transfers from the db.
	if err := acct.loadPendingTransfers(); err != nil {
		return fmt.Errorf("failed to load pending transfers from the db: %w", err)
	}

	// Start the watcher to listen to transfer events from the smart contract.
	if acct.env != GoTestMode {
		if err := supervisor.Run(ctx, "acctwatcher", acct.watcher); err != nil {
			return fmt.Errorf("failed to start watcher: %w", err)
		}
	}

	return nil
}

// SubmitObservation will submit token bridge transfers to the accounting smart contract. This is called from the processor
// loop when a local observation is received from a watcher. It returns true if the observation can be published immediately,
// false if not (because it has been submitted to accounting).
func (acct *Accounting) SubmitObservation(msg *common.MessagePublication) (bool, error) {
	// We only care about token bridges.
	tbk := tokenBridgeKey{emitterChainId: msg.EmitterChain, emitterAddr: msg.EmitterAddress}
	if _, exists := acct.tokenBridges[tbk]; !exists {
		if msg.EmitterChain != vaa.ChainIDPythNet {
			acct.logger.Info("acct: ignoring vaa because it is not a token bridge", zap.String("msgID", msg.MessageIDString()))
		}

		return true, nil
	}

	// We only care about transfers.
	if !vaa.IsTransfer(msg.Payload) {
		if msg.EmitterChain != vaa.ChainIDPythNet {
			acct.logger.Info("acct: ignoring vaa because it is not a transfer", zap.String("msgID", msg.MessageIDString()))
		}
		return true, nil
	}

	acct.mutex.Lock()
	defer acct.mutex.Unlock()

	// If this is already pending, don't send it again.
	pk := pendingKey{emitterChainId: msg.EmitterChain, txHash: msg.TxHash}
	if _, exists := acct.pendingTransfers[pk]; exists {
		acct.logger.Info("acct: blocking previously pending transfer", zap.String("msgID", msg.MessageIDString()))
		return false, nil
	}

	// Add it to the pending map and the database.
	if err := acct.addPendingTransfer(&pk, msg); err != nil {
		acct.logger.Error("acct: failed to persist pending transfer, blocking publishing", zap.String("msgID", msg.MessageIDString()), zap.Error(err))
		return false, err
	}

	acct.logger.Info("acct: submitting transfer to accounting for approval", zap.String("msgID", msg.MessageIDString()), zap.Bool("canPublish", !acct.enforceFlag))

	// This transaction may take a while. Run it as a go routine so we don't block the processor.
	if acct.env != GoTestMode {
		go acct.submitObservationToContract(msg)
		transfersSubmitted.Inc()
	}

	// If we are not enforcing accounting, the event can be published. Otherwise we have to wait to hear back from the contract.
	return !acct.enforceFlag, nil
}

// FinalizeObservation deletes a pending transfer received on the accounting channel. This is called from the processor loop
// when a message is received on the accounting channel. It returns true if the observation should be published, false if not.
func (acct *Accounting) FinalizeObservation(msg *common.MessagePublication) bool {
	acct.mutex.Lock()
	defer acct.mutex.Unlock()

	pk := pendingKey{emitterChainId: msg.EmitterChain, txHash: msg.TxHash}
	if _, exists := acct.pendingTransfers[pk]; !exists {
		acct.logger.Info("acct: dropping pending transfer because it is no longer in the map", zap.String("msgID", msg.MessageIDString()))
		return false
	}

	acct.logger.Info("acct: deleting pending transfer", zap.String("msgID", msg.MessageIDString()))
	acct.deletePendingTransfer(&pk, msg.MessageIDString())

	// If we are enforcing accounting, publish it now. If we are not enforcing accounting, it should already have been published.
	return acct.enforceFlag
}

// transferAlreadyApproved queries the contract to see if a transfer has previously been approved. It assumes the caller holds the lock.
func (acct *Accounting) transferAlreadyApproved(msg *common.MessagePublication) (bool, error) {
	// TODO: How do we this?
	// Do we use QueryMsg::Transfer?
	return false, nil
}

// submitObservationToContract makes a call to the smart contract to submit an observation request.
// It should be called from a go routine because it can block.
func (acct *Accounting) submitObservationToContract(msg *common.MessagePublication) {
	// TODO: How do we this?
	submitFailures.Inc()
}

// AuditPending audits the set of pending transfers for any that can be released, or ones that are stuck. This is called from the processor loop
// each timer interval. Any transfers that can be released will be forwarded to the accounting message channel.
func (acct *Accounting) AuditPendingTransfers() {
	acct.mutex.Lock()
	defer acct.mutex.Unlock()

	for pk, pe := range acct.pendingTransfers {
		if time.Since(pe.updTime) > auditInterval {
			alreadySeen, err := acct.transferAlreadyApproved(pe.msg)
			if err != nil {
				acct.logger.Error("failed to query status of pending transfer", zap.String("msgId", pe.msg.MessageIDString()), zap.Error(err))
				continue
			}

			if alreadySeen {
				acct.logger.Info("acct: pending transfer has previously been approved, dropping it", zap.String("msgId", pe.msg.MessageIDString()))
				acct.publishTransfer(pe)
			} else {
				pe.retryCount += 1
				if pe.retryCount > maxRetries {
					acct.logger.Error("acct: stuck pending transfer has reached the retry limit, dropping it", zap.String("msgId", pe.msg.MessageIDString()))
					acct.deletePendingTransfer(&pk, pe.msg.MessageIDString())
				}
			}
		}
	}
}

// publishTransfer publishes a pending transfer to the accounting channel and updates the timestamp. It assumes the caller holds the lock.
func (acct *Accounting) publishTransfer(pe *pendingEntry) {
	acct.msgChan <- pe.msg
	pe.updTime = time.Now()
}

// addPendingTransfer adds a pending transfer to both the map and the database. It assumes the caller holds the lock.
func (acct *Accounting) addPendingTransfer(pk *pendingKey, msg *common.MessagePublication) error {
	if err := acct.db.AcctStorePendingTransfer(msg); err != nil {
		return err
	}

	pe := &pendingEntry{msg: msg, updTime: time.Now()}
	acct.pendingTransfers[*pk] = pe
	transfersOutstanding.Inc()
	return nil
}

// deletePendingTransfer deletes the transfer from both the map and the database. It assumes the caller holds the lock.
func (acct *Accounting) deletePendingTransfer(pk *pendingKey, msgId string) {
	if _, exists := acct.pendingTransfers[*pk]; exists {
		transfersOutstanding.Dec()
		delete(acct.pendingTransfers, *pk)
	}
	if err := acct.db.AcctDeletePendingTransfer(msgId); err != nil {
		acct.logger.Error("acct: failed to delete pending transfer from the db", zap.String("msgId", msgId), zap.Error(err))
		// Ignore this error and keep going.
	}
}

// loadPendingTransfers loads any pending transfers that are present in the database. Before adding it to the map, it queries
// the smart contract to see if it has already been processed. If so, it drops it. This method assumes the caller holds the lock.
func (acct *Accounting) loadPendingTransfers() error {
	pendingTransfers, err := acct.db.AcctGetData(acct.logger)
	if err != nil {
		return err
	}

	for _, msg := range pendingTransfers {
		acct.logger.Info("acct: reloaded pending transfer", zap.String("msgID", msg.MessageIDString()))
		pk := pendingKey{emitterChainId: msg.EmitterChain, txHash: msg.TxHash}
		pe := &pendingEntry{msg: msg} // Leave the updTime unset so we will query this on the first audit interval.
		acct.pendingTransfers[pk] = pe
		transfersOutstanding.Inc()
	}

	if len(acct.pendingTransfers) != 0 {
		acct.logger.Info("acct: reloaded pending transfers", zap.Int("total", len(acct.pendingTransfers)))
	}

	return nil
}

// notifyOperator sends a notification to the on call / guardian operator in the case of a serious error.
func (acct *Accounting) notifyOperator(err error) {
	// TODO Do something more useful here.
	acct.logger.Error("acct: encountered an error", zap.Error(err))
}
