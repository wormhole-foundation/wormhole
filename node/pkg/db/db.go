package db

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/dgraph-io/badger/v3"

	"go.uber.org/zap"
)

type Database struct {
	db *badger.DB
}

type VAAID struct {
	EmitterChain   vaa.ChainID
	EmitterAddress vaa.Address
	Sequence       uint64
}

// VaaIDFromString parses a <chain>/<address>/<sequence> string into a VAAID.
func VaaIDFromString(s string) (*VAAID, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return nil, errors.New("invalid message id")
	}

	emitterChain, err := strconv.Atoi(parts[0])
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

	msgID := &VAAID{
		EmitterChain:   vaa.ChainID(emitterChain),
		EmitterAddress: emitterAddress,
		Sequence:       sequence,
	}

	return msgID, nil
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

type Transfer struct {
	Timestamp    time.Time
	Value        uint64
	TokenChainID vaa.ChainID
	TokenAddress vaa.Address
	MsgID        string
}

func (t *Transfer) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	vaa.MustWrite(buf, binary.BigEndian, uint32(t.Timestamp.Unix()))
	vaa.MustWrite(buf, binary.BigEndian, t.Value)
	vaa.MustWrite(buf, binary.BigEndian, t.TokenChainID)
	buf.Write(t.TokenAddress[:])
	buf.Write([]byte(t.MsgID))
	return buf.Bytes(), nil
}

func UnmarshalTransfer(data []byte) (*Transfer, error) {
	t := &Transfer{}

	reader := bytes.NewReader(data[:])

	unixSeconds := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &unixSeconds); err != nil {
		return nil, fmt.Errorf("failed to read timestamp: %w", err)
	}
	t.Timestamp = time.Unix(int64(unixSeconds), 0)

	if err := binary.Read(reader, binary.BigEndian, &t.Value); err != nil {
		return nil, fmt.Errorf("failed to read value: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &t.TokenChainID); err != nil {
		return nil, fmt.Errorf("failed to read token chain id: %w", err)
	}

	tokenAddress := vaa.Address{}
	if n, err := reader.Read(tokenAddress[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read emitter address [%d]: %w", n, err)
	}
	t.TokenAddress = tokenAddress

	msgID := make([]byte, 256)
	n, err := reader.Read(msgID)
	if err != nil || n == 0 {
		return nil, fmt.Errorf("failed to read vaa id [%d]: %w", n, err)
	}
	t.MsgID = string(msgID[:n])

	return t, nil
}

const transfer string = "GOV:XFER:"
const transferLen int = len(transfer)

const pending string = "GOV:PENDING:"
const pendingLen int = len(pending)

func TransferMsgID(t *Transfer) []byte {
	return []byte(fmt.Sprintf("%v%v", transfer, t.MsgID))
}

func PendingMsgID(k *common.MessagePublication) []byte {
	return []byte(fmt.Sprintf("%v%v", pending, k.MessageIDString()))
}

func IsTransfer(keyBytes []byte) bool {
	return (len(keyBytes) > transferLen) && (string(keyBytes[0:transferLen]) == transfer)
}

func IsPendingMsg(keyBytes []byte) bool {
	return (len(keyBytes) > pendingLen) && (string(keyBytes[0:pendingLen]) == pending)
}

// This is called by the chain governor on start up to reload status.
func (d *Database) GetChainGovernorData(logger *zap.Logger) (transfers []*Transfer, pending []*common.MessagePublication, err error) {
	err = d.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()
			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			if IsPendingMsg(key) {
				msg, err := common.UnmarshalMessagePublication(val)
				if err != nil {
					return err
				}

				pending = append(pending, msg)
			} else if IsTransfer(key) {
				v, err := UnmarshalTransfer(val)
				if err != nil {
					return err
				}

				transfers = append(transfers, v)
			}
		}
		return nil
	})

	return
}

// This is called by the chain governor to persist a pending transfer.
func (d *Database) StoreTransfer(t *Transfer) error {
	b, _ := t.Marshal()

	err := d.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(TransferMsgID(t), b); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to commit transfer tx: %w", err)
	}

	return nil
}

// This is called by the chain governor to persist a pending transfer.
func (d *Database) StorePendingMsg(k *common.MessagePublication) error {
	b, _ := k.Marshal()

	err := d.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(PendingMsgID(k), b); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to commit pending msg tx: %w", err)
	}

	return nil
}

// This is called by the chain governor to delete a transfer after the time limit has expired.
func (d *Database) DeleteTransfer(t *Transfer) error {
	key := TransferMsgID(t)
	err := d.db.DropPrefix(key)
	if err != nil {
		return fmt.Errorf("failed to delete transfer msg for key [%v]: %w", key, err)
	}

	return nil
}

// This is called by the chain governor to delete a pending transfer.
func (d *Database) DeletePendingMsg(k *common.MessagePublication) error {
	key := PendingMsgID(k)
	err := d.db.DropPrefix(key)
	if err != nil {
		return fmt.Errorf("failed to delete pending msg for key [%v]: %w", key, err)
	}

	return nil
}
