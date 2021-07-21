package common

type BridgeWatcher interface {
	WatchMessages(events chan *MessagePublication) error
}
