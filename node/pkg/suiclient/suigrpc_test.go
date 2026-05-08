package suiclient

import (
	"context"
	"testing"

	pb "github.com/block-vision/sui-go-sdk/pb/sui/rpc/v2"
	"go.uber.org/zap"
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

		// Request GetObject if version is even
		if input_version%2 == 0 {
		} else {
			// Request GetObjectAtVersion if version is odd
			grpcClient.GetObjectAtVersion(context.Background(), input_objectId, input_version) //nolint:errcheck // The function returning an error is expected
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
		grpcClient.GetLatestCheckpointSN(context.Background()) //nolint:errcheck // The function returning an error is expected

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
		grpcClient.GetTransaction(context.Background(), "some digest") //nolint:errcheck // The function returning an error is expected
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
