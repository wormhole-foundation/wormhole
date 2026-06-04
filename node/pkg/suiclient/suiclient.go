package suiclient

import (
	"context"
	"time"
)

const (
	SuiRPCMainnet = "fullnode.mainnet.sui.io:443"
	SuiRPCTestnet = "fullnode.testnet.sui.io:443"
	SuiRPCDevnet  = "fullnode.devnet.sui.io:443"
)

// Field-mask path constants for use with the Get* methods on SuiClient. The
// values match the protobuf field names of ExecutedTransaction, Object, and
// Checkpoint respectively. Only fields actually exposed by the corresponding
// wrapper struct are listed here; the underlying field-mask machinery accepts
// any string, so callers needing a sub-message path (e.g. "contents.value")
// can pass the literal directly.
const (
	TransactionFieldDigest     = "digest"
	TransactionFieldEvents     = "events"
	TransactionFieldCheckpoint = "checkpoint"
	TransactionFieldTimestamp  = "timestamp"
)

const (
	ObjectFieldBcs                 = "bcs"
	ObjectFieldObjectID            = "object_id"
	ObjectFieldVersion             = "version"
	ObjectFieldDigest              = "digest"
	ObjectFieldObjectType          = "object_type"
	ObjectFieldContents            = "contents"
	ObjectFieldPreviousTransaction = "previous_transaction"
	ObjectFieldStorageRebate       = "storage_rebate"
	ObjectFieldBalance             = "balance"
)

const (
	CheckpointFieldSequenceNumber = "sequence_number"
	CheckpointFieldDigest         = "digest"
)

// SuiObject holds the flat-primitive fields of a Sui object. A field is
// populated only when the caller requested it via the field mask AND the
// upstream node returned it; otherwise the field is nil/empty.
//
// Callers MUST nil-check every field they read. The Get* methods do not
// enforce per-field presence — that responsibility belongs to the caller.
type SuiObject struct {
	ObjectID            *string
	Version             *uint64
	Digest              *string
	ObjectType          *string
	BcsType             *string // from `bcs.name`: whole-object BCS type tag
	BcsBytes            []byte  // from `bcs.value`: whole-object BCS bytes
	ContentsType        *string // from `contents.name`: Move struct type tag
	ContentsBytes       []byte  // from `contents.value`: Move struct BCS bytes
	PreviousTransaction *string
	StorageRebate       *uint64
	Balance             *uint64
}

type SuiEvent struct {
	PackageID         string
	TransactionModule string
	Sender            string
	EventType         string
	BcsType           string
	BcsBytes          []byte
}

// SuiTransaction holds the flat-primitive fields of an executed transaction.
// A field is populated only when the caller requested it via the field mask
// AND the upstream node returned it; otherwise the field is nil/empty.
//
// Callers MUST nil-check every field they read. The Get* methods do not
// enforce per-field presence — that responsibility belongs to the caller.
type SuiTransaction struct {
	Digest     *string
	Events     []SuiEvent
	Checkpoint *uint64
	Timestamp  *time.Time
}

// SuiCheckpoint holds the flat-primitive fields of a checkpoint. A field is
// populated only when the caller requested it via the field mask AND the
// upstream node returned it; otherwise the field is nil/empty.
//
// Callers MUST nil-check every field they read. The Get* methods do not
// enforce per-field presence — that responsibility belongs to the caller.
type SuiCheckpoint struct {
	SequenceNumber *uint64
	Digest         *string
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
	// GetObject fetches the latest version of `objectID`. `fields` is the
	// list of protobuf field paths to populate on the returned SuiObject;
	// see the ObjectField* constants. At least one field is required.
	//
	// Fields not requested (and fields requested but missing from the upstream
	// response) come back nil/empty. Callers MUST nil-check every field they
	// read — this method does not enforce per-field presence.
	GetObject(ctx context.Context, objectID string, fields []string) (SuiObject, error)

	// GetObjectAtVersion fetches `objectID` at the given version. A nil
	// `version` requests the latest version. `fields` is the list of
	// protobuf field paths to populate on the returned SuiObject; see the
	// ObjectField* constants. At least one field is required.
	//
	// Fields not requested (and fields requested but missing from the upstream
	// response) come back nil/empty. Callers MUST nil-check every field they
	// read — this method does not enforce per-field presence.
	GetObjectAtVersion(ctx context.Context, objectID string, version *uint64, fields []string) (SuiObject, error)

	// GetLatestCheckpoint fetches the most recent checkpoint. `fields` is the
	// list of protobuf field paths to populate on the returned SuiCheckpoint;
	// see the CheckpointField* constants. At least one field is required.
	//
	// Fields not requested (and fields requested but missing from the upstream
	// response) come back nil/empty. Callers MUST nil-check every field they
	// read — this method does not enforce per-field presence.
	GetLatestCheckpoint(ctx context.Context, fields []string) (SuiCheckpoint, error)

	// GetTransaction fetches the transaction identified by `digest`. `fields`
	// is the list of protobuf field paths to populate on the returned
	// SuiTransaction; see the TransactionField* constants. At least one field
	// is required.
	//
	// Fields not requested (and fields requested but missing from the upstream
	// response) come back nil/empty. Callers MUST nil-check every field they
	// read — this method does not enforce per-field presence.
	GetTransaction(ctx context.Context, digest string, fields []string) (SuiTransaction, error)

	// Subscribe to transaction events of type `eventType`
	SubscribeToTransactionEvent(ctx context.Context, eventType string, eventWriteChannel chan<- SuiEvent) (SuiSubscription, error)
	SubscribeToTransactionEvents(ctx context.Context, eventTypes []string, eventWriteChannel chan<- SuiEvent) (SuiSubscription, error)

	// Close the client
	Close() error
}
