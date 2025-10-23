package db

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/dgraph-io/badger/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var storedVaaTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "wormhole_db_total_vaas",
		Help: "Total number of VAAs added to database",
	})

type Database struct {
	db *badger.DB
}

type VAAID struct {
	EmitterChain   vaa.ChainID
	EmitterAddress vaa.Address
	Sequence       uint64
	Version        *uint8
}

// VaaIDFromString parses a <chain>/<address>/<sequence>[/<version number>] string into a VAAID.
// [/version number] is optional and defaults to vaa.VaaVersion1 (1).
func VaaIDFromString(s string) (*VAAID, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 3 && len(parts) != 4 {
		return nil, errors.New("invalid message id")
	}

	vaaVersion := uint8(vaa.VaaVersion1)
	if len(parts) == 4 {
		v, err := strconv.ParseUint(parts[3], 10, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid vaa version: %w", err)
		}

		vaaVersion = uint8(v)

		if !vaa.SupportedVAAVersions[vaaVersion] {
			return nil, fmt.Errorf("unsupported vaa version: %d", vaaVersion)
		}
	}

	emitterChain, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid emitter chain: %s", err)
	}

	emitterAddress, err := vaa.StringToAddress(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid emitter address: %s", err)
	}

	sequence, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid sequence: %s", err)
	}

	msgId := &VAAID{
		EmitterChain:   vaa.ChainID(emitterChain),
		EmitterAddress: emitterAddress,
		Sequence:       sequence,
		Version:        &vaaVersion,
	}

	return msgId, nil
}

func VaaIDFromVAA(v *vaa.VAA) *VAAID {
	ver := v.Version
	return &VAAID{
		EmitterChain:   v.EmitterChain,
		EmitterAddress: v.EmitterAddress,
		Sequence:       v.Sequence,
		Version:        &ver,
	}
}

var (
	ErrVAANotFound = errors.New("requested VAA not found in store")
	nullAddr       = vaa.Address{}
)

func (i *VAAID) Bytes() []byte {
	tmp := []byte(fmt.Sprintf("signed/%d/%s/%d", i.EmitterChain, i.EmitterAddress, i.Sequence))
	if i.stringRepHasVaaVersion() {
		tmp = append(tmp, fmt.Sprintf("/%d", *i.Version)...)
	}

	return tmp
}

func (i *VAAID) stringRepHasVaaVersion() bool {
	return vaa.VersionHasStringRepresentation(i.Version)
}

func (i *VAAID) EmitterPrefixBytes() []byte {
	if i.EmitterAddress == nullAddr {
		return []byte(fmt.Sprintf("signed/%d", i.EmitterChain))
	}

	return []byte(fmt.Sprintf("signed/%d/%s", i.EmitterChain, i.EmitterAddress))
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) StoreSignedVAA(v *vaa.VAA) error {
	if len(v.Signatures) == 0 {
		panic("StoreSignedVAA called for unsigned VAA")
	}

	b, _ := v.Marshal()

	// We allow overriding of existing VAAs, since there are multiple ways to
	// acquire signed VAA bytes. For instance, the node may have a signed VAA
	// via gossip before it reaches quorum on its own. The new entry may have
	// a different set of signatures, but the same VAA.
	//
	// TODO: panic on non-identical signing digest?

	err := d.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(VaaIDFromVAA(v).Bytes(), b); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	storedVaaTotal.Inc()

	return nil
}

// StoreSignedVAABatch writes multiple VAAs to the database using the BadgerDB batch API.
// Note that the API takes care of splitting up the slice into the maximum allowed count
// and size so we don't need to worry about that.
func (d *Database) StoreSignedVAABatch(vaaBatch []*vaa.VAA) error {
	batchTx := d.db.NewWriteBatch()
	defer batchTx.Cancel()

	for _, v := range vaaBatch {
		if len(v.Signatures) == 0 {
			panic("StoreSignedVAABatch called for unsigned VAA")
		}

		b, err := v.Marshal()
		if err != nil {
			panic("StoreSignedVAABatch failed to marshal VAA")
		}

		err = batchTx.Set(VaaIDFromVAA(v).Bytes(), b)
		if err != nil {
			return err
		}
	}

	// Wait for the batch to finish.
	err := batchTx.Flush()
	storedVaaTotal.Add(float64(len(vaaBatch)))
	return err
}

func (d *Database) HasVAA(id VAAID) (bool, error) {
	err := d.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(id.Bytes())
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

func (d *Database) GetSignedVAABytes(id VAAID) (b []byte, err error) {
	if err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(id.Bytes())
		if err != nil {
			return err
		}
		if val, err := item.ValueCopy(nil); err != nil {
			return err
		} else {
			b = val
		}
		return nil
	}); err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, ErrVAANotFound
		}
		return nil, err
	}
	return
}

func (d *Database) FindEmitterSequenceGap(prefix VAAID) (resp []uint64, firstSeq uint64, lastSeq uint64, err error) {
	resp = make([]uint64, 0)
	if err = d.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := prefix.EmitterPrefixBytes()

		// Find all sequence numbers (the message IDs are ordered lexicographically,
		// rather than numerically, so we need to sort them in-memory).
		seqs := make(map[uint64]bool)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := item.Key()
			err := item.Value(func(val []byte) error {
				v, err := vaa.Unmarshal(val)
				if err != nil {
					return fmt.Errorf("failed to unmarshal VAA for %s: %v", string(key), err)
				}

				seqs[v.Sequence] = true
				return nil
			})
			if err != nil {
				return err
			}
		}

		// Find min/max (yay lack of Go generics)
		first := false
		for k := range seqs {
			if first {
				firstSeq = k
				first = false
			}
			if k < firstSeq {
				firstSeq = k
			}
			if k > lastSeq {
				lastSeq = k
			}
		}

		// Figure out gaps.
		for i := firstSeq; i <= lastSeq; i++ {
			if !seqs[i] {
				resp = append(resp, i)
			}
		}

		return nil
	}); err != nil {
		return
	}
	return
}

// Conn returns a pointer to the underlying database connection.
func (d *Database) Conn() *badger.DB {
	return d.db
}
