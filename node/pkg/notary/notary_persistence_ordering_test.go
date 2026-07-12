// Regression tests for the persist-first ordering in delay(), blackhole(),
// removeDelayed(), and removeBlackholed().
//
// The Notary's whitepaper (whitepapers/0015_notary.md) guarantees that
// delayed and blackholed entries survive guardian restart. That guarantee
// is only enforceable if disk persistence is the source of truth for state
// mutations. Prior to the persist-first refactor, the four listed functions
// mutated in-memory state ahead of the database write and returned the disk
// error without rolling back, producing divergent in-memory / on-disk views
// that only reconverged on restart via loadFromDB() -- silently dropping
// the un-persisted entries in the process.
//
// Each test injects a database failure and asserts that in-memory state is
// NOT mutated. If a future refactor reintroduces the in-memory-first
// ordering, these tests will fail loudly.
//
// Original disclosure: Immunefi report #76768, filed 2026-05-05 by
// @Venator (Matthew S. Hintz).

package notary

import (
	"errors"
	"sync"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// injectingDB is a NotaryDBInterface implementation that returns the exact
// *db.DBError wrapper used by the production db.NotaryDB when BadgerDB
// surfaces "too many open files" or similar I/O failures. Tests toggle the
// per-method failure flags before each Notary call.
type injectingDB struct {
	mu               sync.Mutex
	failStoreDelayed bool
	failStoreBlack   bool
	failDeleteDelay  bool
	failDeleteBlack  bool
	delayed          map[string]*common.PendingMessage
	blackholed       map[string]*common.MessagePublication
}

func newInjectingDB() *injectingDB {
	return &injectingDB{
		delayed:    make(map[string]*common.PendingMessage),
		blackholed: make(map[string]*common.MessagePublication),
	}
}

func (i *injectingDB) StoreDelayed(p *common.PendingMessage) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.failStoreDelayed {
		return &db.DBError{
			Op:  db.OpUpdate,
			Key: []byte("NOTARY:DELAY:V1:" + string(p.Msg.MessageID())),
			Err: errors.New("simulated badger write failure: too many open files"),
		}
	}
	i.delayed[string(p.Msg.MessageID())] = p
	return nil
}

func (i *injectingDB) StoreBlackholed(m *common.MessagePublication) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.failStoreBlack {
		return &db.DBError{
			Op:  db.OpUpdate,
			Key: []byte("NOTARY:BLACKHOLE:V1:" + string(m.MessageID())),
			Err: errors.New("simulated badger write failure: too many open files"),
		}
	}
	i.blackholed[string(m.MessageID())] = m
	return nil
}

func (i *injectingDB) DeleteDelayed(msgID []byte) (*common.PendingMessage, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.failDeleteDelay {
		return nil, &db.DBError{
			Op:  db.OpUpdate,
			Key: []byte("NOTARY:DELAY:V1:" + string(msgID)),
			Err: errors.New("simulated badger delete failure: too many open files"),
		}
	}
	p, ok := i.delayed[string(msgID)]
	if !ok {
		return nil, nil
	}
	delete(i.delayed, string(msgID))
	return p, nil
}

func (i *injectingDB) DeleteBlackholed(msgID []byte) (*common.MessagePublication, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.failDeleteBlack {
		return nil, &db.DBError{
			Op:  db.OpUpdate,
			Key: []byte("NOTARY:BLACKHOLE:V1:" + string(msgID)),
			Err: errors.New("simulated badger delete failure: too many open files"),
		}
	}
	m, ok := i.blackholed[string(msgID)]
	if !ok {
		return nil, nil
	}
	delete(i.blackholed, string(msgID))
	return m, nil
}

func (i *injectingDB) LoadAll(_ *zap.Logger) (*db.NotaryLoadResult, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	delayed := make([]*common.PendingMessage, 0, len(i.delayed))
	for _, p := range i.delayed {
		delayed = append(delayed, p)
	}
	blackholed := make([]*common.MessagePublication, 0, len(i.blackholed))
	for _, m := range i.blackholed {
		blackholed = append(blackholed, m)
	}
	return &db.NotaryLoadResult{Delayed: delayed, Blackholed: blackholed}, nil
}

func newTestNotaryWithInjectingDB(t *testing.T, idb *injectingDB) *Notary {
	t.Helper()
	n := makeTestNotary(t)
	n.database = idb
	return n
}

// TestPersistOrdering_DelayLeavesInMemoryEmptyOnDBFailure asserts that
// n.delay() does NOT mutate the in-memory delayed queue when the disk write
// fails. This is the add-side regression guard for delay().
func TestPersistOrdering_DelayLeavesInMemoryEmptyOnDBFailure(t *testing.T) {
	idb := newInjectingDB()
	n := newTestNotaryWithInjectingDB(t, idb)
	msg := makeUniqueMessagePublication(t)

	idb.mu.Lock()
	idb.failStoreDelayed = true
	idb.mu.Unlock()

	err := n.delay(msg, DefaultDelay)
	require.Error(t, err, "delay() must propagate the disk-write error")

	require.False(t, n.IsDelayed(msg),
		"in-memory delayed queue must NOT contain the entry after a failed disk write; "+
			"a persist-first ordering keeps in-memory and on-disk views consistent across restart")

	idb.mu.Lock()
	require.Empty(t, idb.delayed,
		"on-disk delayed store must be empty after a failed StoreDelayed")
	idb.mu.Unlock()
}

// TestPersistOrdering_BlackholeLeavesInMemoryEmptyOnDBFailure asserts that
// n.blackhole() does NOT mutate the in-memory blackholed set when the disk
// write fails. This is the add-side regression guard for blackhole().
func TestPersistOrdering_BlackholeLeavesInMemoryEmptyOnDBFailure(t *testing.T) {
	idb := newInjectingDB()
	n := newTestNotaryWithInjectingDB(t, idb)
	msg := makeUniqueMessagePublication(t)

	idb.mu.Lock()
	idb.failStoreBlack = true
	idb.mu.Unlock()

	err := n.blackhole(msg)
	require.Error(t, err, "blackhole() must propagate the disk-write error")

	require.False(t, n.IsBlackholed(msg.MessageID()),
		"in-memory blackholed set must NOT contain the entry after a failed disk write; "+
			"a persist-first ordering keeps in-memory and on-disk views consistent across restart")

	idb.mu.Lock()
	require.Empty(t, idb.blackholed,
		"on-disk blackholed store must be empty after a failed StoreBlackholed")
	idb.mu.Unlock()
}

// TestPersistOrdering_RemoveBlackholedKeepsInMemoryOnDBFailure asserts that
// n.removeBlackholed() does NOT remove the entry from in-memory when the
// disk-delete fails. This is the remove-side regression guard for
// removeBlackholed() and closes the pre-restart kill-switch-bypass window.
func TestPersistOrdering_RemoveBlackholedKeepsInMemoryOnDBFailure(t *testing.T) {
	idb := newInjectingDB()
	n := newTestNotaryWithInjectingDB(t, idb)
	msg := makeUniqueMessagePublication(t)

	// Seed both in-memory and on-disk with the blackholed entry.
	require.NoError(t, n.blackhole(msg), "seed blackhole must succeed")
	require.True(t, n.IsBlackholed(msg.MessageID()), "seed blackhole must be in-memory")

	// Now flip the delete-failure flag and try to remove.
	idb.mu.Lock()
	idb.failDeleteBlack = true
	idb.mu.Unlock()

	_, err := n.removeBlackholed(msg.MessageID())
	require.Error(t, err, "removeBlackholed must propagate the disk-delete error")

	require.True(t, n.IsBlackholed(msg.MessageID()),
		"in-memory blackholed set MUST still contain the entry after a failed disk-delete; "+
			"otherwise operator's kill switch is silently bypassable until next restart")

	idb.mu.Lock()
	_, stillOnDisk := idb.blackholed[string(msg.MessageID())]
	require.True(t, stillOnDisk,
		"on-disk blackholed record must still be present after a failed disk delete")
	idb.mu.Unlock()
}

// TestPersistOrdering_RemoveDelayedKeepsInMemoryOnDBFailure asserts that
// n.removeDelayed() does NOT remove the entry from the in-memory delayed
// queue when the disk-delete fails.
func TestPersistOrdering_RemoveDelayedKeepsInMemoryOnDBFailure(t *testing.T) {
	idb := newInjectingDB()
	n := newTestNotaryWithInjectingDB(t, idb)
	msg := makeUniqueMessagePublication(t)

	require.NoError(t, n.delay(msg, DefaultDelay), "seed delay must succeed")
	require.True(t, n.IsDelayed(msg), "seed delayed entry must be in-memory")

	idb.mu.Lock()
	idb.failDeleteDelay = true
	idb.mu.Unlock()

	_, err := n.removeDelayed(msg.MessageID())
	require.Error(t, err, "removeDelayed must propagate the disk-delete error")

	require.True(t, n.IsDelayed(msg),
		"in-memory delayed queue MUST still contain the entry after a failed disk-delete")

	idb.mu.Lock()
	_, stillOnDisk := idb.delayed[string(msg.MessageID())]
	require.True(t, stillOnDisk,
		"on-disk delayed record must still be present after a failed disk delete")
	idb.mu.Unlock()
}

// TestPersistOrdering_HappyPathPersistsAndPopulates confirms that when the
// disk writes succeed, both in-memory and on-disk contain the entry. This
// is the honesty control for the failure-injection tests above.
func TestPersistOrdering_HappyPathPersistsAndPopulates(t *testing.T) {
	idb := newInjectingDB()
	n := newTestNotaryWithInjectingDB(t, idb)
	msg := makeUniqueMessagePublication(t)

	require.NoError(t, n.delay(msg, DefaultDelay))
	require.True(t, n.IsDelayed(msg))
	idb.mu.Lock()
	require.Len(t, idb.delayed, 1)
	idb.mu.Unlock()

	bhMsg := makeUniqueMessagePublication(t)
	require.NoError(t, n.blackhole(bhMsg))
	require.True(t, n.IsBlackholed(bhMsg.MessageID()))
	idb.mu.Lock()
	require.Len(t, idb.blackholed, 1)
	idb.mu.Unlock()
}
