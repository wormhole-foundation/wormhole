package governor

import "sync"

type ChainGovernor struct {
	mutex sync.Mutex
	rw    sync.RWMutex
}

func (gov *ChainGovernor) parseMsgAlreadyLocked(msg []byte) error { return nil }
func (gov *ChainGovernor) loadFromDBAlreadyLocked() error          { return nil }
