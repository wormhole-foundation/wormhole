package tss

import (
	"context"
	"errors"
	"testing"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/stretchr/testify/assert"
	"github.com/xlabs/tss-common/service/signer"
	"go.uber.org/zap"
)

func TestAsyncSign_ChannelFull(t *testing.T) {
	client := &SignerClient{
		conn: &connChans{
			signRequests: make(chan *signer.SignRequest, 1),
		},
	}

	// Fill the channel
	client.conn.signRequests <- &signer.SignRequest{}

	// Attempt to sign
	req := &signer.SignRequest{Digest: []byte("digest"), Protocol: "protocol"}
	err := client.AsyncSign(req)

	assert.Error(t, err)
	assert.Equal(t, ErrSignerClientSignRequestChannelFull, err)
}

func TestGetPublicData_NilClient(t *testing.T) {
	var client *SignerClient
	_, err := client.GetPublicData(context.Background())
	assert.Error(t, err)
	assert.Equal(t, ErrSignerClientNil, err)
}

func TestVerify_NilClient(t *testing.T) {
	var client *SignerClient
	err := client.Verify(context.Background(), &signer.VerifySignatureRequest{})
	assert.Error(t, err)
	assert.Equal(t, ErrSignerClientNil, err)
}

func TestVerify_NilRequest(t *testing.T) {
	client := &SignerClient{}
	err := client.Verify(context.Background(), nil)
	assert.Error(t, err)
	assert.Equal(t, errNilRequest, err)
}

func TestSendUnaryRequest_ContextCancelled(t *testing.T) {
	client := &SignerClient{
		conn: &connChans{
			unaryRequests: make(chan unaryRequest), // Unbuffered
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.sendUnaryRequest(ctx, &signer.PublicDataRequest{})
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestSendUnaryRequest_ResponseError(t *testing.T) {
	client := &SignerClient{
		conn: &connChans{
			unaryRequests: make(chan unaryRequest, 1),
		},
	}

	go func() {
		req := <-client.conn.unaryRequests
		req.responseChan <- unaryResult{
			err: errors.New("mock error"),
		}
	}()

	_, err := client.sendUnaryRequest(context.Background(), &signer.PublicDataRequest{})
	assert.Error(t, err)
	assert.Equal(t, "mock error", err.Error())
}

func TestSendUnaryRequest_NilResponseItem(t *testing.T) {
	client := &SignerClient{
		conn: &connChans{
			unaryRequests: make(chan unaryRequest, 1),
		},
	}

	go func() {
		req := <-client.conn.unaryRequests
		req.responseChan <- unaryResult{
			item: nil,
			err:  nil,
		}
	}()

	_, err := client.sendUnaryRequest(context.Background(), &signer.PublicDataRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "internal error: nil response")
}

func TestUnaryRequestsHandler_UnknownRequest(t *testing.T) {
	client := &SignerClient{
		conn: &connChans{
			unaryRequests: make(chan unaryRequest, 1),
		},
	}
	logger := zap.NewNop()
	errChan := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	respChan := make(chan unaryResult, 1)
	client.conn.unaryRequests <- unaryRequest{
		item:         &gossipv1.Heartbeat{}, // Invalid type
		responseChan: respChan,
	}

	go client.unaryRequestsHandler(ctx, nil, logger, errChan)

	select {
	case res := <-respChan:
		assert.Error(t, res.err)
		assert.Contains(t, res.err.Error(), "unknown unary request type")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for response")
	}
}
