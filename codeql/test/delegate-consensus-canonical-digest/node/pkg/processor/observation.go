package processor

import (
	"encoding/hex"
	"fmt"

	whcommon "github.com/wormhole-foundation/wormhole/node/pkg/common"
	whcrypto "github.com/wormhole-foundation/wormhole/node/pkg/crypto"
	gproto "github.com/wormhole-foundation/wormhole/node/pkg/proto"
)

type DelegateObservation struct {
	IsReobservation bool
	Unreliable      bool
	TxID            []byte
	TxHash          []byte
	GuardianAddr    whcommon.Address
	GuardianAddress whcommon.Address
	Signature       []byte
	Signatures      [][]byte
}

type delegateObservationState struct {
	observations map[string]*DelegateObservation
}

type delegateObservationMap map[string]*delegateObservationState

type delegateState struct {
	observations delegateObservationMap
}

type processor struct {
	delegateState *delegateState
	state         *state
}

type state struct {
	signatures map[string]int
}

func delegateObservationToMessagePublication(m *DelegateObservation) (*whcommon.MessagePublication, error) {
	return &whcommon.MessagePublication{TxID: m.TxID, TxHash: m.TxHash, IsReobservation: m.IsReobservation, Unreliable: m.Unreliable}, nil
}

func (p *processor) currentSignedDelegateEntryPoint(m DelegateObservation) {
	mp, _ := delegateObservationToMessagePublication(&m)
	hash := mp.CreateDigest()
	_ = p.delegateState.observations[hash]
}

func (p *processor) currentCanonicalDelegateEntryPoint(mp *whcommon.MessagePublication) {
	hash := mp.CreateDigest()
	s := p.delegateState.observations[hash]
	if s == nil {
		p.delegateState.observations[hash] = &delegateObservationState{}
	}
}

func (p *processor) equivalentCreateVAASigningDigest(mp *whcommon.MessagePublication) {
	v := mp.CreateVAA(17)
	hash := hex.EncodeToString(v.SigningDigest().Bytes())
	_ = p.delegateState.observations[hash]
}

func delegateConsensusDigest(mp *whcommon.MessagePublication) string {
	return mp.CreateDigest()
}

func (p *processor) thinCompliantHelper(mp *whcommon.MessagePublication) {
	hash := delegateConsensusDigest(mp)
	_ = p.delegateState.observations[hash]
}

type unrelatedDigestBuilder struct{}

func (unrelatedDigestBuilder) SigningDigest() whcommon.Hash { return whcommon.Hash{} }

func (p *processor) unrelatedSameNameSigningDigestNearMiss(b unrelatedDigestBuilder) {
	hash := hex.EncodeToString(b.SigningDigest().Bytes())
	_ = p.delegateState.observations[hash]
}

func (p *processor) historicalMarshalBinaryKeccakShape(mp *whcommon.MessagePublication) {
	buf, _ := mp.MarshalBinary()
	hash := whcrypto.Keccak256Hash(buf).Hex()
	_ = p.delegateState.observations[hash]
}

func (p *processor) deprecatedFullMessageMarshal(mp *whcommon.MessagePublication) {
	buf, _ := mp.Marshal()
	hash := whcrypto.Keccak256Hash(buf).Hex()
	_ = p.delegateState.observations[hash]
}

func (p *processor) serializedDelegateObservation(m *DelegateObservation) {
	buf, _ := gproto.Marshal(m)
	hash := whcrypto.Keccak256Hash(buf).Hex()
	_ = p.delegateState.observations[hash]
}

func (p *processor) signedWrapperSignatureConfusion(m *DelegateObservation) {
	hash := hex.EncodeToString(m.Signature)
	_ = p.delegateState.observations[hash]
}

func (p *processor) explicitIsReobservationDiscriminator(mp *whcommon.MessagePublication, m *DelegateObservation) {
	hash := mp.CreateDigest() + fmt.Sprint(m.IsReobservation)
	_ = p.delegateState.observations[hash]
}

func (p *processor) explicitTxIDDiscriminator(mp *whcommon.MessagePublication, m *DelegateObservation) {
	hash := mp.CreateDigest() + hex.EncodeToString(m.TxHash)
	_ = p.delegateState.observations[hash]
}

func (p *processor) guardianAddressDiscriminator(mp *whcommon.MessagePublication, m *DelegateObservation) {
	hash := fmt.Sprintf("%s/%s", mp.CreateDigest(), m.GuardianAddr.Hex())
	_ = p.delegateState.observations[hash]
}

func manualDigestWithNonVAAField(mp *whcommon.MessagePublication, m *DelegateObservation) string {
	return fmt.Sprintf("%d/%d/%x/%t", mp.Timestamp, mp.Sequence, mp.Payload, m.Unreliable)
}

func (p *processor) manualDigestAddsNonVAAField(mp *whcommon.MessagePublication, m *DelegateObservation) {
	hash := manualDigestWithNonVAAField(mp, m)
	_ = p.delegateState.observations[hash]
}

func (p *processor) messageIDStringOmitFields(mp *whcommon.MessagePublication) {
	hash := mp.MessageIDString()
	_ = p.delegateState.observations[hash]
}

func (p *processor) fallbackToSerializedDelegate(mp *whcommon.MessagePublication, m *DelegateObservation) {
	hash := mp.CreateDigest()
	if hash == "" {
		buf, _ := gproto.Marshal(m)
		hash = whcrypto.Keccak256Hash(buf).Hex()
	}
	_ = p.delegateState.observations[hash]
}

func (p *processor) aliasOfDelegateMap(mp *whcommon.MessagePublication) {
	buf, _ := mp.MarshalBinary()
	hash := whcrypto.Keccak256Hash(buf).Hex()
	obs := p.delegateState.observations
	_ = obs[hash]
}

func (p *processor) reassignedKeyFromNonVAAField(mp *whcommon.MessagePublication, m *DelegateObservation) {
	hash := mp.CreateDigest()
	hash = hex.EncodeToString(m.TxID)
	_ = p.delegateState.observations[hash]
}

func hashSerializedMP(mp *whcommon.MessagePublication) []byte {
	buf, _ := mp.MarshalBinary()
	return buf
}

func hashBytes(buf []byte) string { return whcrypto.Keccak256Hash(buf).Hex() }

func (p *processor) keyAcrossTwoHelpers(mp *whcommon.MessagePublication) {
	hash := hashBytes(hashSerializedMP(mp))
	_ = p.delegateState.observations[hash]
}

func (d *delegateState) getOrCreate(hash string) *delegateObservationState {
	return d.observations[hash]
}

func (p *processor) wrapperMapTypeCallArg(mp *whcommon.MessagePublication) {
	buf, _ := mp.MarshalBinary()
	hash := whcrypto.Keccak256Hash(buf).Hex()
	_ = p.delegateState.getOrCreate(hash)
}

func (d *delegateState) getOrCreateInternal(mp *whcommon.MessagePublication, m *DelegateObservation) *delegateObservationState {
	buf, _ := gproto.Marshal(m)
	hash := whcrypto.Keccak256Hash(buf).Hex()
	return d.observations[hash]
}

func (p *processor) helperComputesKeyInternally(mp *whcommon.MessagePublication, m *DelegateObservation) {
	_ = p.delegateState.getOrCreateInternal(mp, m)
}

func (p *processor) postBucketTxIDWarning(mp *whcommon.MessagePublication, m *DelegateObservation) {
	s := p.delegateState.observations[mp.CreateDigest()]
	if s != nil && len(m.TxHash) > 0 {
		_ = fmt.Sprintf("warn %x", m.TxHash)
	}
}

func (p *processor) postQuorumDeterministicTxID(s *delegateObservationState) {
	for _, obs := range s.observations {
		_ = obs.TxID
	}
}

func (p *processor) metadataNormalization(mp *whcommon.MessagePublication) {
	mp.NormalizeForDelegateConsensus()
}

func (p *processor) loggingMetricsOnly(mp *whcommon.MessagePublication, m *DelegateObservation) {
	buf, _ := mp.MarshalBinary()
	_ = fmt.Sprintf("%x %x %t", buf, m.TxID, m.IsReobservation)
}

func (p *processor) replayDedupAuthenticationOnly(m *DelegateObservation) {
	buf, _ := gproto.Marshal(m)
	_ = whcrypto.Keccak256Hash(buf).Hex()
}

func (p *processor) ordinaryCanonicalAggregation(mp *whcommon.MessagePublication) {
	v := mp.CreateVAA(0)
	hash := hex.EncodeToString(v.SigningDigest().Bytes())
	p.state.signatures[hash]++
}
