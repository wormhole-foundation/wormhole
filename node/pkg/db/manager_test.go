package db

import (
	"bytes"
	"testing"

	"github.com/dgraph-io/badger/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestAggregatedTransactionMarshalUnmarshal(t *testing.T) {
	// Create test data
	original := &AggregatedTransaction{
		VAAHash:          []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		VAAID:            "2/0000000000000000000000000000000000000000/12345",
		DestinationChain: vaa.ChainIDDogecoin,
		ManagerSetIndex:  1,
		Required:         2,
		Total:            3,
		Signatures: map[uint8][][]byte{
			0: {
				{0x10, 0x11, 0x12},
				{0x13, 0x14, 0x15},
			},
			2: {
				{0x20, 0x21, 0x22, 0x23},
			},
		},
	}

	// Marshal
	data, err := original.MarshalBinary()
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Unmarshal
	restored := &AggregatedTransaction{}
	err = restored.UnmarshalBinary(data)
	require.NoError(t, err)

	// Verify all fields
	assert.True(t, bytes.Equal(original.VAAHash, restored.VAAHash))
	assert.Equal(t, original.VAAID, restored.VAAID)
	assert.Equal(t, original.DestinationChain, restored.DestinationChain)
	assert.Equal(t, original.ManagerSetIndex, restored.ManagerSetIndex)
	assert.Equal(t, original.Required, restored.Required)
	assert.Equal(t, original.Total, restored.Total)
	assert.Equal(t, len(original.Signatures), len(restored.Signatures))

	// Verify signatures
	for signerIdx, sigs := range original.Signatures {
		restoredSigs, ok := restored.Signatures[signerIdx]
		require.True(t, ok, "missing signer index %d", signerIdx)
		require.Equal(t, len(sigs), len(restoredSigs))
		for i, sig := range sigs {
			assert.True(t, bytes.Equal(sig, restoredSigs[i]))
		}
	}
}

func TestAggregatedTransactionIsComplete(t *testing.T) {
	tests := []struct {
		name       string
		required   uint8
		numSigs    int
		isComplete bool
	}{
		{"no signatures needed, none provided", 0, 0, true},
		{"one needed, none provided", 1, 0, false},
		{"two needed, one provided", 2, 1, false},
		{"two needed, two provided", 2, 2, true},
		{"two needed, three provided", 2, 3, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tx := &AggregatedTransaction{
				Required:   tc.required,
				Signatures: make(map[uint8][][]byte),
			}
			for i := 0; i < tc.numSigs; i++ {
				tx.Signatures[uint8(i)] = [][]byte{{0x01}} // #nosec G115 -- test code with small values
			}
			assert.Equal(t, tc.isComplete, tx.IsComplete())
		})
	}
}

func TestManagerDBStoreAndGet(t *testing.T) {
	// Create in-memory database
	opts := badger.DefaultOptions("").WithInMemory(true).WithLogger(nil)
	badgerDB, err := badger.Open(opts)
	require.NoError(t, err)
	defer badgerDB.Close()

	managerDB := NewManagerDB(badgerDB)

	// Create test transaction
	tx := &AggregatedTransaction{
		VAAHash:          []byte{0xaa, 0xbb, 0xcc, 0xdd},
		VAAID:            "2/0000000000000000000000000000000000000001/999",
		DestinationChain: vaa.ChainIDDogecoin,
		ManagerSetIndex:  1,
		Required:         2,
		Total:            3,
		Signatures: map[uint8][][]byte{
			0: {{0x01, 0x02, 0x03}},
		},
	}

	hashHex := "aabbccdd"

	// Store
	err = managerDB.StoreAggregatedTransaction(hashHex, tx)
	require.NoError(t, err)

	// Get
	retrieved, err := managerDB.GetAggregatedTransaction(hashHex)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.True(t, bytes.Equal(tx.VAAHash, retrieved.VAAHash))
	assert.Equal(t, tx.VAAID, retrieved.VAAID)
	assert.Equal(t, tx.DestinationChain, retrieved.DestinationChain)
	assert.Equal(t, tx.ManagerSetIndex, retrieved.ManagerSetIndex)
	assert.Equal(t, tx.Required, retrieved.Required)
	assert.Equal(t, tx.Total, retrieved.Total)
}

func TestManagerDBHasAggregatedTransaction(t *testing.T) {
	opts := badger.DefaultOptions("").WithInMemory(true).WithLogger(nil)
	badgerDB, err := badger.Open(opts)
	require.NoError(t, err)
	defer badgerDB.Close()

	managerDB := NewManagerDB(badgerDB)

	hashHex := "deadbeef"

	// Should not exist initially
	exists, err := managerDB.HasAggregatedTransaction(hashHex)
	require.NoError(t, err)
	assert.False(t, exists)

	// Store something
	tx := &AggregatedTransaction{
		VAAHash:    []byte{0xde, 0xad, 0xbe, 0xef},
		Signatures: make(map[uint8][][]byte),
	}
	err = managerDB.StoreAggregatedTransaction(hashHex, tx)
	require.NoError(t, err)

	// Should exist now
	exists, err = managerDB.HasAggregatedTransaction(hashHex)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestManagerDBDeleteAggregatedTransaction(t *testing.T) {
	opts := badger.DefaultOptions("").WithInMemory(true).WithLogger(nil)
	badgerDB, err := badger.Open(opts)
	require.NoError(t, err)
	defer badgerDB.Close()

	managerDB := NewManagerDB(badgerDB)

	hashHex := "cafebabe"
	vaaID := "2/0000000000000000000000000000000000000001/123"

	tx := &AggregatedTransaction{
		VAAHash:    []byte{0xca, 0xfe, 0xba, 0xbe},
		VAAID:      vaaID,
		Signatures: make(map[uint8][][]byte),
	}

	// Store
	err = managerDB.StoreAggregatedTransaction(hashHex, tx)
	require.NoError(t, err)

	// Verify it exists
	exists, err := managerDB.HasAggregatedTransaction(hashHex)
	require.NoError(t, err)
	assert.True(t, exists)

	// Verify index works
	retrieved, err := managerDB.GetAggregatedTransactionByVAAID(vaaID)
	require.NoError(t, err)
	assert.Equal(t, vaaID, retrieved.VAAID)

	// Delete
	err = managerDB.DeleteAggregatedTransaction(hashHex)
	require.NoError(t, err)

	// Verify it's gone
	exists, err = managerDB.HasAggregatedTransaction(hashHex)
	require.NoError(t, err)
	assert.False(t, exists)

	// Get should return not found error
	_, err = managerDB.GetAggregatedTransaction(hashHex)
	assert.ErrorIs(t, err, ErrManagerSigNotFound)

	// Index lookup should also return not found
	_, err = managerDB.GetAggregatedTransactionByVAAID(vaaID)
	assert.ErrorIs(t, err, ErrManagerSigNotFound)
}

func TestManagerDBGetAggregatedTransactionByVAAID(t *testing.T) {
	opts := badger.DefaultOptions("").WithInMemory(true).WithLogger(nil)
	badgerDB, err := badger.Open(opts)
	require.NoError(t, err)
	defer badgerDB.Close()

	managerDB := NewManagerDB(badgerDB)

	hashHex := "11223344"
	vaaID := "2/0000000000000000000000000000000000000002/456"

	tx := &AggregatedTransaction{
		VAAHash:          []byte{0x11, 0x22, 0x33, 0x44},
		VAAID:            vaaID,
		DestinationChain: vaa.ChainIDDogecoin,
		ManagerSetIndex:  1,
		Required:         2,
		Total:            3,
		Signatures:       make(map[uint8][][]byte),
	}

	// Store
	err = managerDB.StoreAggregatedTransaction(hashHex, tx)
	require.NoError(t, err)

	// Lookup by VAA ID (O(1) using index)
	retrieved, err := managerDB.GetAggregatedTransactionByVAAID(vaaID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, vaaID, retrieved.VAAID)
	assert.Equal(t, tx.DestinationChain, retrieved.DestinationChain)
	assert.Equal(t, tx.ManagerSetIndex, retrieved.ManagerSetIndex)

	// Non-existent VAA ID should return not found
	_, err = managerDB.GetAggregatedTransactionByVAAID("nonexistent")
	assert.ErrorIs(t, err, ErrManagerSigNotFound)
}

func TestManagerDBIndexNotDeletedWhenPointingToDifferentHash(t *testing.T) {
	opts := badger.DefaultOptions("").WithInMemory(true).WithLogger(nil)
	badgerDB, err := badger.Open(opts)
	require.NoError(t, err)
	defer badgerDB.Close()

	managerDB := NewManagerDB(badgerDB)

	vaaID := "2/0000000000000000000000000000000000000003/789"
	hashHex1 := "aaaa1111"
	hashHex2 := "bbbb2222"

	tx1 := &AggregatedTransaction{
		VAAHash:    []byte{0xaa, 0xaa, 0x11, 0x11},
		VAAID:      vaaID,
		Signatures: make(map[uint8][][]byte),
	}
	tx2 := &AggregatedTransaction{
		VAAHash:    []byte{0xbb, 0xbb, 0x22, 0x22},
		VAAID:      vaaID,
		Signatures: make(map[uint8][][]byte),
	}

	// Store first transaction
	err = managerDB.StoreAggregatedTransaction(hashHex1, tx1)
	require.NoError(t, err)

	// Store second transaction with same VAA ID (overwrites index)
	err = managerDB.StoreAggregatedTransaction(hashHex2, tx2)
	require.NoError(t, err)

	// Index should now point to hashHex2
	retrieved, err := managerDB.GetAggregatedTransactionByVAAID(vaaID)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(tx2.VAAHash, retrieved.VAAHash))

	// Delete first transaction - should NOT delete the index since it points to hashHex2
	err = managerDB.DeleteAggregatedTransaction(hashHex1)
	require.NoError(t, err)

	// Index should still work and point to tx2
	retrieved, err = managerDB.GetAggregatedTransactionByVAAID(vaaID)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(tx2.VAAHash, retrieved.VAAHash))

	// Delete second transaction - should delete the index since it points to hashHex2
	err = managerDB.DeleteAggregatedTransaction(hashHex2)
	require.NoError(t, err)

	// Index should now return not found
	_, err = managerDB.GetAggregatedTransactionByVAAID(vaaID)
	assert.ErrorIs(t, err, ErrManagerSigNotFound)
}

func TestManagerDBLoadAllAggregatedTransactions(t *testing.T) {
	opts := badger.DefaultOptions("").WithInMemory(true).WithLogger(nil)
	badgerDB, err := badger.Open(opts)
	require.NoError(t, err)
	defer badgerDB.Close()

	managerDB := NewManagerDB(badgerDB)

	// Store multiple transactions
	tx1 := &AggregatedTransaction{
		VAAHash:    []byte{0x01},
		VAAID:      "vaa1",
		Signatures: make(map[uint8][][]byte),
	}
	tx2 := &AggregatedTransaction{
		VAAHash:    []byte{0x02},
		VAAID:      "vaa2",
		Signatures: make(map[uint8][][]byte),
	}

	err = managerDB.StoreAggregatedTransaction("01", tx1)
	require.NoError(t, err)
	err = managerDB.StoreAggregatedTransaction("02", tx2)
	require.NoError(t, err)

	// Load all
	txs, err := managerDB.LoadAllAggregatedTransactions()
	require.NoError(t, err)
	assert.Len(t, txs, 2)

	// Verify contents
	assert.Equal(t, "vaa1", txs["01"].VAAID)
	assert.Equal(t, "vaa2", txs["02"].VAAID)
}
