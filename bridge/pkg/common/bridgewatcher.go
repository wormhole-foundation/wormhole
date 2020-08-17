package common

type BridgeWatcher interface {
	WatchLockups(events chan *ChainLock) error
}
