// The accountant package manages the interface to the accountant smart contract on wormchain. It is passed all VAAs before
// they are signed and published. It determines if the VAA is for a token bridge transfer, and if it is, it submits an observation
// request to the accountant contract. When that happens, the VAA is queued up until the accountant contract responds indicating
// that the VAA has been approved. If the VAA is approved, this module will forward the VAA back to the processor loop to be signed
// and published.

package accountant

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/wormconn"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"

	"go.uber.org/zap"
)

const (
	MainNetMode = 1
	TestNetMode = 2
	DevNetMode  = 3
	GoTestMode  = 4
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

	// pendingEntry is the payload for each pending transfer
	pendingEntry struct {
		msg    *common.MessagePublication
		msgId  string
		digest string

		// stateLock is used to protect the contents of the state struct.
		stateLock sync.Mutex

		// The state struct contains anything that can be modifed. It is protected by the state lock.
		state struct {
			// updTime is the time that the state struct was last updated.
			updTime time.Time

			// submitPending indicates if the observation is either in the channel waiting to be submitted or in an outstanding transaction.
			// The audit should not resubmit anything where submitPending is set to true.
			submitPending bool
		}
	}
)

// Accountant is the object that manages the interface to the wormchain accountant smart contract.
type Accountant struct {
	ctx                  context.Context
	logger               *zap.Logger
	db                   db.AccountantDB
	obsvReqWriteC        chan<- *gossipv1.ObservationRequest
	contract             string
	wsUrl                string
	wormchainConn        *wormconn.ClientConn
	enforceFlag          bool
	gk                   *ecdsa.PrivateKey
	gst                  *common.GuardianSetState
	guardianAddr         ethCommon.Address
	msgChan              chan<- *common.MessagePublication
	tokenBridges         map[tokenBridgeKey]*tokenBridgeEntry
	pendingTransfersLock sync.Mutex
	pendingTransfers     map[string]*pendingEntry // Key is the message ID (emitterChain/emitterAddr/seqNo)
	subChan              chan *common.MessagePublication
	env                  int
}

// On startup, there can be a large number of re-submission requests.
const subChanSize = 500

// NewAccountant creates a new instance of the Accountant object.
func NewAccountant(
	ctx context.Context,
	logger *zap.Logger,
	db db.AccountantDB,
	obsvReqWriteC chan<- *gossipv1.ObservationRequest,
	contract string, // the address of the smart contract on wormchain
	wsUrl string, // the URL of the wormchain websocket interface
	wormchainConn *wormconn.ClientConn, // used for communicating with the smart contract
	enforceFlag bool, // whether or not accountant should be enforced
	gk *ecdsa.PrivateKey, // the guardian key used for signing observation requests
	gst *common.GuardianSetState, // used to get the current guardian set index when sending observation requests
	msgChan chan<- *common.MessagePublication, // the channel where transfers received by the accountant runnable should be published
	env int, // Controls the set of token bridges to be monitored
) *Accountant {
	return &Accountant{
		ctx:              ctx,
		logger:           logger,
		db:               db,
		obsvReqWriteC:    obsvReqWriteC,
		contract:         contract,
		wsUrl:            wsUrl,
		wormchainConn:    wormchainConn,
		enforceFlag:      enforceFlag,
		gk:               gk,
		gst:              gst,
		guardianAddr:     ethCrypto.PubkeyToAddress(gk.PublicKey),
		msgChan:          msgChan,
		tokenBridges:     make(map[tokenBridgeKey]*tokenBridgeEntry),
		pendingTransfers: make(map[string]*pendingEntry),
		subChan:          make(chan *common.MessagePublication, subChanSize),
		env:              env,
	}
}

// Run initializes the accountant and starts the watcher runnable.
func (acct *Accountant) Start(ctx context.Context) error {
	acct.logger.Debug("acct: entering Start", zap.Bool("enforceFlag", acct.enforceFlag))
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()

	emitterMap := sdk.KnownTokenbridgeEmitters
	if acct.env == TestNetMode {
		emitterMap = sdk.KnownTestnetTokenbridgeEmitters
	} else if acct.env == DevNetMode || acct.env == GoTestMode {
		emitterMap = sdk.KnownDevnetTokenbridgeEmitters
	}

	// Build the map of token bridges to be monitored.
	for chainId, emitterAddrBytes := range emitterMap {
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
		if err := supervisor.Run(ctx, "acctworker", common.WrapWithScissors(acct.worker, "acctworker")); err != nil {
			return fmt.Errorf("failed to start submit observation worker: %w", err)
		}

		if err := supervisor.Run(ctx, "acctwatcher", common.WrapWithScissors(acct.watcher, "acctwatcher")); err != nil {
			return fmt.Errorf("failed to start watcher: %w", err)
		}

		if err := supervisor.Run(ctx, "acctaudit", common.WrapWithScissors(acct.audit, "acctaudit")); err != nil {
			return fmt.Errorf("failed to start audit worker: %w", err)
		}
	}

	return nil
}

func (acct *Accountant) Close() {
	if acct.wormchainConn != nil {
		acct.wormchainConn.Close()
		acct.wormchainConn = nil
	}
}

func (acct *Accountant) FeatureString() string {
	if !acct.enforceFlag {
		return "acct:logonly"
	}
	return "acct:enforced"
}

// IsMessageCoveredByAccountant returns `true` if a message should be processed by the Global Accountant, `false` if not.
func (acct *Accountant) IsMessageCoveredByAccountant(msg *common.MessagePublication) bool {
	msgId := msg.MessageIDString()

	// We only care about token bridges.
	tbk := tokenBridgeKey{emitterChainId: msg.EmitterChain, emitterAddr: msg.EmitterAddress}
	if _, exists := acct.tokenBridges[tbk]; !exists {
		if msg.EmitterChain != vaa.ChainIDPythNet {
			acct.logger.Debug("acct: ignoring vaa because it is not a token bridge", zap.String("msgID", msgId))
		}

		return false
	}

	// We only care about transfers.
	if !vaa.IsTransfer(msg.Payload) {
		acct.logger.Info("acct: ignoring vaa because it is not a transfer", zap.String("msgID", msgId))
		return false
	}

	return true
}

// SubmitObservation will submit token bridge transfers to the accountant smart contract. This is called from the processor
// loop when a local observation is received from a watcher. It returns true if the observation can be published immediately,
// false if not (because it has been submitted to accountant).
func (acct *Accountant) SubmitObservation(msg *common.MessagePublication) (bool, error) {
	msgId := msg.MessageIDString()
	acct.logger.Debug("acct: in SubmitObservation", zap.String("msgID", msgId))

	if !acct.IsMessageCoveredByAccountant(msg) {
		return true, nil
	}

	digest := msg.CreateDigest()

	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()

	// If this is already pending, don't send it again.
	if oldEntry, exists := acct.pendingTransfers[msgId]; exists {
		if oldEntry.digest != digest {
			digestMismatches.Inc()
			acct.logger.Error("acct: digest in pending transfer has changed, dropping it",
				zap.String("msgID", msgId),
				zap.String("oldDigest", oldEntry.digest),
				zap.String("newDigest", digest),
				zap.Bool("enforcing", acct.enforceFlag),
			)
		} else {
			acct.logger.Info("acct: blocking transfer because it is already outstanding", zap.String("msgID", msgId), zap.Bool("enforcing", acct.enforceFlag))
		}
		return !acct.enforceFlag, nil
	}

	// Add it to the pending map and the database.
	pe := &pendingEntry{msg: msg, msgId: msgId, digest: digest}
	if err := acct.addPendingTransferAlreadyLocked(pe); err != nil {
		acct.logger.Error("acct: failed to persist pending transfer, blocking publishing", zap.String("msgID", msgId), zap.Error(err))
		return false, err
	}

	// This transaction may take a while. Pass it off to the worker so we don't block the processor.
	if acct.env != GoTestMode {
		acct.logger.Info("acct: submitting transfer to accountant for approval", zap.String("msgID", msgId), zap.Bool("canPublish", !acct.enforceFlag))
		_ = acct.submitObservation(pe)
	}

	// If we are not enforcing accountant, the event can be published. Otherwise we have to wait to hear back from the contract.
	return !acct.enforceFlag, nil
}

// publishTransferAlreadyLocked publishes a pending transfer to the accountant channel and updates the timestamp. It assumes the caller holds the lock.
func (acct *Accountant) publishTransferAlreadyLocked(pe *pendingEntry) {
	if acct.enforceFlag {
		acct.logger.Debug("acct: publishTransferAlreadyLocked: notifying the processor", zap.String("msgId", pe.msgId))
		acct.msgChan <- pe.msg
	}

	acct.deletePendingTransferAlreadyLocked(pe.msgId)
}

// addPendingTransferAlreadyLocked adds a pending transfer to both the map and the database. It assumes the caller holds the lock.
func (acct *Accountant) addPendingTransferAlreadyLocked(pe *pendingEntry) error {
	acct.logger.Debug("acct: addPendingTransferAlreadyLocked", zap.String("msgId", pe.msgId))
	pe.setUpdTime()
	if err := acct.db.AcctStorePendingTransfer(pe.msg); err != nil {
		return err
	}

	acct.pendingTransfers[pe.msgId] = pe
	transfersOutstanding.Set(float64(len(acct.pendingTransfers)))
	return nil
}

// deletePendingTransfer deletes the transfer from both the map and the database. It accquires the lock.
func (acct *Accountant) deletePendingTransfer(msgId string) {
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()
	acct.deletePendingTransferAlreadyLocked(msgId)
}

// deletePendingTransferAlreadyLocked deletes the transfer from both the map and the database. It assumes the caller holds the lock.
func (acct *Accountant) deletePendingTransferAlreadyLocked(msgId string) {
	acct.logger.Debug("acct: deletePendingTransfer", zap.String("msgId", msgId))
	if _, exists := acct.pendingTransfers[msgId]; exists {
		delete(acct.pendingTransfers, msgId)
		transfersOutstanding.Set(float64(len(acct.pendingTransfers)))
	}
	if err := acct.db.AcctDeletePendingTransfer(msgId); err != nil {
		acct.logger.Error("acct: failed to delete pending transfer from the db", zap.String("msgId", msgId), zap.Error(err))
		// Ignore this error and keep going.
	}
}

// loadPendingTransfers loads any pending transfers that are present in the database. This method assumes the caller holds the lock.
func (acct *Accountant) loadPendingTransfers() error {
	pendingTransfers, err := acct.db.AcctGetData(acct.logger)
	if err != nil {
		return err
	}

	for _, msg := range pendingTransfers {
		msgId := msg.MessageIDString()
		acct.logger.Info("acct: reloaded pending transfer", zap.String("msgID", msgId))

		digest := msg.CreateDigest()
		pe := &pendingEntry{msg: msg, msgId: msgId, digest: digest}
		pe.setUpdTime()
		acct.pendingTransfers[msgId] = pe
	}

	transfersOutstanding.Set(float64(len(acct.pendingTransfers)))
	if len(acct.pendingTransfers) != 0 {
		acct.logger.Info("acct: reloaded pending transfers", zap.Int("total", len(acct.pendingTransfers)))
	} else {
		acct.logger.Info("acct: no pending transfers to be reloaded")
	}

	return nil
}

// submitObservation sends an observation request to the worker so it can be submited to the contract.  If the transfer is already
// marked as "submit pending", this function returns false without doing anything. Otherwise it returns true. The return value can
// be used to avoid unnecessary error logging. If writing to the channel would block, this function returns without doing anything,
// assuming the pending transfer will be handled on the next audit interval. This function grabs the state lock.
func (acct *Accountant) submitObservation(pe *pendingEntry) bool {
	pe.stateLock.Lock()
	defer pe.stateLock.Unlock()

	if pe.state.submitPending {
		return false
	}

	pe.state.submitPending = true
	pe.state.updTime = time.Now()

	select {
	case acct.subChan <- pe.msg:
		acct.logger.Debug("acct: submitted observation to channel", zap.String("msgId", pe.msgId))
	default:
		acct.logger.Error("acct: unable to submit observation because the channel is full, will try next interval", zap.String("msgId", pe.msgId))
		pe.state.submitPending = false
	}

	return true
}

// clearSubmitPendingFlags is called after a batch is finished being submitted (success or fail). It clears the submit pending flag for everything in the batch.
// It grabs the pending transfer and state locks.
func (acct *Accountant) clearSubmitPendingFlags(msgs []*common.MessagePublication) {
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()
	for _, msg := range msgs {
		if pe, exists := acct.pendingTransfers[msg.MessageIDString()]; exists {
			pe.setSubmitPending(false)
		}
	}
}

// setSubmitPending sets the submit pending flag on the pending transfer object to the specified value. It grabs the state lock.
func (pe *pendingEntry) setSubmitPending(val bool) {
	pe.stateLock.Lock()
	defer pe.stateLock.Unlock()
	pe.state.submitPending = val
	pe.state.updTime = time.Now()
}

// submitPending returns the "submit pending" flag from the pending transfer object. It grabs the state lock.
func (pe *pendingEntry) submitPending() bool {
	pe.stateLock.Lock()
	defer pe.stateLock.Unlock()
	return pe.state.submitPending
}

// setUpdTime sets the last update time on the pending transfer object to the current time. It grabs the state lock.
func (pe *pendingEntry) setUpdTime() {
	pe.stateLock.Lock()
	defer pe.stateLock.Unlock()
	pe.state.updTime = time.Now()
}

// updTime returns the last update time from the pending transfer object. It grabs the state lock.
func (pe *pendingEntry) updTime() time.Time {
	pe.stateLock.Lock()
	defer pe.stateLock.Unlock()
	return pe.state.updTime
}
