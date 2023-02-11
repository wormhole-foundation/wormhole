package common

import (
	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// xForwardedForStreamServerInterceptor adds the `x-forwarded-for` metadata as a tag to cause it to be logged by grpc_zap.StreamServerInterceptor().
// Note that `x-forwarded-for` can only be trusted if the latest hop in the proxy chain is trusted.
// For JSON-Web requests, the latest hop is the guardian itself (`grpc-gateway`), which is listening on TCP and forwarding to the gRPC publicrpc UNIX socket.
// This can be identified by `"peer.address": "@"` in the logs and `grpc-gateway` correctly sets the `x-forwarded-for` metadata.
func xForwardedForStreamServerInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := stream.Context()

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		grpc_ctxtags.Extract(ctx).Set("x-forwarded-for", md.Get("x-forwarded-for"))
	}

	err := handler(srv, stream)
	return err
}

// xForwardedForServerInterceptor adds the `x-forwarded-for` metadata as a tag to cause it to be logged by grpc_zap.UnaryServerInterceptor().
// Note that `x-forwarded-for` can only be trusted if the latest hop in the proxy chain is trusted.
// For JSON-Web requests, the latest hop is the guardian itself (`grpc-gateway`), which is listening on TCP and forwarding to the gRPC publicrpc UNIX socket.
// This can be identified by `"peer.address": "@"` in the logs and `grpc-gateway` correctly sets the `x-forwarded-for` metadata.
func xForwardedForServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		grpc_ctxtags.Extract(ctx).Set("x-forwarded-for", md.Get("x-forwarded-for"))
	}
	return handler(ctx, req)
}

func NewInstrumentedGRPCServer(logger *zap.Logger) *grpc.Server {
	server := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			xForwardedForStreamServerInterceptor,
			grpc_prometheus.StreamServerInterceptor,
			grpc_zap.StreamServerInterceptor(logger),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			xForwardedForServerInterceptor,
			grpc_prometheus.UnaryServerInterceptor,
			grpc_zap.UnaryServerInterceptor(logger),
		)),
	)

	grpc_prometheus.EnableHandlingTimeHistogram()
	grpc_prometheus.Register(server)

	return server
}
