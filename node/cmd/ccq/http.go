package ccq

import (
	"crypto/ecdsa"
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
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type queryRequest struct {
	ApiKey    string `json:"api_key"`
	Bytes     string `json:"bytes"`
	Signature string `json:"signature"`
}

type queryResponse struct {
	Bytes      string   `json:"bytes"`
	Signatures []string `json:"signatures"`
}

type httpServer struct {
	topic            *pubsub.Topic
	logger           *zap.Logger
	permissions      Permissions
	signerKey        *ecdsa.PrivateKey
	pendingResponses *PendingResponses
}

func (s *httpServer) handleQuery(w http.ResponseWriter, r *http.Request) {
	var q queryRequest
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// There should be one and only one API key in the header.
	apiKey, exists := r.Header["X-Api-Key"]
	if !exists || len(apiKey) != 1 {
		s.logger.Debug("received a request without an api key", zap.Stringer("url", r.URL), zap.Error(err))
		http.Error(w, "api key is missing", http.StatusBadRequest)
		return
	}

	queryRequestBytes, err := hex.DecodeString(q.Bytes)
	if err != nil {
		s.logger.Debug("failed to decode request bytes", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	signature, err := hex.DecodeString(q.Signature)
	if err != nil {
		s.logger.Debug("failed to decode signature bytes", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    signature,
	}

	if err := validateRequest(s.logger, s.permissions, s.signerKey, apiKey[0], signedQueryRequest); err != nil {
		s.logger.Debug("invalid request", zap.String("api_key", apiKey[0]), zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	m := gossipv1.GossipMessage{
		Message: &gossipv1.GossipMessage_SignedQueryRequest{
			SignedQueryRequest: signedQueryRequest,
		},
	}

	b, err := proto.Marshal(&m)
	if err != nil {
		s.logger.Debug("failed to marshal gossip message", zap.Error(err))
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
		s.logger.Debug("failed to publish gossip message", zap.Error(err))
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

func (s *httpServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("health check")
}

func NewHTTPServer(addr string, t *pubsub.Topic, permissions Permissions, signerKey *ecdsa.PrivateKey, p *PendingResponses, logger *zap.Logger) *http.Server {
	s := &httpServer{
		topic:            t,
		permissions:      permissions,
		signerKey:        signerKey,
		pendingResponses: p,
		logger:           logger,
	}
	r := mux.NewRouter()
	r.HandleFunc("/v1/query", s.handleQuery).Methods("PUT")
	r.HandleFunc("/v1/health", s.handleHealth).Methods("GET")
	return &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
