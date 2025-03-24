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
	StoreBlackhole(p *common.PendingMessage) error
	StorePending(p *common.PendingMessage) error
	DeleteBlackhole(p *common.PendingMessage) error
	DeletePending(p *common.PendingMessage) error
}

type NotaryDB struct {
	db *badger.DB
}

// Define prefixes used to isolate different message publications stored in the database.
const (
	delayedPrefix   = "NOTARY:PENDING:V1:"
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
	ErrUpdateErr    = errors.New("notary: could not store msg")
	ErrMarshalErr   = errors.New("notary: could not marshal data")
	ErrUnmarshalErr = errors.New("notary: could not unmarshal data")
	ErrViewErr      = errors.New("notary: error when reading from database")
)

func (d *NotaryDB) StoreDelayed(p *common.PendingMessage) error {
	b, marshalErr := p.MarshalBinary()

	if marshalErr != nil {
		return errors.Join(
			ErrMarshalErr,
			marshalErr,
		)
	}

	if updateErr := d.update(pendingKey(p), b); updateErr != nil {
		return errors.Join(
			ErrUpdateErr,
			updateErr,
		)
	}

	return nil
}

func (d *NotaryDB) StoreBlackhole(m *common.MessagePublication) error {
	b, marshalErr := m.MarshalBinary()

	if marshalErr != nil {
		return errors.Join(
			ErrMarshalErr,
			marshalErr,
		)
	}

	if updateErr := d.update(blackholeKey(m), b); updateErr != nil {
		return errors.Join(
			ErrUpdateErr,
			updateErr,
		)
	}
	return nil
}

func (d *NotaryDB) DeletePending(p *common.PendingMessage) error {
	return d.deleteEntry(pendingKey(p))
}

func (d *NotaryDB) DeleteBlackholed(m *common.MessagePublication) error {
	return d.deleteEntry(blackholeKey(m))
}

type DBLoadResult struct {
	Delayed    []*common.PendingMessage
	Blackholed []*common.MessagePublication
}

const (
	defaultResultCapacity = 10
)

func (d *NotaryDB) LoadAll() (*DBLoadResult, error) {
	result := DBLoadResult{
		Delayed:    make([]*common.PendingMessage, defaultResultCapacity),
		Blackholed: make([]*common.MessagePublication, defaultResultCapacity),
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
						ErrUnmarshalErr,
						unmarshalErr,
					)
				}
				result.Blackholed = append(result.Blackholed, &msgPub)
			case Delayed:
				var pMsg common.PendingMessage
				unmarshalErr := pMsg.UnmarshalBinary(data)
				if unmarshalErr != nil {
					return errors.Join(
						ErrUnmarshalErr,
						unmarshalErr,
					)
				}
				result.Delayed = append(result.Delayed, &pMsg)
			default:
				return errors.New("unknown data type")
			}

		}
		return nil
	})

	if viewErr != nil {
		return nil, errors.Join(
			ErrViewErr,
			viewErr,
		)
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
		return fmt.Errorf("failed to store data for key %s: %w", key, updateErr)
	}

	return nil
}

func (d *NotaryDB) deleteEntry(key []byte) error {
	if updateErr := d.db.Update(func(txn *badger.Txn) error {
		deleteErr := txn.Delete(key)
		return deleteErr
	}); updateErr != nil {
		return fmt.Errorf("failed to delete entry with key %x: %w", key, updateErr)
	}

	return nil
}

// pendingKey returns a unique prefix for pending message publications to be stored in the Notary's database.
func pendingKey(p *common.PendingMessage) []byte {
	return key(delayedPrefix, p.Msg.MessageIDString())
}

// pendingKey returns a unique prefix for blackholed message publications to be stored in the Notary's database.
func blackholeKey(m *common.MessagePublication) []byte {
	return key(blackholePrefix, m.MessageIDString())
}

// key returns a unique prefix for message publication to be stored in the Notary's database.
func key(prefix string, msgID string) []byte {
	return []byte(fmt.Sprintf("%v%v", prefix, msgID))
}
