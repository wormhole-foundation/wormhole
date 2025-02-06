package db

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/dgraph-io/badger/v3"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

type AccountantDB interface {
	AcctStorePendingTransfer(msg *common.MessagePublication) error
	AcctDeletePendingTransfer(msgId string) error
	AcctGetData(logger *zap.Logger) ([]*common.MessagePublication, error)
}

type MockAccountantDB struct {
}

func (d *MockAccountantDB) AcctStorePendingTransfer(msg *common.MessagePublication) error {
	return nil
}

func (d *MockAccountantDB) AcctDeletePendingTransfer(msgId string) error {
	return nil
}

func (d *MockAccountantDB) AcctGetData(logger *zap.Logger) ([]*common.MessagePublication, error) {
	return nil, nil
}

const acctOldPendingTransfer = "ACCT:PXFER2:"
const acctOldPendingTransferLen = len(acctOldPendingTransfer)

const acctPendingTransfer = "ACCT:PXFER3:"
const acctPendingTransferLen = len(acctPendingTransfer)

const acctMinMsgIdLen = len("1/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0")

func acctOldPendingTransferMsgID(msgId string) []byte {
	return []byte(fmt.Sprintf("%v%v", acctOldPendingTransfer, msgId))
}

func acctIsOldPendingTransfer(keyBytes []byte) bool {
	return (len(keyBytes) >= acctOldPendingTransferLen+acctMinMsgIdLen) && (string(keyBytes[0:acctOldPendingTransferLen]) == acctOldPendingTransfer)
}

func acctPendingTransferMsgID(msgId string) []byte {
	return []byte(fmt.Sprintf("%v%v", acctPendingTransfer, msgId))
}

func acctIsPendingTransfer(keyBytes []byte) bool {
	return (len(keyBytes) >= acctPendingTransferLen+acctMinMsgIdLen) && (string(keyBytes[0:acctPendingTransferLen]) == acctPendingTransfer)
}

// This is called by the accountant on start up to reload pending transfers.
func (d *Database) AcctGetData(logger *zap.Logger) ([]*common.MessagePublication, error) {
	pendingTransfers := []*common.MessagePublication{}
	var err error

	if err = d.convertOldTransfersToNewFormat(logger); err != nil {
		return pendingTransfers, fmt.Errorf("failed to convert old pending transfers to the new format: %w", err)
	}

	{
		prefixBytes := []byte(acctPendingTransfer)
		err = d.db.View(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.PrefetchSize = 10
			it := txn.NewIterator(opts)
			defer it.Close()
			for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
				item := it.Item()
				key := item.Key()
				val, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}

				if acctIsPendingTransfer(key) {
					var pt common.MessagePublication
					err := json.Unmarshal(val, &pt)
					if err != nil {
						logger.Error("failed to unmarshal pending transfer for key", zap.String("key", string(key[:])), zap.Error(err))
						continue
					}

					pendingTransfers = append(pendingTransfers, &pt)
				} else {
					return fmt.Errorf("failed to load accountant pending transfer, unexpected key '%s'", string(key))
				}
			}

			return nil
		})
	}

	return pendingTransfers, err
}

func (d *Database) AcctStorePendingTransfer(msg *common.MessagePublication) error {
	b, _ := json.Marshal(msg)

	err := d.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(acctPendingTransferMsgID(msg.MessageIDString()), b); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to commit accountant pending transfer for tx %s: %w", msg.MessageIDString(), err)
	}

	return nil
}

func (d *Database) AcctDeletePendingTransfer(msgId string) error {
	key := acctPendingTransferMsgID(msgId)
	if err := d.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(key)
		return err
	}); err != nil {
		return fmt.Errorf("failed to delete accountant pending transfer for tx %s: %w", msgId, err)
	}

	return nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// The code below here is used to read and convert old Pending transfers. Once the db has been migrated away from those, this can be deleted.
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// OldMessagePublication is used to unmarshal old JSON which has the TxHash rather than the TxID.
type OldMessagePublication struct {
	TxHash    ethCommon.Hash
	Timestamp time.Time

	Nonce            uint32
	Sequence         uint64
	ConsistencyLevel uint8
	EmitterChain     vaa.ChainID
	EmitterAddress   vaa.Address
	Payload          []byte
	IsReobservation  bool
	Unreliable       bool
}

func (msg *OldMessagePublication) UnmarshalJSON(data []byte) error {
	type Alias OldMessagePublication
	aux := &struct {
		Timestamp int64
		*Alias
	}{
		Alias: (*Alias)(msg),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	msg.Timestamp = time.Unix(aux.Timestamp, 0)
	return nil
}

// convertOldToNew converts an OldMessagePublication to a MessagePublication.
func convertOldToNew(old *OldMessagePublication) *common.MessagePublication {
	return &common.MessagePublication{
		TxID:             old.TxHash.Bytes(),
		Timestamp:        old.Timestamp,
		Nonce:            old.Nonce,
		Sequence:         old.Sequence,
		EmitterChain:     old.EmitterChain,
		EmitterAddress:   old.EmitterAddress,
		Payload:          old.Payload,
		ConsistencyLevel: old.ConsistencyLevel,
		IsReobservation:  old.IsReobservation,
		Unreliable:       old.Unreliable,
	}
}

// convertOldTransfersToNewFormat loads any pending transfers in the old format, writes them in the new format and deletes the old ones.
func (d *Database) convertOldTransfersToNewFormat(logger *zap.Logger) error {
	pendingTransfers := []*common.MessagePublication{}
	prefixBytes := []byte(acctOldPendingTransfer)
	err := d.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			key := item.Key()
			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			if acctIsOldPendingTransfer(key) {
				var pt OldMessagePublication
				err := json.Unmarshal(val, &pt)
				if err != nil {
					return fmt.Errorf("failed to unmarshal old pending transfer for key '%s': %w", string(key), err)
				}

				pendingTransfers = append(pendingTransfers, convertOldToNew(&pt))
			} else {
				return fmt.Errorf("failed to convert old accountant pending transfer, unexpected key '%s'", string(key))
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(pendingTransfers) != 0 {
		for _, pt := range pendingTransfers {
			logger.Info("converting old pending transfer to new format", zap.String("msgId", pt.MessageIDString()))
			if err := d.AcctStorePendingTransfer(pt); err != nil {
				return fmt.Errorf("failed to convert old pending transfer for key [%v]: %w", pt, err)
			}
		}

		for _, pt := range pendingTransfers {
			key := acctOldPendingTransferMsgID(pt.MessageIDString())
			logger.Info("deleting old pending transfer", zap.String("msgId", pt.MessageIDString()), zap.String("key", string(key)))
			if err := d.db.Update(func(txn *badger.Txn) error {
				err := txn.Delete(key)
				return err
			}); err != nil {
				return fmt.Errorf("failed to delete old pending transfer for key [%v]: %w", pt, err)
			}
		}
	}

	return nil
}
