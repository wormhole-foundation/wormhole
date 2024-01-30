package ccq

import (
	"encoding/hex"
	"sync"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
)

type PendingResponse struct {
	req      *gossipv1.SignedQueryRequest
	userName string
	ch       chan *SignedResponse
	errCh    chan *ErrorEntry
}

type ErrorEntry struct {
	err    error
	status int
}

func NewPendingResponse(req *gossipv1.SignedQueryRequest, userName string) *PendingResponse {
	return &PendingResponse{
		req:      req,
		userName: userName,
		ch:       make(chan *SignedResponse),
		errCh:    make(chan *ErrorEntry),
	}
}

type PendingResponses struct {
	pendingResponses map[string]*PendingResponse
	mu               sync.RWMutex
}

func NewPendingResponses() *PendingResponses {
	return &PendingResponses{
		// Make this channel bigger than the number of responses we ever expect to get for a query.
		pendingResponses: make(map[string]*PendingResponse, 100),
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
}

func (p *PendingResponses) NumPending() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.pendingResponses)
}
