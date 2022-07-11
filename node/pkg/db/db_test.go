package db

import (
	"crypto/ecdsa"
	"crypto/rand"
	"os"

	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/ethereum/go-ethereum/crypto"

	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func TestEmitterPrefixBytes(t *testing.T) {
	vaaIdString := "1/0000000000000000000000000000000000000000000000000000000000000004/1"
	vaaID, _ := VaaIDFromString(vaaIdString)
	expected := []byte{0x73, 0x69, 0x67, 0x6e, 0x65, 0x64, 0x2f, 0x31, 0x2f, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x34}

	assert.Equal(t, expected, vaaID.EmitterPrefixBytes())
}

func TestStoreSignedVAAUnsigned(t *testing.T) {
	dbPath := t.TempDir()
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
	defer db.Close()
	defer os.Remove(dbPath)

	testVaa := getVAA()

	// Should panic because the VAA is not signed
	assert.Panics(t, func() { db.StoreSignedVAA(&testVaa) }, "The code did not panic") //nolint:errcheck
}

func TestStoreSignedVAASigned(t *testing.T) {
	dbPath := t.TempDir()
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
	defer db.Close()
	defer os.Remove(dbPath)

	testVaa := getVAA()

	privKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	testVaa.AddSignature(privKey, 0)

	err2 := db.StoreSignedVAA(&testVaa)
	assert.NoError(t, err2)
}

func TestGetSignedVAABytes(t *testing.T) {
	dbPath := t.TempDir()
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
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
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
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
