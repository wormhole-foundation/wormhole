package nearapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/mr-tron/base58"
)

const (
	nearRPCTimeout = 5 * time.Second
	/*
		NEAR JSON RPC node is starting up with 4 workers
		(https://github.com/near/nearcore/blob/8dc9a0bab8aa4648fc7af777e9fa7e3e545c95a5/chain/jsonrpc/src/lib.rs#L1372)
		and actix_web by default supports 256 concurrent TLS connections per worker and 25k non-TLS
		(https://actix.rs/actix-web/actix_web/struct.HttpServer.html#method.workers).

		Therefore, the Guardian NEAR RPC node should allow at least 500 concurrent connections.
		According to https://explorer.near.org/stats, NEAR blockchain has bursts of up to 2M tx/day,
		so 500 concurrent RPC connections should be sufficient.
	*/
	nearRPCConcurrentConnections = 500
)

type (
	NearRpc interface {
		Query(ctx context.Context, s string) ([]byte, error)
	}
	HttpNearRpc struct {
		nearRpc        string
		nearHttpClient *http.Client
	}
	NearApi interface {
		GetBlock(ctx context.Context, blockId string) (Block, error)
		GetBlockByHeight(ctx context.Context, blockHeight uint64) (Block, error)
		GetFinalBlock(ctx context.Context) (Block, error)
		GetChunk(ctx context.Context, chunkHeader ChunkHeader) (Chunk, error)
		GetTxStatus(ctx context.Context, txHash string, senderAccountId string) ([]byte, error)
	}
	NearApiImpl struct {
		nearRPC NearRpc
	}
)

func NewHttpNearRpc(nearRPC string) HttpNearRpc {
	// Customize the Transport to have larger connection pool (default is only 2 per host)
	t := http.DefaultTransport.(*http.Transport).Clone() //nolint:forcetypeassert
	t.MaxConnsPerHost = nearRPCConcurrentConnections
	t.MaxIdleConnsPerHost = nearRPCConcurrentConnections
	var httpClient = &http.Client{
		Timeout:   nearRPCTimeout,
		Transport: t,
	}

	return HttpNearRpc{nearRPC, httpClient}
}

func NewNearApiImpl(nearRpc NearRpc) NearApiImpl {
	return NearApiImpl{nearRpc}
}

func (n HttpNearRpc) Query(ctx context.Context, s string) ([]byte, error) {
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
			req, _ := http.NewRequestWithContext(timeout, http.MethodPost, n.nearRpc, bytes.NewBuffer([]byte(s)))
			req.Header.Add("Content-Type", "application/json")
			resp, err := n.nearHttpClient.Do(req)

			if err == nil {
				defer resp.Body.Close()
				result, err := io.ReadAll(resp.Body)
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
func (n NearApiImpl) GetBlock(ctx context.Context, blockId string) (Block, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"block_id": "%s"}}`, blockId)
	blockBytes, err := n.nearRPC.Query(ctx, s)
	if err != nil {
		return Block{}, err
	}

	newBlock, err := NewBlockFromBytes(blockBytes)
	if err != nil {
		return Block{}, err
	}

	// SECURITY defense-in-depth
	if newBlock.Header.Hash != blockId {
		return Block{}, errors.New("Returned block hash does not equal queried block hash")
	}

	return newBlock, err
}

// getBlockByHeight calls the NEAR RPC API to retrieve a block by its height (https://docs.near.org/api/rpc/block-chunk#block-details)
func (n NearApiImpl) GetBlockByHeight(ctx context.Context, blockHeight uint64) (Block, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"block_id": %d}}`, blockHeight)
	blockBytes, err := n.nearRPC.Query(ctx, s)
	if err != nil {
		return Block{}, err
	}
	newBlock, err := NewBlockFromBytes(blockBytes)
	if err != nil {
		return Block{}, err
	}

	// SECURITY defense-in-depth
	if newBlock.Header.Height != blockHeight {
		return Block{}, errors.New("Returned block height not equal queried block height")
	}
	return newBlock, nil
}

// getFinalBlock gets a finalized block from the NEAR RPC API using the parameter "finality": "final" (https://docs.near.org/api/rpc/block-chunk)
func (n NearApiImpl) GetFinalBlock(ctx context.Context) (Block, error) {
	s := `{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"finality": "final"}}`
	blockBytes, err := n.nearRPC.Query(ctx, s)
	if err != nil {
		return Block{}, err
	}
	return NewBlockFromBytes(blockBytes)
}

// getChunk gets a chunk from the NEAR RPC API: https://docs.near.org/api/rpc/block-chunk#chunk-details
func (n NearApiImpl) GetChunk(ctx context.Context, chunkHeader ChunkHeader) (Chunk, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "chunk", "params": {"chunk_id": "%s"}}`, chunkHeader.Hash)
	bytes, err := n.nearRPC.Query(ctx, s)
	if err != nil {
		return Chunk{}, err
	}
	newChunk, err := NewChunkFromBytes(bytes)
	if err != nil {
		return Chunk{}, err
	}

	// SECURITY defense-in-depth
	if newChunk.Hash != chunkHeader.Hash {
		fmt.Printf("queried hash=%s, return_hash=%s", chunkHeader.Hash, newChunk.Hash)
		return Chunk{}, errors.New("Returned chunk hash does not equal queried chunk hash")
	}
	return newChunk, nil
}

// getTxStatus queries status of a transaction by hash, returning the transaction_outcomes and receipts_outcomes
// sender_account_id is used to determine which shard to query for the transaction
// See https://docs.near.org/api/rpc/transactions#transaction-status
func (n NearApiImpl) GetTxStatus(ctx context.Context, txHash string, senderAccountId string) ([]byte, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "tx", "params": ["%s", "%s"]}`, txHash, senderAccountId)
	return n.nearRPC.Query(ctx, s)
}

func IsWellFormedHash(hash string) error {
	hashBytes, err := base58.Decode(hash)
	if err != nil {
		return err
	}
	if len(hashBytes) != 32 {
		return errors.New("base58-decoded hash is not 32 bytes")
	}
	return nil
}
