package common

type BridgeWatcher interface {
	WatchLockups(events chan *MessagePublication) error
}
