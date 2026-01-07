package tss

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/stretchr/testify/require"
	tsscommon "github.com/xlabs/tss-common"
	"github.com/xlabs/tss-common/service/signer"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const bufSize = 1024 * 1024

type mockSignerServer struct {
	signer.UnimplementedSignerServer
	signRequests  chan *signer.SignRequest
	signResponses chan *signer.SignResponse
	publicData    *signer.PublicData
}

func (m *mockSignerServer) SignMessage(stream signer.Signer_SignMessageServer) error {
	// Handle sending responses
	go func() {
		for resp := range m.signResponses {
			if err := stream.Send(resp); err != nil {
				return
			}
		}
	}()

	// Handle receiving requests
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		select {
		case m.signRequests <- req:
		default:
			// Drop if full to avoid blocking test
		}
	}
}

func (m *mockSignerServer) GetPublicData(ctx context.Context, req *signer.PublicDataRequest) (*signer.PublicData, error) {
	if m.publicData == nil {
		return nil, errors.New("no public data")
	}

	return m.publicData, nil
}

func (m *mockSignerServer) VerifySignature(ctx context.Context, req *signer.VerifySignatureRequest) (*signer.VerifySignatureResponse, error) {
	return &signer.VerifySignatureResponse{IsValid: true}, nil
}

func TestSignerClient(t *testing.T) {
	a := require.New(t)

	lis, err := net.Listen("tcp", "localhost:0") // listen on a random available port
	a.NoError(err)

	s := grpc.NewServer()
	mock := &mockSignerServer{
		signRequests:  make(chan *signer.SignRequest, 10),
		signResponses: make(chan *signer.SignResponse, 10),
		publicData: &signer.PublicData{
			FrostPublicData: []byte("frost_key"),
			EcdsaPublicData: []byte("ecdsa_key"),
		},
	}
	signer.RegisterSignerServer(s, mock)

	go func() {
		if err := s.Serve(lis); err != nil {
			t.Errorf("Server exited with error: %v", err)
		}
	}()
	defer s.Stop()

	// Create client with bufconn dialer and insecure credentials
	client, err := NewSigner(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	a.NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect in background, ctx will be used for cancellation
	supervisor.New(ctx, zap.L(), client.Connect)

	waitForConnection(t, client)

	t.Run("GetPublicData", func(t *testing.T) {
		pd, err := client.GetPublicData(ctx)
		a.NoError(err)

		a.Equal(string(pd.FrostPublicData), "frost_key")
	})

	t.Run("AsyncSign", func(t *testing.T) {
		req := &signer.SignRequest{Digest: []byte("test_digest")}

		a.NoError(client.AsyncSign(req))

		// Verify request received by server
		select {
		case received := <-mock.signRequests:
			a.Equal(string(received.Digest), "test_digest")
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for sign request")
		}

		// Send response from server
		resp := &signer.SignResponse{
			Response: &signer.SignResponse_Signature{
				Signature: &tsscommon.SignatureData{
					Signature: []byte("test_signature"),
				},
			},
		}
		mock.signResponses <- resp

		// Verify response received by client
		select {
		case received := <-client.Response():
			sig := received.GetSignature()
			a.NotNil(sig)
			a.Equal(string(sig.Signature), "test_signature")

		case <-time.After(time.Second):
			t.Fatal("timeout waiting for sign response")
		}
	})

	t.Run("Verify", func(t *testing.T) {
		err := client.Verify(ctx, &signer.VerifySignatureRequest{})
		a.NoError(err)
	})
}

func waitForConnection(t *testing.T, client *signerClient) {
	t.Log("waiting for client to connect...")
	for range 5 {
		if client.isConnected() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	require.True(t, client.isConnected(), "client should be connected")
}
