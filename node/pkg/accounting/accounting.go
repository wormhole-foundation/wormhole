package accounting

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/wormconn"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"

	wasmdtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"

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
		v          *vaa.VAA
		digest     string
		updTime    time.Time
		retryCount int
	}
)

// Accounting is the object that manages the interface to the wormchain accounting smart contract.
type Accounting struct {
	ctx              context.Context
	logger           *zap.Logger
	db               db.AccountingDB
	contract         string
	wsUrl            string
	lcdUrl           string
	wormchainConn    *wormconn.ClientConn
	enforceFlag      bool
	gk               *ecdsa.PrivateKey
	gst              *common.GuardianSetState
	msgChan          chan<- *common.MessagePublication
	mutex            sync.Mutex
	tokenBridges     map[tokenBridgeKey]*tokenBridgeEntry
	pendingTransfers map[pendingKey]*pendingEntry
	env              int
}

// NewAccounting creates a new instance of the Accounting object.
func NewAccounting(
	ctx context.Context,
	logger *zap.Logger,
	db db.AccountingDB,
	contract string, // the address of the smart contract on wormchain
	wsUrl string, // the URL of the wormchain websocket interface
	lcdUrl string, // the URL of the wormchain LCD interface
	wormchainConn *wormconn.ClientConn, // used for communicating with the smart contract
	enforceFlag bool, // whether or not accounting should be enforced
	gk *ecdsa.PrivateKey, // the guardian key used for signing observation requests
	gst *common.GuardianSetState, // used to get the current guardian set index when sending observation requests
	msgChan chan<- *common.MessagePublication, // the channel where transfers received by the accounting runnable should be published
	env int, // Controls the set of token bridges to be monitored
) *Accounting {
	return &Accounting{
		ctx:              ctx,
		logger:           logger,
		db:               db,
		contract:         contract,
		wsUrl:            wsUrl,
		lcdUrl:           lcdUrl,
		wormchainConn:    wormchainConn,
		enforceFlag:      enforceFlag,
		gk:               gk,
		gst:              gst,
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

func (acct *Accounting) Close() {
	acct.mutex.Lock()
	defer acct.mutex.Unlock()

	if acct.wormchainConn != nil {
		acct.wormchainConn.Close()
		acct.wormchainConn = nil
	}
}

func (acct *Accounting) FeatureString() string {
	if !acct.enforceFlag {
		return "acct:logonly"
	}
	return "acct:enforced"
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

	gs := acct.gst.Get()
	if gs == nil {
		acct.logger.Error("acct: failed to look up guardian set, blocking publishing", zap.String("msgID", msg.MessageIDString()))
		return false, nil
	}

	v := msg.CreateVAA(gs.Index)
	db := v.SigningMsg()
	digest := hex.EncodeToString(db.Bytes())

	// If this is already pending, don't send it again.
	pk := pendingKey{emitterChainId: msg.EmitterChain, txHash: msg.TxHash}
	if oldPk, exists := acct.pendingTransfers[pk]; exists {
		if oldPk.digest != digest {
			acct.logger.Error("acct: digest in pending transfer has changed",
				zap.String("msgID", msg.MessageIDString()),
				zap.String("oldDigest", oldPk.digest),
				zap.String("newDigest", digest),
			)
		} else {
			acct.logger.Info("acct: blocking previously pending transfer", zap.String("msgID", msg.MessageIDString()))
		}
		return false, nil
	}

	// Add it to the pending map and the database.
	if err := acct.addPendingTransfer(&pk, msg, v, digest); err != nil {
		acct.logger.Error("acct: failed to persist pending transfer, blocking publishing", zap.String("msgID", msg.MessageIDString()), zap.Error(err))
		return false, err
	}

	acct.logger.Info("acct: submitting transfer to accounting for approval", zap.String("msgID", msg.MessageIDString()), zap.Bool("canPublish", !acct.enforceFlag))

	// This transaction may take a while. Run it as a go routine so we don't block the processor.
	if acct.env != GoTestMode {
		go acct.submitObservationToContract(msg, gs.Index)
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

type (
	SubmitObservationsMsg struct {
		Params SubmitObservationsParams `json:"submit_observations"`
	}

	SubmitObservationsParams struct {
		// A serialized `Vec<Observation>`. Multiple observations can be submitted together to reduce  transaction overhead.
		Observations string `json:"observations"`

		// The index of the guardian set used to sign the observations.
		GuardianSetIndex uint32 `json:"guardian_set_index"`

		// A signature for `observations`.
		Signature SignatureType `json:"signature"`
	}

	SignatureType struct {
		Index     uint32         `json:"index"`
		Signature SignatureBytes `json:"signature"`
	}

	SignatureBytes []uint8

	Observation struct {
		// The key that uniquely identifies the Observation.
		Key TransferKey `json:"key"`

		// The nonce for the transfer.
		Nonce uint32 `json:"nonce"`

		// The serialized tokenbridge payload.
		Payload string `json:"payload"`

		// The hash of the transaction on the emitter chain in which the transfer was performed.
		TxHash string `json:"tx_hash"`
	}

	TransferKey struct {
		// The chain id of the chain on which this transfer originated.
		EmitterChain uint16 `json:"emitter_chain"`

		// The address on the emitter chain that created this transfer.
		EmitterAddress string `json:"emitter_address"`

		// The sequence number of the transfer.
		Sequence uint64 `json:"sequence"`
	}
)

func (sb SignatureBytes) MarshalJSON() ([]byte, error) {
	var result string
	if sb == nil {
		result = "null"
	} else {
		result = strings.Join(strings.Fields(fmt.Sprintf("%d", sb)), ",")
	}
	return []byte(result), nil
}

// submitObservationToContract makes a call to the smart contract to submit an observation request.
// It should be called from a go routine because it can block.
func (acct *Accounting) submitObservationToContract(msg *common.MessagePublication, gsIndex uint32) {
	obs := []Observation{
		Observation{
			Key: TransferKey{
				EmitterChain:   uint16(msg.EmitterChain),
				EmitterAddress: base64.StdEncoding.EncodeToString(msg.EmitterAddress.Bytes()),
				Sequence:       msg.Sequence,
			},
			Nonce:   msg.Nonce,
			TxHash:  strings.Trim(string(msg.TxHash.String()), `0x`),
			Payload: base64.StdEncoding.EncodeToString(msg.Payload),
		},
	}

	if _, err := SubmitObservationToContract(acct.ctx, acct.gk, gsIndex, acct.wormchainConn, acct.contract, obs); err != nil {
		// Should allow TransferError::DuplicateTransfer - Just publish it (probably reobservation).
		// Should handle DuplicateSignatureError - Don't publish it, just keep waiting.
		acct.logger.Error("acct: failed to submit observation request", zap.String("msgId", msg.MessageIDString()), zap.Error(err))
		submitFailures.Inc()
		return
	}
}

// SubmitObservationToContract is a free function to make a call to the smart contract to submit an observation request.
func SubmitObservationToContract(
	ctx context.Context,
	gk *ecdsa.PrivateKey,
	gsIndex uint32,
	wormchainConn *wormconn.ClientConn,
	contract string,
	obs []Observation,
) (*sdktx.BroadcastTxResponse, error) {
	bytes, err := json.Marshal(obs)
	if err != nil {
		err = fmt.Errorf("acct: failed to marshal accounting observation request: %w", err)
		panic(err)
	}

	b64String := base64.StdEncoding.EncodeToString(bytes)

	digest := vaa.SigningMsg(bytes)

	SignatureBytes, err := ethCrypto.Sign(digest.Bytes(), gk)
	if err != nil {
		err = fmt.Errorf("acct: failed to sign accounting Observation request: %w", err)
		panic(err)
	}

	sig := SignatureType{Index: 0, Signature: SignatureBytes}

	msgData := SubmitObservationsMsg{
		Params: SubmitObservationsParams{
			Observations:     b64String,
			GuardianSetIndex: gsIndex,
			Signature:        sig,
		},
	}

	msgBytes, err := json.Marshal(msgData)
	if err != nil {
		err = fmt.Errorf("acct: failed to marshal accounting observation request: %w", err)
		panic(err)
	}

	subMsg := wasmdtypes.MsgExecuteContract{
		Sender:   wormchainConn.PublicKey(),
		Contract: contract,
		Msg:      msgBytes,
		Funds:    sdktypes.Coins{},
	}

	return wormchainConn.SignAndBroadcastTx(ctx, &subMsg)
}

// AuditPending audits the set of pending transfers for any that can be released, or ones that are stuck. This is called from the processor loop
// each timer interval. Any transfers that can be released will be forwarded to the accounting message channel.
func (acct *Accounting) AuditPendingTransfers() {
	acct.mutex.Lock()
	defer acct.mutex.Unlock()

	if len(acct.pendingTransfers) == 0 {
		return
	}

	gs := acct.gst.Get()
	if gs == nil {
		acct.logger.Error("acct: failed to look up guardian set, unable to audit pending transfers", zap.Int("numPending", len(acct.pendingTransfers)))
		return
	}

	for pk, pe := range acct.pendingTransfers {
		if time.Since(pe.updTime) > auditInterval {
			pe.retryCount += 1
			if pe.retryCount > maxRetries {
				acct.logger.Error("acct: stuck pending transfer has reached the retry limit, dropping it", zap.String("msgId", pe.msg.MessageIDString()))
				acct.deletePendingTransfer(&pk, pe.msg.MessageIDString())
			}

			acct.logger.Error("acct: resubmitting pending transfer", zap.String("msgId", pe.msg.MessageIDString()), zap.Stringer("lastUpdateTime", pe.updTime))
			pe.updTime = time.Now()
			go acct.submitObservationToContract(pe.msg, gs.Index)
			transfersSubmitted.Inc()
		}
	}
}

// publishTransfer publishes a pending transfer to the accounting channel and updates the timestamp. It assumes the caller holds the lock.
func (acct *Accounting) publishTransfer(pe *pendingEntry) {
	acct.msgChan <- pe.msg
	pe.updTime = time.Now()
}

// addPendingTransfer adds a pending transfer to both the map and the database. It assumes the caller holds the lock.
func (acct *Accounting) addPendingTransfer(pk *pendingKey, msg *common.MessagePublication, v *vaa.VAA, digest string) error {
	if err := acct.db.AcctStorePendingTransfer(msg); err != nil {
		return err
	}

	pe := &pendingEntry{msg: msg, v: v, digest: digest, updTime: time.Now()}
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

		v := msg.CreateVAA(0) // TODO: Need to persist the gsIndex!
		db := v.SigningMsg()
		digest := hex.EncodeToString(db.Bytes())

		pe := &pendingEntry{msg: msg, v: v, digest: digest} // Leave the updTime unset so we will query this on the first audit interval.
		acct.pendingTransfers[pk] = pe
		transfersOutstanding.Inc()
	}

	if len(acct.pendingTransfers) != 0 {
		acct.logger.Info("acct: reloaded pending transfers", zap.Int("total", len(acct.pendingTransfers)))
	}

	return nil
}

/* TODO:
- Pending map key should include the digest? Or not, still being debated.
- Peg a metric on accounting failures.
*/
