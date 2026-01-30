package notary

import (
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"go.uber.org/zap"
)

// MockNotaryDB is a mock implementation of the NotaryDB interface.
// It returns nil for all operations, so it can be used to test the Notary's
// core logic but certain DB-related operations are not covered.
// Where possible, these should be tested in the Notary database's own unit tests, not here.
type MockNotaryDB struct{}

func (md MockNotaryDB) StoreBlackholed(m *common.MessagePublication) error { return nil }
func (md MockNotaryDB) StoreDelayed(p *common.PendingMessage) error        { return nil }
func (md MockNotaryDB) DeleteBlackholed(msgID []byte) (*common.MessagePublication, error) {
	return nil, nil
}
func (md MockNotaryDB) DeleteDelayed(msgID []byte) (*common.PendingMessage, error) { return nil, nil }
func (md MockNotaryDB) LoadAll(l *zap.Logger) (*db.NotaryLoadResult, error)        { return nil, nil }

