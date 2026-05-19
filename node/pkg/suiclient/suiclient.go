package suiclient

import (
	"context"
)

const (
	SuiRPCMainnet = "fullnode.mainnet.sui.io:443"
	SuiRPCTestnet = "fullnode.testnet.sui.io:443"
	SuiRPCDevnet  = "fullnode.devnet.sui.io:443"
)

type SuiObject struct {
	ID         string
	ObjectType string
	// NOTE: This *should* match the ObjectType, but since the API is capable of returning both, both are
	// stored in the event that a future update breaks things.
	BcsType  string
	BcsBytes []byte
}

type SuiEvent struct {
	PackageID         string
	TransactionModule string
	Sender            string
	EventType         string
	BcsType           string
	BcsBytes          []byte
}

type SuiObjectChange struct{}

type SuiTransaction struct {
	Digest        string
	ObjectChanges []SuiObjectChange
	Events        []SuiEvent
}

type SuiSubscription struct {
	// Channel for communicating errors during streaming.
	err chan error
	// Closed when the subscription's background goroutine has exited.
	done chan struct{}
	// Context cancellation function to stop the subscription
	ctxCancel context.CancelFunc
}

func (sub *SuiSubscription) Err() <-chan error {
	return sub.err
}

// Done returns a channel that is closed when the subscription's background
// goroutine has fully exited (either after Unsubscribe() or a stream error).
func (sub *SuiSubscription) Done() <-chan struct{} {
	return sub.done
}

func (sub *SuiSubscription) Unsubscribe() {
	sub.ctxCancel()
}

type SuiClient interface {
	// Get the latest version of object `objectID`
	GetObject(ctx context.Context, objectID string) (SuiObject, error)
	// Get version `version` of object `objectID`
	GetObjectAtVersion(ctx context.Context, objectID string, version uint64) (SuiObject, error)
	// Get the latest checkpoint sequence number
	GetLatestCheckpointSN(ctx context.Context) (uint64, error)
	// Get the transaction data for `digest`
	GetTransaction(ctx context.Context, digest string) (SuiTransaction, error)

	// Subscribe to events of type `eventType`
	SubscribeToEvent(ctx context.Context, eventType string, eventWriteChannel chan<- SuiEvent) (SuiSubscription, error)
	SubscribeToEvents(ctx context.Context, eventTypes []string, eventWriteChannel chan<- SuiEvent) (SuiSubscription, error)

	// Close the client
	Close() error
}
