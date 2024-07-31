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
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"

	"go.uber.org/zap"
)

// MsgChannelCapacity specifies the capacity of the message channel used to publish messages released from the accountant.
// This channel should not back up, but if it does, the accountant will start dropping messages, which would require reobservations.
const MsgChannelCapacity = 5 * batchSize

type (
	AccountantWormchainConn interface {
		Close()
		SenderAddress() string
		SubmitQuery(ctx context.Context, contractAddress string, query []byte) ([]byte, error)
		SignAndBroadcastTx(ctx context.Context, msg sdktypes.Msg) (*sdktx.BroadcastTxResponse, error)
		BroadcastTxResponseToString(txResp *sdktx.BroadcastTxResponse) string
	}

	// emitterKey is the key to a map of emitters to be monitored
	emitterKey struct {
		emitterChainId vaa.ChainID
		emitterAddr    vaa.Address
	}

	// validEmitters is a set of supported emitter chain / address pairs. The payload is the enforcement flag.
	validEmitters map[emitterKey]bool

	// pendingEntry is the payload for each pending transfer
	pendingEntry struct {
		msg         *common.MessagePublication
		msgId       string
		digest      string
		isNTT       bool
		enforceFlag bool

		// stateLock is used to protect the contents of the state struct.
		stateLock sync.Mutex

		// The state struct contains anything that can be modified. It is protected by the state lock.
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
	wormchainConn        AccountantWormchainConn
	enforceFlag          bool
	gk                   *ecdsa.PrivateKey
	gst                  *common.GuardianSetState
	guardianAddr         ethCommon.Address
	msgChan              chan<- *common.MessagePublication
	tokenBridges         validEmitters
	pendingTransfersLock sync.Mutex
	pendingTransfers     map[string]*pendingEntry // Key is the message ID (emitterChain/emitterAddr/seqNo)
	subChan              chan *common.MessagePublication
	env                  common.Environment

	nttContract       string
	nttWormchainConn  AccountantWormchainConn
	nttDirectEmitters validEmitters
	nttArEmitters     validEmitters
	nttSubChan        chan *common.MessagePublication
}

// On startup, there can be a large number of re-submission requests.
const subChanSize = 500

// baseEnabled returns true if the base accountant is enabled, false if not.
func (acct *Accountant) baseEnabled() bool {
	return acct.contract != ""
}

// NewAccountant creates a new instance of the Accountant object.
func NewAccountant(
	ctx context.Context,
	logger *zap.Logger,
	db db.AccountantDB,
	obsvReqWriteC chan<- *gossipv1.ObservationRequest,
	contract string, // the address of the smart contract on wormchain
	wsUrl string, // the URL of the wormchain websocket interface
	wormchainConn AccountantWormchainConn, // used for communicating with the smart contract
	enforceFlag bool, // whether or not accountant should be enforced
	nttContract string, // the address of the NTT smart contract on wormchain
	nttWormchainConn AccountantWormchainConn, // used for communicating with the NTT smart contract
	gk *ecdsa.PrivateKey, // the guardian key used for signing observation requests
	gst *common.GuardianSetState, // used to get the current guardian set index when sending observation requests
	msgChan chan<- *common.MessagePublication, // the channel where transfers received by the accountant runnable should be published
	env common.Environment, // Controls the set of token bridges to be monitored
) *Accountant {
	return &Accountant{
		ctx:              ctx,
		logger:           logger.With(zap.String("component", "gacct")),
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
		tokenBridges:     make(validEmitters),
		pendingTransfers: make(map[string]*pendingEntry),
		subChan:          make(chan *common.MessagePublication, subChanSize),
		env:              env,

		nttContract:       nttContract,
		nttWormchainConn:  nttWormchainConn,
		nttDirectEmitters: make(validEmitters),
		nttArEmitters:     make(validEmitters),
		nttSubChan:        make(chan *common.MessagePublication, subChanSize),
	}
}

// Start initializes the accountant and starts the worker and watcher runnables.
func (acct *Accountant) Start(ctx context.Context) error {
	acct.logger.Debug("entering Start", zap.Bool("enforceFlag", acct.enforceFlag), zap.Bool("baseEnabled", acct.baseEnabled()), zap.Bool("nttEnabled", acct.nttEnabled()))
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()

	if !acct.baseEnabled() && !acct.nttEnabled() {
		return fmt.Errorf("start should not be called when neither base nor NTT accountant are enabled")
	}

	if acct.baseEnabled() {
		emitterMap := sdk.KnownTokenbridgeEmitters
		if acct.env == common.TestNet {
			emitterMap = sdk.KnownTestnetTokenbridgeEmitters
		} else if acct.env == common.UnsafeDevNet || acct.env == common.GoTest || acct.env == common.AccountantMock {
			emitterMap = sdk.KnownDevnetTokenbridgeEmitters
		}

		// Build the map of token bridges to be monitored.
		for chainId, emitterAddrBytes := range emitterMap {
			emitterAddr, err := vaa.BytesToAddress(emitterAddrBytes)
			if err != nil {
				return fmt.Errorf("failed to convert emitter address for chain: %v", chainId)
			}

			tbk := emitterKey{emitterChainId: chainId, emitterAddr: emitterAddr}
			_, exists := acct.tokenBridges[tbk]
			if exists {
				return fmt.Errorf("detected duplicate token bridge for chain: %v", chainId)
			}

			acct.tokenBridges[tbk] = acct.enforceFlag
			acct.logger.Info("will monitor token bridge:", zap.Stringer("emitterChainId", tbk.emitterChainId), zap.Stringer("emitterAddr", tbk.emitterAddr))
		}
	}

	// The NTT data structures should be set up before we reload from the db.
	if acct.nttEnabled() {
		if err := acct.nttStart(ctx); err != nil {
			return fmt.Errorf("failed to start ntt accountant: %w", err)
		}
	}

	// Load any existing pending transfers from the db.
	if err := acct.loadPendingTransfers(); err != nil {
		return fmt.Errorf("failed to load pending transfers from the db: %w", err)
	}

	// Start the watcher to listen to transfer events from the smart contract.
	if acct.baseEnabled() {
		if acct.env == common.AccountantMock {
			// We're not in a runnable context, so we can't use supervisor.
			go func() {
				_ = acct.baseWorker(ctx)
			}()
		} else if acct.env != common.GoTest {
			if err := supervisor.Run(ctx, "acctworker", common.WrapWithScissors(acct.baseWorker, "acctworker")); err != nil {
				return fmt.Errorf("failed to start submit observation worker: %w", err)
			}

			if err := supervisor.Run(ctx, "acctwatcher", common.WrapWithScissors(acct.baseWatcher, "acctwatcher")); err != nil {
				return fmt.Errorf("failed to start watcher: %w", err)
			}

			if err := supervisor.Run(ctx, "acctaudit", common.WrapWithScissors(acct.audit, "acctaudit")); err != nil {
				return fmt.Errorf("failed to start audit worker: %w", err)
			}
		}
	}

	return nil
}

func (acct *Accountant) Close() {
	if acct.wormchainConn != nil {
		acct.wormchainConn.Close()
		acct.wormchainConn = nil
	}
	if acct.nttWormchainConn != nil {
		acct.nttWormchainConn.Close()
		acct.nttWormchainConn = nil
	}
}

func (acct *Accountant) FeatureString() string {
	var ret string
	if !acct.enforceFlag {
		ret = "acct-logonly"
	} else {
		ret = "acct"
	}
	if acct.nttEnabled() {
		if ret != "" {
			ret += ":"
		}
		ret += "ntt-acct"
	}

	return ret
}

// IsMessageCoveredByAccountant returns `true` if a message should be processed by the Global Accountant, `false` if not.
func (acct *Accountant) IsMessageCoveredByAccountant(msg *common.MessagePublication) bool {
	ret, _, _ := acct.isMessageCoveredByAccountant(msg)
	return ret
}

// isMessageCoveredByAccountant returns true if a message should be processed by the Global Accountant, false if not.
// It also returns whether or not it is a Native Token Transfer and whether or not accounting is being enforced for this emitter.
func (acct *Accountant) isMessageCoveredByAccountant(msg *common.MessagePublication) (bool, bool, bool) {
	isTBT, enforceFlag := acct.isTokenBridgeTransfer(msg)
	if isTBT {
		return true, false, enforceFlag
	}

	isNTT, enforceFlag := nttIsMsgDirectNTT(msg, acct.nttDirectEmitters)
	if isNTT {
		return true, true, enforceFlag
	}

	isNTT, enforceFlag = nttIsMsgArNTT(msg, acct.nttArEmitters, acct.nttDirectEmitters)
	if isNTT {
		return true, true, enforceFlag
	}

	return false, false, false
}

// isTokenBridgeTransfer returns true if a message is a token bridge transfer and whether or not accounting is being enforced for this emitter.
func (acct *Accountant) isTokenBridgeTransfer(msg *common.MessagePublication) (bool, bool) {
	msgId := msg.MessageIDString()

	// We only care about token bridges.
	enforceFlag, exists := acct.tokenBridges[emitterKey{emitterChainId: msg.EmitterChain, emitterAddr: msg.EmitterAddress}]
	if !exists {
		return false, false
	}

	// We only care about transfers.
	if !vaa.IsTransfer(msg.Payload) {
		acct.logger.Info("ignoring vaa because it is not a transfer", zap.String("msgID", msgId))
		return false, false
	}

	return true, enforceFlag
}

// SubmitObservation will submit token bridge transfers to the accountant smart contract. This is called from the processor
// loop when a local observation is received from a watcher. It returns true if the observation can be published immediately,
// false if not (because it has been submitted to the accountant).
func (acct *Accountant) SubmitObservation(msg *common.MessagePublication) (bool, error) {
	msgId := msg.MessageIDString()
	acct.logger.Debug("in SubmitObservation", zap.String("msgID", msgId))

	coveredByAcct, isNTT, enforceFlag := acct.isMessageCoveredByAccountant(msg)
	if !coveredByAcct {
		return true, nil
	}

	digest := msg.CreateDigest()

	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()

	// If this is already pending, don't send it again.
	if oldEntry, exists := acct.pendingTransfers[msgId]; exists {
		if oldEntry.digest != digest {
			digestMismatches.Inc()
			acct.logger.Error("digest in pending transfer has changed, dropping it",
				zap.String("msgID", msgId),
				zap.String("oldDigest", oldEntry.digest),
				zap.String("newDigest", digest),
				zap.Bool("enforcing", enforceFlag),
			)
		} else {
			acct.logger.Info("blocking transfer because it is already outstanding", zap.String("msgID", msgId), zap.Bool("enforcing", enforceFlag))
		}
		return !enforceFlag, nil
	}

	// Add it to the pending map and the database.
	pe := &pendingEntry{msg: msg, msgId: msgId, digest: digest, isNTT: isNTT, enforceFlag: enforceFlag}
	if err := acct.addPendingTransferAlreadyLocked(pe); err != nil {
		acct.logger.Error("failed to persist pending transfer, blocking publishing", zap.String("msgID", msgId), zap.Error(err))
		return false, err
	}

	// This transaction may take a while. Pass it off to the worker so we don't block the processor.
	if acct.env != common.GoTest {
		tag := "accountant"
		if isNTT {
			tag = "ntt-accountant"
		}
		acct.logger.Info(fmt.Sprintf("submitting transfer to %s for approval", tag), zap.String("msgID", msgId), zap.Bool("canPublish", !enforceFlag))
		_ = acct.submitObservation(pe)
	}

	// If we are not enforcing accountant, the event can be published. Otherwise we have to wait to hear back from the contract.
	return !enforceFlag, nil
}

// publishTransferAlreadyLocked publishes a pending transfer to the accountant channel and deletes it from the pending map. It assumes the caller holds the lock.
func (acct *Accountant) publishTransferAlreadyLocked(pe *pendingEntry) {
	if pe.enforceFlag {
		select {
		case acct.msgChan <- pe.msg:
			acct.logger.Debug("published transfer to channel", zap.String("msgId", pe.msgId))
		default:
			acct.logger.Error("unable to publish transfer because the channel is full", zap.String("msgId", pe.msgId))
		}
	}

	acct.deletePendingTransferAlreadyLocked(pe.msgId)
}

// addPendingTransferAlreadyLocked adds a pending transfer to both the map and the database. It assumes the caller holds the lock.
func (acct *Accountant) addPendingTransferAlreadyLocked(pe *pendingEntry) error {
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
	acct.logger.Debug("deletePendingTransfer", zap.String("msgId", msgId))
	if _, exists := acct.pendingTransfers[msgId]; exists {
		delete(acct.pendingTransfers, msgId)
		transfersOutstanding.Set(float64(len(acct.pendingTransfers)))
	}
	if err := acct.db.AcctDeletePendingTransfer(msgId); err != nil {
		acct.logger.Error("failed to delete pending transfer from the db", zap.String("msgId", msgId), zap.Error(err))
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
		coveredByAcct, isNTT, enforceFlag := acct.isMessageCoveredByAccountant(msg)
		if !coveredByAcct {
			acct.logger.Error("dropping reloaded pending transfer because it is not covered by the accountant", zap.String("msgID", msgId))
			if err := acct.db.AcctDeletePendingTransfer(msgId); err != nil {
				acct.logger.Error("failed to delete pending transfer from the db", zap.String("msgId", msgId), zap.Error(err))
				// Ignore this error and keep going.
			}
			continue
		}
		acct.logger.Info("reloaded pending transfer", zap.String("msgID", msgId))

		digest := msg.CreateDigest()
		pe := &pendingEntry{msg: msg, msgId: msgId, digest: digest, isNTT: isNTT, enforceFlag: enforceFlag}
		pe.setUpdTime()
		acct.pendingTransfers[msgId] = pe
	}

	transfersOutstanding.Set(float64(len(acct.pendingTransfers)))
	if len(acct.pendingTransfers) != 0 {
		acct.logger.Info("reloaded pending transfers", zap.Int("total", len(acct.pendingTransfers)))
	} else {
		acct.logger.Info("no pending transfers to be reloaded")
	}

	return nil
}

// submitObservation sends an observation request to the worker so it can be submitted to the contract.  If the transfer is already
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

	if pe.isNTT {
		acct.submitToChannel(pe, acct.nttSubChan, "ntt-accountant")
	} else {
		acct.submitToChannel(pe, acct.subChan, "accountant")
	}

	return true
}

// submitToChannel submits an observation to the specified channel. If the submission fails because the channel is full,
// it marks the transfer as pending so it will be resubmitted by the audit.
func (acct *Accountant) submitToChannel(pe *pendingEntry, subChan chan *common.MessagePublication, tag string) {
	select {
	case subChan <- pe.msg:
		acct.logger.Debug(fmt.Sprintf("submitted observation to channel for %s", tag), zap.String("msgId", pe.msgId))
	default:
		acct.logger.Error(fmt.Sprintf("unable to submit observation to %s because the channel is full, will try next interval", tag), zap.String("msgId", pe.msgId))
		pe.state.submitPending = false
	}
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
