package db

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"testing"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func storeVAA(db *Database, v *vaa.VAA) error {
	privKey, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	v.AddSignature(privKey, 0)
	return db.StoreSignedVAA(v)
}

func countVAAs(d *Database, chainId vaa.ChainID) (numThisChain int, numOtherChains int, err error) { //nolint:unparam
	if err = d.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()
			err := item.Value(func(val []byte) error {
				v, err := vaa.Unmarshal(val)
				if err != nil {
					return fmt.Errorf("failed to unmarshal VAA for %s: %v", string(key), err)
				}

				if v.EmitterChain == chainId {
					numThisChain++
				} else {
					numOtherChains++
				}

				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return
	}

	return
}

func TestPurgingPythnetVAAs(t *testing.T) {
	var payload = []byte{97, 97, 97, 97, 97, 97}
	var emitterAddress = vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}

	dbPath := t.TempDir()
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
	defer db.Close()
	defer os.Remove(dbPath)

	now := time.Now()

	// Create 50 VAAs each for Pythnet and Solana that are more than three days old.
	timeStamp := now.Add(-time.Hour * time.Duration(3*24+1))
	pythnetSeqNum := uint64(10000)
	solanaSeqNum := uint64(20000)
	for count := 0; count < 50; count++ {
		err = storeVAA(db, &vaa.VAA{
			Version:          uint8(1),
			GuardianSetIndex: uint32(1),
			Signatures:       nil,
			Timestamp:        timeStamp,
			Nonce:            uint32(1),
			Sequence:         pythnetSeqNum,
			ConsistencyLevel: uint8(32),
			EmitterChain:     vaa.ChainIDPythNet,
			EmitterAddress:   emitterAddress,
			Payload:          payload,
		})
		require.NoError(t, err)
		pythnetSeqNum++

		err = storeVAA(db, &vaa.VAA{
			Version:          uint8(1),
			GuardianSetIndex: uint32(1),
			Signatures:       nil,
			Timestamp:        timeStamp,
			Nonce:            uint32(1),
			Sequence:         solanaSeqNum,
			ConsistencyLevel: uint8(32),
			EmitterChain:     vaa.ChainIDSolana,
			EmitterAddress:   emitterAddress,
			Payload:          payload,
		})
		require.NoError(t, err)
		solanaSeqNum++
	}

	// Create 75 VAAs each for Pythnet and Solana that are less than three days old.
	timeStamp = now.Add(-time.Hour * time.Duration(3*24-1))
	for count := 0; count < 75; count++ {
		err = storeVAA(db, &vaa.VAA{
			Version:          uint8(1),
			GuardianSetIndex: uint32(1),
			Signatures:       nil,
			Timestamp:        timeStamp,
			Nonce:            uint32(1),
			Sequence:         pythnetSeqNum,
			ConsistencyLevel: uint8(32),
			EmitterChain:     vaa.ChainIDPythNet,
			EmitterAddress:   emitterAddress,
			Payload:          payload,
		})
		require.NoError(t, err)
		pythnetSeqNum++

		err = storeVAA(db, &vaa.VAA{
			Version:          uint8(1),
			GuardianSetIndex: uint32(1),
			Signatures:       nil,
			Timestamp:        timeStamp,
			Nonce:            uint32(1),
			Sequence:         solanaSeqNum,
			ConsistencyLevel: uint8(32),
			EmitterChain:     vaa.ChainIDSolana,
			EmitterAddress:   emitterAddress,
			Payload:          payload,
		})
		require.NoError(t, err)
		solanaSeqNum++
	}

	// Before we do the purge, make sure the database contains what we expect.
	numPythnet, numOther, err := countVAAs(db, vaa.ChainIDPythNet)
	require.NoError(t, err)
	assert.Equal(t, 125, numPythnet)
	assert.Equal(t, 125, numOther)

	// Purge PythNet VAAs that are more than three days old.
	oldestTime := now.Add(-time.Hour * time.Duration(3*24))
	prefix := VAAID{EmitterChain: vaa.ChainIDPythNet}
	_, err = db.PurgeVaas(prefix, oldestTime, false)
	require.NoError(t, err)

	// Make sure we deleted the old PythNet VAAs but didn't touch the Solana ones.
	numPythnet, numOther, err = countVAAs(db, vaa.ChainIDPythNet)
	require.NoError(t, err)
	assert.Equal(t, 75, numPythnet)
	assert.Equal(t, 125, numOther)
}

func TestPurgingVAAsForOneEmitterAddress(t *testing.T) {
	var payload = []byte{97, 97, 97, 97, 97, 97}
	var pythnetEmitterAddress1 = vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	var pythnetEmitterAddress2 = vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}
	var solanaEmitterAddress1 = vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	dbPath := t.TempDir()
	db, err := Open(dbPath)
	if err != nil {
		t.Error("failed to open database")
	}
	defer db.Close()
	defer os.Remove(dbPath)

	now := time.Now()

	// Create 50 VAAs each for each emitter that are more than three days old.
	timeStamp := now.Add(-time.Hour * time.Duration(3*24+1))
	pythnetSeqNum := uint64(10000)
	solanaSeqNum := uint64(20000)
	for count := 0; count < 50; count++ {
		err = storeVAA(db, &vaa.VAA{
			Version:          uint8(1),
			GuardianSetIndex: uint32(1),
			Signatures:       nil,
			Timestamp:        timeStamp,
			Nonce:            uint32(1),
			Sequence:         pythnetSeqNum,
			ConsistencyLevel: uint8(32),
			EmitterChain:     vaa.ChainIDPythNet,
			EmitterAddress:   pythnetEmitterAddress1,
			Payload:          payload,
		})
		require.NoError(t, err)

		err = storeVAA(db, &vaa.VAA{
			Version:          uint8(1),
			GuardianSetIndex: uint32(1),
			Signatures:       nil,
			Timestamp:        timeStamp,
			Nonce:            uint32(1),
			Sequence:         pythnetSeqNum,
			ConsistencyLevel: uint8(32),
			EmitterChain:     vaa.ChainIDPythNet,
			EmitterAddress:   pythnetEmitterAddress2,
			Payload:          payload,
		})
		require.NoError(t, err)

		pythnetSeqNum++

		err = storeVAA(db, &vaa.VAA{
			Version:          uint8(1),
			GuardianSetIndex: uint32(1),
			Signatures:       nil,
			Timestamp:        timeStamp,
			Nonce:            uint32(1),
			Sequence:         solanaSeqNum,
			ConsistencyLevel: uint8(32),
			EmitterChain:     vaa.ChainIDSolana,
			EmitterAddress:   solanaEmitterAddress1,
			Payload:          payload,
		})
		require.NoError(t, err)
		solanaSeqNum++
	}

	// Create 75 VAAs each for each emitter that are less than three days old.
	timeStamp = now.Add(-time.Hour * time.Duration(3*24-1))
	for count := 0; count < 75; count++ {
		err = storeVAA(db, &vaa.VAA{
			Version:          uint8(1),
			GuardianSetIndex: uint32(1),
			Signatures:       nil,
			Timestamp:        timeStamp,
			Nonce:            uint32(1),
			Sequence:         pythnetSeqNum,
			ConsistencyLevel: uint8(32),
			EmitterChain:     vaa.ChainIDPythNet,
			EmitterAddress:   pythnetEmitterAddress1,
			Payload:          payload,
		})
		require.NoError(t, err)

		err = storeVAA(db, &vaa.VAA{
			Version:          uint8(1),
			GuardianSetIndex: uint32(1),
			Signatures:       nil,
			Timestamp:        timeStamp,
			Nonce:            uint32(1),
			Sequence:         pythnetSeqNum,
			ConsistencyLevel: uint8(32),
			EmitterChain:     vaa.ChainIDPythNet,
			EmitterAddress:   pythnetEmitterAddress2,
			Payload:          payload,
		})
		require.NoError(t, err)

		pythnetSeqNum++

		err = storeVAA(db, &vaa.VAA{
			Version:          uint8(1),
			GuardianSetIndex: uint32(1),
			Signatures:       nil,
			Timestamp:        timeStamp,
			Nonce:            uint32(1),
			Sequence:         solanaSeqNum,
			ConsistencyLevel: uint8(32),
			EmitterChain:     vaa.ChainIDSolana,
			EmitterAddress:   solanaEmitterAddress1,
			Payload:          payload,
		})
		require.NoError(t, err)
		solanaSeqNum++
	}

	// Before we do the purge, make sure the database contains what we expect.
	numPythnet, numOther, err := countVAAs(db, vaa.ChainIDPythNet)
	require.NoError(t, err)
	assert.Equal(t, 250, numPythnet)
	assert.Equal(t, 125, numOther)

	// Purge VAAs for a single PythNet emitter that are more than three days old.
	oldestTime := now.Add(-time.Hour * time.Duration(3*24))
	prefix := VAAID{EmitterChain: vaa.ChainIDPythNet, EmitterAddress: pythnetEmitterAddress1}
	_, err = db.PurgeVaas(prefix, oldestTime, false)
	require.NoError(t, err)

	// Make sure we deleted the old PythNet VAAs but didn't touch the Solana ones.
	numPythnet, numOther, err = countVAAs(db, vaa.ChainIDPythNet)
	require.NoError(t, err)
	assert.Equal(t, 200, numPythnet)
	assert.Equal(t, 125, numOther)
}
