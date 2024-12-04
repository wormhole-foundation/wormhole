package tss

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/tss/internal"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/yossigi/tss-lib/v2/common"
	tssutil "github.com/yossigi/tss-lib/v2/ecdsa/ethereum"
	"github.com/yossigi/tss-lib/v2/ecdsa/keygen"
	"github.com/yossigi/tss-lib/v2/ecdsa/party"
	"github.com/yossigi/tss-lib/v2/tss"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type uuid digest // distinguishing between types to avoid confusion.

// Engine is the implementation of reliableTSS, it is a wrapper for the
// tss-lib fullParty and adds reliable broadcast logic
// to the message sending and receiving.
type Engine struct {
	ctx context.Context

	logger *zap.Logger
	GuardianStorage

	fpParams *party.Parameters
	fp       party.FullParty

	fpOutChan      chan tss.Message
	fpSigOutChan   chan *common.SignatureData // output inspected in fpListener.
	sigOutChan     chan *common.SignatureData // actual sig output.
	messageOutChan chan Sendable
	fpErrChannel   chan *tss.Error // used to log issues from the FullParty.

	started         atomic.Uint32
	msgSerialNumber uint64

	// used to perform reliable broadcast:
	mtx      *sync.Mutex
	received map[uuid]*broadcaststate

	sigCounter activeSigCounter

	// used for fault-tolerance:
	// informs a central tracker of the guardian's actions.
	// used to ensure the guardian is in the loop, and which guardians are active and on which chain.
	//
	// If the guardian attempted to sign previously, but wasn't part of the comittee, on some cases might change this case and add this
	// guardian to the committee.
	ftCommandChan chan ftCommand
}

type PEM []byte

// Contains the TSS related configurations.
type Configurations struct {
	maxSimultaneousSignatures int
	// MaxSignerTTL is the maximum time a signature is allowed to be active.
	// used to release resources.
	MaxSignerTTL   time.Duration
	DelayGraceTime time.Duration // the duration a guardian is allowed not to start the protocol for some digest (after f+1 of its peers did).

	// not exported so it can't be changed.
	guardianDownTime time.Duration // once a guardian is marked as faulty, this is the time it isn't allowed into the protocol.
	maxJitter        time.Duration // jitter is used to reduce the chance guardians get back at the same time from DownTime.
}

// GuardianStorage is a struct that holds the data needed for a guardian to participate in the TSS protocol
// including its signing key, and the shared symmetric keys with other guardians.
// should be loaded from a file.
type GuardianStorage struct {
	Configurations

	Self *tss.PartyID

	// should be a certificate generated with SecretKey
	TlsX509    PEM
	PrivateKey PEM
	tlsCert    *tls.Certificate
	signingKey *ecdsa.PrivateKey // should be the unmarshalled value of PriavteKey.

	// Stored sorted by Key. include Self.
	Guardians []*tss.PartyID

	// guardianCert[i] should be the x509.Cert of guardians[i]. (uses p256, since golang x509 doesn't support secp256k1)
	GuardianCerts  []PEM
	guardiansCerts []*x509.Certificate

	// Assumes threshold = 2f+1, where f is the maximal expected number of faulty nodes.
	Threshold int

	// all secret keys should be generated with specific value.
	SavedSecretParameters *keygen.LocalPartySaveData

	LoadDistributionKey []byte

	// data structures to ensure quick lookups:
	guardiansProtoIDs []*tsscommv1.PartyId
	guardianToCert    map[string]*x509.Certificate
	pemkeyToGuardian  map[string]*tss.PartyID
}

func (g *GuardianStorage) contains(pid *tss.PartyID) bool {
	for _, v := range g.Guardians {
		if equalPartyIds(pid, v) {
			return true
		}
	}

	return false
}

// GuardianStorageFromFile loads a guardian storage from a file.
// If the storage file hadn't contained symetric keys, it'll compute them.
func NewGuardianStorageFromFile(storagePath string) (*GuardianStorage, error) {
	var storage GuardianStorage
	if err := storage.load(storagePath); err != nil {
		return nil, err
	}

	return &storage, nil
}

// ProducedSignature lets a listener receive the output signatures once they're ready.
func (t *Engine) ProducedSignature() <-chan *common.SignatureData {
	return t.sigOutChan
}

// ProducedOutputMessages ensures a listener can send the output messages to the network.
func (t *Engine) ProducedOutputMessages() <-chan Sendable {
	return t.messageOutChan
}

func (st *GuardianStorage) fetchPartyIdFromBytes(pk []byte) *tsscommv1.PartyId {
	pid, ok := st.pemkeyToGuardian[string(pk)]
	if !ok {
		return nil
	}

	return partyIdToProto(pid)
}

func (st *GuardianStorage) FetchCertificate(pid *tsscommv1.PartyId) (*x509.Certificate, error) {
	if pid == nil {
		return nil, ErrNilPartyId
	}

	cert, ok := st.guardianToCert[partyIdToString(protoToPartyId(pid))]
	if !ok {
		return nil, fmt.Errorf("partyID certificate not found: %v", pid)
	}

	return cert, nil
}

// FetchPartyId implements ReliableTSS.
func (st *GuardianStorage) FetchPartyId(cert *x509.Certificate) (*tsscommv1.PartyId, error) {
	var pid *tsscommv1.PartyId

	switch key := cert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		publicKeyPem, err := internal.PublicKeyToPem(key)
		if err != nil {
			return nil, err
		}

		pid = st.fetchPartyIdFromBytes(publicKeyPem)
	case []byte:
		pid = st.fetchPartyIdFromBytes(key)
	default:
		return nil, fmt.Errorf("unsupported public key type")
	}

	if pid == nil {
		return nil, fmt.Errorf("certificate owner is unknown")
	}

	return pid, nil
}

// GetCertificate implements ReliableTSS.
func (st *GuardianStorage) GetCertificate() *tls.Certificate {
	return st.tlsCert
}

// GetPeers implements ReliableTSS.
func (st *GuardianStorage) GetPeers() []*x509.Certificate {
	return st.guardiansCerts
}

var (
	errNilTssEngine        = fmt.Errorf("tss engine is nil")
	errTssEngineNotStarted = fmt.Errorf("tss engine hasn't started")
)

// BeginAsyncThresholdSigningProtocol used to start the TSS protocol over a specific msg.
func (t *Engine) BeginAsyncThresholdSigningProtocol(vaaDigest []byte, chainID vaa.ChainID) error {
	if t == nil {
		return errNilTssEngine
	}

	if t.started.Load() != started {
		return errTssEngineNotStarted
	}

	if t.fp == nil {
		return fmt.Errorf("tss engine is not set up correctly, use NewReliableTSS to create a new engine")
	}

	if len(vaaDigest) != digestSize {
		return fmt.Errorf("vaaDigest length is not 32 bytes")
	}

	d := party.Digest{}
	copy(d[:], vaaDigest)

	cmd := getInactiveGuardiansCommand{
		ChainID: chainID,
		reply:   make(chan inactives, 1),
	}

	if err := intoChannelOrDone[ftCommand](t.ctx, t.ftCommandChan, &cmd); err != nil {
		return fmt.Errorf("failed to request for inactive guardians: %w", err)
	}

	// waiting for the reply.
	inactiveParties, err := outOfChannelOrDone(t.ctx, cmd.reply)
	if err != nil {
		return fmt.Errorf("failed to get inactive guardians: %w", err)
	}

	dgstStr := fmt.Sprintf("%x", vaaDigest)

	for _, faulties := range inactiveParties.getFaultiesLists() {

		if len(t.Guardians)-len(faulties) <= t.Threshold {
			t.logger.Error(
				"too many faulty guardians to start the signing protocol",
				zap.String("digest", dgstStr),
				zap.String("chainID", chainID.String()),
				zap.Strings("faulties", getCommitteeIDs(faulties)),
			)

			continue // not a failure of the method, so it should continue, instead of returning an error.
		}

		info, err := t.fp.AsyncRequestNewSignature(makeSigningRequest(d, faulties, chainID))

		if err != nil {
			// note, we don't inform the fault-tolerance tracker of the error, so it can put this guardian in timeoout.
			return err
		}

		flds := []zap.Field{
			zap.String("trackingID", info.TrackingID.ToString()),
			zap.String("ChainID", chainID.String()),
			zap.Any("committee", getCommitteeIDs(info.SigningCommittee)),
		}

		if len(faulties) > 0 {
			flds = append(flds, zap.Int("num-removed", len(faulties)))
			flds = append(flds, zap.Any("removed-from-committee", getCommitteeIDs(faulties)))
		}

		t.logger.Info(
			"guardian started signing protocol",
			flds...,
		)

		intoChannelOrDone[ftCommand](t.ctx, t.ftCommandChan, &signCommand{SigningInfo: info})

		if info.IsSigner {
			inProgressSigs.Inc()
		}

	}

	return nil
}

func makeSigningRequest(d party.Digest, faulties []*tss.PartyID, chainID vaa.ChainID) party.SigningTask {
	return party.SigningTask{
		Digest: d,
		// indicating the reviving guardian will be given a chance to join the protocol.
		Faulties:     faulties,
		AuxilaryData: chainIDToBytes(chainID),
	}
}

func NewReliableTSS(storage *GuardianStorage) (ReliableTSS, error) {
	if storage == nil {
		return nil, fmt.Errorf("the guardian's tss storage is nil")
	}

	if storage.maxSimultaneousSignatures < 0 {
		storage.maxSimultaneousSignatures = defaultMaxLiveSignatures
	}

	if storage.MaxSignerTTL == 0 {
		storage.MaxSignerTTL = defaultMaxSignerTTL
	}

	if storage.DelayGraceTime == 0 {
		storage.DelayGraceTime = defaultDelayGraceTime
	}

	if storage.guardianDownTime == 0 {
		storage.guardianDownTime = defaultGuardianDownTime
	}

	if storage.maxJitter == 0 {
		storage.maxJitter = defaultMaxDownTimeJitter
	}

	if storage.maxSimultaneousSignatures == 0 {
		storage.maxSimultaneousSignatures = defaultMaxLiveSignatures
	}

	fpParams := &party.Parameters{
		SavedSecrets:         storage.SavedSecretParameters,
		PartyIDs:             storage.Guardians,
		Self:                 storage.Self,
		Threshold:            storage.Threshold,
		WorkDir:              "", // set to empty since we don't support DKG/reshare protocol yet.
		MaxSignerTTL:         storage.MaxSignerTTL,
		LoadDistributionSeed: storage.LoadDistributionKey,
	}

	fp, err := party.NewFullParty(fpParams)
	if err != nil {
		return nil, err
	}

	expectedMsgs := storage.maxSimultaneousSignatures *
		(numBroadcastsPerSignature + numUnicastsRounds*storage.Threshold)
	t := &Engine{
		ctx: nil,

		logger:          &zap.Logger{},
		GuardianStorage: *storage,

		fpParams:     fpParams,
		fp:           fp,
		fpOutChan:    make(chan tss.Message, expectedMsgs),
		fpSigOutChan: make(chan *common.SignatureData, storage.maxSimultaneousSignatures),
		sigOutChan:   make(chan *common.SignatureData, storage.maxSimultaneousSignatures),

		fpErrChannel:    make(chan *tss.Error),
		messageOutChan:  make(chan Sendable),
		msgSerialNumber: 0,
		mtx:             &sync.Mutex{},
		received:        map[uuid]*broadcaststate{},

		started: atomic.Uint32{}, // default value is 0

		sigCounter: newSigCounter(),

		ftCommandChan: make(chan ftCommand, expectedMsgs),
	}

	return t, nil
}

// Start starts the TSS engine, and listens for the outputs of the full party.
func (t *Engine) Start(ctx context.Context) error {
	if t == nil {
		return fmt.Errorf("tss engine is nil")
	}

	if !t.started.CompareAndSwap(notStarted, started) {
		return fmt.Errorf("tss engine has already started")
	}

	t.ctx = ctx
	t.logger = supervisor.Logger(ctx).
		With(zap.String("ID", t.GuardianStorage.Self.Id)).
		Named("tss")

	if err := t.fp.Start(t.fpOutChan, t.fpSigOutChan, t.fpErrChannel); err != nil {
		t.started.Store(notStarted)

		return err
	}

	// closing the t.fp.start inside th listener
	go t.fpListener()

	go t.ftTracker()

	t.logger.Info(
		"tss engine started",
		zap.Any("configs", t.GuardianStorage.Configurations),
	)

	return nil
}

func (t *Engine) GetPublicKey() *ecdsa.PublicKey {
	return t.fp.GetPublic()
}

func (t *Engine) GetEthAddress() ethcommon.Address {
	pubkey := t.fp.GetPublic()
	ethAddBytes := ethcommon.LeftPadBytes(
		crypto.Keccak256(tssutil.EcdsaPublicKeyToBytes(pubkey)[1:])[12:], 32)

	return ethcommon.BytesToAddress(ethAddBytes)
}

func (st *GuardianStorage) maxSignerTTL() time.Duration {
	// SECURITY NOTE: when we clean the guardian map from received Echo's
	// we must use TTL > FullParty.TTL to ensure guardians can't use
	// the deletion time to perform equivication attacks (since a message
	// has no record after it was deleted).
	// *2 is to account for possible offset in the time of the guardian.
	return st.MaxSignerTTL * 2
}

// fpListener serves as a listining loop for the full party outputs.
// ensures the FP isn't being blocked on writing to fpOutChan, and wraps the result into a gossip message.
func (t *Engine) fpListener() {
	maxTTL := t.GuardianStorage.maxSignerTTL()

	cleanUpTicker := time.NewTicker(maxTTL)

	for {
		select {
		case <-t.ctx.Done():
			t.logger.Info(
				"shutting down TSS Engine",
			)

			t.fp.Stop()
			cleanUpTicker.Stop()

			return
		case m := <-t.fpOutChan:
			t.handleFpOutput(m)

		case err := <-t.fpErrChannel:
			t.handleFpError(err)

		case sig := <-t.fpSigOutChan:
			t.handleFpSignature(sig)

		case <-cleanUpTicker.C:
			t.cleanup(maxTTL)
		}
	}
}

func (t *Engine) handleFpSignature(sig *common.SignatureData) {
	if sig == nil {
		return
	}

	t.logger.Debug("signature complete. updating inner state and forwarding it", zap.String("trackingId", sig.TrackingId.ToString()))

	sigProducedCntr.Inc()
	inProgressSigs.Dec()

	t.sigCounter.remove(sig.TrackingId)
	if err := intoChannelOrDone[ftCommand](t.ctx, t.ftCommandChan, &SigEndCommand{sig.TrackingId}); err != nil {
		t.logger.Error("couldn't inform the fault-tolerance tracker of the signature end", zap.Error(err), zap.String("trackingId", sig.TrackingId.ToString()))
	}

	if err := intoChannelOrDone(t.ctx, t.sigOutChan, sig); err != nil {
		t.logger.Error("couldn't deliver outside of engine the signature", zap.Error(err), zap.String("trackingId", sig.TrackingId.ToString()))
	}
}

func (t *Engine) handleFpError(err *tss.Error) {
	if err == nil {
		return
	}

	trackid := err.TrackingId()
	if trackid == nil {
		t.logger.Error("error (without trackingID) in signing protocol ", zap.Error(err.Cause()))
		return
	}

	if err := intoChannelOrDone[ftCommand](t.ctx, t.ftCommandChan, &SigEndCommand{trackid}); err != nil {
		t.logger.Error("couldn't inform the fault-tolerance tracker of signature end due to error",
			zap.Error(err),
			zap.String("trackingId", trackid.ToString()),
		)
	}

	// if someone sent a message that caused an error -> we don't
	// accept an override to that message, therefore, we can remove it, since it won't change.
	t.sigCounter.remove(trackid)
	inProgressSigs.Dec()

	logErr(t.logger, &logableError{
		fmt.Errorf("error in signing protocol: %w", err.Cause()),
		trackid,
		intToRound(err.Round()),
	})
}

func (t *Engine) handleFpOutput(m tss.Message) {
	tssMsg, err := t.intoSendable(m)
	if err == nil {
		sentMsgCntr.Inc()

		if err := intoChannelOrDone(t.ctx, t.messageOutChan, tssMsg); err != nil {
			t.logger.Error("couldn't output message to be sent via network",
				zap.Error(err),
				zap.String("trackingId", m.WireMsg().GetTrackingID().ToString()),
			)
		}

		return
	}

	// else log error:
	lgErr := logableError{
		fmt.Errorf("failed to convert tss message and send it to network: %w", err),
		m.WireMsg().GetTrackingID(),
		"",
	}

	// The following should always pass, since FullParty outputs a
	// tss.ParsedMessage and a valid message with a specific round.
	if parsed, ok := m.(tss.ParsedMessage); ok {
		if rnd, e := getRound(parsed); e == nil {
			lgErr.round = rnd
		}
	}

	logErr(t.logger, lgErr)
}

func (t *Engine) cleanup(maxTTL time.Duration) {
	now := time.Now()

	t.mtx.Lock()
	defer t.mtx.Unlock()

	for k, v := range t.received {
		if now.Sub(v.timeReceived) > maxTTL {
			delete(t.received, k)

			// since the fullParty deleted its state, we can remove the sigCounter entry.
			t.sigCounter.remove(v.trackingId)
		}
	}
}

func (t *Engine) intoSendable(m tss.Message) (Sendable, error) {
	bts, routing, err := m.WireBytes()
	if err != nil {
		return nil, err
	}

	content := &tsscommv1.SignedMessage_TssContent{
		TssContent: &tsscommv1.TssContent{
			Payload:         bts,
			MsgSerialNumber: atomic.AddUint64(&t.msgSerialNumber, 1),
		},
	}

	var sendable Sendable

	if routing.IsBroadcast || len(routing.To) == 0 {
		msgToSend := &tsscommv1.SignedMessage{
			Content:   content,
			Sender:    partyIdToProto(t.Self),
			Signature: nil,
		}

		if err := t.sign(msgToSend); err != nil {
			return nil, err
		}

		sendable = newEcho(msgToSend, t.guardiansProtoIDs)
	} else {
		indices := make([]*tsscommv1.PartyId, 0, len(routing.To))
		for _, pId := range routing.To {
			indices = append(indices, partyIdToProto(pId))
		}

		sendable = &Unicast{
			Unicast:     content.TssContent,
			Receipients: indices,
		}
	}

	return sendable, nil
}

func (t *Engine) HandleIncomingTssMessage(msg Incoming) {
	if t == nil {
		return // TODO: Consider what to do.
	}

	if t.started.Load() != started {
		return // TODO: Consider what to do.
	}

	receivedMsgCntr.Inc()

	if err := t.handleIncomingTssMessage(msg); err != nil {
		logErr(t.logger, err)
	}
}

var (
	errNilIncoming                = fmt.Errorf("received nil incoming message")
	errNilSource                  = fmt.Errorf("no source in incoming message")
	errNeitherBroadcastNorUnicast = fmt.Errorf("received incoming message which is neither broadcast nor unicast")
)

func (t *Engine) handleIncomingTssMessage(msg Incoming) error {
	if msg == nil {
		return errNilIncoming
	}

	if msg.GetSource() == nil {
		return errNilSource
	}

	if msg.IsUnicast() {
		return t.handleUnicast(msg)
	} else if !msg.IsBroadcast() {
		return errNeitherBroadcastNorUnicast
	}

	shouldEcho, err := t.handleEcho(msg)
	if err != nil {
		return err
	}

	if !shouldEcho {
		return nil // not an error, just don't echo.
	}

	return t.sendEchoOut(msg)
}

func (t *Engine) sendEchoOut(m Incoming) error {
	content, ok := proto.Clone(m.toEcho()).(*tsscommv1.Echo)
	if !ok {
		return fmt.Errorf("failed to clone echo message")
	}

	ech := newEcho(content.Message, t.guardiansProtoIDs)

	if err := intoChannelOrDone[Sendable](t.ctx, t.messageOutChan, ech); err != nil {
		return fmt.Errorf("couldn't output echo to be sent via network: %w", err)
	}

	return nil
}

var errBadRoundsInEcho = fmt.Errorf("cannot receive echos for rounds: %v,%v", round1Message1, round2Message)

func (t *Engine) handleEcho(m Incoming) (bool, error) {
	parsed, err := t.parseEcho(m)
	if err != nil {
		if parsed != nil {
			err = parsed.wrapError(err)
		}

		return false, err
	}

	shouldEcho, shouldDeliver, err := t.relbroadcastInspection(parsed, m)
	if err != nil {
		return false, parsed.wrapError(err)
	}

	if !shouldDeliver {
		return shouldEcho, nil
	}

	switch v := parsed.(type) {
	case *parsedProblem:
		intoChannelOrDone[ftCommand](t.ctx, t.ftCommandChan, &reportProblemCommand{*v}) // received delivery status.
	case *parsedTssContent:

		deliveredMsgCntr.Inc()
		if err := t.feedIncomingToFp(v.ParsedMessage); err != nil {
			return shouldEcho, parsed.wrapError(fmt.Errorf("failed to update the full party: %w", err))
		}
	}

	return shouldEcho, nil
}

func (t *Engine) feedIncomingToFp(parsed tss.ParsedMessage) error {
	trackId := parsed.WireMsg().TrackingID
	from := parsed.GetFrom()
	maxLiveSignatures := t.GuardianStorage.maxSimultaneousSignatures

	if ok := t.sigCounter.add(trackId, from, maxLiveSignatures); ok {
		// TODO: Should I update that a delivery was made even if sigCounter blocked it?

		intoChannelOrDone[ftCommand](t.ctx, t.ftCommandChan, &deliveryCommand{
			parsedMsg: parsed,
			from:      from,
		})

		return t.fp.Update(parsed)
	}

	tooManySimulSigsErrCntr.Inc()

	return fmt.Errorf("guardian %v has reached the maximum number of simultaneous signatures", from.Id)
}

var errUnicastBadRound = fmt.Errorf("bad round for unicast (can accept round1Message1 and round2Message)")

func (t *Engine) handleUnicast(m Incoming) error {
	parsed, err := t.parseUnicast(m)
	if err != nil {
		err = fmt.Errorf("couldn't parse unicast payload: %w", err)
		if parsed != nil {
			err = parsed.wrapError(err)
		}

		return err
	}

	fpmsg, ok := parsed.(*parsedTssContent)
	if !ok {
		return parsed.wrapError(fmt.Errorf("unicast casting issue"))
	}

	err = t.validateUnicastDoesntExist(fpmsg)
	if err == errUnicastAlreadyReceived {
		return nil
	}

	if err != nil {
		return parsed.wrapError(fmt.Errorf("failed to ensure no equivication present in unicast: %w, sender:%v", err, m.GetSource().Id))
	}

	if err := t.feedIncomingToFp(fpmsg); err != nil {
		return parsed.wrapError(fmt.Errorf("unicast failed to update the full party: %w", err))
	}

	return nil
}

var errUnicastAlreadyReceived = fmt.Errorf("unicast already received")

func (t *Engine) validateUnicastDoesntExist(parsed tss.ParsedMessage) error {
	id, err := getMessageUUID(parsed, t.LoadDistributionKey)
	if err != nil {
		return err
	}

	bts, _, err := parsed.WireBytes()
	if err != nil {
		return fmt.Errorf("failed storing the unicast: %w", err)
	}

	msgDigest := hash(bts)

	t.mtx.Lock()
	defer t.mtx.Unlock()

	if stored, ok := t.received[id]; ok {
		if stored.messageDigest != msgDigest {
			return ErrEquivicatingGuardian
		}

		return errUnicastAlreadyReceived
	}

	t.received[id] = &broadcaststate{
		timeReceived:  time.Now(), // used for GC.
		messageDigest: hash(bts),  // used to ensure no equivocation.
		votes:         nil,        // no votes should be stored for a unicast.
		echoedAlready: true,       // ensuring this never echoed since it is a unicast.
		mtx:           nil,        // no need to lock this, just store it.
	}

	return nil
}

var (
	ErrUnkownEchoer = fmt.Errorf("echoer is not a known guardian")
	ErrUnkownSender = fmt.Errorf("sender is not a known guardian")
)

func (t *Engine) parseEcho(m Incoming) (processedMessage, error) {
	echoMsg := m.toEcho()
	if err := vaidateEchoCorrectForm(echoMsg); err != nil {
		return nil, err
	}

	senderPid := protoToPartyId(echoMsg.Message.Sender)
	if !t.GuardianStorage.contains(senderPid) {
		return nil, fmt.Errorf("%w: %v", ErrUnkownSender, senderPid)
	}

	switch cntnt := echoMsg.Message.Content.(type) {
	case *tsscommv1.SignedMessage_Problem:
		return &parsedProblem{
			Problem: cntnt.Problem,
			issuer:  echoMsg.Message.Sender,
		}, nil

	case *tsscommv1.SignedMessage_TssContent:
		p, err := tss.ParseWireMessage(cntnt.TssContent.Payload, senderPid, true)
		if err != nil {
			return nil, err
		}

		parsed := &parsedTssContent{p, ""}

		rnd, err := getRound(parsed)
		if err != nil {
			return parsed, fmt.Errorf("couldn't extract round from echo: %w", err)
		}

		parsed.signingRound = rnd

		// according to gg18 (tss ecdsa paper), unicasts are sent in these rounds.
		if rnd == round1Message1 || rnd == round2Message {
			return parsed, errBadRoundsInEcho
		}

		if err := t.validateTrackingIDForm(parsed.getTrackingID()); err != nil {
			return parsed, err
		}

		return parsed, nil
	default:
		return nil, fmt.Errorf("unknown content type: %T", cntnt)
	}
}

// SECURITY NOTE: this function sets a sessionID to a message. Used to ensure no equivocation.
//
// We don't add the content of the message to the uuid, instead we collect all data that can put this message in a context.
// this is used by the reliable broadcast to check no two messages from the same sender will be used to update the full party
// in the same round for the specific session of the protocol.
func getMessageUUID(msg tss.ParsedMessage, loadDistKey []byte) (uuid, error) {
	// The TackingID of a parsed message is tied to the run of the protocol for a single
	//  signature, thus we use it as a sessionID.
	messageTrackingID := [trackingIDHexStrSize]byte{}
	copy(messageTrackingID[:], []byte(msg.WireMsg().GetTrackingID().ToString()))

	fromId := [hostnameSize]byte{}
	copy(fromId[:], msg.GetFrom().Id)

	fromKey := [pemKeySize]byte{}
	copy(fromKey[:], msg.GetFrom().Key)

	// Adding the round allows the same sender to send messages for different rounds.
	// but, sender j is not allowed to send two different messages to the same round.
	rnd, err := getRound(msg)
	if err != nil {
		return uuid{}, err
	}

	round := [signingRoundSize]byte{}
	copy(round[:], rnd)

	d := make([]byte, 0, len(tssContentDomain)+len(loadDistKey)+int(trackingIDHexStrSize)+hostnameSize+pemKeySize)

	d = append(d, tssContentDomain...)
	d = append(d, loadDistKey...)
	d = append(d, messageTrackingID[:]...)
	d = append(d, fromId[:]...)
	d = append(d, fromKey[:]...)
	d = append(d, round[:]...)

	return uuid(hash(d)), nil
}

func (t *Engine) parseUnicast(m Incoming) (processedMessage, error) {
	if err := validateContentCorrectForm(m.toUnicast()); err != nil {
		return nil, err
	}

	p, err := tss.ParseWireMessage(m.toUnicast().Payload, protoToPartyId(m.GetSource()), false)
	if err != nil {
		return nil, err
	}

	parsed := &parsedTssContent{p, ""}

	// ensuring the reported source of the message matches the claimed source. (parsed.GetFrom() used by the tss-lib)
	if !equalPartyIds(parsed.GetFrom(), protoToPartyId(m.GetSource())) {
		return parsed, fmt.Errorf("parsed message sender doesn't match the source of the message")
	}

	rnd, err := getRound(parsed)
	if err != nil {
		return parsed, fmt.Errorf("unicast parsing error: %w", err)
	}

	parsed.signingRound = rnd

	// only round 1 and round 2 are unicasts.
	if rnd != round1Message1 && rnd != round2Message {
		return parsed, errUnicastBadRound
	}

	if err := t.validateTrackingIDForm(parsed.getTrackingID()); err != nil {
		return parsed, err
	}

	return parsed, nil
}

func (st *GuardianStorage) sign(msg *tsscommv1.SignedMessage) error {
	digest := hashSignedMessage(msg)

	sig, err := st.signingKey.Sign(rand.Reader, digest[:], nil)
	msg.Signature = sig

	return err
}

var ErrInvalidSignature = fmt.Errorf("invalid signature")

var errEmptySignature = fmt.Errorf("empty signature")

func (st *GuardianStorage) verifySignedMessage(msg *tsscommv1.SignedMessage) error {
	if msg == nil {
		return fmt.Errorf("nil signed message")
	}

	if msg.Signature == nil {
		return errEmptySignature
	}

	cert, err := st.FetchCertificate(msg.Sender)
	if err != nil {
		return err
	}

	pk, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("certificated stored with non-ecdsa public key, guardian storage is corrupted")
	}

	digest := hashSignedMessage(msg)

	isValid := ecdsa.VerifyASN1(pk, digest[:], msg.Signature)

	if !isValid {
		return ErrInvalidSignature
	}

	return nil
}
