package common

import (
	"context"
	"fmt"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// metadataStreamServerInterceptor adds the `x-forwarded-for` and `x-trace-id` metadata as a tag to cause it to be logged by grpc_zap.StreamServerInterceptor().
// Note that `x-forwarded-for` can only be trusted if the latest hop in the proxy chain is trusted.
// For JSON-Web requests, the latest hop is the guardian itself (`grpc-gateway`), which is listening on TCP and forwarding to the gRPC publicrpc UNIX socket.
// This can be identified by `"peer.address": "@"` in the logs and `grpc-gateway` correctly sets the `x-forwarded-for` metadata.
func metadataStreamServerInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := stream.Context()

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		grpc_ctxtags.Extract(ctx).
			Set("x-forwarded-for", md.Get("x-forwarded-for")).
			Set("x-trace-id", md.Get("x-trace-id"))
	}

	err := handler(srv, stream)
	return err
}

// metadataServerInterceptor adds the `x-forwarded-for` and `x-trace-id` metadata as a tag to cause it to be logged by grpc_zap.UnaryServerInterceptor().
// Note that `x-forwarded-for` can only be trusted if the latest hop in the proxy chain is trusted.
// For JSON-Web requests, the latest hop is the guardian itself (`grpc-gateway`), which is listening on TCP and forwarding to the gRPC publicrpc UNIX socket.
// This can be identified by `"peer.address": "@"` in the logs and `grpc-gateway` correctly sets the `x-forwarded-for` metadata.
func metadataServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		tags := grpc_ctxtags.Extract(ctx)
		tags.Set("x-forwarded-for", md.Get("x-forwarded-for"))

		if len(md.Get("x-trace-id")) > 0 {
			tags.Set("x-trace-id", md.Get("x-trace-id"))
		}
	}
	return handler(ctx, req)
}

type protojsonObjectMarshaler struct {
	pb protoreflect.ProtoMessage
}

func (j *protojsonObjectMarshaler) MarshalLogObject(e zapcore.ObjectEncoder) error {
	// ZAP jsonEncoder deals with AddReflect by using json.MarshalObject. The same thing applies for consoleEncoder.
	return e.AddReflected("msg", j)
}

func (j *protojsonObjectMarshaler) MarshalJSON() ([]byte, error) {
	b, err := protojson.Marshal(j.pb)
	if err != nil {
		return nil, fmt.Errorf("jsonpb serializer failed: %v", err)
	}
	return b, nil
}

func requestPayloadServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if p, ok := req.(protoreflect.ProtoMessage); ok {
		if b, err := protojson.Marshal(p); err == nil {
			if len(b) <= 200 {
				grpc_ctxtags.Extract(ctx).Set("grpc.requestbody", &protojsonObjectMarshaler{pb: p})
			} else {
				grpc_ctxtags.Extract(ctx).Set("grpc.requestbody", "too long")
			}
		}
	}
	return handler(ctx, req)
}

func NewInstrumentedGRPCServer(logger *zap.Logger) *grpc.Server {
	streamInterceptors := []grpc.StreamServerInterceptor{
		grpc_ctxtags.StreamServerInterceptor(),
		grpc_prometheus.StreamServerInterceptor,
	}

	unaryInterceptors := []grpc.UnaryServerInterceptor{
		grpc_ctxtags.UnaryServerInterceptor(),
		grpc_prometheus.UnaryServerInterceptor,
	}

	if logger != nil {
		logger = logger.With(zap.Bool("_privateLogEntry", true))
		streamInterceptors = append(streamInterceptors,
			metadataStreamServerInterceptor,
			grpc_zap.PayloadStreamServerInterceptor(logger, func(ctx context.Context, fullMethodName string, servingObject interface{}) bool { return true }),
		)

		unaryInterceptors = append(unaryInterceptors,
			metadataServerInterceptor,
			requestPayloadServerInterceptor,
			grpc_zap.UnaryServerInterceptor(logger),
		)
	}

	server := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(streamInterceptors...)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(unaryInterceptors...)),
	)

	grpc_prometheus.EnableHandlingTimeHistogram()
	grpc_prometheus.Register(server)

	return server
}
