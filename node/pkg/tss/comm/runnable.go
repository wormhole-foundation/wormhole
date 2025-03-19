package comm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"

	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/certusone/wormhole/node/pkg/tss"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type DirectLink interface {
	tsscommv1.DirectLinkServer

	Run(ctx context.Context) error
}

func NewServer(socketPath string, logger *zap.Logger, tssMessenger tss.ReliableMessenger) (DirectLink, error) {
	peers := tssMessenger.GetPeers()
	partyIds := make([]*tss.Identity, len(peers))
	peerToCert := make(map[string]*x509.Certificate, len(peers))

	var err error
	for i, peer := range peers {
		partyIds[i], err = tssMessenger.FetchPartyId(peer)
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

		tssMessenger: tssMessenger,

		peers:         partyIds,
		peerToCert:    peerToCert,
		connections:   make(map[string]*connection, len(peers)),
		requestRedial: make(chan redialRequest, len(peers)),
		redials:       make(chan redialResponse, 1),
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
