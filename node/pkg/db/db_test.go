package db

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	math_rand "math/rand"
	"os"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/dgraph-io/badger/v3"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getVAA() vaa.VAA {
	return getVAAWithSeqNum(1)
}

func getVAAWithSeqNum(seqNum uint64) vaa.VAA {
	var payload = []byte{97, 97, 97, 97, 97, 97}
	var governanceEmitter = vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}

	return vaa.VAA{
		Version:          uint8(1),
		GuardianSetIndex: uint32(1),
		Signatures:       nil,
		Timestamp:        time.Unix(0, 0),
		Nonce:            uint32(1),
		Sequence:         seqNum,
		ConsistencyLevel: uint8(32),
		EmitterChain:     vaa.ChainIDSolana,
		EmitterAddress:   governanceEmitter,
		Payload:          payload,
	}
}

// Testing the expected default behavior of a CreateGovernanceVAA
func TestVaaIDFromString(t *testing.T) {
	vaaIdString := "1/0000000000000000000000000000000000000000000000000000000000000004/1"
	vaaID, _ := VaaIDFromString(vaaIdString)
	expectAddr := vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}

	assert.Equal(t, vaa.ChainIDSolana, vaaID.EmitterChain)
	assert.Equal(t, expectAddr, vaaID.EmitterAddress)
	assert.Equal(t, uint64(1), vaaID.Sequence)
}

func TestVaaIDFromVAA(t *testing.T) {
	testVaa := getVAA()
	vaaID := VaaIDFromVAA(&testVaa)
	expectAddr := vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}

	assert.Equal(t, vaa.ChainIDSolana, vaaID.EmitterChain)
	assert.Equal(t, expectAddr, vaaID.EmitterAddress)
	assert.Equal(t, uint64(1), vaaID.Sequence)
}

func TestBytes(t *testing.T) {
	vaaIdString := "1/0000000000000000000000000000000000000000000000000000000000000004/1"
	vaaID, _ := VaaIDFromString(vaaIdString)
	expected := []byte{0x73, 0x69, 0x67, 0x6e, 0x65, 0x64, 0x2f, 0x31, 0x2f, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x34, 0x2f, 0x31}

	assert.Equal(t, expected, vaaID.Bytes())
}

func TestEmitterPrefixBytesWithChainIDAndAddress(t *testing.T) {
	vaaIdString := "1/0000000000000000000000000000000000000000000000000000000000000004/1"
	vaaID, _ := VaaIDFromString(vaaIdString)
	expected := []byte{0x73, 0x69, 0x67, 0x6e, 0x65, 0x64, 0x2f, 0x31, 0x2f, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x34}

	assert.Equal(t, expected, vaaID.EmitterPrefixBytes())
}

func TestEmitterPrefixBytesWithOnlyChainID(t *testing.T) {
	vaaID := VAAID{EmitterChain: vaa.ChainID(26)}
	assert.Equal(t, []byte("signed/26"), vaaID.EmitterPrefixBytes())
}

func TestStoreSignedVAAUnsigned(t *testing.T) {
	dbPath := t.TempDir()
	db := OpenDb(zap.NewNop(), &dbPath)
	defer db.Close()
	defer os.Remove(dbPath)

	testVaa := getVAA()

	// Should panic because the VAA is not signed
	assert.Panics(t, func() { db.StoreSignedVAA(&testVaa) }, "The code did not panic") //nolint:errcheck
}

func TestStoreSignedVAASigned(t *testing.T) {
	dbPath := t.TempDir()
	db := OpenDb(zap.NewNop(), &dbPath)
	defer db.Close()
	defer os.Remove(dbPath)

	testVaa := getVAA()

	privKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	testVaa.AddSignature(privKey, 0)

	err2 := db.StoreSignedVAA(&testVaa)
	assert.NoError(t, err2)
}

func TestStoreSignedVAABatch(t *testing.T) {
	dbPath := t.TempDir()
	db := OpenDb(zap.NewNop(), &dbPath)
	defer db.Close()
	defer os.Remove(dbPath)

	privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	require.NoError(t, err)

	require.Less(t, int64(0), db.db.MaxBatchCount()) // In testing this was 104857.
	require.Less(t, int64(0), db.db.MaxBatchSize())  // In testing this was 10066329.

	// Make sure we exceed the max batch size.
	numVAAs := uint64(db.db.MaxBatchCount() + 1) // #nosec G115 -- This is safe given the testing values noted above

	// Build the VAA batch.
	vaaBatch := make([]*vaa.VAA, 0, numVAAs)
	for seqNum := uint64(0); seqNum < numVAAs; seqNum++ {
		v := getVAAWithSeqNum(seqNum)
		v.AddSignature(privKey, 0)
		vaaBatch = append(vaaBatch, &v)
	}

	// Store the batch in the database.
	err = db.StoreSignedVAABatch(vaaBatch)
	require.NoError(t, err)

	// Verify all the VAAs are in the database.
	for _, v := range vaaBatch {
		storedBytes, err := db.GetSignedVAABytes(*VaaIDFromVAA(v))
		require.NoError(t, err)

		origBytes, err := v.Marshal()
		require.NoError(t, err)

		assert.True(t, bytes.Equal(origBytes, storedBytes))
	}

	// Verify that updates work as well by tweaking the VAAs and rewriting them.
	for _, v := range vaaBatch {
		v.Nonce += 1
	}

	// Store the updated batch in the database.
	err = db.StoreSignedVAABatch(vaaBatch)
	require.NoError(t, err)

	// Verify all the updated VAAs are in the database.
	for _, v := range vaaBatch {
		storedBytes, err := db.GetSignedVAABytes(*VaaIDFromVAA(v))
		require.NoError(t, err)

		origBytes, err := v.Marshal()
		require.NoError(t, err)

		assert.True(t, bytes.Equal(origBytes, storedBytes))
	}
}

func TestGetSignedVAABytes(t *testing.T) {
	dbPath := t.TempDir()
	db := OpenDb(zap.NewNop(), &dbPath)
	defer db.Close()
	defer os.Remove(dbPath)

	testVaa := getVAA()

	vaaID := VaaIDFromVAA(&testVaa)

	privKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	testVaa.AddSignature(privKey, 0)

	// Store full VAA
	err2 := db.StoreSignedVAA(&testVaa)
	assert.NoError(t, err2)

	// Retrieve it using vaaID
	vaaBytes, err2 := db.GetSignedVAABytes(*vaaID)
	assert.NoError(t, err2)

	testVaaBytes, err3 := testVaa.Marshal()
	assert.NoError(t, err3)

	assert.Equal(t, testVaaBytes, vaaBytes)
}

func TestFindEmitterSequenceGap(t *testing.T) {
	dbPath := t.TempDir()
	db := OpenDb(zap.NewNop(), &dbPath)
	defer db.Close()
	defer os.Remove(dbPath)

	testVaa := getVAA()

	vaaID := VaaIDFromVAA(&testVaa)

	privKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	testVaa.AddSignature(privKey, 0)

	// Store full VAA
	err2 := db.StoreSignedVAA(&testVaa)
	assert.NoError(t, err2)

	resp, firstSeq, lastSeq, err := db.FindEmitterSequenceGap(*vaaID)

	assert.Equal(t, []uint64{0x0}, resp)
	assert.Equal(t, uint64(0x0), firstSeq)
	assert.Equal(t, uint64(0x1), lastSeq)
	assert.NoError(t, err)
}

// BenchmarkVaaLookup benchmarks db.GetSignedVAABytes
// You need to set the environment variable WH_DBPATH to a path with a populated BadgerDB.
// You may want to play with the CONCURRENCY parameter.
func BenchmarkVaaLookup(b *testing.B) {
	CONCURRENCY := runtime.NumCPU()
	dbPath := os.Getenv("WH_DBPATH")
	require.NotEqual(b, dbPath, "")

	// open DB
	optionsDB := badger.DefaultOptions(dbPath)
	optionsDB.Logger = nil
	badgerDb, err := badger.Open(optionsDB)
	require.NoError(b, err)
	db := &Database{
		db: badgerDb,
	}

	if err != nil {
		b.Error("failed to open database")
	}
	defer db.Close()

	vaaIds := make(chan *VAAID, b.N)

	for i := 0; i < b.N; i++ {
		randId := math_rand.Intn(250000) //nolint
		randId = 250000 - (i / 18)
		vaaId, err := VaaIDFromString(fmt.Sprintf("4/000000000000000000000000b6f6d86a8f9879a9c87f643768d9efc38c1da6e7/%d", randId))
		assert.NoError(b, err)
		vaaIds <- vaaId
	}

	b.ResetTimer()

	// actual timed code
	var errCtr atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < CONCURRENCY; i++ {
		wg.Add(1)
		go func() {
			for {
				select {
				case vaaId := <-vaaIds:
					_, err = db.GetSignedVAABytes(*vaaId)
					if err != nil {
						fmt.Printf("error retrieving %s/%s/%d: %s\n", vaaId.EmitterChain, vaaId.EmitterAddress, vaaId.Sequence, err)
						errCtr.Add(1)
					}
				default:
					wg.Done()
					return
				}
			}
		}()
	}

	wg.Wait()

	if int(errCtr.Load()) > b.N/3 {
		b.Error("More than 1/3 of GetSignedVAABytes failed.")
	}
}
