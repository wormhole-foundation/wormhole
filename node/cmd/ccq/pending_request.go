package ccq

import (
	"encoding/hex"
	"sync"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type PendingResponse struct {
	req          *gossipv1.SignedQueryRequest
	userName     string
	queryRequest *query.QueryRequest
	ch           chan *SignedResponse
	errCh        chan *ErrorEntry
}

type ErrorEntry struct {
	err    error
	status int
}

func NewPendingResponse(req *gossipv1.SignedQueryRequest, userName string, queryRequest *query.QueryRequest) *PendingResponse {
	return &PendingResponse{
		req:          req,
		userName:     userName,
		queryRequest: queryRequest,
		ch:           make(chan *SignedResponse),
		errCh:        make(chan *ErrorEntry),
	}
}

type PendingResponses struct {
	pendingResponses map[string]*PendingResponse
	mu               sync.RWMutex
	logger           *zap.Logger
}

func NewPendingResponses(logger *zap.Logger) *PendingResponses {
	return &PendingResponses{
		// Make this channel bigger than the number of responses we ever expect to get for a query.
		pendingResponses: make(map[string]*PendingResponse, 100),
		logger:           logger,
	}
}

func (p *PendingResponses) Add(r *PendingResponse) bool {
	signature := hex.EncodeToString(r.req.Signature)
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.pendingResponses[signature]; ok {
		// the request w/ this signature is already being handled
		// don't overwrite
		return false
	}
	p.pendingResponses[signature] = r
	p.updateMetricsAlreadyLocked(nil)
	return true
}

func (p *PendingResponses) Get(signature string) *PendingResponse {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if r, ok := p.pendingResponses[signature]; ok {
		return r
	}
	return nil
}

func (p *PendingResponses) Remove(r *PendingResponse) {
	signature := hex.EncodeToString(r.req.Signature)
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.pendingResponses, signature)
	p.updateMetricsAlreadyLocked(r)
}

func (p *PendingResponses) NumPending() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.pendingResponses)
}

func (p *PendingResponses) updateMetricsAlreadyLocked(reqRemoved *PendingResponse) {
	counts := make(map[vaa.ChainID]float64)
	if reqRemoved != nil {
		// We may have removed the last request for a chain. Make sure we always update that chain.
		for _, pcr := range reqRemoved.queryRequest.PerChainQueries {
			counts[pcr.ChainId] = 0
		}
	}
	for _, pr := range p.pendingResponses {
		for _, pcr := range pr.queryRequest.PerChainQueries {
			counts[pcr.ChainId] = counts[pcr.ChainId] + 1
		}
	}

	for chainId, count := range counts {
		currentNumConcurrentQueriesByChain.WithLabelValues(chainId.String()).Set(count)
		currVal, err := getGaugeValue(maxConcurrentQueriesByChain.WithLabelValues(chainId.String()))
		if err != nil {
			p.logger.Error("failed to read current value of max concurrent queries metric", zap.String("chainId", chainId.String()), zap.Error(err))
			continue
		}
		if count > currVal {
			p.logger.Info("updating max concurrent queries metric", zap.String("chain", chainId.String()), zap.Float64("oldMax", currVal), zap.Float64("newMax", count))
			maxConcurrentQueriesByChain.WithLabelValues(chainId.String()).Set(count)
		}
	}
}
