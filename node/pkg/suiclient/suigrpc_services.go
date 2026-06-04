package suiclient

import (
	"context"

	pb "github.com/block-vision/sui-go-sdk/pb/sui/rpc/v2"
)

// Production implementation of GrpcLedgerServiceClient
type GrpcLedgerServiceClient struct {
	pbLedgerServiceClient pb.LedgerServiceClient
}

func (c *GrpcLedgerServiceClient) GetObject(ctx context.Context, req *pb.GetObjectRequest) (*pb.GetObjectResponse, error) {
	return c.pbLedgerServiceClient.GetObject(ctx, req)
}
func (c *GrpcLedgerServiceClient) GetCheckpoint(ctx context.Context, req *pb.GetCheckpointRequest) (*pb.GetCheckpointResponse, error) {
	return c.pbLedgerServiceClient.GetCheckpoint(ctx, req)
}
func (c *GrpcLedgerServiceClient) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.GetTransactionResponse, error) {
	return c.pbLedgerServiceClient.GetTransaction(ctx, req)
}

// Production implementation of GrpcSubscriptionServiceClientInterface
type GrpcSubscriptionServiceClient struct {
	pbSubscriptionServiceClient pb.SubscriptionServiceClient
}

func (c *GrpcSubscriptionServiceClient) SubscribeCheckpoints(ctx context.Context, req *pb.SubscribeCheckpointsRequest) (pb.SubscriptionService_SubscribeCheckpointsClient, error) {
	return c.pbSubscriptionServiceClient.SubscribeCheckpoints(ctx, req)
}
