package db

import (
	"errors"
	"fmt"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/dgraph-io/badger/v3"
)

type Database struct {
	db *badger.DB
}

type VAAID struct {
	EmitterChain   vaa.ChainID
	EmitterAddress vaa.Address
	Sequence       uint64
}

func VaaIDFromVAA(v *vaa.VAA) *VAAID {
	return &VAAID{
		EmitterChain:   v.EmitterChain,
		EmitterAddress: v.EmitterAddress,
		Sequence:       v.Sequence,
	}
}

var (
	ErrVAANotFound = errors.New("requested VAA not found in store")
)

func (i *VAAID) Bytes() []byte {
	return []byte(fmt.Sprintf("signed/%d/%s/%d", i.EmitterChain, i.EmitterAddress, i.Sequence))
}

func (i *VAAID) EmitterPrefixBytes() []byte {
	return []byte(fmt.Sprintf("signed/%d/%s", i.EmitterChain, i.EmitterAddress))
}

func Open(path string) (*Database, error) {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return &Database{
		db: db,
	}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) StoreSignedVAA(v *vaa.VAA) error {
	if len(v.Signatures) == 0 {
		panic("StoreSignedVAA called for unsigned VAA")
	}

	b, _ := v.Marshal()

	// TODO: panic if same VAA is stored with different value

	err := d.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(VaaIDFromVAA(v).Bytes(), b); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
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
		if err == badger.ErrKeyNotFound {
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
				fmt.Printf("missing: %d\n", i)
				resp = append(resp, i)
			}
		}

		return nil
	}); err != nil {
		return
	}
	return
}
