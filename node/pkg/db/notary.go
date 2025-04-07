// TODO explain the meaning of pending and blackholed.
// SECURITY: The calling code is responsible for handling mutex operations when
// working with this package.
package db

import (
	"errors"
	"fmt"
	"strings"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/dgraph-io/badger/v3"
)

type NotaryDBInterface interface {
	StoreBlackhole(m *common.MessagePublication) error
	StoreDelayed(p *common.PendingMessage) error
	DeleteBlackholed(m *common.MessagePublication) error
	DeleteDelayed(p *common.PendingMessage) error
	LoadAll() (*NotaryLoadResult, error)
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
	delayedPrefix   = "NOTARY:DELAYED:V1:"
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
		return ErrMarshal
	}

	key := delayedKey(p)
	if updateErr := d.update(key, b); updateErr != nil {
		return &DBError{Op: OpUpdate, Key: key, Err: updateErr}
	}

	return nil
}

func (d *NotaryDB) StoreBlackhole(m *common.MessagePublication) error {
	b, marshalErr := m.MarshalBinary()

	if marshalErr != nil {
		return ErrMarshal
	}

	key := blackholeKey(m)
	if updateErr := d.update(key, b); updateErr != nil {
		return &DBError{Op: OpUpdate, Key: key, Err: updateErr}
	}
	return nil
}

func (d *NotaryDB) DeleteDelayed(p *common.PendingMessage) error {
	return d.deleteEntry(delayedKey(p))
}

func (d *NotaryDB) DeleteBlackholed(m *common.MessagePublication) error {
	return d.deleteEntry(blackholeKey(m))
}

type NotaryLoadResult struct {
	Delayed    []*common.PendingMessage
	Blackholed []*common.MessagePublication
}

const (
	defaultResultCapacity = 10
)

func (d *NotaryDB) LoadAll() (*NotaryLoadResult, error) {
	result := NotaryLoadResult{
		Delayed:    make([]*common.PendingMessage, 0, defaultResultCapacity),
		Blackholed: make([]*common.MessagePublication, 0, defaultResultCapacity),
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
			default:
				return fmt.Errorf("unknown data type for key: %x", key)
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

func (d *NotaryDB) deleteEntry(key []byte) error {
	if updateErr := d.db.Update(func(txn *badger.Txn) error {
		deleteErr := txn.Delete(key)
		return deleteErr
	}); updateErr != nil {
		return &DBError{Op: OpDelete, Key: key, Err: updateErr}
	}

	return nil
}

// delayedKey returns a unique prefix for pending messages to be stored in the Notary's database.
func delayedKey(p *common.PendingMessage) []byte {
	return key(delayedPrefix, p.Msg.MessageIDString())
}

// blackholeKey returns a unique prefix for blackholed message publications to be stored in the Notary's database.
func blackholeKey(m *common.MessagePublication) []byte {
	return key(blackholePrefix, m.MessageIDString())
}

// key returns a unique prefix for different data types stored in the Notary's database.
func key(prefix string, msgID string) (key []byte) {
	return fmt.Appendf(key, "%v%v", prefix, msgID)
}
