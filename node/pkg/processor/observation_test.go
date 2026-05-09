package processor

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func getVAA() vaa.VAA {
	var payload = []byte{97, 97, 97, 97, 97, 97}
	var governanceEmitter = vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}

	return vaa.VAA{
		Version:          uint8(1),
		GuardianSetIndex: uint32(1),
		Signatures:       nil,
		Timestamp:        time.Unix(0, 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		ConsistencyLevel: uint8(32),
		EmitterChain:     vaa.ChainIDSolana,
		EmitterAddress:   governanceEmitter,
		Payload:          payload,
	}
}

func TestHandleInboundSignedVAAWithQuorum_NilGuardianSet(t *testing.T) {
	testVAA := getVAA()
	marshalVAA, _ := testVAA.Marshal()

	// Stub out the minimum to get processor to dance
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	observedLogger := zap.New(observedZapCore)

	signedVAAWithQuorum := &gossipv1.SignedVAAWithQuorum{Vaa: marshalVAA}
	processor := Processor{}
	processor.logger = observedLogger

	processor.handleInboundSignedVAAWithQuorum(signedVAAWithQuorum)

	// Check to see if we got an error, which we should have,
	// because a `gs` is not defined on processor
	assert.Equal(t, 1, observedLogs.Len())
	firstLog := observedLogs.All()[0]
	errorString := "dropping SignedVAAWithQuorum message since we haven't initialized our guardian set yet"
	assert.Equal(t, errorString, firstLog.Message)
}

func TestHandleInboundSignedVAAWithQuorum(t *testing.T) {
	goodPrivateKey1, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	goodAddr1 := crypto.PubkeyToAddress(goodPrivateKey1.PublicKey)
	badPrivateKey1, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)

	tests := []struct {
		label      string
		keyOrder   []*ecdsa.PrivateKey
		indexOrder []uint8
		addrs      []ethcommon.Address
		errString  string
	}{
		{label: "GuardianSetNoKeys", keyOrder: []*ecdsa.PrivateKey{}, indexOrder: []uint8{}, addrs: []ethcommon.Address{},
			errString: "dropping SignedVAAWithQuorum message since we have a guardian set without keys"},
		{label: "VAANoSignatures", keyOrder: []*ecdsa.PrivateKey{}, indexOrder: []uint8{0}, addrs: []ethcommon.Address{goodAddr1},
			errString: "dropping SignedVAAWithQuorum message because it failed verification: VAA was not signed"},
		{label: "VAAInvalidSignatures", keyOrder: []*ecdsa.PrivateKey{badPrivateKey1}, indexOrder: []uint8{0}, addrs: []ethcommon.Address{goodAddr1},
			errString: "dropping SignedVAAWithQuorum message because it failed verification: VAA had bad signatures"},
		{label: "DuplicateGoodSignaturesNonMonotonic", keyOrder: []*ecdsa.PrivateKey{goodPrivateKey1, goodPrivateKey1, goodPrivateKey1, goodPrivateKey1}, indexOrder: []uint8{0, 0, 0, 0}, addrs: []ethcommon.Address{goodAddr1},
			errString: "dropping SignedVAAWithQuorum message because it failed verification: VAA had bad signatures"},
		{label: "DuplicateGoodSignaturesMonotonic", keyOrder: []*ecdsa.PrivateKey{goodPrivateKey1, goodPrivateKey1, goodPrivateKey1, goodPrivateKey1}, indexOrder: []uint8{0, 1, 2, 3}, addrs: []ethcommon.Address{goodAddr1},
			errString: "dropping SignedVAAWithQuorum message because it failed verification: VAA had bad signatures"},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			testVAA := getVAA()

			// Define a GuardianSet from test addrs
			guardianSet := common.GuardianSet{
				Keys:  tc.addrs,
				Index: 1,
			}

			// Sign with the keys at the proper index
			for i, key := range tc.keyOrder {
				testVAA.AddSignature(key, tc.indexOrder[i])
			}

			marshalVAA, err := testVAA.Marshal()
			if err != nil {
				panic(err)
			}

			// Stub out the minimum to get processor to dance
			observedZapCore, observedLogs := observer.New(zap.InfoLevel)
			observedLogger := zap.New(observedZapCore)

			signedVAAWithQuorum := &gossipv1.SignedVAAWithQuorum{Vaa: marshalVAA}
			processor := Processor{}
			processor.gs = &guardianSet
			processor.logger = observedLogger

			processor.handleInboundSignedVAAWithQuorum(signedVAAWithQuorum)

			// Check to see if we got an error, which we should have
			assert.Equal(t, 1, observedLogs.Len())
			firstLog := observedLogs.All()[0]
			assert.Equal(t, tc.errString, firstLog.Message)
		})
	}
}

// makeDelegateObs builds a minimal DelegateObservation for the same VAA
// (chain/emitter/sequence/payload), parameterized by the per-guardian fields
// that don't affect the VAA digest: signer address, TxHash, IsReobservation.
func makeDelegateObs(t *testing.T, signer ethcommon.Address, txHash []byte, isReobservation bool) *gossipv1.DelegateObservation {
	t.Helper()
	emitter, err := vaa.StringToAddress("000000000000000000000000b1731c586ca89a23809861c6103f0b96b3f57d92")
	if err != nil {
		t.Fatal(err)
	}
	return &gossipv1.DelegateObservation{
		EmitterChain:      uint32(vaa.ChainIDMoonbeam),
		EmitterAddress:    emitter[:],
		Sequence:          95838,
		TxHash:            txHash,
		Timestamp:         1746026862,
		Nonce:             0,
		ConsistencyLevel:  1,
		Payload:           []byte("payload"),
		GuardianAddr:      signer.Bytes(),
		IsReobservation:   isReobservation,
		Unreliable:        false,
		VerificationState: 0,
	}
}

// TestHandleCanonicalDelegateObservation_BucketMergesAcrossIsReobservation feeds
// a delegated guardian set's worth of observations split across IsReobservation
// values and verifies that:
//   - All observations land in the SAME bucket (the bucket-key fix at
//     observation.go:CreateDigest is invariant to IsReobservation).
//   - No "delegate TxID disagreement" warning is emitted, because every
//     observation carries the same TxID.
func TestHandleCanonicalDelegateObservation_BucketMergesAcrossIsReobservation(t *testing.T) {
	signers := []ethcommon.Address{
		ethcommon.HexToAddress("0x000ac0076727b35fbea2dac28fee5ccb0fea768e"),
		ethcommon.HexToAddress("0x178e21ad2e77ae06711549cfbb1f9c7a9d8096e8"),
		ethcommon.HexToAddress("0xda798f6896a3331f64b48c12d1d57fd9cbe70811"),
		ethcommon.HexToAddress("0x938f104aeb5581293216ce97d771e0cb721221b1"),
		ethcommon.HexToAddress("0x42579bffbcf4276e290ab8e4c162bd4052b97970"),
	}
	cfg, err := NewDelegatedGuardianChainConfig(signers, vaa.CalculateQuorum(len(signers)))
	if err != nil {
		t.Fatal(err)
	}

	observedCore, observedLogs := observer.New(zap.InfoLevel)
	p := &Processor{
		logger:        zap.New(observedCore),
		delegateState: &delegateAggregationState{delegateObservationMap{}},
	}

	txID := ethcommon.HexToHash("0x39c2f7f67fbce903d49bb24147668095f1b726acef3c19460da39e83c6929a2b").Bytes()
	// Mix IsReobservation values across the set; under the old bucket-key this
	// would split the signatures across buckets and prevent quorum.
	flags := []bool{false, false, true, true, true}
	for i, signer := range signers {
		obs := makeDelegateObs(t, signer, txID, flags[i])
		// We don't drive checkForDelegateQuorum's downstream call here (that
		// would require a full Processor); we stop after the bucket update.
		// To do that, mark submitted on the bucket once it exists so the
		// quorum-triggered downstream path is short-circuited.
		_ = p.handleCanonicalDelegateObservation(t.Context(), cfg, obs)
		if i == 0 {
			// After the first observation, force submitted=true so subsequent
			// calls return early without invoking the consensus handler.
			for _, s := range p.delegateState.observations {
				s.submitted = true
			}
		}
	}

	// Exactly one bucket — the IsReobservation split is gone.
	assert.Len(t, p.delegateState.observations, 1,
		"all delegate observations must land in a single bucket regardless of IsReobservation")

	for _, s := range p.delegateState.observations {
		assert.Len(t, s.observations, len(signers),
			"every signer's observation should be in the bucket")
	}

	// No TxID-disagreement warnings — every signer used the same TxID.
	for _, entry := range observedLogs.All() {
		assert.NotEqual(t, "delegate TxID disagreement", entry.Message)
	}
}

// TestConsensusTxID covers the per-bucket majority-TxID picker:
//   - all observations agree → that TxID wins
//   - simple majority → majority wins
//   - tie → lex-smallest TxID wins (independent of cfg, delegate-set updates,
//     and map iteration order)
//   - empty bucket → nil
func TestConsensusTxID(t *testing.T) {
	signers := []ethcommon.Address{
		ethcommon.HexToAddress("0x000ac0076727b35fbea2dac28fee5ccb0fea768e"),
		ethcommon.HexToAddress("0x178e21ad2e77ae06711549cfbb1f9c7a9d8096e8"),
		ethcommon.HexToAddress("0xda798f6896a3331f64b48c12d1d57fd9cbe70811"),
		ethcommon.HexToAddress("0x938f104aeb5581293216ce97d771e0cb721221b1"),
		ethcommon.HexToAddress("0x42579bffbcf4276e290ab8e4c162bd4052b97970"),
	}
	txA := ethcommon.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa").Bytes()
	txB := ethcommon.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb").Bytes()

	t.Run("unanimous", func(t *testing.T) {
		s := &delegateState{observations: map[ethcommon.Address]*gossipv1.DelegateObservation{}}
		for _, sg := range signers {
			s.observations[sg] = &gossipv1.DelegateObservation{TxHash: txA}
		}
		assert.Equal(t, txA, s.consensusTxID())
	})

	t.Run("majority_wins", func(t *testing.T) {
		s := &delegateState{observations: map[ethcommon.Address]*gossipv1.DelegateObservation{}}
		// 4 sigs txA, 1 sig txB. txB is lex-smaller than txA, but minority loses.
		s.observations[signers[0]] = &gossipv1.DelegateObservation{TxHash: txA}
		s.observations[signers[1]] = &gossipv1.DelegateObservation{TxHash: txA}
		s.observations[signers[2]] = &gossipv1.DelegateObservation{TxHash: txA}
		s.observations[signers[3]] = &gossipv1.DelegateObservation{TxHash: txA}
		s.observations[signers[4]] = &gossipv1.DelegateObservation{TxHash: txB}
		assert.Equal(t, txA, s.consensusTxID())
	})

	t.Run("tie_lex_smaller_wins", func(t *testing.T) {
		// 2 sigs each — txA (0xaa…) is lex-smaller than txB (0xbb…), so txA wins.
		// Repeated to defeat map-iteration randomness.
		for i := 0; i < 50; i++ {
			s := &delegateState{observations: map[ethcommon.Address]*gossipv1.DelegateObservation{}}
			s.observations[signers[0]] = &gossipv1.DelegateObservation{TxHash: txB}
			s.observations[signers[1]] = &gossipv1.DelegateObservation{TxHash: txA}
			s.observations[signers[2]] = &gossipv1.DelegateObservation{TxHash: txB}
			s.observations[signers[3]] = &gossipv1.DelegateObservation{TxHash: txA}
			assert.Equal(t, txA, s.consensusTxID())
		}
	})

	t.Run("tie_independent_of_signer_order", func(t *testing.T) {
		// Swapping which signers carry which TxID must not change the result —
		// the tie-break is on TxID bytes, not signer position.
		for i := 0; i < 50; i++ {
			s := &delegateState{observations: map[ethcommon.Address]*gossipv1.DelegateObservation{}}
			s.observations[signers[0]] = &gossipv1.DelegateObservation{TxHash: txA}
			s.observations[signers[1]] = &gossipv1.DelegateObservation{TxHash: txB}
			s.observations[signers[2]] = &gossipv1.DelegateObservation{TxHash: txA}
			s.observations[signers[3]] = &gossipv1.DelegateObservation{TxHash: txB}
			assert.Equal(t, txA, s.consensusTxID())
		}
	})

	t.Run("empty_bucket", func(t *testing.T) {
		s := &delegateState{observations: map[ethcommon.Address]*gossipv1.DelegateObservation{}}
		assert.Nil(t, s.consensusTxID())
	})
}

// TestHandleCanonicalDelegateObservation_TxIDDisagreementWarn verifies that
// when two delegates report the same VAA from different transactions, the
// canonical emits the expected warning. TxID is not part of consensus, so the
// observations still land in the same bucket — but the disagreement is logged
// for operators.
func TestHandleCanonicalDelegateObservation_TxIDDisagreementWarn(t *testing.T) {
	signerA := ethcommon.HexToAddress("0x000ac0076727b35fbea2dac28fee5ccb0fea768e")
	signerB := ethcommon.HexToAddress("0x178e21ad2e77ae06711549cfbb1f9c7a9d8096e8")
	cfg, err := NewDelegatedGuardianChainConfig(
		[]ethcommon.Address{signerA, signerB,
			ethcommon.HexToAddress("0xda798f6896a3331f64b48c12d1d57fd9cbe70811")},
		vaa.CalculateQuorum(3),
	)
	if err != nil {
		t.Fatal(err)
	}

	observedCore, observedLogs := observer.New(zap.WarnLevel)
	p := &Processor{
		logger:        zap.New(observedCore),
		delegateState: &delegateAggregationState{delegateObservationMap{}},
	}

	txA := ethcommon.HexToHash("0x39c2f7f67fbce903d49bb24147668095f1b726acef3c19460da39e83c6929a2b").Bytes()
	txB := ethcommon.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa").Bytes()

	if err := p.handleCanonicalDelegateObservation(t.Context(), cfg, makeDelegateObs(t, signerA, txA, false)); err != nil {
		t.Fatalf("first obs returned error: %v", err)
	}
	// Force submitted to short-circuit the quorum path on the second obs.
	for _, s := range p.delegateState.observations {
		s.submitted = true
	}
	if err := p.handleCanonicalDelegateObservation(t.Context(), cfg, makeDelegateObs(t, signerB, txB, false)); err != nil {
		t.Fatalf("second obs returned error: %v", err)
	}

	// Both observations are in the same bucket — TxID is not part of the digest.
	assert.Len(t, p.delegateState.observations, 1,
		"different TxIDs must not split the bucket — TxID is not part of the VAA digest")

	// Exactly one disagreement warning, naming both signers and TxIDs.
	disagreementLogs := observedLogs.FilterMessage("delegate TxID disagreement").All()
	if assert.Len(t, disagreementLogs, 1, "expected one TxID disagreement warning") {
		entry := disagreementLogs[0]
		fields := map[string]string{}
		for _, f := range entry.Context {
			fields[f.Key] = f.String
		}
		assert.Contains(t, fields["txid_a"], "39c2f7f6")
		assert.Contains(t, fields["txid_b"], "aaaaaaaa")
		assert.NotEmpty(t, fields["guardian_a"])
		assert.NotEmpty(t, fields["guardian_b"])
		assert.NotEmpty(t, fields["msgID"])
	}
}
