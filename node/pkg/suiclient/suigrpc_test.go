package suiclient

import (
	"context"
	"errors"
	"io"
	"testing"

	pb "github.com/block-vision/sui-go-sdk/pb/sui/rpc/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

// A mock LedgerService client for testing. Each of the requests (GetObject, GetCheckpoint, GetTransaction) can
// be simulated by calling the appropriate `SetNext*Response` function, to return a prepared response.
type MockLedgerServiceClient struct {
	nextGetObjectResponse      *pb.GetObjectResponse
	nextGetCheckpointResponse  *pb.GetCheckpointResponse
	nextGetTransactionResponse *pb.GetTransactionResponse
}

func (m *MockLedgerServiceClient) SetNextGetObjectResponse(next *pb.GetObjectResponse) {
	m.nextGetObjectResponse = next
}

func (m *MockLedgerServiceClient) GetObject(ctx context.Context, req *pb.GetObjectRequest) (*pb.GetObjectResponse, error) {
	return m.nextGetObjectResponse, nil
}

func (m *MockLedgerServiceClient) SetNextGetCheckpointResponse(next *pb.GetCheckpointResponse) {
	m.nextGetCheckpointResponse = next
}

func (m *MockLedgerServiceClient) GetCheckpoint(ctx context.Context, req *pb.GetCheckpointRequest) (*pb.GetCheckpointResponse, error) {
	return m.nextGetCheckpointResponse, nil
}

func (m *MockLedgerServiceClient) SetNextGetTransactionResponse(next *pb.GetTransactionResponse) {
	m.nextGetTransactionResponse = next
}

func (m *MockLedgerServiceClient) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.GetTransactionResponse, error) {
	return m.nextGetTransactionResponse, nil
}

func FuzzSuiGrpcClientGetObject(f *testing.F) {

	ledgerService := &MockLedgerServiceClient{}

	grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

	corpusString := "random string"
	corpusUint := uint64(0)

	// Add a seed input for each property that can be nil
	f.Add(true, false, false, corpusString, false, corpusString, false, false, corpusString, []byte{0x41, 0x41}, corpusString, corpusUint)
	f.Add(false, true, false, corpusString, false, corpusString, false, false, corpusString, []byte{0x41, 0x41}, corpusString, corpusUint)
	f.Add(false, false, true, corpusString, false, corpusString, false, false, corpusString, []byte{0x41, 0x41}, corpusString, corpusUint)
	f.Add(false, false, false, corpusString, true, corpusString, false, false, corpusString, []byte{0x41, 0x41}, corpusString, corpusUint)
	f.Add(false, false, false, corpusString, false, corpusString, true, false, corpusString, []byte{0x41, 0x41}, corpusString, corpusUint)
	f.Add(false, false, false, corpusString, false, corpusString, false, true, corpusString, []byte{0x41, 0x41}, corpusString, corpusUint)

	// The following properties of a GetObjectResponse are acually used, and are fuzzed:
	// resp	*pb.GetObjectResponse
	//	.Object	*pb.Object
	//		.ObjectId 	*string
	//		.ObjectType	*string
	//		.Contents 	*pb,Bcs
	//			.Name	*string
	//			.Value	[]byte
	f.Fuzz(func(t *testing.T,
		// Response Object
		respNilOrNot bool, // determines if response itself is nil
		objectNilOrNot bool, // determines if resp.Object should be nil
		object_objectIdNilOrNot bool,
		object_objectId string,
		object_objectTypeNilOrNot bool,
		object_objectType string,
		object_contentsNilOrNot bool, // determines if resp.Object.Contents should be nil
		object_contents_nameNilOrNot bool,
		object_contents_name string,
		object_contents_value []byte,

		// GetObject input
		input_objectId string,
		input_version uint64,
	) {
		// This fuzz harness only checks for panics; returned errors are expected
		// and explicitly discarded via `_, _ = ...` below.
		resp := &pb.GetObjectResponse{}

		// set resp to nil or not
		if respNilOrNot {
			resp = nil
		} else {
			// set resp.Object to nil or not
			if objectNilOrNot {
				resp.Object = nil
			} else {
				resp.Object = &pb.Object{}

				// Set resp.Object.ObjectId
				if !object_objectIdNilOrNot {
					resp.Object.ObjectId = &object_objectId
				}

				// Set resp.Object.ObjectType
				if !object_objectTypeNilOrNot {
					resp.Object.ObjectType = &object_objectType
				}

				// set resp.Object.Contents to nil or not
				if object_contentsNilOrNot {
					resp.Object.Contents = nil
				} else {
					resp.Object.Contents = &pb.Bcs{}

					// Set resp.Object.Contents.Name
					if !object_contents_nameNilOrNot {
						resp.Object.Contents.Name = &object_contents_name
					}

					// Set resp.Object.Contents.Value
					if len(object_contents_value) == 0 || object_contents_value == nil {
						resp.Object.Contents.Value = nil
					} else {
						resp.Object.Contents.Value = object_contents_value
					}
				}

			}
		}

		ledgerService.SetNextGetObjectResponse(resp)

		fields := []string{
			ObjectFieldObjectID,
			ObjectFieldObjectType,
			ObjectFieldContents,
		}

		// Request GetObject if version is even
		if input_version%2 == 0 {
			_, _ = grpcClient.GetObject(context.Background(), input_objectId, fields)
		} else {
			// Request GetObjectAtVersion if version is odd
			_, _ = grpcClient.GetObjectAtVersion(context.Background(), input_objectId, &input_version, fields)
		}

		ledgerService.SetNextGetObjectResponse(nil)
	})
}

func FuzzSuiGrpcClientGetCheckpoint(f *testing.F) {

	ledgerService := &MockLedgerServiceClient{}
	grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

	// Add a seed input for each property that can be nil
	f.Add(true, false, uint64(0))
	f.Add(false, true, uint64(0))

	// Properties being used:
	// resp *pb.GetCheckpointResponse
	//	Checkpoint *pb.Checkpoint
	//		SequenceNumber *uint64
	f.Fuzz(func(t *testing.T,
		respNilOrNot bool,
		checkpointNilOrNot bool,
		sequenceNumber uint64,
	) {
		// This fuzz harness only checks for panics; returned errors are expected
		// and explicitly discarded via `_, _ = ...` below.
		resp := &pb.GetCheckpointResponse{}

		if respNilOrNot {
			resp = nil
		} else {
			if checkpointNilOrNot {
				resp.Checkpoint = nil
			} else {
				resp.Checkpoint = &pb.Checkpoint{}

				if sequenceNumber%2 == 0 {
					resp.Checkpoint.SequenceNumber = nil
				} else {
					resp.Checkpoint.SequenceNumber = &sequenceNumber
				}
			}
		}

		ledgerService.SetNextGetCheckpointResponse(resp)
		_, _ = grpcClient.GetLatestCheckpoint(context.Background(), []string{CheckpointFieldSequenceNumber})

		ledgerService.SetNextGetCheckpointResponse(nil)

	})
}

func FuzzSuiGrpcClientGetTransactionNoEvents(f *testing.F) {
	// The mock responses don't include events in the transaction. This is deliberate, since
	// the datatypes become unnecessarily complicated, and fuzzing the event parsing can be
	// done separately.

	ledgerService := &MockLedgerServiceClient{}
	grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, ledgerService, nil)

	// Add a seed input for each property that can be nil
	f.Add(true, false, false, "random string")
	f.Add(false, true, false, "random string")
	f.Add(false, false, true, "random string")

	f.Fuzz(func(t *testing.T,
		respNilOrNot bool,
		transactionNilOrNot bool,
		transaction_digestNilOrNot bool,
		transaction_digest string,
	) {
		// This fuzz harness only checks for panics; returned errors are expected
		// and explicitly discarded via `_, _ = ...` below.
		resp := &pb.GetTransactionResponse{}

		if respNilOrNot {
			resp = nil
		} else {
			if transactionNilOrNot {
				resp.Transaction = nil
			} else {
				resp.Transaction = &pb.ExecutedTransaction{}

				if !transaction_digestNilOrNot {
					resp.Transaction.Digest = &transaction_digest
				}
			}
		}

		ledgerService.SetNextGetTransactionResponse(resp)
		_, _ = grpcClient.GetTransaction(context.Background(), "some digest", []string{
			TransactionFieldDigest,
			TransactionFieldEvents,
		})
		ledgerService.SetNextGetTransactionResponse(nil)
	})
}

func FuzzExecutedTransactionToSuiTransaction(f *testing.F) {

	// This fuzzer accepts a number for each property within a pb.Event. The
	// maximum of the set is then used to determine how many events are created,
	// and for each `property` only `numProperty` amount of entries in the list
	// will be non-nil.

	u8_0 := uint8(0)
	u8_1 := uint8(1)

	// Add seed inputs for each case where a single property exists, but the others are nil.
	f.Add(u8_1, u8_0, u8_0, u8_0, u8_0, u8_0, u8_0)
	f.Add(u8_0, u8_1, u8_0, u8_0, u8_0, u8_0, u8_0)
	f.Add(u8_0, u8_0, u8_1, u8_0, u8_0, u8_0, u8_0)
	f.Add(u8_0, u8_0, u8_0, u8_1, u8_0, u8_0, u8_0)
	f.Add(u8_0, u8_0, u8_0, u8_0, u8_1, u8_0, u8_0)
	f.Add(u8_0, u8_0, u8_0, u8_0, u8_0, u8_1, u8_0)
	f.Add(u8_0, u8_0, u8_0, u8_0, u8_0, u8_0, u8_1)

	txDigest := "0xDigest"

	// Default values for properties
	defaultPackageId := "PackageId"
	defaultModule := "Module"
	defaultSender := "Sender"
	defaultEventType := "EventType"
	defaultContentsName := "Contents.Name"
	defaultContentsValue := []byte{0x13, 0x37}

	f.Fuzz(func(t *testing.T,
		numPackageIds uint8,
		numModules uint8,
		numSenders uint8,
		numEventTypes uint8,
		numContents uint8,
		numContentsName uint8,
		numContentsBcs uint8,
	) {

		entries := max(numPackageIds, numModules, numSenders, numEventTypes, numContents, numContentsName, numContentsBcs)

		grpcTransaction := &pb.ExecutedTransaction{
			Digest: &txDigest,
		}

		grpcTransaction.Events = &pb.TransactionEvents{}

		for idx := range entries {
			grpcEvent := &pb.Event{}

			if idx < numPackageIds {
				grpcEvent.PackageId = &defaultPackageId
			}

			if idx < numModules {
				grpcEvent.Module = &defaultModule
			}

			if idx < numSenders {
				grpcEvent.Sender = &defaultSender
			}

			if idx < numEventTypes {
				grpcEvent.EventType = &defaultEventType
			}

			if idx < numContents {
				grpcEvent.Contents = &pb.Bcs{}

				if idx < numContentsName {
					grpcEvent.Contents.Name = &defaultContentsName
				}

				if idx < numContentsBcs {
					grpcEvent.Contents.Value = defaultContentsValue
				}

			}

			grpcTransaction.Events.Events = append(grpcTransaction.Events.Events, grpcEvent)
		}

		grpcExecutedTransactionToSuiTransaction(grpcTransaction)

	})
}

// MockSubscribeCheckpointsStream is a mock server-streaming client for the Sui
// SubscribeCheckpoints RPC. Recv() returns each queued response in order, and once
// the queue is exhausted it returns recvErr. recvErr must be non-nil so that the
// subscription's background goroutine terminates deterministically during fuzzing.
type MockSubscribeCheckpointsStream struct {
	responses []*pb.SubscribeCheckpointsResponse
	idx       int
	recvErr   error
}

func (m *MockSubscribeCheckpointsStream) Recv() (*pb.SubscribeCheckpointsResponse, error) {
	if m.idx < len(m.responses) {
		resp := m.responses[m.idx]
		m.idx++
		return resp, nil
	}
	return nil, m.recvErr
}

// The remaining methods satisfy the grpc.ClientStream portion of the
// SubscriptionService_SubscribeCheckpointsClient interface. None of them are
// exercised by the subscription logic under test, so they are trivial stubs.
func (m *MockSubscribeCheckpointsStream) Header() (metadata.MD, error) { return nil, nil }
func (m *MockSubscribeCheckpointsStream) Trailer() metadata.MD         { return nil }
func (m *MockSubscribeCheckpointsStream) CloseSend() error             { return nil }
func (m *MockSubscribeCheckpointsStream) Context() context.Context     { return context.Background() }
func (m *MockSubscribeCheckpointsStream) SendMsg(_ any) error          { return nil }
func (m *MockSubscribeCheckpointsStream) RecvMsg(_ any) error          { return nil }

// MockSubscriptionServiceClient is a mock SubscriptionService client. SubscribeCheckpoints
// returns the configured stream and error, which allows both the stream-creation failure
// path and the streaming path of SubscribeToEvents to be exercised.
type MockSubscriptionServiceClient struct {
	nextStream pb.SubscriptionService_SubscribeCheckpointsClient
	nextError  error
}

func (m *MockSubscriptionServiceClient) SubscribeCheckpoints(ctx context.Context, req *pb.SubscribeCheckpointsRequest) (pb.SubscriptionService_SubscribeCheckpointsClient, error) {
	return m.nextStream, m.nextError
}

func FuzzSuiGrpcClientSubscribeToEvents(f *testing.F) {
	// Default values for event properties.
	txDigest := "0xDigest"
	defaultPackageId := "PackageId"
	defaultModule := "Module"
	defaultSender := "Sender"
	defaultEventType := "EventType"
	defaultContentsName := "Contents.Name"
	defaultContentsValue := []byte{0x13, 0x37}

	// Seed inputs covering: stream-creation failure, a nil checkpoint response, a nil
	// checkpoint, a fully-populated matching event, the single-event Subscribe variant,
	// and early unsubscription with multiple transactions.
	f.Add(true, false, false, false, false, false, uint8(0), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0))
	f.Add(false, false, false, true, false, false, uint8(1), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0))
	f.Add(false, false, false, false, true, false, uint8(1), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0))
	f.Add(false, false, false, false, false, true, uint8(1), uint8(1), uint8(1), uint8(1), uint8(1), uint8(1), uint8(1), uint8(1))
	f.Add(false, true, false, false, false, true, uint8(1), uint8(1), uint8(1), uint8(1), uint8(1), uint8(1), uint8(1), uint8(1))
	f.Add(false, false, true, false, false, true, uint8(3), uint8(2), uint8(2), uint8(2), uint8(2), uint8(2), uint8(2), uint8(2))

	// The structure of the events mirrors FuzzExecutedTransactionToSuiTransaction: the
	// maximum of the `num*` inputs determines how many events each transaction holds, and
	// for each property only `numProperty` of those events have that property set.
	f.Fuzz(func(t *testing.T,
		streamCreationFails bool, // SubscribeCheckpoints returns an error
		useSingleSubscribe bool, // call SubscribeToEvent instead of SubscribeToEvents
		unsubscribeEarly bool, // cancel the subscription context before draining
		respNil bool, // the streamed SubscribeCheckpointsResponse is nil
		checkpointNil bool, // the response's Checkpoint is nil
		matchEventType bool, // the subscribed event type matches the events' type
		numTransactions uint8,
		numPackageIds uint8,
		numModules uint8,
		numSenders uint8,
		numEventTypes uint8,
		numContents uint8,
		numContentsName uint8,
		numContentsBcs uint8,
	) {
		// Bound the work so a single fuzz input cannot create an unbounded number of events.
		const maxTransactions = 8
		const maxEventsPerTx = 16
		txCount := int(min(numTransactions, maxTransactions))
		entries := int(min(max(numPackageIds, numModules, numSenders, numEventTypes, numContents, numContentsName, numContentsBcs), maxEventsPerTx))

		// Build the checkpoint response that the mock stream will emit once.
		var resp *pb.SubscribeCheckpointsResponse
		if !respNil {
			resp = &pb.SubscribeCheckpointsResponse{}
			if !checkpointNil {
				checkpoint := &pb.Checkpoint{}
				for range txCount {
					grpcTx := &pb.ExecutedTransaction{
						Digest: &txDigest,
						Events: &pb.TransactionEvents{},
					}
					for idx := range entries {
						grpcEvent := &pb.Event{}

						if idx < int(numPackageIds) {
							grpcEvent.PackageId = &defaultPackageId
						}
						if idx < int(numModules) {
							grpcEvent.Module = &defaultModule
						}
						if idx < int(numSenders) {
							grpcEvent.Sender = &defaultSender
						}
						if idx < int(numEventTypes) {
							grpcEvent.EventType = &defaultEventType
						}
						if idx < int(numContents) {
							grpcEvent.Contents = &pb.Bcs{}

							if idx < int(numContentsName) {
								grpcEvent.Contents.Name = &defaultContentsName
							}
							if idx < int(numContentsBcs) {
								grpcEvent.Contents.Value = defaultContentsValue
							}
						}

						grpcTx.Events.Events = append(grpcTx.Events.Events, grpcEvent)
					}
					checkpoint.Transactions = append(checkpoint.Transactions, grpcTx)
				}
				resp.Checkpoint = checkpoint
			}
		}

		// Configure the mock subscription service.
		subscriptionService := &MockSubscriptionServiceClient{}
		if streamCreationFails {
			subscriptionService.nextError = errors.New("stream creation failed")
		} else {
			subscriptionService.nextStream = &MockSubscribeCheckpointsStream{
				responses: []*pb.SubscribeCheckpointsResponse{resp},
				// io.EOF terminates the subscription goroutine after the single response.
				recvErr: io.EOF,
			}
		}

		grpcClient := newSuiGrpcClientWithServices(zap.NewNop(), nil, nil, subscriptionService)

		// Buffer the channel generously so the subscription goroutine never blocks while
		// writing events. At most maxTransactions*maxEventsPerTx events can be produced.
		eventChan := make(chan SuiEvent, maxTransactions*maxEventsPerTx+1)

		eventTypes := []string{"non-matching-event-type"}
		if matchEventType {
			eventTypes = []string{defaultEventType}
		}

		var subscription SuiSubscription
		var err error
		if useSingleSubscribe {
			subscription, err = grpcClient.SubscribeToTransactionEvent(context.Background(), eventTypes[0], eventChan)
		} else {
			subscription, err = grpcClient.SubscribeToTransactionEvents(context.Background(), eventTypes, eventChan)
		}

		// When stream creation fails there is no background goroutine to wait on.
		if err != nil {
			return
		}

		if unsubscribeEarly {
			subscription.Unsubscribe()
		}

		// Wait for the subscription's background goroutine to fully exit. The error channel
		// is buffered, so the goroutine never blocks even though it is not drained here.
		<-subscription.Done()

		// Unsubscribe again to confirm it is safe to call after the goroutine has exited.
		subscription.Unsubscribe()
	})
}

func FuzzNewSuiGrpcClient(f *testing.F) {
	// grpc.NewClient is lazy: it validates the target and constructs the client without
	// dialing, so this exercises NewSuiGrpcClient and Close() with no network access.
	f.Add("fullnode.mainnet.sui.io:443")
	f.Add("localhost:443")
	f.Add("")
	f.Add(":::::")
	f.Add("dns:///example.com:443")

	f.Fuzz(func(t *testing.T, rpcURL string) {
		client, err := NewSuiGrpcClient(rpcURL, zap.NewNop())
		if err != nil {
			return
		}
		client.Close() //nolint:errcheck // The Close error is not relevant for the fuzz harness
	})
}
