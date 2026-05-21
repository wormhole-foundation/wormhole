package processor

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	guardianDB "github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	"github.com/certusone/wormhole/node/pkg/gwrelayer"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"

	// gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var benchmarkConsensusTxIDSink []byte

/*
No gwrelayer:
average time to do  1800000  observations:  51.246µs
there were  1100000  under quorum, taking an average time of  62.888µs
there were  100000  quorum reached, taking an average time of  68.049µs
there were  600000  over quorum, taking an average time of  27.101µs
there were  100000  handle message calls, taking an average time of  28.723µs

With gwrelayer (where observations are not relayed):
average time to do  1800000  observations:  51.18µs
there were  1100000  under quorum, taking an average time of  62.713µs
there were  100000  quorum reached, taking an average time of  68.316µs
there were  600000  over quorum, taking an average time of  27.182µs
there were  100000  handle message calls, taking an average time of  28.704µs
*/

// go test -bench ^BenchmarkHandleObservation -benchtime=1x
func BenchmarkHandleObservation(b *testing.B) {
	const NumObservations = 100000
	ctx := context.Background()
	db := guardianDB.OpenDb(nil, nil)
	defer db.Close()
	p, pd := createProcessorForTest(b, NumObservations, ctx, db)
	require.NotNil(b, p)
	require.NotNil(b, pd)

	var totalTime, underQuorumTime, quorumReachedTime, overQuorumTime, handleMsgTime time.Duration
	var totalCount, underQuorumCount, quorumReachedCount, overQuorumCount int
	for count := range NumObservations {
		k := pd.createMessagePublication(b, uint64(count)) // #nosec G115 -- Safe as NumObservations hard coded above
		start := time.Now()
		p.handleMessage(ctx, k)
		handleMsgTime += time.Since(start)

		for guardianIdx := 1; guardianIdx < 19; guardianIdx++ {
			start := time.Now()
			p.handleSingleObservation(pd.guardianAddrs[guardianIdx], pd.createObservation(b, guardianIdx, k))
			duration := time.Since(start)
			totalCount++
			totalTime += duration
			if guardianIdx < 12 {
				underQuorumCount++
				underQuorumTime += duration
			} else if guardianIdx == 12 {
				quorumReachedCount++
				quorumReachedTime += duration
			} else {
				overQuorumCount++
				overQuorumTime += duration
			}
		}
	}
	require.Equal(b, NumObservations, len(pd.gossipVaaSendC))
	// This won't work once batching is enabled.
	// require.Equal(b, NumObservations, len(pd.gossipAttestationSendC))
	fmt.Println("average time to do ", totalCount, " observations: ", totalTime/time.Duration(totalCount))
	fmt.Println("there were ", underQuorumCount, " under quorum, taking an average time of ", underQuorumTime/time.Duration(underQuorumCount))
	fmt.Println("there were ", quorumReachedCount, " quorum reached, taking an average time of ", quorumReachedTime/time.Duration(quorumReachedCount))
	fmt.Println("there were ", overQuorumCount, " over quorum, taking an average time of ", overQuorumTime/time.Duration(overQuorumCount))
	fmt.Println("there were ", NumObservations, " handle message calls, taking an average time of ", handleMsgTime/time.Duration(NumObservations))
}

// go test -bench ^BenchmarkProfileHandleObservation -benchtime=1x
// To view profiling results:
//   go install github.com/google/pprof@latest
//   sudo apt install graphviz
//   pprof -http=:8080 handleObs.prof

func BenchmarkProfileHandleObservation(b *testing.B) {
	// return
	const NumObservations = 100000
	f, err := os.Create("handleObs.prof")
	require.NoError(b, err)
	err = pprof.StartCPUProfile(f)
	require.NoError(b, err)
	defer pprof.StopCPUProfile()

	ctx := context.Background()
	db := guardianDB.OpenDb(nil, nil)
	defer db.Close()
	p, pd := createProcessorForTest(b, NumObservations, ctx, db)
	require.NotNil(b, p)
	require.NotNil(b, pd)

	for count := 0; count < NumObservations; count++ {
		k := pd.createMessagePublication(b, uint64(count)) // #nosec G115 -- Safe as NumObservations hard coded above
		p.handleMessage(ctx, k)

		for guardianIdx := 1; guardianIdx < 19; guardianIdx++ {
			p.handleSingleObservation(pd.guardianAddrs[guardianIdx], pd.createObservation(b, guardianIdx, k))
		}
	}
	require.Equal(b, NumObservations, len(pd.gossipVaaSendC))
}

func BenchmarkCheckForDelegateQuorum(b *testing.B) {
	// Thirteen delegates is a realistic upper bound for delegated guardian sets;
	// the protocol-level hard bound is the 19-member normal guardian set.
	const numDelegates = 13

	signers := make([]ethCommon.Address, 0, numDelegates)
	for i := 1; i <= numDelegates; i++ {
		signers = append(signers, ethCommon.BigToAddress(big.NewInt(int64(i))))
	}

	cfg, err := NewDelegatedGuardianChainConfig(signers, vaa.CalculateQuorum(len(signers)))
	require.NoError(b, err)

	txHash := ethCommon.HexToHash("0x39c2f7f67fbce903d49bb24147668095f1b726acef3c19460da39e83c6929a2b").Bytes()
	bucket := &delegateState{
		firstObserved: time.Now(),
		observations:  make(map[ethCommon.Address]*gossipv1.DelegateObservation, cfg.Quorum()),
		cfg:           cfg,
	}
	for _, signer := range signers[:cfg.Quorum()] {
		bucket.observations[signer] = makeDelegateObs(b, signer, txHash, false)
	}

	emitter, err := vaa.StringToAddress("000000000000000000000000b1731c586ca89a23809861c6103f0b96b3f57d92")
	require.NoError(b, err)
	mp := &common.MessagePublication{
		TxID:             txHash,
		Timestamp:        time.Unix(1746026862, 0),
		Nonce:            0,
		Sequence:         95838,
		ConsistencyLevel: 1,
		EmitterChain:     vaa.ChainIDMoonbeam,
		EmitterAddress:   emitter,
		Payload:          []byte("payload"),
	}

	// Keep the processor deliberately minimal. checkForDelegateQuorum dispatches
	// to the downstream consensus path after quorum, but a nil guardian set makes
	// handleMessage return before signing or VAA aggregation. That keeps this
	// benchmark focused on delegate quorum accounting, TxID selection,
	// normalization, and dispatch overhead.
	p := &Processor{logger: zap.NewNop()}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		bucket.submitted = false
		if err := p.checkForDelegateQuorum(ctx, mp, bucket, cfg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConsensusTxID(b *testing.B) {
	// Match the realistic delegated-set size used by BenchmarkCheckForDelegateQuorum.
	const numDelegates = 13

	signers := make([]ethCommon.Address, 0, numDelegates)
	for i := 1; i <= numDelegates; i++ {
		signers = append(signers, ethCommon.BigToAddress(big.NewInt(int64(i))))
	}

	b.Run("unanimous", func(b *testing.B) {
		txHash := ethCommon.HexToHash("0x39c2f7f67fbce903d49bb24147668095f1b726acef3c19460da39e83c6929a2b").Bytes()
		bucket := &delegateState{observations: make(map[ethCommon.Address]*gossipv1.DelegateObservation, len(signers))}
		for _, signer := range signers {
			bucket.observations[signer] = makeDelegateObs(b, signer, txHash, false)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			benchmarkConsensusTxIDSink = bucket.consensusTxID()
		}
	})

	b.Run("unique", func(b *testing.B) {
		bucket := &delegateState{observations: make(map[ethCommon.Address]*gossipv1.DelegateObservation, len(signers))}
		for i, signer := range signers {
			txHash := ethCommon.BigToHash(big.NewInt(int64(i + 1))).Bytes()
			bucket.observations[signer] = makeDelegateObs(b, signer, txHash, false)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			benchmarkConsensusTxIDSink = bucket.consensusTxID()
		}
	})

	b.Run("majority", func(b *testing.B) {
		txMaj := ethCommon.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb").Bytes()
		txMin := ethCommon.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa").Bytes()
		bucket := &delegateState{observations: make(map[ethCommon.Address]*gossipv1.DelegateObservation, len(signers))}
		for i, signer := range signers {
			txHash := txMin
			if i < 7 {
				txHash = txMaj
			}
			bucket.observations[signer] = makeDelegateObs(b, signer, txHash, false)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			benchmarkConsensusTxIDSink = bucket.consensusTxID()
		}
	})
}

func BenchmarkHandleSignedDelegateObservation(b *testing.B) {
	// Thirteen delegates is a realistic upper bound for delegated guardian sets;
	// this benchmark sends one valid observation, keeping the bucket below quorum
	// so it measures signed-delegate validation, conversion, digesting, and bucket
	// update without downstream consensus publication.
	const numDelegates = 13

	signers := make([]ethCommon.Address, 0, numDelegates)
	for i := 1; i <= numDelegates; i++ {
		signers = append(signers, ethCommon.BigToAddress(big.NewInt(int64(i))))
	}
	cfg, err := NewDelegatedGuardianChainConfig(signers, vaa.CalculateQuorum(len(signers)))
	require.NoError(b, err)

	dgc := NewDelegatedGuardianConfig()
	require.NoError(b, dgc.Set(map[vaa.ChainID]*DelegatedGuardianChainConfig{
		vaa.ChainIDMoonbeam: cfg,
	}))

	txHash := ethCommon.HexToHash("0x39c2f7f67fbce903d49bb24147668095f1b726acef3c19460da39e83c6929a2b").Bytes()
	obs := makeDelegateObs(b, signers[0], txHash, false)
	obsBytes, err := proto.Marshal(obs)
	require.NoError(b, err)
	signedObs := &gossipv1.SignedDelegateObservation{
		DelegateObservation: obsBytes,
		GuardianAddr:        signers[0].Bytes(),
	}

	p := &Processor{
		logger:        zap.NewNop(),
		dgc:           dgc,
		delegateState: &delegateAggregationState{delegateObservationMap{}},
	}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if err := p.handleSignedDelegateObservation(ctx, signedObs); err != nil {
			b.Fatal(err)
		}
	}
}

type ProcessorData struct {
	gossipAttestationSendC chan []byte
	gossipVaaSendC         chan []byte
	emitterChain           vaa.ChainID
	emitterAddress         vaa.Address
	guardianSigners        []guardiansigner.GuardianSigner
	guardianAddrs          [][]byte
}

func (pd *ProcessorData) messageID(seqNum uint64) string {
	return fmt.Sprintf("%d/%s/%d", pd.emitterChain, pd.emitterAddress, seqNum)
}

// createProcessorForTest creates a processor for benchmarking. It assumes we are index zero in the guardian set.
func createProcessorForTest(b *testing.B, numVAAs int, ctx context.Context, db *guardianDB.Database) (*Processor, *ProcessorData) {
	b.Helper()
	logger := zap.NewNop()

	var ourSigner guardiansigner.GuardianSigner
	keys := []ethCommon.Address{}
	guardianSigners := []guardiansigner.GuardianSigner{}
	guardianAddrs := [][]byte{}

	for count := 0; count < 19; count++ {
		guardianSigner, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(nil)
		require.NoError(b, err)
		keys = append(keys, crypto.PubkeyToAddress(guardianSigner.PublicKey(ctx)))
		guardianSigners = append(guardianSigners, guardianSigner)
		guardianAddrs = append(guardianAddrs, crypto.PubkeyToAddress(guardianSigner.PublicKey(ctx)).Bytes())
		if count == 0 {
			ourSigner = guardianSigner
		}
	}

	gs := common.NewGuardianSet(keys, 0)
	gst := common.NewGuardianSetState(nil)
	gst.Set(gs)

	emitterAddress, err := vaa.StringToAddress("0x3ee18B2214AFF97000D974cf647E7C347E8fa585")
	require.NoError(b, err)

	gwRelayer := gwrelayer.NewGatewayRelayer(ctx, logger, "wormhole14ejqjyq8um4p3xfqj74yld5waqljf88fz25yxnma0cngspxe3les00fpj", nil, common.MainNet)
	require.NoError(b, gwRelayer.Start(ctx))

	pd := &ProcessorData{
		gossipAttestationSendC: make(chan []byte, numVAAs+100),
		gossipVaaSendC:         make(chan []byte, numVAAs+100),
		emitterChain:           vaa.ChainIDEthereum,
		emitterAddress:         emitterAddress,
		guardianSigners:        guardianSigners,
		guardianAddrs:          guardianAddrs,
	}

	p := &Processor{
		gossipAttestationSendC: pd.gossipAttestationSendC,
		gossipVaaSendC:         pd.gossipVaaSendC,
		guardianSigner:         ourSigner,
		gs:                     gs,
		gst:                    gst,
		db:                     db,
		logger:                 logger,
		state:                  &aggregationState{observationMap{}},
		ourAddr:                crypto.PubkeyToAddress(ourSigner.PublicKey(context.Background())),
		pythnetVaas:            make(map[string]PythNetVaaEntry),
		updatedVAAs:            make(map[string]*updateVaaEntry),
		gatewayRelayer:         gwRelayer,
	}

	go func() { _ = p.vaaWriter(ctx) }()
	go func() { _ = p.batchProcessor(ctx) }()

	return p, pd
}

func (pd *ProcessorData) createMessagePublication(b *testing.B, sequence uint64) *common.MessagePublication {
	b.Helper()
	return &common.MessagePublication{
		TxID:             ethCommon.HexToHash(fmt.Sprintf("%064x", sequence)).Bytes(),
		Timestamp:        time.Now(),
		Nonce:            42,
		Sequence:         sequence,
		EmitterChain:     pd.emitterChain,
		EmitterAddress:   pd.emitterAddress,
		Payload:          []byte{0x01, 0x02, 0x03, 0x04},
		ConsistencyLevel: 32,
		// Unreliable and IsReobservation are not relevant for the benchmark.
	}
}

func (pd *ProcessorData) createObservation(b *testing.B, guardianIdx int, k *common.MessagePublication) *gossipv1.Observation {
	b.Helper()
	v := &VAA{
		VAA: vaa.VAA{
			Version:          vaa.SupportedVAAVersion,
			GuardianSetIndex: uint32(guardianIdx), // #nosec G115 -- Safe as number of guardians constrained to 19 in these tests
			Signatures:       nil,
			Timestamp:        k.Timestamp,
			Nonce:            k.Nonce,
			EmitterChain:     k.EmitterChain,
			EmitterAddress:   k.EmitterAddress,
			Payload:          k.Payload,
			Sequence:         k.Sequence,
			ConsistencyLevel: k.ConsistencyLevel,
		},
		// Unreliable and IsReobservation are not relevant for the benchmark.
	}

	// Generate digest of the unsigned VAA.
	digest := v.SigningDigest()

	// Sign the digest using our node's guardian signer
	guardianSigner := pd.guardianSigners[guardianIdx]
	signature, err := guardianSigner.Sign(context.Background(), digest.Bytes())
	require.NoError(b, err)

	return &gossipv1.Observation{
		Hash:      digest.Bytes(),
		Signature: signature,
		TxHash:    k.TxID,
		MessageId: pd.messageID(k.Sequence),
	}
}
