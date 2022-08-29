package db

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/dgraph-io/badger/v3"

	"go.uber.org/zap"
)

type GovernorDB interface {
	StoreTransfer(t *Transfer) error
	StorePendingMsg(k *PendingTransfer) error
	DeleteTransfer(t *Transfer) error
	DeletePendingMsg(k *PendingTransfer) error
	GetChainGovernorData(logger *zap.Logger) (transfers []*Transfer, pending []*PendingTransfer, err error)
}

type MockGovernorDB struct {
}

func (d *MockGovernorDB) StoreTransfer(t *Transfer) error {
	return nil
}

func (d *MockGovernorDB) StorePendingMsg(k *PendingTransfer) error {
	return nil
}

func (d *MockGovernorDB) DeleteTransfer(t *Transfer) error {
	return nil
}

func (d *MockGovernorDB) DeletePendingMsg(pending *PendingTransfer) error {
	return nil
}

func (d *MockGovernorDB) GetChainGovernorData(logger *zap.Logger) (transfers []*Transfer, pending []*PendingTransfer, err error) {
	return nil, nil, nil
}

type Transfer struct {
	Timestamp      time.Time
	Value          uint64
	OriginChain    vaa.ChainID
	OriginAddress  vaa.Address
	EmitterChain   vaa.ChainID
	EmitterAddress vaa.Address
	MsgID          string
}

func (t *Transfer) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	vaa.MustWrite(buf, binary.BigEndian, uint32(t.Timestamp.Unix()))
	vaa.MustWrite(buf, binary.BigEndian, t.Value)
	vaa.MustWrite(buf, binary.BigEndian, t.OriginChain)
	buf.Write(t.OriginAddress[:])
	vaa.MustWrite(buf, binary.BigEndian, t.EmitterChain)
	buf.Write(t.EmitterAddress[:])
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

	if err := binary.Read(reader, binary.BigEndian, &t.OriginChain); err != nil {
		return nil, fmt.Errorf("failed to read token chain id: %w", err)
	}

	originAddress := vaa.Address{}
	if n, err := reader.Read(originAddress[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read emitter address [%d]: %w", n, err)
	}
	t.OriginAddress = originAddress

	if err := binary.Read(reader, binary.BigEndian, &t.EmitterChain); err != nil {
		return nil, fmt.Errorf("failed to read token chain id: %w", err)
	}

	emitterAddress := vaa.Address{}
	if n, err := reader.Read(emitterAddress[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read emitter address [%d]: %w", n, err)
	}
	t.EmitterAddress = emitterAddress

	msgID := make([]byte, 256)
	n, err := reader.Read(msgID)
	if err != nil || n == 0 {
		return nil, fmt.Errorf("failed to read vaa id [%d]: %w", n, err)
	}
	t.MsgID = string(msgID[:n])

	return t, nil
}

type PendingTransfer struct {
	ReleaseTime time.Time
	Msg         common.MessagePublication
}

func (p *PendingTransfer) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	vaa.MustWrite(buf, binary.BigEndian, uint32(p.ReleaseTime.Unix()))

	b, err := p.Msg.Marshal()
	if err != nil {
		return buf.Bytes(), fmt.Errorf("failed to marshal pending transfer: %w", err)
	}

	vaa.MustWrite(buf, binary.BigEndian, b)

	return buf.Bytes(), nil
}

func UnmarshalPendingTransfer(data []byte) (*PendingTransfer, error) {
	p := &PendingTransfer{}

	reader := bytes.NewReader(data[:])

	unixSeconds := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &unixSeconds); err != nil {
		return nil, fmt.Errorf("failed to read pending transfer release time: %w", err)
	}

	p.ReleaseTime = time.Unix(int64(unixSeconds), 0)

	buf := make([]byte, reader.Len())
	n, err := reader.Read(buf)
	if err != nil || n == 0 {
		return nil, fmt.Errorf("failed to read pending transfer msg [%d]: %w", n, err)
	}

	msg, err := common.UnmarshalMessagePublication(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal pending transfer msg: %w", err)
	}

	p.Msg = *msg
	return p, nil
}

const transfer = "GOV:XFER:"
const transferLen = len(transfer)

// Since we are changing the DB format of pending entries, we will use a new tag in the pending key field.
// The first time we run this new release, any existing entries with the "GOV:PENDING" tag will get converted
// to the new format and given the "GOV:PENDING2" format. In a future release, the "GOV:PENDING" code can be deleted.

const oldPending = "GOV:PENDING:"
const oldPendingLen = len(oldPending)

const pending = "GOV:PENDING2:"
const pendingLen = len(pending)

const minMsgIdLen = len("1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0")

func TransferMsgID(t *Transfer) []byte {
	return []byte(fmt.Sprintf("%v%v", transfer, t.MsgID))
}

func PendingMsgID(k *common.MessagePublication) []byte {
	return []byte(fmt.Sprintf("%v%v", pending, k.MessageIDString()))
}

func oldPendingMsgID(k *common.MessagePublication) []byte {
	return []byte(fmt.Sprintf("%v%v", oldPending, k.MessageIDString()))
}

func IsTransfer(keyBytes []byte) bool {
	return (len(keyBytes) >= transferLen+minMsgIdLen) && (string(keyBytes[0:transferLen]) == transfer)
}

func IsPendingMsg(keyBytes []byte) bool {
	return (len(keyBytes) >= pendingLen+minMsgIdLen) && (string(keyBytes[0:pendingLen]) == pending)
}

func isOldPendingMsg(keyBytes []byte) bool {
	return (len(keyBytes) >= oldPendingLen+minMsgIdLen) && (string(keyBytes[0:oldPendingLen]) == oldPending)
}

// This is called by the chain governor on start up to reload status.
func (d *Database) GetChainGovernorData(logger *zap.Logger) (transfers []*Transfer, pending []*PendingTransfer, err error) {
	return d.GetChainGovernorDataForTime(logger, time.Now())
}

func (d *Database) GetChainGovernorDataForTime(logger *zap.Logger, now time.Time) (transfers []*Transfer, pending []*PendingTransfer, err error) {
	oldPendingToUpdate := []*PendingTransfer{}
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
				p, err := UnmarshalPendingTransfer(val)
				if err != nil {
					return err
				}

				pending = append(pending, p)
			} else if IsTransfer(key) {
				v, err := UnmarshalTransfer(val)
				if err != nil {
					return err
				}

				transfers = append(transfers, v)
			} else if isOldPendingMsg(key) {
				msg, err := common.UnmarshalMessagePublication(val)
				if err != nil {
					return err
				}

				p := &PendingTransfer{ReleaseTime: now.Add(time.Duration(time.Hour * 72)), Msg: *msg}
				pending = append(pending, p)
				oldPendingToUpdate = append(oldPendingToUpdate, p)
			}
		}

		if len(oldPendingToUpdate) != 0 {
			for _, pending := range oldPendingToUpdate {
				logger.Info("cgov: updating format of database entry for pending vaa", zap.String("msgId", pending.Msg.MessageIDString()))
				err := d.StorePendingMsg(pending)
				if err != nil {
					return fmt.Errorf("failed to write new pending msg for key [%v]: %w", pending.Msg.MessageIDString(), err)
				}

				key := oldPendingMsgID(&pending.Msg)
				err = d.db.DropPrefix(key)
				if err != nil {
					return fmt.Errorf("failed to delete old pending msg for key [%v]: %w", pending.Msg.MessageIDString(), err)
				}
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
func (d *Database) StorePendingMsg(pending *PendingTransfer) error {
	b, _ := pending.Marshal()

	err := d.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(PendingMsgID(&pending.Msg), b); err != nil {
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
func (d *Database) DeletePendingMsg(pending *PendingTransfer) error {
	key := PendingMsgID(&pending.Msg)
	err := d.db.DropPrefix(key)
	if err != nil {
		return fmt.Errorf("failed to delete pending msg for key [%v]: %w", key, err)
	}

	return nil
}
