package tss

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"sync/atomic"

	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	common "github.com/xlabs/tss-common"
	"github.com/xlabs/tss-common/service/signer"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type SignerConnection interface {
	// Connect establishes a connection to the signer service.
	// It blocks until the connection is established or an error occurs.
	// Can be used for a supervisor.Runnable.
	Connect(ctx context.Context) error

	Signer
}

/*
The Signer interface represents a TSS signer service that can be used to request signatures.
*/
type Signer interface {
	/*
		AsyncSign starts an asynchronous signing request.
		The produced signature can be received from ProducedSignatures() channel.
		The request must contain the digest to be signed and the protocol to use.

		If a specific committee is needed, it can be specified via rq.CommitteeMembers. (nil means use pseudo-random committee).
		The committee members are specified via their public keys (guardian public keys).
		If the signer is not configured to recognize the specified committee members, it'll return an error.
	*/
	AsyncSign(rq *signer.SignRequest) error
	// Outputs signatures as they are produced. doesn't guarantee order.
	Response() <-chan *signer.SignResponse

	// TODO: Missing signerService API! The SignerService should support both GetPublicData data and verify.
	// GetPublicKey gets the public information of the signer.
	GetPublicData(context.Context) error
	Verify(context.Context, *common.SignatureData) error

	// Witness new VAA is used by a LEADER to inform all peers of a new VAA observed on the network and to sign it!
	// If this signer is a leader: it'll use the p2p network to tell all peers to sign the
	WitnessNewVaa(v *vaa.VAA) error // TODO: need to specify from whom it came, to ensure leader forwarded this VAA.
}

// Ensure interfaces are implemented.
var (
	_ SignerConnection = (*signerClient)(nil)
	_ Signer           = (*signerClient)(nil)
)

func TODO() SignerConnection { // TODO: remove
	return &signerClient{}
}

// TODO: Consider letting the signer have its own logs and inspect the status of sign requests, responses, errors, etc.
func NewSigner(socketPath string) *signerClient {
	// todo: create a goroutine with a map that will match requests to responses and output them to the out channel.
	// it will also use a logger to log errors, etc.
	// closes once the context is cancelled.

	return &signerClient{
		socketPath: socketPath,
		started:    atomic.Int32{},
		ctx:        nil,                // filled in start.
		cert:       &tls.Certificate{}, // TODO
		logger:     zap.NewNop(),       // filled in start.
		conn: &connChans{
			signRequests:  make(chan *signer.SignRequest, 100),  // TODO: buffer sizes?
			signResponses: make(chan *signer.SignResponse, 100), // TODO: buffer sizes?
		},
	}
}

type signerClient struct {
	// immutable fields:
	ctx        context.Context
	cert       *tls.Certificate
	socketPath string
	out        chan *common.SignatureData // outputs signatures.

	// used to communicate with the signer service.
	conn *connChans

	// initialized in start. Might be changed during a restart of the runnable.
	started atomic.Int32 // 0 != means started.
	logger  *zap.Logger
}

// This might fail suddenly. if it does, the runnable should restart it.
type connChans struct {
	// streams for sign request/response.
	signRequests  chan *signer.SignRequest
	signResponses chan *signer.SignResponse

	// TODO: unary requests like public-data, verify, etc. should have a single channel that accepts a type and a response channel with 1 capacity to respond into.
	//   the signer will attempt to send them through this channel, if its full/ blocked it will return an error.
	//   unaryRequests chan UnaryRequest{item, responseChan with 1 buffer}
}

func (s *signerClient) Start(ctx context.Context) error {
	return s.Connect(ctx)
}

// a blocking call that connects to the signer service and maintains the connection.
//
// Connect implements a connection that can be used for the supervisor.Runnable interface.
// it connects to the signer service, forwards requests from the in channel, and outputs responses to the out channel.
// It runs until the context is cancelled or an error occurs.
// (expects the supervisor to restart it on failure).
func (s *signerClient) Connect(ctx context.Context) error {
	logger := supervisor.Logger(ctx).Named("tss-signer-connection")
	logger.Info("setting connection to signer service.")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // we cancel on exit.

	// setup conn:
	cc, err := s.makeConn()
	if err != nil {
		return err
	}
	defer cc.Close()

	client := signer.NewSignerClient(cc)

	// Setting up the stream for signing requests and responses.
	stream, err := client.SignMessage(ctx)
	if err != nil {
		return err
	}
	defer stream.CloseSend()

	errchan := make(chan error, 3) // buffer to avoid goroutine leak if both fail simultaneously.

	// listener
	go func() {
		for {
			resp, err := stream.Recv()
			if err != nil {
				errchan <- err
				return
			}

			// TODO: Consider inspecting the type of response (signature or error. on error log and continue?)
			select {
			case <-ctx.Done(): // context cancelled, stop processing.
				return
			case s.conn.signResponses <- resp:
			}
		}
	}()

	// sender
	go func() {
		for {
			select {
			case <-ctx.Done(): // context cancelled, or error from other peer.
				return
			case rq := <-s.conn.signRequests:
				if err := stream.Send(rq); err != nil {
					errchan <- err
					return
				}
			}
		}
	}()

	// TODO: add unary request handling goroutine.
	select {
	case <-ctx.Done():
		return nil
	case err := <-errchan:
		logger.Error("closing connection", zap.Error(err))
		return err
	}
}

func (s *signerClient) makeConn() (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	if s.cert != nil {
		pool := x509.NewCertPool()
		pool.AddCert(s.cert.Leaf) // same cert used for server verification.

		creds := credentials.NewTLS(&tls.Config{
			MinVersion:   tls.VersionTLS13,
			Certificates: []tls.Certificate{*s.cert}, // this is what the client presents to the server.
			RootCAs:      pool,                       // this is what the client uses to verify the server.
		})

		opts = append(opts, grpc.WithTransportCredentials(creds))
	}

	return grpc.NewClient(s.socketPath, opts...)
}

var ErrSignerClientSignRequestChannelFull = errors.New("signer client sign request channel is full")

// AsyncSign implements Signer.
func (s *signerClient) AsyncSign(rq *signer.SignRequest) error {
	select {
	case s.conn.signRequests <- rq:
		return nil
	default:
		return ErrSignerClientSignRequestChannelFull
	}
}

// GetPublicData implements Signer.
func (s *signerClient) GetPublicData(context.Context) error {
	panic("unimplemented")
}

// outputs the SignerService responses.
func (s *signerClient) Response() <-chan *signer.SignResponse {
	return s.conn.signResponses
}

// Verify implements Signer.
func (s *signerClient) Verify(context.Context, *common.SignatureData) error {
	panic("unimplemented")
}

// WitnessNewVaa implements Signer.
func (s *signerClient) WitnessNewVaa(v *vaa.VAA) error {
	panic("unimplemented")
}
