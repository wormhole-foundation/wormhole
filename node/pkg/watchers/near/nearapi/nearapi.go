package nearapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/certusone/wormhole/node/pkg/supervisor"
	"go.uber.org/zap"
)

const (
	nearRPCTimeout               = 5 * time.Second
	nearRPCConcurrentConnections = 10
)

type (
	NearAPI struct {
		nearRPC        string
		nearHttpClient *http.Client
	}
)

func NewNearAPI(nearRPC string) NearAPI {
	// Customize the Transport to have larger connection pool (default is only 2 per host)
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxConnsPerHost = nearRPCConcurrentConnections
	t.MaxIdleConnsPerHost = nearRPCConcurrentConnections
	var httpClient = &http.Client{
		Timeout:   nearRPCTimeout,
		Transport: t,
	}

	return NearAPI{nearRPC, httpClient}
}

func (n *NearAPI) nearRPCQuery(ctx context.Context, s string) ([]byte, error) {
	timeout, cancelFunc := context.WithTimeout(ctx, nearRPCTimeout)
	defer cancelFunc()

	timer := time.NewTimer(time.Nanosecond)
	var backoffMilliseconds int = 100

	for {
		select {
		case <-timeout.Done():
			return nil, errors.New("HTTP timeout")
		case <-timer.C:
			// perform HTTP request
			req, _ := http.NewRequestWithContext(timeout, http.MethodPost, n.nearRPC, bytes.NewBuffer([]byte(s)))
			req.Header.Add("Content-Type", "application/json")
			resp, err := n.nearHttpClient.Do(req)

			if err == nil {
				defer resp.Body.Close()
				result, err := ioutil.ReadAll(resp.Body)
				if resp.StatusCode == 200 {
					return result, err
				}
			}
			// retry if there was a server error
			backoffMilliseconds += int((float64(backoffMilliseconds)) * (rand.Float64() * 2.5)) //#nosec G404 no CSPRNG needed here for jitter computation
			timer.Reset(time.Millisecond * time.Duration(backoffMilliseconds))
		}
	}
}

// getBlock calls the NEAR RPC API to retrieve a block by its hash (https://docs.near.org/api/rpc/block-chunk#block-details)
func (n *NearAPI) GetBlock(ctx context.Context, blockId string) (Block, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"block_id": "%s"}}`, blockId)
	blockBytes, err := n.nearRPCQuery(ctx, s)
	if err != nil {
		return Block{}, err
	}
	// TODO cleanup
	logger := supervisor.Logger(ctx)
	logger.Debug("block json", zap.String("json", string(blockBytes)))

	return newBlockFromBytes(blockBytes)
}

// getBlockByHeight calls the NEAR RPC API to retrieve a block by its height (https://docs.near.org/api/rpc/block-chunk#block-details)
func (n *NearAPI) GetBlockByHeight(ctx context.Context, blockHeight uint64) (Block, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"block_id": %d}}`, blockHeight)
	blockBytes, err := n.nearRPCQuery(ctx, s)
	if err != nil {
		return Block{}, err
	}
	return newBlockFromBytes(blockBytes)
}

// getFinalBlock gets a finalized block from the NEAR RPC API using the parameter "finality": "final" (https://docs.near.org/api/rpc/block-chunk)
func (n *NearAPI) GetFinalBlock(ctx context.Context) (Block, error) {
	s := `{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"finality": "final"}}`
	blockBytes, err := n.nearRPCQuery(ctx, s)
	if err != nil {
		return Block{}, err
	}
	return newBlockFromBytes(blockBytes)
}

// getChunk gets a chunk from the NEAR RPC API: https://docs.near.org/api/rpc/block-chunk#chunk-details
func (n *NearAPI) GetChunk(ctx context.Context, chunkHeader ChunkHeader) (Chunk, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "chunk", "params": {"chunk_id": "%s"}}`, chunkHeader.Hash)
	bytes, err := n.nearRPCQuery(ctx, s)
	if err != nil {
		return Chunk{}, err
	}
	return newChunkFromBytes(bytes)
}

// getTxStatus queries status of a transaction by hash, returning the transaction_outcomes and receipts_outcomes
// sender_account_id is used to determine which shard to query for the transaction
// See https://docs.near.org/api/rpc/transactions#transaction-status
func (n *NearAPI) GetTxStatus(ctx context.Context, txHash string, senderAccountId string) ([]byte, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "EXPERIMENTAL_tx_status", "params": ["%s", "%s"]}`, txHash, senderAccountId)
	return n.nearRPCQuery(ctx, s)
}
