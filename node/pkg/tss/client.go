package tss

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/gogo/status"
	tsscommon "github.com/xlabs/tss-common"
	"github.com/xlabs/tss-common/service/signer"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

type vaaHandling struct {
	isLeader    bool
	leaderIndex int // used to verify what content came from the leader.
	gst         *common.GuardianSetState
	guardiansigner.GuardianSigner

	gossipOutput   chan *gossipv1.TSSGossipMessage // channel to send outgoing gossip messages.
	incomingGossip chan *gossipv1.TSSGossipMessage // channel to receive incoming gossip messages.
}

type signerClient struct {
	// immutable fields:
	dialOpts   []grpc.DialOption
	socketPath string
	out        chan *tsscommon.SignatureData // outputs signatures.

	// used to communicate with the signer service.
	conn *connChans

	vaaData vaaHandling

	connected atomic.Int64 // 0 is not connected, 1 is connected.
}

type unaryResult struct {
	item proto.Message
	err  error
}

type unaryRequest struct {
	item         proto.Message
	responseChan chan unaryResult
}

// This might fail suddenly. if it does, the runnable should restart it.
type connChans struct {
	// streams for sign request/response.
	signRequests  chan *signer.SignRequest
	signResponses chan *signer.SignResponse

	// unary requests (GetPublicData, VerifySignature).
	unaryRequests chan unaryRequest
}

type signatureStream grpc.BidiStreamingClient[signer.SignRequest, signer.SignResponse]

const (
	notConnected = iota
	connected
)

var (
	ErrSignerClientSignRequestChannelFull = errors.New("signer client sign request channel is full")
	ErrSignerClientNil                    = errors.New("tss signer client is nil")
	errInvalidUnaryResponseError          = errors.New("internal error: invalid response type from signer service")
)

// a blocking call that connects to the signer service and maintains the connection.
//
// Connect implements a connection that can be used for the supervisor.Runnable interface.
// it connects to the signer service, forwards requests from the in channel, and outputs responses to the out channel.
// It runs until the context is cancelled or an error occurs.
// (expects the supervisor to restart it on failure).
func (s *signerClient) Connect(ctx context.Context) error {
	return s.connect(ctx, supervisor.Logger(ctx).Named("tss-signer-connection"))
}

// AsyncSign implements Signer.
func (s *signerClient) AsyncSign(rq *signer.SignRequest) error {
	if s == nil {
		return ErrSignerClientNil
	}

	select {
	case s.conn.signRequests <- rq:
		return nil
	default:
		return ErrSignerClientSignRequestChannelFull
	}
}

// GetPublicData implements Signer.
func (s *signerClient) GetPublicData(ctx context.Context) (*signer.PublicData, error) {
	if s == nil {
		return nil, ErrSignerClientNil
	}

	response, err := s.sendUnaryRequest(ctx, &signer.PublicDataRequest{})
	if err != nil {
		return nil, err
	}

	publicData, ok := response.(*signer.PublicData)
	if !ok {
		return nil, errInvalidUnaryResponseError
	}

	return publicData, nil
}

// outputs the SignerService responses.
func (s *signerClient) Response() <-chan *signer.SignResponse {
	if s == nil {
		return nil // ensure we don't panic, but return nil channel (which blocks forever, and ignored in select).
	}

	return s.conn.signResponses
}

// Verify implements Signer.
func (s *signerClient) Verify(ctx context.Context, toVerify *signer.VerifySignatureRequest) error {
	if s == nil {
		return ErrSignerClientNil
	}

	response, err := s.sendUnaryRequest(ctx, toVerify)
	if err != nil {
		return err
	}

	verifyResult, ok := response.(*signer.VerifySignatureResponse)
	if !ok {
		return errInvalidUnaryResponseError
	}

	if !verifyResult.IsValid {
		return errors.New("signature verification failed")
	}

	return nil // no error. signature is valid.
}

func (s *signerClient) sendUnaryRequest(ctx context.Context, request proto.Message) (proto.Message, error) {
	chn := make(chan unaryResult, 1)

	select {
	case s.conn.unaryRequests <- unaryRequest{
		item:         request,
		responseChan: chn,
	}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	select {
	case response := <-chn:
		if response.err != nil {
			return nil, response.err
		}

		if response.item == nil {
			return nil, errors.New("internal error: nil response from signer service")
		}

		return response.item, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// mainly used for tests.
func (s *signerClient) isConnected() bool {
	if s == nil {
		return false
	}

	return s.connected.Load() == connected
}

func (s *signerClient) connect(ctx context.Context, logger *zap.Logger) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // we cancel on exit to ensure all goroutines exit.

	logger.Info("connecting to signer service...")

	// setup conn:
	cc, err := grpc.NewClient(s.socketPath, s.dialOpts...)
	if err != nil {
		logger.Error("connecting to signer service failed", zap.Error(err))

		return err
	}
	defer cc.Close()

	client := signer.NewSignerClient(cc)

	// Setting up the stream for signing requests and responses.
	stream, err := client.SignMessage(ctx)
	if err != nil {
		logger.Error("stream setup failed", zap.Error(err))

		return err
	}
	defer stream.CloseSend()

	logger.Info("connection to signer service established")

	s.connected.Store(connected)
	defer s.connected.Store(notConnected)

	// buffer to avoid goroutine leaks.
	errchan := make(chan error, 3)

	go s.receivingStream(ctx, stream, errchan)
	go s.sendingStream(ctx, stream, errchan)
	go s.unaryRequestsHandler(ctx, client, logger, errchan)

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errchan:
		logger.Error("closing connection", zap.Error(err))

		return err
	}
}

func (s *signerClient) receivingStream(ctx context.Context, stream signatureStream, errchan chan error) {
	for {
		resp, err := stream.Recv()
		if err != nil {
			errchan <- err // error from stream is stream-fatal.

			return
		}

		select {
		case <-ctx.Done():
			return
		// incoming response from signer service is sent to the signResponses channel.
		case s.conn.signResponses <- resp:
			// TODO: Consider inspecting the type of response (signature or error. on error log and continue?)
		}
	}
}

// responsible to send sign requests to the signer-service.
func (s *signerClient) sendingStream(ctx context.Context, stream signatureStream, errchan chan error) {
	for {
		select {
		case <-ctx.Done(): // context cancelled, or error from other peer.
			return
		case rq := <-s.conn.signRequests:
			if err := stream.Send(rq); err != nil {
				errchan <- err // error from stream is stream-fatal.

				return
			}
		}
	}
}

// responsible to receive unary requests and send the to the signer-service for processing.
func (s *signerClient) unaryRequestsHandler(ctx context.Context, client signer.SignerClient, logger *zap.Logger, errchan chan error) {
	for {
		select {
		case <-ctx.Done():
			return
		case urq := <-s.conn.unaryRequests:
			if urq.item == nil {
				continue // malformed request, ignore.
			}

			var resp proto.Message
			var errResponse error

			switch req := urq.item.(type) {
			case *signer.PublicDataRequest:
				resp, errResponse = client.GetPublicData(ctx, req)
			case *signer.VerifySignatureRequest:
				resp, errResponse = client.VerifySignature(ctx, req)
			default:
				errResponse = errors.New("unknown unary request type")
			}

			select {
			case urq.responseChan <- unaryResult{item: resp, err: errResponse}:
			default:
				logger.Error("unary response channel full, dropping response")
			}

			if isFatalError(errResponse) {
				errchan <- errResponse

				return
			}
		}
	}
}

func isFatalError(err error) bool {
	if err == nil {
		return false
	}

	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	switch st.Code() {
	case codes.Unavailable, codes.Internal:
		return true
	default:
		return false
	}
}
