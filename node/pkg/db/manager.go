package db

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/dgraph-io/badger/v3"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// ManagerDB is a wrapper for the database connection used by the manager service.
// It provides methods for storing and retrieving aggregated manager signatures.
type ManagerDB struct {
	db *badger.DB
}

// NewManagerDB creates a new ManagerDB instance.
func NewManagerDB(dbConn *badger.DB) *ManagerDB {
	return &ManagerDB{
		db: dbConn,
	}
}

// Define prefixes used to isolate manager data in the database.
const (
	managerSigPrefix   = "MANAGER:SIG:V1:"
	managerIndexPrefix = "MANAGER:IDX:V1:"
)

var (
	ErrManagerSigNotFound = errors.New("manager signature not found in store")
)

// AggregatedTransaction holds signatures from multiple signers for a single VAA.
// It is used to collect signatures until we have M-of-N required for broadcast.
type AggregatedTransaction struct {
	// VAAHash is the hash of the VAA that triggered this signing.
	VAAHash []byte
	// VAAID is the VAA ID in format "{chain}/{emitter}/{sequence}".
	VAAID string
	// DestinationChain is the target chain (e.g., Dogecoin).
	DestinationChain vaa.ChainID
	// ManagerSetIndex is the delegated manager set index from the payload.
	ManagerSetIndex uint32
	// Required is the M value (number of signatures needed).
	Required uint8
	// Total is the N value (total number of possible signers).
	Total uint8
	// Signatures maps signer index to their signatures.
	// Each entry contains the per-input signatures from that signer.
	Signatures map[uint8][][]byte
}

// IsComplete returns true if this aggregated transaction has enough signatures.
func (a *AggregatedTransaction) IsComplete() bool {
	return uint8(len(a.Signatures)) >= a.Required // #nosec G115 -- Signatures map is bounded by N (uint8)
}

// managerSigKey returns the database key for a manager signature.
func managerSigKey(vaaHashHex string) []byte {
	return []byte(managerSigPrefix + vaaHashHex)
}

// managerIndexKey returns the database key for the VAA ID -> hash index.
func managerIndexKey(vaaID string) []byte {
	return []byte(managerIndexPrefix + vaaID)
}

// MarshalBinary serializes an AggregatedTransaction to bytes.
//
//nolint:unparam // error return kept for encoding.BinaryMarshaler interface compatibility
func (s *AggregatedTransaction) MarshalBinary() ([]byte, error) {
	// Format:
	// [4 bytes] VAAHash length
	// [n bytes] VAAHash
	// [4 bytes] VAAID length
	// [n bytes] VAAID
	// [2 bytes] DestinationChain
	// [4 bytes] ManagerSetIndex
	// [1 byte]  Required
	// [1 byte]  Total
	// [4 bytes] Number of signers
	// For each signer:
	//   [1 byte]  SignerIndex
	//   [4 bytes] Number of input signatures
	//   For each input signature:
	//     [4 bytes] Signature length
	//     [n bytes] Signature

	// Calculate total size
	size := 4 + len(s.VAAHash) + // VAAHash
		4 + len(s.VAAID) + // VAAID
		2 + // DestinationChain
		4 + // ManagerSetIndex
		1 + // Required
		1 + // Total
		4 // Number of signers

	for _, sigs := range s.Signatures {
		size += 1 + 4 // SignerIndex + Number of input signatures
		for _, sig := range sigs {
			size += 4 + len(sig) // Signature length + Signature
		}
	}

	buf := make([]byte, size)
	offset := 0

	// VAAHash
	// #nosec G115 -- VAAHash length is bounded by practical limits (32 bytes for SHA256)
	binary.BigEndian.PutUint32(buf[offset:], uint32(len(s.VAAHash)))
	offset += 4
	copy(buf[offset:], s.VAAHash)
	offset += len(s.VAAHash)

	// VAAID
	// #nosec G115 -- VAAID length is bounded by VAA ID format constraints
	binary.BigEndian.PutUint32(buf[offset:], uint32(len(s.VAAID)))
	offset += 4
	copy(buf[offset:], s.VAAID)
	offset += len(s.VAAID)

	// DestinationChain
	binary.BigEndian.PutUint16(buf[offset:], uint16(s.DestinationChain))
	offset += 2

	// ManagerSetIndex
	binary.BigEndian.PutUint32(buf[offset:], s.ManagerSetIndex)
	offset += 4

	// Required
	buf[offset] = s.Required
	offset++

	// Total
	buf[offset] = s.Total
	offset++

	// Number of signers
	// #nosec G115 -- Signatures map is bounded by N (uint8)
	binary.BigEndian.PutUint32(buf[offset:], uint32(len(s.Signatures)))
	offset += 4

	// Signatures
	for signerIdx, sigs := range s.Signatures {
		buf[offset] = signerIdx
		offset++

		// #nosec G115 -- Number of inputs per transaction is bounded by practical limits
		binary.BigEndian.PutUint32(buf[offset:], uint32(len(sigs)))
		offset += 4

		for _, sig := range sigs {
			// #nosec G115 -- DER signature length is bounded (~72 bytes)
			binary.BigEndian.PutUint32(buf[offset:], uint32(len(sig)))
			offset += 4
			copy(buf[offset:], sig)
			offset += len(sig)
		}
	}

	return buf, nil
}

// UnmarshalBinary deserializes an AggregatedTransaction from bytes.
func (s *AggregatedTransaction) UnmarshalBinary(data []byte) error {
	if len(data) < 18 { // Minimum size: 4+0+4+0+2+4+1+1+4 = 20, but allow smaller VAAHash/VAAID
		return errors.New("data too short for AggregatedTransaction")
	}

	offset := 0

	// VAAHash
	if offset+4 > len(data) {
		return errors.New("data too short for VAAHash length")
	}
	vaaHashLen := binary.BigEndian.Uint32(data[offset:])
	offset += 4
	if offset+int(vaaHashLen) > len(data) {
		return errors.New("data too short for VAAHash")
	}
	s.VAAHash = make([]byte, vaaHashLen)
	copy(s.VAAHash, data[offset:offset+int(vaaHashLen)])
	offset += int(vaaHashLen)

	// VAAID
	if offset+4 > len(data) {
		return errors.New("data too short for VAAID length")
	}
	vaaIDLen := binary.BigEndian.Uint32(data[offset:])
	offset += 4
	if offset+int(vaaIDLen) > len(data) {
		return errors.New("data too short for VAAID")
	}
	s.VAAID = string(data[offset : offset+int(vaaIDLen)])
	offset += int(vaaIDLen)

	// DestinationChain
	if offset+2 > len(data) {
		return errors.New("data too short for DestinationChain")
	}
	s.DestinationChain = vaa.ChainID(binary.BigEndian.Uint16(data[offset:]))
	offset += 2

	// ManagerSetIndex
	if offset+4 > len(data) {
		return errors.New("data too short for ManagerSetIndex")
	}
	s.ManagerSetIndex = binary.BigEndian.Uint32(data[offset:])
	offset += 4

	// Required
	if offset+1 > len(data) {
		return errors.New("data too short for Required")
	}
	s.Required = data[offset]
	offset++

	// Total
	if offset+1 > len(data) {
		return errors.New("data too short for Total")
	}
	s.Total = data[offset]
	offset++

	// Number of signers
	if offset+4 > len(data) {
		return errors.New("data too short for number of signers")
	}
	numSigners := binary.BigEndian.Uint32(data[offset:])
	offset += 4

	s.Signatures = make(map[uint8][][]byte)

	for i := uint32(0); i < numSigners; i++ {
		// SignerIndex
		if offset+1 > len(data) {
			return errors.New("data too short for signer index")
		}
		signerIdx := data[offset]
		offset++

		// Number of input signatures
		if offset+4 > len(data) {
			return errors.New("data too short for number of input signatures")
		}
		numInputSigs := binary.BigEndian.Uint32(data[offset:])
		offset += 4

		inputSigs := make([][]byte, numInputSigs)
		for j := uint32(0); j < numInputSigs; j++ {
			// Signature length
			if offset+4 > len(data) {
				return errors.New("data too short for signature length")
			}
			sigLen := binary.BigEndian.Uint32(data[offset:])
			offset += 4

			// Signature
			if offset+int(sigLen) > len(data) {
				return errors.New("data too short for signature")
			}
			inputSigs[j] = make([]byte, sigLen)
			copy(inputSigs[j], data[offset:offset+int(sigLen)])
			offset += int(sigLen)
		}

		s.Signatures[signerIdx] = inputSigs
	}

	return nil
}

// StoreAggregatedTransaction stores an aggregated transaction in the database.
func (d *ManagerDB) StoreAggregatedTransaction(vaaHashHex string, tx *AggregatedTransaction) error {
	b, err := tx.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal aggregated transaction: %w", err)
	}

	sigKey := managerSigKey(vaaHashHex)
	indexKey := managerIndexKey(tx.VAAID)

	return d.db.Update(func(txn *badger.Txn) error {
		// Store the aggregated transaction
		if err := txn.Set(sigKey, b); err != nil {
			return err
		}
		// Store the VAA ID -> hash index
		return txn.Set(indexKey, []byte(vaaHashHex))
	})
}

// GetAggregatedTransaction retrieves an aggregated transaction from the database.
func (d *ManagerDB) GetAggregatedTransaction(vaaHashHex string) (*AggregatedTransaction, error) {
	var tx AggregatedTransaction

	key := managerSigKey(vaaHashHex)
	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(tx.UnmarshalBinary)
	})

	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, ErrManagerSigNotFound
		}
		return nil, err
	}

	return &tx, nil
}

// HasAggregatedTransaction checks if an aggregated transaction exists in the database.
func (d *ManagerDB) HasAggregatedTransaction(vaaHashHex string) (bool, error) {
	key := managerSigKey(vaaHashHex)
	err := d.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		return err
	})
	if err == nil {
		return true, nil
	}
	if errors.Is(err, badger.ErrKeyNotFound) {
		return false, nil
	}
	return false, err
}

// GetAggregatedTransactionByVAAID retrieves an aggregated transaction by VAA ID using the index.
// This is O(1) lookup using the VAA ID -> hash index.
func (d *ManagerDB) GetAggregatedTransactionByVAAID(vaaID string) (*AggregatedTransaction, error) {
	indexKey := managerIndexKey(vaaID)

	var hashHex string
	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(indexKey)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			hashHex = string(val)
			return nil
		})
	})

	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, ErrManagerSigNotFound
		}
		return nil, err
	}

	return d.GetAggregatedTransaction(hashHex)
}

// DeleteAggregatedTransaction removes an aggregated transaction from the database.
func (d *ManagerDB) DeleteAggregatedTransaction(vaaHashHex string) error {
	// First, get the transaction to find the VAA ID for the index
	tx, err := d.GetAggregatedTransaction(vaaHashHex)
	if err != nil {
		if errors.Is(err, ErrManagerSigNotFound) {
			return nil // Already deleted
		}
		return err
	}

	sigKey := managerSigKey(vaaHashHex)
	indexKey := managerIndexKey(tx.VAAID)

	return d.db.Update(func(txn *badger.Txn) error {
		// Delete the aggregated transaction
		if err := txn.Delete(sigKey); err != nil {
			return err
		}
		// Only delete the index if it points to the same hash
		item, err := txn.Get(indexKey)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil // Index doesn't exist, nothing to delete
			}
			return err
		}
		return item.Value(func(val []byte) error {
			if string(val) == vaaHashHex {
				return txn.Delete(indexKey)
			}
			return nil // Index points to different hash, don't delete
		})
	})
}

// LoadAllAggregatedTransactions loads all aggregated transactions from the database.
// This is useful for restoring state after a restart.
func (d *ManagerDB) LoadAllAggregatedTransactions() (map[string]*AggregatedTransaction, error) {
	result := make(map[string]*AggregatedTransaction)

	err := d.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(managerSigPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := string(item.Key())

			// Extract the VAA hash hex from the key
			vaaHashHex := strings.TrimPrefix(key, managerSigPrefix)

			var tx AggregatedTransaction
			err := item.Value(tx.UnmarshalBinary)
			if err != nil {
				return fmt.Errorf("failed to unmarshal aggregated transaction for key %s: %w", key, err)
			}

			result[vaaHashHex] = &tx
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
