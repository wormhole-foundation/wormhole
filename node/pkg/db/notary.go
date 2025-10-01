// SECURITY: The calling code is responsible for handling mutex operations when
// working with this package.
package db

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/dgraph-io/badger/v3"
	"go.uber.org/zap"
)

type NotaryDBInterface interface {
	StoreBlackholed(m *common.MessagePublication) error
	StoreDelayed(p *common.PendingMessage) error
	DeleteBlackholed(msgID []byte) (*common.MessagePublication, error)
	DeleteDelayed(msgID []byte) (*common.PendingMessage, error)
	LoadAll(logger *zap.Logger) (*NotaryLoadResult, error)
}

// NotaryDB is a wrapper struct for a database connection.
// Its main purpose is to provide some separation from the Notary's functionality
// and the general functioning of db.Database
type NotaryDB struct {
	db *badger.DB
}

func NewNotaryDB(dbConn *badger.DB) *NotaryDB {
	return &NotaryDB{
		db: dbConn,
	}
}

// Define prefixes used to isolate different message publications stored in the database.
const (
	delayedPrefix   = "NOTARY:DELAY:V1:"
	blackholePrefix = "NOTARY:BLACKHOLE:V1:"
)

// The type of data stored in the Notary's database.
type dataType string

const (
	Unknown    dataType = "unknown"
	Delayed    dataType = "delayed"
	Blackholed dataType = "blackholed"
)

var (
	ErrMarshal   = errors.New("notary: marshal")
	ErrUnmarshal = errors.New("notary: unmarshal")
)

// Operation represents a database operation type
type Operation string

const (
	OpRead   Operation = "read"
	OpUpdate Operation = "update"
	OpDelete Operation = "delete"
)

type DBError struct {
	Op  Operation
	Key []byte
	Err error
}

func (e *DBError) Unwrap() error {
	return e.Err
}

func (e *DBError) Error() string {
	return fmt.Sprintf("notary database: %s key: %x error: %v", e.Op, e.Key, e.Err)
}

func (d *NotaryDB) StoreDelayed(p *common.PendingMessage) error {
	b, marshalErr := p.MarshalBinary()

	if marshalErr != nil {
		return errors.Join(ErrMarshal, marshalErr)
	}

	key := delayKey(p.Msg.MessageID())
	if updateErr := d.update(key, b); updateErr != nil {
		return &DBError{Op: OpUpdate, Key: key, Err: updateErr}
	}

	return nil
}

func (d *NotaryDB) StoreBlackholed(m *common.MessagePublication) error {
	b, marshalErr := m.MarshalBinary()

	if marshalErr != nil {
		return errors.Join(ErrMarshal, marshalErr)
	}

	key := blackholeKey(m.MessageID())
	if updateErr := d.update(key, b); updateErr != nil {
		return &DBError{Op: OpUpdate, Key: key, Err: updateErr}
	}
	return nil
}

// DeleteDelayed deletes a delayed message from the database and returns the value that was deleted.
func (d *NotaryDB) DeleteDelayed(msgID []byte) (*common.PendingMessage, error) {
	deleted, err := d.deleteEntry(delayKey(msgID))
	if err != nil {
		return nil, err
	}

	var pendingMsg common.PendingMessage
	unmarshalErr := pendingMsg.UnmarshalBinary(deleted)
	if unmarshalErr != nil {
		return nil, errors.Join(
			ErrUnmarshal,
			unmarshalErr,
		)
	}

	// Sanity check that the message ID matches the one that was deleted.
	if !bytes.Equal(pendingMsg.Msg.MessageID(), msgID) {
		return &pendingMsg, errors.New("notary: delete pending message from notary database: removed message publication had different message ID compared to query")
	}

	return &pendingMsg, nil
}

// DeleteBlackholed deletes a blackholed message from the database and returns the value that was deleted.
func (d *NotaryDB) DeleteBlackholed(msgID []byte) (*common.MessagePublication, error) {
	deleted, err := d.deleteEntry(blackholeKey(msgID))
	if err != nil {
		return nil, err
	}

	var msgPub common.MessagePublication
	unmarshalErr := msgPub.UnmarshalBinary(deleted)
	if unmarshalErr != nil {
		return nil, errors.Join(
			ErrUnmarshal,
			unmarshalErr,
		)
	}

	// Sanity check that the message ID matches the one that was deleted.
	if !bytes.Equal(msgPub.MessageID(), msgID) {
		return &msgPub, errors.New("notary: delete blackholed message from notary database: removed message publication had different message ID compared to query")
	}

	return &msgPub, nil
}

type NotaryLoadResult struct {
	Delayed    []*common.PendingMessage
	Blackholed []*common.MessagePublication
}

// LoadAll retrieves all keys from the database.
func (d *NotaryDB) LoadAll(logger *zap.Logger) (*NotaryLoadResult, error) {
	result := NotaryLoadResult{
		Delayed:    make([]*common.PendingMessage, 0),
		Blackholed: make([]*common.MessagePublication, 0),
	}
	viewErr := d.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()
			data, copyErr := item.ValueCopy(nil)
			if copyErr != nil {
				return copyErr
			}

			switch dbDataType(key) {
			case Blackholed:
				var msgPub common.MessagePublication
				unmarshalErr := msgPub.UnmarshalBinary(data)
				if unmarshalErr != nil {
					return errors.Join(
						ErrUnmarshal,
						unmarshalErr,
					)
				}
				result.Blackholed = append(result.Blackholed, &msgPub)
			case Delayed:
				var pMsg common.PendingMessage
				unmarshalErr := pMsg.UnmarshalBinary(data)
				if unmarshalErr != nil {
					return errors.Join(
						ErrUnmarshal,
						unmarshalErr,
					)
				}
				result.Delayed = append(result.Delayed, &pMsg)
			case Unknown:
				// The key-value store is shared across other modules and message types (e.g. Governor, Accountant).
				// If another key is discovered, just ignore it.
				logger.Debug("notary: load database ignoring unknown key type", zap.String("key", string(key)))
				continue
			}

		}
		return nil
	})

	if viewErr != nil {
		// No key provided here since the View function is iterating over every entry.
		return nil, &DBError{Op: OpRead, Err: viewErr}
	}

	return &result, nil
}

// dbDataType returns the data type for an entry in the database based on its key.
func dbDataType(key []byte) dataType {
	if strings.HasPrefix(string(key), blackholePrefix) {
		return Blackholed
	}
	if strings.HasPrefix(string(key), delayedPrefix) {
		return Delayed
	}
	return Unknown

}

func (d *NotaryDB) update(key []byte, data []byte) error {
	updateErr := d.db.Update(func(txn *badger.Txn) error {
		if setErr := txn.Set(key, data); setErr != nil {
			return setErr
		}
		return nil
	})

	if updateErr != nil {
		return &DBError{Op: OpUpdate, Key: key, Err: updateErr}
	}

	return nil
}

// deleteEntry deletes a key-value pair from the database and returns the value that was deleted.
func (d *NotaryDB) deleteEntry(key []byte) ([]byte, error) {
	var deletedValue []byte

	if updateErr := d.db.Update(func(txn *badger.Txn) error {
		// Get the item first
		item, getErr := txn.Get(key)
		if getErr != nil {
			return getErr
		}

		// Copy the value before deleting
		valueCopy, copyErr := item.ValueCopy(nil)
		if copyErr != nil {
			return copyErr
		}
		deletedValue = valueCopy

		// Now delete the key
		deleteErr := txn.Delete(key)
		return deleteErr
	}); updateErr != nil {
		return nil, &DBError{Op: OpDelete, Key: key, Err: updateErr}
	}

	if len(deletedValue) == 0 {
		return nil, &DBError{Op: OpDelete, Key: key, Err: errors.New("notary: delete operation did not return a value")}
	}

	return deletedValue, nil
}

// delayKey returns a unique prefix for pending messages to be stored in the Notary's database.
func delayKey(msgID []byte) []byte {
	return key(delayedPrefix, string(msgID))
}

// blackholeKey returns a unique prefix for blackholed message publications to be stored in the Notary's database.
func blackholeKey(msgID []byte) []byte {
	return key(blackholePrefix, string(msgID))
}

// key returns a unique prefix for different data types stored in the Notary's database.
func key(prefix string, msgID string) (key []byte) {
	return fmt.Appendf(key, "%v%v", prefix, msgID)
}
