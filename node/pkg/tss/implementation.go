package tss

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strconv"

	"sync"
	"sync/atomic"
	"time"

	whcommon "github.com/certusone/wormhole/node/pkg/common"
	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/tss/internal"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/xlabs/tss-lib/v2/common"
	tssutil "github.com/xlabs/tss-lib/v2/ecdsa/ethereum"
	"github.com/xlabs/tss-lib/v2/ecdsa/keygen"
	"github.com/xlabs/tss-lib/v2/ecdsa/party"
	"github.com/xlabs/tss-lib/v2/tss"
	"go.uber.org/zap"
)

type uuid digest // distinguishing between types to avoid confusion.

// Engine is the implementation of reliableTSS, it is a wrapper for the
// tss-lib fullParty and adds  hash-broadcast logic
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

	// used to perform  hash-broadcast:
	mtx      *sync.Mutex
	received map[uuid]*broadcaststate

	sigCounter activeSigCounter

	// informs a central tracker of the guardian's actions.
	ftCommandChan chan ftCommand

	SignatureMetrics sync.Map

	gst *whcommon.GuardianSetState
}

type PEM []byte

// Contains the TSS related configurations.
type Configurations struct {
	maxSimultaneousSignatures int
	// MaxSignerTTL is the maximum time a signature is allowed to be active.
	// used to release resources.
	MaxSignerTTL time.Duration

	ChainsWithNoSelfReport []uint16

	// LeaderIdentity is used by the TSS engine protocol to determine who is responsible for telling
	// the other guardians about a new VAAv1.
	LeaderIdentity PEM // The public key of the leader in PEM format.
}

type Identity struct {
	Pid     *tss.PartyID // used for tss protocol.
	KeyPEM  PEM
	Key     *ecdsa.PublicKey `json:"-"` // ensuring this isn't stored in non-pem format.
	CertPem PEM
	Cert    *x509.Certificate `json:"-"` // ensuring this isn't stored in non-pem format.

	CommunicationIndex SenderIndex // the number representing the guardian when passing messages.
	Hostname           string
	Port               int    // the port the guardian is listening on. if 0 -> use the default port.
	networkname        string // the combination of the two.
}

func (id *Identity) Copy() *Identity {
	keypem := make([]byte, len(id.KeyPEM))
	copy(keypem, id.KeyPEM)

	certPem := make([]byte, len(id.CertPem))
	copy(certPem, id.CertPem)

	c, k, _ := extractCertAndKeyFromPem(certPem)
	cpy := &Identity{
		Pid:                id.getPidCopy(),
		KeyPEM:             keypem,
		CertPem:            certPem,
		CommunicationIndex: id.CommunicationIndex,
		Hostname:           id.Hostname,
		Port:               id.Port,
		Key:                k,
		Cert:               c,
		networkname:        id.networkname,
	}

	return cpy
}

func (id *Identity) NetworkName() string {
	if id.networkname != "" {
		return id.networkname
	}

	return id.portAndHostToNetName()
}

func (id *Identity) portAndHostToNetName() string {
	var port string
	if id.Port <= 0 || id.Port > (1<<16) {
		port = DefaultPort
	} else {
		port = strconv.Itoa(id.Port)
	}

	return net.JoinHostPort(id.Hostname, port)
}

func (id *Identity) getPidCopy() *tss.PartyID {
	keyCpy := make([]byte, len(id.Pid.Key))
	copy(keyCpy, id.Pid.Key)

	// return a copy, tss-lib might modify this object.
	return &tss.PartyID{
		MessageWrapper_PartyID: &tss.MessageWrapper_PartyID{
			Id:      id.Pid.Id,
			Moniker: id.Pid.Moniker,
			Key:     keyCpy,
		},

		Index: id.Pid.Index,
	}
}

type Identities struct {
	Identities []*Identity

	// maps and slices to ensure quick lookups.
	indexToIdendity  map[SenderIndex]int
	pemkeyToGuardian map[string]int
	peerCerts        []*x509.Certificate // avoid
	partyIds         []*tss.PartyID
}

func (i Identities) Len() int {
	return len(i.Identities)
}

// GuardianStorage is a struct that holds the data needed for a guardian to participate in the TSS protocol
// including its signing key, and the shared symmetric keys with other guardians.
// should be loaded from a file.
type GuardianStorage struct {
	Configurations

	Self *Identity

	// should be a certificate generated with SecretKey
	TlsX509    PEM
	PrivateKey PEM
	tlsCert    *tls.Certificate
	signingKey *ecdsa.PrivateKey // should be the unmarshalled value of PriavteKey.

	// Stored sorted by Key. include Self.
	// Guardians []*tss.PartyID
	Guardians Identities

	// Assumes threshold = 2f+1, where f is the maximal expected number of faulty nodes.
	Threshold int

	// all secret keys should be generated with specific value.
	SavedSecretParameters *keygen.LocalPartySaveData

	LoadDistributionKey []byte

	isleader bool
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

func (st *GuardianStorage) fetchPartyIdFromBytes(pk []byte) *Identity {
	pos, ok := st.Guardians.pemkeyToGuardian[string(pk)]
	if !ok {
		return nil
	}

	return st.Guardians.Identities[pos]
}

// FetchPartyId implements ReliableTSS.
func (st *GuardianStorage) FetchPartyId(cert *x509.Certificate) (*Identity, error) {
	var id *Identity

	switch key := cert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		publicKeyPem, err := internal.PublicKeyToPem(key)
		if err != nil {
			return nil, err
		}

		id = st.fetchPartyIdFromBytes(publicKeyPem)
	case []byte:
		id = st.fetchPartyIdFromBytes(key)
	default:
		return nil, fmt.Errorf("unsupported public key type")
	}

	if id == nil {
		return nil, fmt.Errorf("certificate owner is unknown")
	}

	return id, nil
}

// GetCertificate implements ReliableTSS.
func (st *GuardianStorage) GetCertificate() *tls.Certificate {
	return st.tlsCert
}

// GetPeers implements ReliableTSS.
func (st *GuardianStorage) GetPeers() []*x509.Certificate {
	return st.Guardians.peerCerts
}

var (
	errNilTssEngine        = fmt.Errorf("tss engine is nil")
	errTssEngineNotStarted = fmt.Errorf("tss engine hasn't started")
)

// BeginAsyncThresholdSigningProtocol used to start the TSS protocol over a specific msg.

func (t *Engine) BeginAsyncThresholdSigningProtocol(vaaDigest []byte, chainID vaa.ChainID, consistencyLvl uint8) error {
	return t.beginTSSSign(vaaDigest, chainID, consistencyLvl, signingMeta{})
}

type signingMeta struct {
	isFromVaav1 bool
	isRetry     bool
}

func (t *Engine) beginTSSSign(vaaDigest []byte, chainID vaa.ChainID, consistencyLvl uint8, mt signingMeta) error {
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

	if err := t.prepareThenAnounceNewDigest(d, chainID, consistencyLvl, mt); err != nil {
		return err
	}

	sigPrepInfo, err := t.getSigPrepInfo(chainID, d)
	if err != nil {
		return err
	}

	t.logger.Info("signature for VAA requested",
		zap.String("digest", fmt.Sprintf("%x", vaaDigest)),
		zap.String("chainID", chainID.String()),
		zap.Uint8("consistency", consistencyLvl),
		zap.Bool("isFromVaav1", mt.isFromVaav1),
		zap.Bool("isRetry", mt.isRetry),
		zap.Int("numMatchingTrackIDS", len(sigPrepInfo.alreadyStartedSigningTrackingIDs)),
	)

	t.createSignatureMetrics(vaaDigest, chainID)

	// TODO: Once we can map between the signers of VAAv1 and the TSS party, we can
	// select a specific committee by stating that anyone who hasn’t signed VAAv1 is a faultier.
	sigTask := makeSigningRequest(d, nil, chainID)

	info, err := t.fp.GetSigningInfo(sigTask)
	if err != nil {
		return fmt.Errorf("couldnt generate signing task: %w", err)
	}

	if sigPrepInfo.alreadyStartedSigningTrackingIDs[trackidStr(info.TrackingID.ToString())] {
		return nil // skipping signing.
	}

	// TODO: cosider not recomputing the info, and just used it from `t.fp.GetSigningInfo(sigTask)`
	info, err = t.fp.AsyncRequestNewSignature(sigTask)

	if err != nil {
		return err
	}

	flds := []zap.Field{
		zap.String("trackingID", info.TrackingID.ToString()),
		zap.String("ChainID", chainID.String()),
		zap.Any("committee", getCommitteeIDs(info.SigningCommittee)),
	}

	t.logger.Info(
		"guardian started signing protocol",
		flds...,
	)

	scmd := signCommand{SigningInfo: info, passedToFP: true, signingMeta: mt, digestconsistancy: consistencyLvl}
	if err := intoChannelOrDone[ftCommand](t.ctx, t.ftCommandChan, &scmd); err != nil {
		t.logger.Error("couldn't inform the tracker of the signature start",
			zap.Error(err),
			zap.String("trackingID", info.TrackingID.ToString()),
		)

		return err
	}

	return nil
}

func (t *Engine) SetGuardianSetState(gss *whcommon.GuardianSetState) error {
	if gss == nil {
		return fmt.Errorf("guardian set state is nil")
	}

	if t == nil {
		return errNilTssEngine
	}

	if t.started.Load() != notStarted {
		return fmt.Errorf("tss engine has started, and cannot receive new guardian set state")
	}

	t.gst = gss

	return nil
}

func (t *Engine) getSigPrepInfo(chainID vaa.ChainID, d party.Digest) (sigPreparationInfo, error) {
	cmd := prepareToSignCommand{
		ChainID: chainID,
		Digest:  d,
		reply:   make(chan sigPreparationInfo, 1),
	}

	if err := intoChannelOrDone[ftCommand](t.ctx, t.ftCommandChan, &cmd); err != nil {
		return sigPreparationInfo{}, fmt.Errorf("failed to request for inactive guardians: %w", err)
	}

	// waiting for the reply.
	sigPrepInfo, err := outOfChannelOrDone(t.ctx, cmd.reply)
	if err != nil {
		return sigPreparationInfo{}, fmt.Errorf("failed to get inactive guardians: %w", err)
	}

	return sigPrepInfo, nil
}

// prepareThenAnounceNewDigest updates the inner state of the engine before announcing to others about a new digest seen.
func (t *Engine) prepareThenAnounceNewDigest(d party.Digest, chainID vaa.ChainID, consistencyLvl uint8, mt signingMeta) error {
	signinginfo, err := t.fp.GetSigningInfo(party.SigningTask{
		Digest:       d,
		Faulties:     []*tss.PartyID{}, // no faulties
		AuxilaryData: chainIDToBytes(chainID),
	})

	if err != nil {
		return fmt.Errorf("couldnt generate signing task: %w", err)
	}

	sgCmd := &signCommand{
		SigningInfo:       signinginfo,
		passedToFP:        false, // set to true only after FP actually received the message.
		digestconsistancy: consistencyLvl,
		signingMeta:       mt,
	}

	if err := intoChannelOrDone[ftCommand](t.ctx, t.ftCommandChan, sgCmd); err != nil {
		return fmt.Errorf("couldn't inform the tracker of the signature start: %w", err)
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

	if storage.maxSimultaneousSignatures == 0 {
		storage.maxSimultaneousSignatures = defaultMaxLiveSignatures
	}

	if bytes.Equal(storage.Self.CertPem, storage.LeaderIdentity) {
		storage.isleader = true
	}

	fpParams := &party.Parameters{
		SavedSecrets:         storage.SavedSecretParameters,
		PartyIDs:             storage.Guardians.partyIds,
		Self:                 storage.Self.Pid,
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
		(numBroadcastsPerSignature + numUnicastsRounds*storage.Guardians.Len()) * 2 // times 2 to stay on the safe side.
	t := &Engine{
		ctx: nil,

		logger:          &zap.Logger{},
		GuardianStorage: *storage,

		fpParams:     fpParams,
		fp:           fp,
		fpOutChan:    make(chan tss.Message, expectedMsgs),
		fpSigOutChan: make(chan *common.SignatureData, storage.maxSimultaneousSignatures),
		sigOutChan:   make(chan *common.SignatureData, storage.maxSimultaneousSignatures),

		fpErrChannel:    make(chan *tss.Error, storage.maxSimultaneousSignatures),
		messageOutChan:  make(chan Sendable, expectedMsgs),
		msgSerialNumber: 0,
		mtx:             &sync.Mutex{},
		received:        map[uuid]*broadcaststate{},

		started: atomic.Uint32{}, // default value is 0

		sigCounter: newSigCounter(),

		ftCommandChan: make(chan ftCommand, expectedMsgs),
	}

	return t, nil
}

func (t *Engine) MaxTTL() time.Duration {
	return t.GuardianStorage.maxSignerTTL()
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
		With(zap.String("ID", t.GuardianStorage.Self.Pid.Id)).
		Named("tss")

	if err := t.fp.Start(t.fpOutChan, t.fpSigOutChan, t.fpErrChannel); err != nil {
		t.started.Store(notStarted)

		return err
	}

	// closing the t.fp.start inside th listener
	go t.fpListener()

	go t.sigTracker()

	leaderIdentityPos, ok := t.Guardians.pemkeyToGuardian[string(t.LeaderIdentity)]
	if !ok {
		return fmt.Errorf("leader identity not found in guardian storage")
	}

	leaderIdentity := t.Guardians.Identities[leaderIdentityPos]

	t.logger.Info(
		"tss engine started",
		zap.Any("configs", t.GuardianStorage.Configurations),
		zap.Bool("hasGuardianSet", t.gst != nil),
		zap.String("leaderID", leaderIdentity.Hostname),
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
// IMPORTANT: the fpListener should not wait on writing to other channels!
// if the channel is full, the message should be dropped.
func (t *Engine) fpListener() {
	maxTTL := t.MaxTTL()

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

	t.sigCounter.remove(sig.TrackingId)

	select {
	case t.ftCommandChan <- &SigEndCommand{sig.TrackingId}:
	default:
		// This is a warning, since the ftTracker will eventually clean the sigState matching the trackingID.
		t.logger.Warn(
			"couldn't inform the tracker of the signature end",
			zap.String("trackingId", sig.TrackingId.ToString()),
		)
	}

	select {
	case t.sigOutChan <- sig:
	default:
		// if the signature can't be delivered, we can't do much about it.
		t.logger.Error(
			"Couldn't deliver the signature, signature output channel buffer is full",
			zap.String("trackingId", sig.TrackingId.ToString()),
		)
	}

	t.sigMetricDone(sig.TrackingId, false) // false since there were no issues.
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

	select {
	case t.ftCommandChan <- &SigEndCommand{trackid}:
	default:
		t.logger.Error("couldn't inform the tracker of signature end due to error",
			zap.Error(err),
			zap.String("trackingId", trackid.ToString()),
		)
	}

	// if someone sent a message that caused an error -> we don't
	// accept an override to that message, therefore, we can remove it, since it won't change.
	t.sigCounter.remove(trackid)

	logErr(t.logger, &logableError{
		fmt.Errorf("error in signing protocol: %w", err.Cause()),
		trackid,
		intToRound(err.Round()),
	})

	t.sigMetricDone(trackid, true)
}

func (t *Engine) handleFpOutput(m tss.Message) {
	tssMsg, err := t.intoSendable(m)
	if err == nil {

		select {
		case t.messageOutChan <- tssMsg:
		default:
			t.logger.Error("couldn't output tss message, network output channel buffer is full",
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

	keysToBeRemoved := make([]any, 0)

	t.SignatureMetrics.Range(func(k, v any) bool {
		mt, ok := v.(*signatureMetadata)
		if !ok {
			keysToBeRemoved = append(keysToBeRemoved, k)

			return true
		}

		tmp := now.Sub(mt.timeOfCreation)
		if tmp > maxTTL {
			keysToBeRemoved = append(keysToBeRemoved, k)
		}

		return true
	})

	for _, k := range keysToBeRemoved {
		t.SignatureMetrics.Delete(k)
	}

	t.sigCounter.cleanSelf(maxTTL)

	t.mtx.Lock()
	defer t.mtx.Unlock()

	for k, v := range t.received {
		if now.Sub(v.timeReceived) > maxTTL {
			delete(t.received, k)
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
			Sender:    t.Self.CommunicationIndex.toProto(),
			Signature: nil,
		}

		tmp := serializeableMessage{&tssMessageWrapper{m}}

		if err := t.sign(tmp.getUUID(t.LoadDistributionKey), msgToSend); err != nil {
			return nil, err
		}

		sendable = newEcho(msgToSend, t.Guardians.Identities)
	} else {
		recipients := make([]*Identity, 0, len(routing.To))
		for _, pId := range routing.To {
			recipients = append(recipients, t.GuardianStorage.fetchPartyIdFromBytes(pId.Key))
		}

		sendable = &Unicast{
			Unicast: &tsscommv1.Unicast{
				Content: &tsscommv1.Unicast_Tss{
					Tss: content.TssContent,
				},
			},
			Receipients: recipients,
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

	if err := t.handleBroadcast(msg); err != nil {
		return err
	}

	return nil
}

func (t *Engine) sendEchoOut(parsed broadcastMessage, m Incoming) {
	e := m.toBroadcastMsg()

	uuid := parsed.getUUID(t.LoadDistributionKey)
	contentDigest := hashSignedMessage(e.Message)

	content := &tsscommv1.SignedMessage{
		Sender:    e.Message.Sender,
		Signature: e.Message.Signature,
		Content: &tsscommv1.SignedMessage_HashEcho{
			HashEcho: &tsscommv1.HashEcho{
				SessionUuid:          uuid[:],
				OriginalContetDigest: contentDigest[:],
			},
		},
	}

	select {
	case t.messageOutChan <- newEcho(content, t.Guardians.Identities):
	default:
		t.logger.Warn("couldn't echo the message, network output channel buffer is full")
	}
}

var errBadRoundsInBroadcast = fmt.Errorf("cannot receive broadcast for rounds: %v,%v", round1Message1, round2Message)

func (t *Engine) handleBroadcast(m Incoming) error {
	parsed, err := t.parseBroadcast(m)
	if err != nil {
		return err
	}

	shouldEcho, deliverable, err := t.broadcastInspection(parsed, m)
	if err != nil {
		return err
	}

	if shouldEcho {
		t.sendEchoOut(parsed, m)
	}

	if deliverable == nil {
		return nil
	}

	return deliverable.deliver(t)
}

func (t *Engine) feedIncomingToFp(parsed tss.ParsedMessage) error {
	trackId := parsed.WireMsg().TrackingID
	from := parsed.GetFrom()
	maxLiveSignatures := t.GuardianStorage.maxSimultaneousSignatures

	if ok := t.sigCounter.add(trackId, from, maxLiveSignatures); ok {
		return t.fp.Update(parsed)
	}

	tooManySimulSigsErrCntr.Inc()

	return fmt.Errorf("guardian %v has reached the maximum number of simultaneous signatures", from.Id)
}

var errUnicastBadRound = fmt.Errorf("bad round for unicast (can accept round1Message1 and round2Message)")

// handleUnicast is responsible to handle any incoming unicast messages.
func (t *Engine) handleUnicast(m Incoming) error {
	unicast := m.toUnicast()
	if err := validateUnicastCorrectForm(unicast); err != nil {
		return err
	}

	switch v := unicast.Content.(type) {
	case *tsscommv1.Unicast_Vaav1:
		if err := t.handleUnicastVaaV1(v); err != nil {
			return fmt.Errorf("failed to handle unicast vaav1: %w", err)
		}
	case *tsscommv1.Unicast_Tss:
		if err := t.handleUnicastTSS(v, m.GetSource()); err != nil {
			return fmt.Errorf("failed to handle unicast tss message: %w", err)
		}
	default:
		return fmt.Errorf("received unicast with unknown content type: %T", v)
	}

	return nil
}

// handleUnicastTSS is helper function. responsible for handling unicast.TSS messages.
func (t *Engine) handleUnicastTSS(v *tsscommv1.Unicast_Tss, src *Identity) error {
	fpmsg, err := t.parseTssContent(v.Tss, src)
	if err != nil {
		err = fmt.Errorf("couldn't parse unicast_tss payload: %w", err)
		if fpmsg != nil {
			err = fpmsg.wrapError(err)
		}

		return err
	}

	if err = t.validateUnicastDoesntExist(fpmsg); err == errUnicastAlreadyReceived {
		return nil
	}

	if err != nil {
		return fpmsg.wrapError(fmt.Errorf("failed to ensure no equivication present in unicast: %w, sender:%v", err, src.Hostname))
	}

	if err := t.feedIncomingToFp(fpmsg); err != nil {
		return fpmsg.wrapError(fmt.Errorf("unicast failed to update the full party: %w", err))
	}

	return nil
}

var errUnicastAlreadyReceived = fmt.Errorf("unicast already received")

func (t *Engine) validateUnicastDoesntExist(parsed tss.ParsedMessage) error {
	tmp := serializeableMessage{&tssMessageWrapper{parsed}}
	id := tmp.getUUID(t.LoadDistributionKey)

	bts, _, err := parsed.WireBytes()
	if err != nil {
		return fmt.Errorf("failed storing the unicast: %w", err)
	}

	msgDigest := hash(bts)

	t.mtx.Lock()
	defer t.mtx.Unlock()

	if stored, ok := t.received[id]; ok {
		if stored.verifiedDigest == nil {
			return fmt.Errorf("internal error. Unicast stored without verified hash")
		}

		if *stored.verifiedDigest != msgDigest {
			return fmt.Errorf("%w. (duration from prev unicast %v)", ErrEquivicatingGuardian, time.Since(stored.timeReceived))
		}

		return errUnicastAlreadyReceived
	}

	t.received[id] = &broadcaststate{
		timeReceived:   time.Now(), // used for GC.
		verifiedDigest: &msgDigest, // used to ensure no equivocation.
		votes:          nil,        // no votes should be stored for a unicast.
		echoedAlready:  true,       // ensuring this never echoed since it is a unicast.
		mtx:            nil,        // no need to lock this, just store it.
	}

	return nil
}

var (
	ErrUnkownEchoer = fmt.Errorf("echoer is not a known guardian")
	ErrUnkownSender = fmt.Errorf("sender is not a known guardian")
)

func (st *GuardianStorage) sign(uuid uuid, msg *tsscommv1.SignedMessage) error {
	tmp := hashSignedMessage(msg)
	digest := hash(append(uuid[:], tmp[:]...))

	sig, err := st.signingKey.Sign(rand.Reader, digest[:], nil)
	msg.Signature = sig

	return err
}

var ErrInvalidSignature = fmt.Errorf("invalid signature")

var errEmptySignature = fmt.Errorf("empty signature")

func (st *GuardianStorage) verifySignedMessage(uid uuid, msg *tsscommv1.SignedMessage) error {
	if msg == nil {
		return fmt.Errorf("nil signed message")
	}

	if msg.Signature == nil {
		return errEmptySignature
	}

	cert, err := st.fetchCertificate(SenderIndex(msg.Sender))
	if err != nil {
		return err
	}

	pk, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("certificated stored with non-ecdsa public key, guardian storage is corrupted")
	}

	tmp := hashSignedMessage(msg)
	digest := hash(append(uid[:], tmp[:]...))

	isValid := ecdsa.VerifyASN1(pk, digest[:], msg.Signature)

	if !isValid {
		return ErrInvalidSignature
	}

	return nil
}
