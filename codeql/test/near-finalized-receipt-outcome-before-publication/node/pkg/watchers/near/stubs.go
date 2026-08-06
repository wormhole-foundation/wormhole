package near

import "context"

type Result struct{}

func (Result) Get(string) Result { return Result{} }
func (Result) Array() []Result { return nil }
func (Result) Exists() bool { return true }
func (Result) String() string { return "" }

type Header struct{}
type Logger struct{}
type Job struct {
	blockHash string
	isReobservation bool
}
type Finalizer struct{}
type OtherFinalizer struct{}
type MessagePublication struct{}

func (Finalizer) isFinalized(*Logger, context.Context, string) (Header, bool) { return Header{}, true }
func (Finalizer) setFinalized(string) {}
func (OtherFinalizer) isFinalized(*Logger, context.Context, string) (Header, bool) { return Header{}, true }

type Watcher struct {
	finalizer Finalizer
	msgC chan *MessagePublication
}

func parseReceipts() []Result { return nil }

func (e *Watcher) processWormholeLog(_ *Logger, _ context.Context, _ *Job, _ Header, _ string, _ Result) error {
	observation := &MessagePublication{}
	e.msgC <- observation
	return nil
}
