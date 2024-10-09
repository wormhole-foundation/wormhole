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
	"github.com/yossigi/tss-lib/v2/common"
	tssutil "github.com/yossigi/tss-lib/v2/ecdsa/ethereum"
	"github.com/yossigi/tss-lib/v2/ecdsa/keygen"
	"github.com/yossigi/tss-lib/v2/ecdsa/party"
	"github.com/yossigi/tss-lib/v2/tss"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

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
	fpSigOutChan   chan *common.SignatureData
	messageOutChan chan Sendable
	fpErrChannel   chan *tss.Error // used to log issues from the FullParty.

	started         atomic.Uint32
	msgSerialNumber uint64

	// used to perform reliable broadcast:
	mtx      *sync.Mutex
	received map[digest]*broadcaststate
}

type PEM []byte

// GuardianStorage is a struct that holds the data needed for a guardian to participate in the TSS protocol
// including its signing key, and the shared symmetric keys with other guardians.
// should be loaded from a file.
type GuardianStorage struct {
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
	return t.fpSigOutChan
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
func (t *Engine) BeginAsyncThresholdSigningProtocol(vaaDigest []byte) error {
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

	t.logger.Info(
		"guardian started signing protocol",
		zap.String("guardian", t.GuardianStorage.Self.Id),
		zap.String("digest", fmt.Sprintf("%x", vaaDigest)),
	)

	d := party.Digest{}
	copy(d[:], vaaDigest)

	return t.fp.AsyncRequestNewSignature(d)
}

func NewReliableTSS(storage *GuardianStorage) (ReliableTSS, error) {
	if storage == nil {
		return nil, fmt.Errorf("the guardian's tss storage is nil")
	}

	fpParams := &party.Parameters{
		SavedSecrets:         storage.SavedSecretParameters,
		PartyIDs:             storage.Guardians,
		Self:                 storage.Self,
		Threshold:            storage.Threshold,
		WorkDir:              "",
		MaxSignerTTL:         time.Minute * 5,
		LoadDistributionSeed: storage.LoadDistributionKey,
	}

	fp, err := party.NewFullParty(fpParams)
	if err != nil {
		return nil, err
	}

	fpOutChan := make(chan tss.Message) // this one must listen on it, and output to the p2p network.
	fpSigOutChan := make(chan *common.SignatureData)
	fpErrChannel := make(chan *tss.Error)

	t := &Engine{
		ctx: nil,

		logger:          &zap.Logger{},
		GuardianStorage: *storage,

		fpParams:        fpParams,
		fp:              fp,
		fpOutChan:       fpOutChan,
		fpSigOutChan:    fpSigOutChan,
		fpErrChannel:    fpErrChannel,
		messageOutChan:  make(chan Sendable),
		msgSerialNumber: 0,
		mtx:             &sync.Mutex{},
		received:        map[digest]*broadcaststate{},

		started: atomic.Uint32{}, // default value is 0
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
	t.logger = supervisor.Logger(ctx)

	if err := t.fp.Start(t.fpOutChan, t.fpSigOutChan, t.fpErrChannel); err != nil {
		t.started.Store(notStarted)

		return err
	}

	// closing the t.fp.start inside th listener
	go t.fpListener()

	t.logger.Info(
		"tss engine started",
		zap.String("guardian", t.GuardianStorage.Self.Id),
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

// fpListener serves as a listining loop for the full party outputs.
// ensures the FP isn't being blocked on writing to fpOutChan, and wraps the result into a gossip message.
func (t *Engine) fpListener() {
	// using a few more seconds to ensure
	cleanUpTicker := time.NewTicker(t.fpParams.MaxSignerTTL + time.Second*5)

	for {
		select {
		case <-t.ctx.Done():
			t.logger.Info(
				"shutting down TSS Engine",
				zap.String("guardian", t.GuardianStorage.Self.Id),
			)

			t.fp.Stop()
			cleanUpTicker.Stop()

			return

		case m := <-t.fpOutChan:
			tssMsg, err := t.intoSendable(m)
			if err == nil {
				t.messageOutChan <- tssMsg

				continue
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

		case err := <-t.fpErrChannel:
			if err == nil {
				continue // shouldn't happen. safety.
			}

			logErr(t.logger, &logableError{
				fmt.Errorf("error in signing protocol: %w", err.Cause()),
				err.TrackingId(),
				intToRound(err.Round()),
			})

		case <-cleanUpTicker.C:
			t.cleanup()
		}
	}
}

func (t *Engine) cleanup() {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	for k, v := range t.received {
		if time.Since(v.timeReceived) > time.Minute*5 {
			// althoug delete doesn't reduce the size of the underlying map
			// it is good enough since this map contains many entries, and it'll be wastefull to let a new map grow again.
			delete(t.received, k)
		}
	}
}

func (t *Engine) intoSendable(m tss.Message) (Sendable, error) {
	bts, routing, err := m.WireBytes()
	if err != nil {
		return nil, err
	}

	content := &tsscommv1.TssContent{
		Payload:         bts,
		MsgSerialNumber: atomic.AddUint64(&t.msgSerialNumber, 1),
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
			Unicast:     content,
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

	select {
	case <-t.ctx.Done():
		return t.ctx.Err()
	case t.messageOutChan <- newEcho(content.Message, t.guardiansProtoIDs):
	}

	return nil
}

var errBadRoundsInEcho = fmt.Errorf("cannot receive echos for rounds: %v,%v", round1Message1, round2Message)

func (t *Engine) handleEcho(m Incoming) (bool, error) {
	parsed, err := t.parseEcho(m)
	if err != nil {
		return false,
			logableError{
				fmt.Errorf("couldn't parse echo payload: %w", err),
				nil,
				"",
			}
	}

	rnd, err := getRound(parsed)
	if err != nil {
		return false,
			logableError{
				fmt.Errorf("couldn't extract round from echo: %w", err),
				parsed.WireMsg().GetTrackingID(),
				"",
			}
	}

	// according to gg18 (tss ecdsa paper), unicasts are sent in these rounds.
	if rnd == round1Message1 || rnd == round2Message {
		return false,
			logableError{
				errBadRoundsInEcho,
				parsed.WireMsg().GetTrackingID(),
				rnd,
			}
	}

	shouldEcho, shouldDeliver, err := t.relbroadcastInspection(parsed, m)
	if err != nil {
		return false,
			logableError{
				fmt.Errorf("reliable broadcast inspection issue: %w", err),
				parsed.WireMsg().GetTrackingID(),
				rnd,
			}
	}

	if !shouldDeliver {
		return shouldEcho, nil
	}

	if err := t.fp.Update(parsed); err != nil {
		return shouldEcho, logableError{
			fmt.Errorf("failed to update the full party: %w", err),
			parsed.WireMsg().GetTrackingID(),
			rnd,
		}
	}

	return shouldEcho, nil
}

var errUnicastBadRound = fmt.Errorf("bad round for unicast (can accept round1Message1 and round2Message)")

func (t *Engine) handleUnicast(m Incoming) error {
	parsed, err := t.parseUnicast(m)
	if err != nil {
		return logableError{fmt.Errorf("couldn't parse unicast payload: %w", err), nil, ""}
	}

	// ensuring the reported source of the message matches the claimed source. (parsed.GetFrom() used by the tss-lib)
	if !equalPartyIds(parsed.GetFrom(), protoToPartyId(m.GetSource())) {
		return logableError{
			fmt.Errorf("parsed message sender doesn't match the source of the message"),
			parsed.WireMsg().GetTrackingID(),
			"",
		}
	}

	rnd, err := getRound(parsed)
	if err != nil {
		return logableError{
			fmt.Errorf("unicast parsing error: %w", err),
			parsed.WireMsg().GetTrackingID(),
			"",
		}
	}

	// only round 1 and round 2 are unicasts.
	if rnd != round1Message1 && rnd != round2Message {
		return logableError{
			errUnicastBadRound,
			parsed.WireMsg().GetTrackingID(),
			rnd,
		}
	}

	err = t.validateUnicastDoesntExist(parsed)
	if err == errUnicastAlreadyReceived {
		return nil
	}

	if err != nil {
		return logableError{
			fmt.Errorf("failed to ensure no equivication present in unicast: %w, sender:%v", err, m.GetSource().Id),
			parsed.WireMsg().GetTrackingID(),
			rnd,
		}
	}

	if err := t.fp.Update(parsed); err != nil {
		return logableError{
			fmt.Errorf("unicast failed to update the full party: %w", err),
			parsed.WireMsg().GetTrackingID(),
			rnd,
		}
	}

	return nil
}

var errUnicastAlreadyReceived = fmt.Errorf("unicast already received")

func (t *Engine) validateUnicastDoesntExist(parsed tss.ParsedMessage) error {
	id, err := t.getMessageUUID(parsed)
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

func (t *Engine) parseEcho(m Incoming) (tss.ParsedMessage, error) {
	echoMsg := m.toEcho()
	if err := vaidateEchoCorrectForm(echoMsg); err != nil {
		return nil, err
	}

	senderPid := protoToPartyId(echoMsg.Message.Sender)
	if !t.GuardianStorage.contains(senderPid) {
		return nil, fmt.Errorf("%w: %v", ErrUnkownSender, senderPid)
	}

	return tss.ParseWireMessage(echoMsg.Message.Content.Payload, senderPid, true)
}

// SECURITY NOTE: this function sets a sessionID to a message. Used to ensure no equivocation.
//
// We don't add the content of the message to the uuid, instead we collect all data that can put this message in a context.
// this is used by the reliable broadcast to check no two messages from the same sender will be used to update the full party
// in the same round for the specific session of the protocol.
func (t *Engine) getMessageUUID(msg tss.ParsedMessage) (digest, error) {
	// The TackingID of a parsed message is tied to the run of the protocol for a single
	//  signature, thus we use it as a sessionID.
	messageTrackingID := [trackingIDSize]byte{}
	copy(messageTrackingID[:], msg.WireMsg().GetTrackingID())

	fromId := [hostnameSize]byte{}
	copy(fromId[:], msg.GetFrom().Id)

	fromKey := [pemKeySize]byte{}
	copy(fromKey[:], msg.GetFrom().Key)

	// Adding the round allows the same sender to send messages for different rounds.
	// but, sender j is not allowed to send two different messages to the same round.
	rnd, err := getRound(msg)
	if err != nil {
		return digest{}, err
	}

	round := [signingRoundSize]byte{}
	copy(round[:], rnd)

	d := append([]byte("tssMsgUUID:"), t.GuardianStorage.LoadDistributionKey...)
	d = append(d, messageTrackingID[:]...)
	d = append(d, fromId[:]...)
	d = append(d, fromKey[:]...)
	d = append(d, round[:]...)

	return hash(d), nil
}

func (t *Engine) parseUnicast(m Incoming) (tss.ParsedMessage, error) {
	if err := validateContentCorrectForm(m.toUnicast()); err != nil {
		return nil, err
	}

	return tss.ParseWireMessage(m.toUnicast().Payload, protoToPartyId(m.GetSource()), false)
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
