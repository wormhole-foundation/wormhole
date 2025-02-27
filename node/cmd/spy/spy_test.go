package spy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	spyv1 "github.com/certusone/wormhole/node/pkg/proto/spy/v1"
	ipfslog "github.com/ipfs/go-log/v2"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

var govEmitter = vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}

// govAddress is the string representation of the govEmitter Address.
// Leaving here in case it is helpful for future tests.
// const govAddress = "0000000000000000000000000000000000000000000000000000000000000004"

// helper method for *vaa.VAA creation
func getVAA(chainID vaa.ChainID, emitterAddr vaa.Address) *vaa.VAA {
	var payload = []byte{97, 97, 97, 97, 97, 97}

	vaa := &vaa.VAA{
		Version:          vaa.SupportedVAAVersion,
		GuardianSetIndex: uint32(1),
		Signatures:       nil,
		Timestamp:        time.Unix(0, 0),
		Nonce:            1,
		Sequence:         1,
		ConsistencyLevel: uint8(32),
		EmitterChain:     chainID,
		EmitterAddress:   emitterAddr,
		Payload:          payload,
	}

	return vaa
}

// wait for the server to establish a client subscription before returning.
func waitForClientSubscriptionInit(server *spyServer) {
	for {
		server.subsSignedVaaMu.Lock()
		subs := len(server.subsSignedVaa)
		server.subsSignedVaaMu.Unlock()

		time.Sleep(time.Millisecond * 10)

		if subs > 0 {
			return
		}
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

// Tests a subscription to spySever with no filters will return message(s)
func TestSpyHandleGossipVAA(t *testing.T) {
	ctx, conn, client := grpcClientSetup(t)
	defer conn.Close()

	vaaToSend := getVAA(vaa.ChainIDEthereum, govEmitter)

	req := &spyv1.SubscribeSignedVAARequest{Filters: []*spyv1.FilterEntry{}}

	stream, err := client.SubscribeSignedVAA(ctx, req)
	if err != nil {
		t.Fatalf("SubscribeSignedVAA failed: %v", err)
	}

	doneCh := make(chan bool)
	go func() {
		defer close(doneCh)
		// receive is a blocking call, it will keep receiving/looping until the pipe breaks.
		signedVAA, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			t.Log("the SignedVAA stream has closed, err == io.EOF. going to break.")
			t.Fail()
			return
		}
		if err != nil {
			t.Log("SubscribeSignedVAA returned an error.")
			t.Fail()
			return
		}
		parsedRes, err := vaa.Unmarshal(signedVAA.VaaBytes)
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
	}()
	waitForClientSubscriptionInit(mockedSpyServer)

	vaaBytes, err := vaaToSend.Marshal()
	if err != nil {
		t.Fatal("failed marshaling VAA to bytes")
	}
	err = mockedSpyServer.PublishSignedVAA(vaaBytes)
	if err != nil {
		t.Fatal("failed HandleGossipVAA")
	}

	<-doneCh
}

// Tests spySever's implementation of the spyv1.EmitterFilter
func TestSpyHandleEmitterFilter(t *testing.T) {
	ctx, conn, client := grpcClientSetup(t)
	defer conn.Close()

	vaaToSend := getVAA(vaa.ChainIDEthereum, govEmitter)
	vaaBytes, err := vaaToSend.Marshal()
	if err != nil {
		t.Fatal("failed marshaling VAA to bytes")
	}
	// create an EmitterFilter request
	emitterFilter := &spyv1.EmitterFilter{
		ChainId:        publicrpcv1.ChainID(vaa.ChainIDEthereum),
		EmitterAddress: govEmitter.String(),
	}
	filterEntryEmitter := &spyv1.FilterEntry_EmitterFilter{EmitterFilter: emitterFilter}
	emitterFilterEnvelope := &spyv1.FilterEntry{Filter: filterEntryEmitter}
	emitterFilterReq := &spyv1.SubscribeSignedVAARequest{Filters: []*spyv1.FilterEntry{emitterFilterEnvelope}}

	emitterFilterStream, err := client.SubscribeSignedVAA(ctx, emitterFilterReq)
	if err != nil {
		t.Fatalf("SubscribeSignedVAA failed: %v", err)
	}

	doneCh := make(chan bool)
	go func() {
		defer close(doneCh)
		// receive is a blocking call, it will keep receiving/looping until the pipe breaks.
		signedVAA, err := emitterFilterStream.Recv()
		if errors.Is(err, io.EOF) {
			t.Log("the SignedVAA stream has closed, err == io.EOF. going to break.")
			t.Fail()
			return
		}
		if err != nil {
			t.Log("SubscribeSignedVAA returned an error.")
			t.Fail()
			return
		}
		_, err = vaa.Unmarshal(signedVAA.VaaBytes)
		if err != nil {
			t.Log("failed unmarshaling VAA from response")
			t.Fail()
			return
		}
		if !bytes.Equal(signedVAA.VaaBytes, vaaBytes) {
			t.Log("vaaBytes of the response does not match what was sent")
			t.Fail()
			return
		}
	}()
	waitForClientSubscriptionInit(mockedSpyServer)

	// should not be sent to us by the server
	// everything passes the filter except for chainID
	msg1 := getVAA(vaa.ChainIDSolana, govEmitter)
	msg1Bytes, err := msg1.Marshal()
	if err != nil {
		t.Fatal("failed marshaling VAA to bytes")
	}
	err = mockedSpyServer.PublishSignedVAA(msg1Bytes)
	if err != nil {
		t.Fatal("failed to publish signed VAA")
	}
	// should not be sent to us by the server
	// everything passes the filter except for emitterAddress
	differentEmitter := vaa.Address{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	msg2 := getVAA(vaa.ChainIDEthereum, differentEmitter)
	msg2Bytes, err := msg2.Marshal()
	if err != nil {
		t.Fatal("failed marshaling VAA to bytes")
	}
	err = mockedSpyServer.PublishSignedVAA(msg2Bytes)
	if err != nil {
		t.Fatal("failed to publish signed VAA")
	}

	// passes the filter - should be sent back to us by the server
	err = mockedSpyServer.PublishSignedVAA(vaaBytes)
	if err != nil {
		t.Fatal("failed to publish signed VAA")
	}

	<-doneCh
}
