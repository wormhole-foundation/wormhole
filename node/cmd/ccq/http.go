package ccq

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/gorilla/mux"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"google.golang.org/protobuf/proto"
)

type queryRequest struct {
	Bytes     string `json:"bytes"`
	Signature string `json:"signature"`
}

type queryResponse struct {
	Bytes      string   `json:"bytes"`
	Signatures []string `json:"signatures"`
}

type httpServer struct {
	topic            *pubsub.Topic
	pendingResponses *PendingResponses
}

func (s *httpServer) handleQuery(w http.ResponseWriter, r *http.Request) {
	var q queryRequest
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	queryRequestBytes, err := hex.DecodeString(q.Bytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: check if request signer is authorized on Wormchain

	signature, err := hex.DecodeString(q.Signature)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    signature,
	}

	// TODO: validate request before publishing

	m := gossipv1.GossipMessage{
		Message: &gossipv1.GossipMessage_SignedQueryRequest{
			SignedQueryRequest: signedQueryRequest,
		},
	}

	b, err := proto.Marshal(&m)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pendingResponse := NewPendingResponse(signedQueryRequest)
	added := s.pendingResponses.Add(pendingResponse)
	if !added {
		http.Error(w, "Duplicate request", http.StatusInternalServerError)
		return
	}

	err = s.topic.Publish(r.Context(), b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		s.pendingResponses.Remove(pendingResponse)
		return
	}

	// Wait for the response or timeout
	select {
	case <-time.After(query.RequestTimeout + 5*time.Second):
		http.Error(w, "Timed out waiting for response", http.StatusGatewayTimeout)
	case res := <-pendingResponse.ch:
		resBytes, err := res.Response.Marshal()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			break
		}
		// Signature indices must be ascending for on-chain verification
		sort.Slice(res.Signatures, func(i, j int) bool {
			return res.Signatures[i].Index < res.Signatures[j].Index
		})
		signatures := make([]string, 0, len(res.Signatures))
		for _, s := range res.Signatures {
			// ECDSA signature + a byte for the index of the guardian in the guardian set
			signature := fmt.Sprintf("%s%02x", s.Signature, uint8(s.Index))
			signatures = append(signatures, signature)
		}
		w.Header().Add("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(&queryResponse{
			Signatures: signatures,
			Bytes:      hex.EncodeToString(resBytes),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

	s.pendingResponses.Remove(pendingResponse)
}

func NewHTTPServer(addr string, t *pubsub.Topic, p *PendingResponses) *http.Server {
	s := &httpServer{
		topic:            t,
		pendingResponses: p,
	}
	r := mux.NewRouter()
	r.HandleFunc("/v1/query", s.handleQuery).Methods("PUT")
	return &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
