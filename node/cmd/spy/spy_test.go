package spy

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	spyv1 "github.com/certusone/wormhole/node/pkg/proto/spy/v1"
	ipfslog "github.com/ipfs/go-log/v2"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const vaaNonce = uint32(1)
const vaaSequence = uint64(1)
const tx = "39c5940431b1507c2a496e945dfb6b6760771fb3c19f2531c5976decc16814ca"

var govEmitter = vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}

// govAddress is the string representation of the govEmitter Address.
// Leaving here in case it is helpful for future tests.
// const govAddress = "0000000000000000000000000000000000000000000000000000000000000004"

func getTxBytes(tx string) []byte {
	bytes, err := hex.DecodeString(tx)
	if err != nil {
		panic("failed decoding Tx string to bytes")
	}
	return bytes
}

// helper method for *vaa.VAA creation
func getVAA(chainID vaa.ChainID, emitterAddr vaa.Address, nonce uint32) *vaa.VAA {
	var payload = []byte{97, 97, 97, 97, 97, 97}

	vaa := &vaa.VAA{
		Version:          vaa.SupportedVAAVersion,
		GuardianSetIndex: uint32(1),
		Signatures:       nil,
		Timestamp:        time.Unix(0, 0),
		Nonce:            nonce,
		Sequence:         vaaSequence,
		ConsistencyLevel: uint8(32),
		EmitterChain:     chainID,
		EmitterAddress:   emitterAddr,
		Payload:          payload,
	}

	return vaa
}

// helper method for *vaa.BatchVAA creation
func getBatchVAA(chainID vaa.ChainID, _ []byte, nonce uint32, emitterAddr vaa.Address) *vaa.BatchVAA {
	v := getVAA(chainID, emitterAddr, nonce)
	obs := &vaa.Observation{
		Index:       uint8(0),
		Observation: v,
	}
	var obsList = []*vaa.Observation{}
	obsList = append(obsList, obs)

	batchVAA := &vaa.BatchVAA{
		Version:          vaa.BatchVAAVersion,
		GuardianSetIndex: uint32(1),
		Signatures:       nil,
		Observations:     obsList,
	}

	return batchVAA
}

// helper method for *gossipv1.SignedBatchVAAWithQuorum creation
func getBatchVAAQuorumMessage(chainID vaa.ChainID, txID []byte, nonce uint32, emitterAddr vaa.Address) *gossipv1.SignedBatchVAAWithQuorum {
	batchVaa := getBatchVAA(chainID, txID, nonce, emitterAddr)
	vaaBytes, err := batchVaa.Marshal()
	if err != nil {
		panic("failed marshaling batchVAA.")
	}

	msg := &gossipv1.SignedBatchVAAWithQuorum{
		BatchVaa: vaaBytes,
		ChainId:  uint32(chainID),
		TxId:     txID,
		Nonce:    nonce,
	}
	return msg
}

// wait for the server to establish a client subscription before returning.
func waitForClientSubscriptionInit(server *spyServer) {
	for {
		server.subsAllVaaMu.Lock()
		subs := len(server.subsAllVaa)
		server.subsAllVaaMu.Unlock()

		if subs > 0 {
			return
		}
		time.Sleep(time.Millisecond * 10)
	}
}

//
// unit tests
//

// returns a gossip message and the pertinant values for filtering
func batchMsg() (msg *gossipv1.SignedBatchVAAWithQuorum, chainID vaa.ChainID, txID []byte, nonce uint32, emitterAddr vaa.Address) {
	chainID = vaa.ChainIDEthereum
	txID = getTxBytes(tx)
	nonce = vaaNonce
	emitterAddr = govEmitter
	msg = getBatchVAAQuorumMessage(chainID, txID, nonce, emitterAddr)
	return msg, chainID, txID, nonce, emitterAddr
}

// happy path of VAA and spyv1.FilterEntry comparison
func TestSpyTransactionIdMatches(t *testing.T) {
	batchMsg, chainID, txID, _, _ := batchMsg()

	filter := &spyv1.BatchFilter{
		ChainId: publicrpcv1.ChainID(chainID),
		TxId:    txID,
	}
	matches := TransactionIdMatches(batchMsg, filter)
	if !matches {
		t.FailNow()
	}
}

// filter does not match the VAA
func TestSpyTransactionIdMatchesNoMatch(t *testing.T) {
	batchMsg, chainID, _, _, _ := batchMsg()
	filter := &spyv1.BatchFilter{
		ChainId: publicrpcv1.ChainID(chainID),
		TxId:    []byte{1, 1, 1, 1}, // anything that does not match txBytes
	}
	matches := TransactionIdMatches(batchMsg, filter)
	if matches {
		// should not return true
		t.FailNow()
	}
}

// happy path
func TestSpyBatchMatchesFilter(t *testing.T) {
	batchMsg, chainID, txID, nonce, _ := batchMsg()

	// filter without nonce
	filter := &spyv1.BatchFilter{
		ChainId: publicrpcv1.ChainID(chainID),
		TxId:    txID,
	}
	matches := BatchMatchesFilter(batchMsg, filter)
	if !matches {
		t.FailNow()
	}

	// filter with nonce
	nonceFilter := &spyv1.BatchFilter{
		ChainId: publicrpcv1.ChainID(chainID),
		TxId:    txID,
		Nonce:   nonce,
	}
	nonceMatches := BatchMatchesFilter(batchMsg, nonceFilter)
	if !nonceMatches {
		t.FailNow()
	}
}

// filter is valid, but does not match the VAA
func TestSpyBatchMatchesFilterNoMatch(t *testing.T) {
	batchMsg, chainID, txID, _, _ := batchMsg()

	// different chainID
	solFilter := &spyv1.BatchFilter{
		ChainId: publicrpcv1.ChainID(vaa.ChainIDSolana),
		TxId:    txID,
	}
	solMatches := BatchMatchesFilter(batchMsg, solFilter)
	if solMatches {
		// should not return true
		t.FailNow()
	}

	// different transaction identifier
	txFilter := &spyv1.BatchFilter{
		ChainId: publicrpcv1.ChainID(chainID),
		TxId:    []byte{1, 1, 1, 1}, // anything that does not match txBytes
	}
	txMatches := BatchMatchesFilter(batchMsg, txFilter)
	if txMatches {
		// should not return true
		t.FailNow()
	}
}

//
// gRPC server setup for e2e tests
//

// for protobuf gRPC testing
const bufSize = 1024 * 1024

var lis *bufconn.Listener
var mockedSpyServer *spyServer

// mock the rpc server so it can run in CI without needing ports and such.
func init() {
	// setup the spyServer as it is setup in prod, only mock what is necessary.
	logger := ipfslog.Logger("wormhole-spy-mocked-in-ci").Desugar()

	// only print PANIC logs from the server's logger
	_ = ipfslog.SetLogLevel("wormhole-spy-mocked-in-ci", "PANIC")

	lis = bufconn.Listen(bufSize)

	grpcServer := common.NewInstrumentedGRPCServer(logger, common.GrpcLogDetailFull)

	mockedSpyServer = newSpyServer(logger)
	spyv1.RegisterSpyRPCServiceServer(grpcServer, mockedSpyServer)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			fmt.Println("server error", err)
			logger.Fatal("Server exited with error", zap.Error(err))
		}
	}()
}

// creates a network connection in memory rather than with the host's network
// stack, for CI. See https://pkg.go.dev/google.golang.org/grpc/test/bufconn for info.
func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

//
// e2e gRPC server tests
// assert clients can connect, supply filters, and responses are as expected
//

// subscription tests - make sure the spy allows subscribing with and without filters

// helper function that setups a gRPC client for spySever
func grpcClientSetup(t *testing.T) (context.Context, *grpc.ClientConn, spyv1.SpyRPCServiceClient) {
	ctx := context.Background()
	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), creds)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	client := spyv1.NewSpyRPCServiceClient(conn)

	return ctx, conn, client
}

// Tests creating a subscription to spyServer with no filters succeeds
func TestSpySubscribeNoFilter(t *testing.T) {
	ctx, conn, client := grpcClientSetup(t)
	defer conn.Close()

	req := &spyv1.SubscribeSignedVAARequest{Filters: []*spyv1.FilterEntry{}}

	_, err := client.SubscribeSignedVAA(ctx, req)
	if err != nil {
		t.Fatalf("SubscribeSignedVAA failed: %v", err)
	}
	// just check the subscription can be created without returning an error
}

// Tests creating a subscription to spyServer with a spyv1.EmitterFilter succeeds
func TestSpySubscribeEmitterFilter(t *testing.T) {
	ctx, conn, client := grpcClientSetup(t)
	defer conn.Close()

	// create EmitterFilter
	emitterFilter := &spyv1.EmitterFilter{ChainId: publicrpcv1.ChainID(vaa.ChainIDEthereum), EmitterAddress: ""}
	filterEntryEmitter := &spyv1.FilterEntry_EmitterFilter{EmitterFilter: emitterFilter}
	filter := &spyv1.FilterEntry{Filter: filterEntryEmitter}

	req := &spyv1.SubscribeSignedVAARequest{Filters: []*spyv1.FilterEntry{filter}}

	_, err := client.SubscribeSignedVAA(ctx, req)
	if err != nil {
		t.Fatalf("SubscribeSignedVAA failed: %v", err)
	}
	// just check the subscription can be created without returning an error
}

// Tests creating a subscription to spyServer with a spyv1.BatchFilter succeeds
func TestSpySubscribeBatchFilter(t *testing.T) {
	ctx, conn, client := grpcClientSetup(t)
	defer conn.Close()

	batchFilter := &spyv1.BatchFilter{
		ChainId: publicrpcv1.ChainID(vaa.ChainIDEthereum),
		TxId:    getTxBytes(tx),
		Nonce:   vaaNonce,
	}
	filterEntryBatch := &spyv1.FilterEntry_BatchFilter{BatchFilter: batchFilter}
	filter := &spyv1.FilterEntry{Filter: filterEntryBatch}
	req := &spyv1.SubscribeSignedVAAByTypeRequest{Filters: []*spyv1.FilterEntry{filter}}

	_, err := client.SubscribeSignedVAAByType(ctx, req)
	if err != nil {
		t.Fatalf("SubscribeSignedVAAByType failed: %v", err)
	}
	// just check the subscription can be created without returning an error
}

// Tests creating a subscription to spyServer with a spyv1.BatchTransactionFilter succeeds
func TestSpySubscribeBatchTransactionFilter(t *testing.T) {

	ctx, conn, client := grpcClientSetup(t)
	defer conn.Close()

	batchTxFilter := &spyv1.BatchTransactionFilter{
		ChainId: publicrpcv1.ChainID(vaa.ChainIDEthereum),
		TxId:    getTxBytes(tx),
	}
	filterEntryBatchTx := &spyv1.FilterEntry_BatchTransactionFilter{
		BatchTransactionFilter: batchTxFilter,
	}
	filter := &spyv1.FilterEntry{Filter: filterEntryBatchTx}
	req := &spyv1.SubscribeSignedVAAByTypeRequest{Filters: []*spyv1.FilterEntry{filter}}

	_, err := client.SubscribeSignedVAAByType(ctx, req)
	if err != nil {
		t.Fatalf("SubscribeSignedVAAByType failed: %v", err)
	}
	// just check the subscription can be created without returning an error
}

// Tests a subscription to spySever with no filters will return message(s)
func TestSpyHandleGossipVAA(t *testing.T) {
	ctx, conn, client := grpcClientSetup(t)
	defer conn.Close()

	vaaToSend := getVAA(vaa.ChainIDEthereum, govEmitter, vaaNonce)

	req := &spyv1.SubscribeSignedVAAByTypeRequest{Filters: []*spyv1.FilterEntry{}}

	stream, err := client.SubscribeSignedVAAByType(ctx, req)
	if err != nil {
		t.Fatalf("SubscribeSignedVAA failed: %v", err)
	}

	doneCh := make(chan bool)
	go func() {
		defer close(doneCh)
		for {
			// recieve is a blocking call, it will keep recieving/looping until the pipe breaks.
			signedVAA, err := stream.Recv()
			if err == io.EOF {
				t.Log("the SignedVAA stream has closed, err == io.EOF. going to break.")
				t.Fail()
				return
			}
			if err != nil {
				t.Log("SubscribeSignedVAAByType returned an error.")
				t.Fail()
				return
			}

			vaaRes := signedVAA.GetVaaType()
			switch resType := vaaRes.(type) {
			case *spyv1.SubscribeSignedVAAByTypeResponse_SignedVaa:
				bytes := resType.SignedVaa.Vaa
				parsedRes, err := vaa.Unmarshal(bytes)
				if err != nil {
					t.Log("failed unmarshaling VAA from response")
					t.Fail()
					return
				}
				if parsedRes.MessageID() != vaaToSend.MessageID() {
					t.Log("parsedRes.MessageID() does not equal vaaToSend.MessageID()")
					t.Fail()
					return
				}
				// this test only expects one response, so return
				return
			}
		}
	}()
	waitForClientSubscriptionInit(mockedSpyServer)

	vaaBytes, err := vaaToSend.Marshal()
	if err != nil {
		t.Fatal("failed marshaling VAA to bytes")
	}
	msg := &gossipv1.SignedVAAWithQuorum{
		Vaa: vaaBytes,
	}
	err = mockedSpyServer.HandleGossipVAA(msg)
	if err != nil {
		t.Fatal("failed HandleGossipVAA")
	}

	<-doneCh
}

// Tests spySever's implementation of the spyv1.EmitterFilter
func TestSpyHandleGossipBatchVAAEmitterFilter(t *testing.T) {
	ctx, conn, client := grpcClientSetup(t)
	defer conn.Close()

	// This test creates a message (gossipMsg) that will be published, and a filter that will allow the
	// message through (emitterFilter). Then it subscribes to messages from the server (SubscribeSignedVAAByType),
	// and publishes the gossipMsg, along with some messages that do not pass the filter (msg1, msg2, etc).
	// This test passes if only the expected gossipMsg is recieved from the client stream (emitterFilterStream.Recv()).

	// gossipMsg will be published to the spyServer
	gossipMsg, chainID, txID, nonce, emitter := batchMsg()

	// create an EmitterFilter request
	emitterFilter := &spyv1.EmitterFilter{
		ChainId:        publicrpcv1.ChainID(chainID),
		EmitterAddress: emitter.String(),
	}
	filterEntryEmitter := &spyv1.FilterEntry_EmitterFilter{EmitterFilter: emitterFilter}
	emitterFilterEnvelope := &spyv1.FilterEntry{Filter: filterEntryEmitter}
	emitterFilterReq := &spyv1.SubscribeSignedVAAByTypeRequest{Filters: []*spyv1.FilterEntry{emitterFilterEnvelope}}

	emitterFilterStream, err := client.SubscribeSignedVAAByType(ctx, emitterFilterReq)
	if err != nil {
		t.Fatalf("SubscribeSignedVAAByType failed: %v", err)
	}

	doneCh := make(chan bool)
	go func() {
		defer close(doneCh)
		for {
			// recieve is a blocking call, it will keep recieving/looping until the pipe breaks.
			res, err := emitterFilterStream.Recv()
			if err == io.EOF {
				t.Log("the SignedVAA stream has closed, err == io.EOF. going to break.")
				t.Fail()
				return
			}
			if err != nil {
				t.Log("SubscribeSignedVAAByType returned an error.")
				t.Fail()
				return
			}
			vaaRes := res.GetVaaType()
			switch resType := vaaRes.(type) {
			case *spyv1.SubscribeSignedVAAByTypeResponse_SignedBatchVaa:
				vaaBytes := resType.SignedBatchVaa.BatchVaa

				// just unmarshal to smoke test
				_, err := vaa.UnmarshalBatch(vaaBytes)
				if err != nil {
					t.Log("failed unmarshaling BatchVAA from response")
					t.Fail()
					return
				}

				// check the response is the expected vaa (one that passes the supplied filter)
				if !bytes.Equal(resType.SignedBatchVaa.TxId, txID) {
					t.Log("Got a VAA from the server stream not matching the supplied filter.")
					t.Fail()
				}

				// check to make sure msg1 did not make it through
				if resType.SignedBatchVaa.ChainId != uint32(chainID) {
					t.Fail()
				}

				// check that the VAA we got is exactly what we expect
				if !bytes.Equal(vaaBytes, gossipMsg.BatchVaa) {
					t.Log("vaaBytes of the response does not match what was sent")
					t.Fail()
				}

				return
			case *spyv1.SubscribeSignedVAAByTypeResponse_SignedVaa:
				t.Log("got SubscribeSignedVAAByTypeResponse_SignedVaa")
				return
			}
		}
	}()
	waitForClientSubscriptionInit(mockedSpyServer)

	// should not be sent to us by the server
	// everything passes the filter except for chainID
	msg1 := getBatchVAAQuorumMessage(vaa.ChainIDSolana, txID, nonce, emitter)
	err = mockedSpyServer.HandleGossipBatchVAA(msg1)
	if err != nil {
		t.Fatal("failed HandleGossipBatchVAA for msg1")
	}

	// should not be sent to us by the server
	// everything passes the filter except except for emitterAddress
	differentEmitter := vaa.Address{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	msg2 := getBatchVAAQuorumMessage(chainID, txID, nonce, differentEmitter)
	err = mockedSpyServer.HandleGossipBatchVAA(msg2)
	if err != nil {
		t.Fatal("failed HandleGossipBatchVAA for msg2")
	}

	// passes the filter - should be sent back to us by the server
	err = mockedSpyServer.HandleGossipBatchVAA(gossipMsg)
	if err != nil {
		t.Fatal("failed HandleGossipBatchVAA for gossipMsg")
	}

	<-doneCh
}

// Tests spySever's implementation of the spyv1.BatchTransactionFilter
func TestSpyHandleGossipBatchVAABatchTxFilter(t *testing.T) {
	ctx, conn, client := grpcClientSetup(t)
	defer conn.Close()

	// This test creates a message (gossipMsg) that will be published, and a filter that will allow the
	// message through (batchTxFilter). Then it subscribes to messages from the server (SubscribeSignedVAAByType),
	// and publishes the gossipMsg, along with some messages that do not pass the filter (msg1, msg2, etc).
	// This test passes if only the expected gossipMsg is recieved from the client stream (batchTxStream.Recv()).

	// gossipMsg will be published to the spyServer
	// the other values returned are used for setting up the filters we expect this message to pass
	gossipMsg, chainID, txID, nonce, emitter := batchMsg()

	// create a BatchTransactionFilter request
	batchTxFilter := &spyv1.BatchTransactionFilter{
		ChainId: publicrpcv1.ChainID(chainID),
		TxId:    txID,
	}
	filterEntryBatchTx := &spyv1.FilterEntry_BatchTransactionFilter{
		BatchTransactionFilter: batchTxFilter,
	}
	batchTxFilterEnvelope := &spyv1.FilterEntry{Filter: filterEntryBatchTx}
	batchTxReq := &spyv1.SubscribeSignedVAAByTypeRequest{Filters: []*spyv1.FilterEntry{batchTxFilterEnvelope}}

	batchTxStream, err := client.SubscribeSignedVAAByType(ctx, batchTxReq)
	if err != nil {
		t.Fatalf("SubscribeSignedVAAByType failed: %v", err)
	}

	doneCh := make(chan bool)
	go func() {
		defer close(doneCh)
		for {
			// recieve is a blocking call, it will keep recieving/looping until the pipe breaks.
			res, err := batchTxStream.Recv()
			if err == io.EOF {
				t.Log("the SignedVAA stream has closed, err == io.EOF. going to break.")
				t.Fail()
				return
			}
			if err != nil {
				t.Log("SubscribeSignedVAAByType returned an error.")
				t.Fail()
				return
			}

			vaaRes := res.GetVaaType()
			switch resType := vaaRes.(type) {
			case *spyv1.SubscribeSignedVAAByTypeResponse_SignedBatchVaa:
				vaaBytes := resType.SignedBatchVaa.BatchVaa

				// just unmarshal to smoke test
				_, err := vaa.UnmarshalBatch(vaaBytes)
				if err != nil {
					t.Log("failed unmarshaling BatchVAA from response")
					t.Fail()
					return
				}

				// check the response is the expected vaa (one that passes the supplied filter)
				if !bytes.Equal(resType.SignedBatchVaa.TxId, txID) {
					t.Log("Got a VAA from the server stream not matching the supplied filter.")
					t.Fail()
				}

				if resType.SignedBatchVaa.ChainId != uint32(chainID) {
					t.Fail()
				}

				// check that the VAA we got is exactly what we expect
				if !bytes.Equal(vaaBytes, gossipMsg.BatchVaa) {
					t.Log("vaaBytes of the response does not match what was sent")
					t.Fail()
				}

				return
			case *spyv1.SubscribeSignedVAAByTypeResponse_SignedVaa:
				t.Log("got SubscribeSignedVAAByTypeResponse_SignedVaa")
				return
			}
		}
	}()
	waitForClientSubscriptionInit(mockedSpyServer)

	// should not be sent to us by the server
	// everything passes the filter except for chainID
	msg1 := getBatchVAAQuorumMessage(vaa.ChainIDSolana, txID, nonce, emitter)
	err = mockedSpyServer.HandleGossipBatchVAA(msg1)
	if err != nil {
		t.Fatal("failed HandleGossipBatchVAA for msg1")
	}

	// should not be sent to us by the server
	// everything passes the filter except except for txID
	differentTx := []byte{1, 1, 1, 1} // anything that does not match txBytes
	msg2 := getBatchVAAQuorumMessage(chainID, differentTx, nonce, emitter)
	err = mockedSpyServer.HandleGossipBatchVAA(msg2)
	if err != nil {
		t.Fatal("failed HandleGossipBatchVAA for msg2")
	}

	// passes the filter - should be sent back to us by the server
	err = mockedSpyServer.HandleGossipBatchVAA(gossipMsg)
	if err != nil {
		t.Fatal("failed HandleGossipBatchVAA for gossipMsg")
	}

	<-doneCh
}

// Tests spySever's implementation of the spyv1.BatchFilter
func TestSpyHandleGossipBatchVAABatchFilter(t *testing.T) {
	ctx, conn, client := grpcClientSetup(t)
	defer conn.Close()

	// This test creates a message (gossipMsg) that will be published, and a filter that will allow the
	// message through (batchFilter). Then it subscribes to messages from the server (SubscribeSignedVAAByType),
	// and publishes the gossipMsg, along with some messages that do not pass the filter (msg1, msg2, etc).
	// This test passes if only the expected gossipMsg is recieved from the client stream (batchStream.Recv()).

	// gossipMsg will be published to the spyServer
	// the other values returned are used for setting up the filters we expect this message to pass
	gossipMsg, chainID, txID, nonce, emitter := batchMsg()

	// create a BatchTransactionFilter request
	batchFilter := &spyv1.BatchFilter{
		ChainId: publicrpcv1.ChainID(chainID),
		TxId:    txID,
		Nonce:   nonce,
	}
	filterEntryBatch := &spyv1.FilterEntry_BatchFilter{
		BatchFilter: batchFilter,
	}
	batchFilterEnvelope := &spyv1.FilterEntry{Filter: filterEntryBatch}
	batchReq := &spyv1.SubscribeSignedVAAByTypeRequest{Filters: []*spyv1.FilterEntry{batchFilterEnvelope}}

	batchStream, err := client.SubscribeSignedVAAByType(ctx, batchReq)
	if err != nil {
		t.Fatalf("SubscribeSignedVAAByType failed: %v", err)
	}

	doneCh := make(chan bool)
	go func() {
		defer close(doneCh)
		for {
			// recieve is a blocking call, it will keep recieving/looping until the pipe breaks.
			res, err := batchStream.Recv()
			if err == io.EOF {
				t.Log("the SignedVAA stream has closed, err == io.EOF. going to break.")
				t.Fail()
				return
			}
			if err != nil {
				t.Log("SubscribeSignedVAAByType returned an error.")
				t.Fail()
				return
			}

			vaaRes := res.GetVaaType()
			switch resType := vaaRes.(type) {
			case *spyv1.SubscribeSignedVAAByTypeResponse_SignedBatchVaa:
				vaaBytes := resType.SignedBatchVaa.BatchVaa

				// just unmarshal to smoke test
				_, err := vaa.UnmarshalBatch(vaaBytes)
				if err != nil {
					t.Log("failed unmarshaling BatchVAA from response")
					t.Fail()
					return
				}

				if resType.SignedBatchVaa.Nonce != nonce {
					t.Fail()
				}

				// check the response is the expected vaa (one that passes the supplied filter)
				if !bytes.Equal(resType.SignedBatchVaa.TxId, txID) {
					t.Log("Got a VAA from the server stream not matching the supplied filter.")
					t.Fail()
				}

				if resType.SignedBatchVaa.ChainId != uint32(chainID) {
					t.Fail()
				}

				// check that the VAA we got is exactly what we expect
				if !bytes.Equal(vaaBytes, gossipMsg.BatchVaa) {
					t.Log("vaaBytes of the response does not match what was sent")
					t.Fail()
				}

				return
			case *spyv1.SubscribeSignedVAAByTypeResponse_SignedVaa:
				t.Log("got SubscribeSignedVAAByTypeResponse_SignedVaa")
				return
			}
		}
	}()
	waitForClientSubscriptionInit(mockedSpyServer)

	// should not be sent to us by the server
	// everything passes the filter except for chainID
	msg1 := getBatchVAAQuorumMessage(vaa.ChainIDSolana, txID, nonce, emitter)
	err = mockedSpyServer.HandleGossipBatchVAA(msg1)
	if err != nil {
		t.Fatal("failed HandleGossipBatchVAA for msg1")
	}

	// should not be sent to us by the server
	// everything passes the filter except except for txID
	differentTx := []byte{1, 1, 1, 1} // anything that does not match txBytes
	msg2 := getBatchVAAQuorumMessage(chainID, differentTx, nonce, emitter)
	err = mockedSpyServer.HandleGossipBatchVAA(msg2)
	if err != nil {
		t.Fatal("failed HandleGossipBatchVAA for msg2")
	}

	// should not be sent to us by the server
	// everything passes the filter except except for nonce
	differentNonce := uint32(8888) // anything that does not match txBytes
	msg3 := getBatchVAAQuorumMessage(chainID, txID, differentNonce, emitter)
	err = mockedSpyServer.HandleGossipBatchVAA(msg3)
	if err != nil {
		t.Fatal("failed HandleGossipBatchVAA for msg3")
	}

	// passes the filter - should be sent back to us by the server
	err = mockedSpyServer.HandleGossipBatchVAA(gossipMsg)
	if err != nil {
		t.Fatal("failed HandleGossipBatchVAA for gossipMsg")
	}

	<-doneCh
}
