package db

import (
	"encoding/json"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/dgraph-io/badger/v3"

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

const acctOldPendingTransfer = "ACCT:PXFER:"
const acctOldPendingTransferLen = len(acctOldPendingTransfer)

const acctPendingTransfer = "ACCT:PXFER2:"
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
					return fmt.Errorf("unexpected accountant pending transfer key '%s'", string(key))
				}
			}

			return nil
		})
	}

	// See if we have any old format pending transfers.
	if err == nil {
		oldPendingTransfers := []*common.MessagePublication{}
		prefixBytes := []byte(acctOldPendingTransfer)
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

				if acctIsOldPendingTransfer(key) {
					pt, err := common.UnmarshalOldMessagePublicationBeforeIsReobservation(val)
					if err != nil {
						logger.Error("failed to unmarshal old pending transfer for key", zap.String("key", string(key[:])), zap.Error(err))
						continue
					}

					oldPendingTransfers = append(oldPendingTransfers, pt)
				} else {
					return fmt.Errorf("unexpected accountant pending transfer key '%s'", string(key))
				}
			}

			return nil
		})

		if err == nil && len(oldPendingTransfers) != 0 {
			pendingTransfers = append(pendingTransfers, oldPendingTransfers...)
			for _, pt := range oldPendingTransfers {
				logger.Info("updating format of database entry for pending vaa", zap.String("msgId", pt.MessageIDString()))
				err := d.AcctStorePendingTransfer(pt)
				if err != nil {
					return pendingTransfers, fmt.Errorf("failed to write new pending msg for key [%v]: %w", pt.MessageIDString(), err)
				}

				key := acctOldPendingTransferMsgID(pt.MessageIDString())
				if err := d.db.Update(func(txn *badger.Txn) error {
					err := txn.Delete(key)
					return err
				}); err != nil {
					return pendingTransfers, fmt.Errorf("failed to delete old pending msg for key [%v]: %w", pt.MessageIDString(), err)
				}
			}
		}
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
