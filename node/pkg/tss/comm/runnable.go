package comm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"strconv"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/certusone/wormhole/node/pkg/tss"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type DirectLink interface {
	tsscommv1.DirectLinkServer

	Run(ctx context.Context) error

	// Will wait until either the context is expired, or the number of active connections
	// reaches the DirectLink Server target number.
	// NOTICE: this function might return, and the server might still lose a connection to a peer.
	// This is a best-effort function, and it is not guaranteed that all connections are active
	// when it returns.
	WaitForConnections(ctx context.Context) error
}

func NewServer(logger *zap.Logger, tssMessenger tss.ReliableMessenger) (DirectLink, error) {
	cert := tssMessenger.GetCertificate()
	if cert == nil {
		return nil, errors.New("tssMessenger returned nil certificate")
	}

	selfID, err := tssMessenger.FetchIdentity(cert.Leaf)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch self identity from tssMessenger: %w", err)
	}

	port := selfID.Port
	if port == 0 {
		p, err := strconv.Atoi(tss.DefaultPort)
		if err != nil {
			return nil, fmt.Errorf("failed to parse default port: %w", err)
		}

		port = p
	}

	return newServer(fmt.Sprintf("[::]:%d", port), logger, tssMessenger)
}

func newServer(socketPath string, logger *zap.Logger, tssMessenger tss.ReliableMessenger) (DirectLink, error) {
	if socketPath == "" {
		return nil, errors.New("can't create DirectLink server: socketPath is empty")
	}
	if logger == nil {
		return nil, errors.New("can't create DirectLink server: logger is nil")
	}
	if tssMessenger == nil {
		return nil, errors.New("can't create DirectLink server: tssMessenger is nil")
	}

	peers := tssMessenger.GetPeers()
	partyIds := make([]*tss.Identity, len(peers))
	peerToCert := make(map[string]*x509.Certificate, len(peers))

	var err error
	for i, peer := range peers {
		partyIds[i], err = tssMessenger.FetchIdentity(peer)
		if err != nil {
			return nil, err
		}

		peerToCert[partyIds[i].NetworkName()] = peer
	}

	return &server{
		UnimplementedDirectLinkServer: tsscommv1.UnimplementedDirectLinkServer{},
		ctx:                           nil, // set up in Run(ctx)
		logger:                        logger,
		socketPath:                    socketPath,
		tssMessenger:                  tssMessenger,
		peers:                         partyIds,
		peerToCert:                    peerToCert,
		connections:                   make(map[string]*connection, len(peers)),
		requestRedial:                 make(chan redialRequest, len(peers)),
		redials:                       make(chan redialResponse, 1),
		fullyConnected:                make(chan struct{}, 1), // buffered to avoid blocking
	}, nil
}

// Run initialise the server and starts listening on the socket.
// In addition, it will set up connections to all given peers (guardians).
func (s *server) Run(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("tsscomm.server is nil")
	}

	s.ctx = ctx

	listener, err := net.Listen("tcp", s.socketPath)
	if err != nil {
		return err
	}

	errC := make(chan error)
	gserver := grpc.NewServer(
		s.makeServerCredentials(),
	)

	tsscommv1.RegisterDirectLinkServer(gserver, s)

	go func() {
		errC <- gserver.Serve(listener)
	}()
	s.run()

	s.logger.Info("tsscomm.server listening on", zap.String("path", s.socketPath))

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-errC:
	}

	gserver.Stop()

	if err := listener.Close(); err != nil {
		s.logger.Error("failed to close listener", zap.Error(err))
	}

	return err
}

func (s *server) makeServerCredentials() grpc.ServerOption {
	certPool := x509.NewCertPool()
	for _, peer := range s.tssMessenger.GetPeers() {
		certPool.AddCert(peer)
	}

	creds := grpc.Creds(credentials.NewTLS(
		&tls.Config{
			MinVersion:   tls.VersionTLS13, // version 1.3
			Certificates: []tls.Certificate{*s.tssMessenger.GetCertificate()},

			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs:  certPool, // treating each peer as its own CA, will use the given cert as the ID of the peer.
		},
	))

	return creds
}
