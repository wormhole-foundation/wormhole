package comm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"time"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/certusone/wormhole/node/pkg/tss"
	"github.com/gogo/status"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

type connection struct {
	cc     *grpc.ClientConn
	stream tsscommv1.DirectLink_SendClient
}

type redialResponse struct {
	name string
	conn *connection
}

type redialRequest struct {
	hostname   string
	immediatly bool //used to skip waiting in the dialer backoff mechanism
}

type server struct {
	tsscommv1.UnimplementedDirectLinkServer
	ctx        context.Context
	logger     *zap.Logger
	socketPath string

	tssMessenger tss.ReliableMessenger

	peers      []*tsscommv1.PartyId
	peerToCert map[string]*x509.Certificate
	// to ensure thread-safety without locks, only the sender is allowed to change this map.
	connections   map[string]*connection
	requestRedial chan redialRequest
	redials       chan redialResponse
}

func (s *server) run() {
	go s.dialer()
	go s.sender()

	for _, pid := range s.peers {
		s.enqueueRedialRequest(redialRequest{
			hostname:   pid.Id,
			immediatly: false,
		})
	}
}

const connectionCheckTime = time.Second * 5

func (s *server) sender() {
	connectionCheckTicker := time.NewTicker(connectionCheckTime)

	for {
		select {
		case <-s.ctx.Done():
			for _, con := range s.connections {
				s.closeConnection(con)
			}

			return

		case o := <-s.tssMessenger.ProducedOutputMessages():
			s.send(o)

		case redial := <-s.redials:
			if _, ok := s.connections[redial.name]; ok {
				// shouldn't open the same connection twice.
				// if a redial request is still needed, it will be enqueued again either
				// on the next send attempt, or once the ticker pops.
				s.closeConnection(redial.conn)

				continue
			}

			s.connections[redial.name] = redial.conn

		case <-connectionCheckTicker.C:
			s.forceDialIfNotConnected()
		}
	}
}

func (s *server) closeConnection(con *connection) {
	if err := con.cc.Close(); err != nil {
		s.logger.Error(
			"couldn't close connection while shutting down",
			zap.Error(err),
		)
	}
}

func (s *server) forceDialIfNotConnected() {
	if len(s.connections) != len(s.peers) {
		for _, pid := range s.peers {
			hostname := pid.Id
			if _, ok := s.connections[hostname]; !ok {
				s.enqueueRedialRequest(redialRequest{
					hostname:   hostname,
					immediatly: true,
				})
			}
		}
	}
}

func (s *server) send(msg tss.Sendable) {
	for _, recipient := range msg.GetDestinations() {
		hostname := recipient.Id

		conn, ok := s.connections[hostname]
		if !ok {
			s.enqueueRedialRequest(redialRequest{
				hostname:   hostname,
				immediatly: false,
			})

			s.logger.Warn(
				"Couldn't send message to peer. No connection found.",
				zap.String("hostname", hostname),
			)

			continue
		}

		if err := conn.stream.Send(msg.GetNetworkMessage()); err != nil {
			delete(s.connections, hostname)
			s.enqueueRedialRequest(redialRequest{
				hostname:   hostname,
				immediatly: false,
			})

			s.logger.Error(
				"couldn't send message to peer due to error.",
				zap.Error(err),
				zap.String("hostname", hostname),
			)
		}
	}
}

func (s *server) enqueueRedialRequest(rqst redialRequest) {
	select {
	case <-s.ctx.Done():
		return
	case s.requestRedial <- rqst:
		s.logger.Debug("requested redial", zap.String("hostname", rqst.hostname))

		return
	default:
		s.logger.Warn("couldn't send request to redial", zap.String("hostname", rqst.hostname))
	}
}

func (s *server) dialer() {
	// using a heap instead of time.AfterFunc/ After to reduce the number of
	// goroutines generated to 0 (not including the dialer itself).
	waiters := newBackoffHeap()

	for {
		dialTo := ""

		select {
		case <-s.ctx.Done():
			return
		case <-waiters.timer.C:
			dialTo = waiters.Dequeue()
		case rqst := <-s.requestRedial:
			if rqst.immediatly {
				dialTo = rqst.hostname // will drop down to the dialing section.
			} else {
				waiters.Enqueue(rqst.hostname)
			}
		}

		if dialTo == "" {
			continue // skip (nothing to dial to)
		}

		if err := s.dial(dialTo); err != nil {
			s.logger.Error(
				"couldn't create direct link to peer",
				zap.Error(err),
				zap.String("hostname", dialTo),
			)

			waiters.Enqueue(dialTo) // ensuring a retry.

			continue
		}

		s.logger.Info("dialed to peer", zap.String("hostname", dialTo))
		waiters.ResetAttempts(dialTo)
	}
}

func (s *server) dial(hostname string) error {
	pool := x509.NewCertPool()
	pool.AddCert(s.peerToCert[hostname]) // dialing to peer and accepting his cert only.

	cc, err := grpc.Dial(hostname,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			MinVersion:   tls.VersionTLS13,                                    // tls 1.3
			Certificates: []tls.Certificate{*s.tssMessenger.GetCertificate()}, // our cert to be sent to the peer.
			RootCAs:      pool,
		})),
	)
	if err != nil {
		return err
	}

	stream, err := tsscommv1.NewDirectLinkClient(cc).Send(s.ctx)
	if err != nil {
		cc.Close()

		return err
	}

	s.redials <- redialResponse{
		name: hostname,
		conn: &connection{
			cc:     cc,
			stream: stream,
		},
	}

	return nil
}

func (s *server) Send(inStream tsscommv1.DirectLink_SendServer) error {
	clientId, err := s.getIdentityFromIncomingStream(inStream)
	if err != nil {
		s.logger.Warn(
			"did not accept incoming peer connection",
			zap.Error(err),
		)

		return err
	}

	for {
		m, err := inStream.Recv()
		if err != nil {
			if err == io.EOF {
				s.logger.Info(
					"closing input stream",
					zap.String("peer", clientId.Id),
				)

				return nil
			}

			s.logger.Error(
				"error receiving from guardian. Closing connection",
				zap.Error(err),
				zap.String("peer", clientId.Id),
			)

			return err
		}

		s.tssMessenger.HandleIncomingTssMessage(&tss.IncomingMessage{
			Source:  clientId,
			Content: m,
		})
	}
}

// getIdentityFromIncomingStream extracts the peer identity from the
// incoming TLS certificate embbeded into the stream.
// adds various checks to ensure the client is a valid guardian.
func (s *server) getIdentityFromIncomingStream(inStream tsscommv1.DirectLink_SendServer) (*tsscommv1.PartyId, error) {
	p, ok := peer.FromContext(inStream.Context())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "unable to retrieve peer from context")
	}

	// Extract AuthInfo (TLS information)
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "unexpected peer transport credentials type, please use tls")
	}

	// check incoming TLS cert doesn't contain a chain (should be a leaf cert).
	// this is more of a precaution.
	if len(tlsInfo.State.PeerCertificates) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no client certificate provided")
	}

	if len(tlsInfo.State.PeerCertificates) != 1 {
		return nil, status.Error(codes.PermissionDenied, "expected certificate to be a CA")
	}

	for _, chain := range tlsInfo.State.VerifiedChains {
		if len(chain) != 1 {
			return nil, status.Error(codes.PermissionDenied, "certificate has a chain")
		}
	}

	// Get the peer's certificate: The first element is the leaf certificate
	// that the connection is verified against
	clientCert := tlsInfo.State.PeerCertificates[0]

	if clientCert.PublicKeyAlgorithm != x509.ECDSA {
		return nil, status.Error(codes.InvalidArgument, "certificate must use ECDSA")
	}

	if !clientCert.IsCA {
		return nil, status.Error(codes.PermissionDenied, "client certificate is not a CA, but a leaf certificate")
	}

	// fetch the party ID according to the public key used to verify this certificate (embbded in the cert).
	clientId, err := s.tssMessenger.FetchPartyId(clientCert)
	if err != nil {
		return nil, fmt.Errorf("client certificate wasn't found: %w", err)
	}

	return clientId, nil
}
