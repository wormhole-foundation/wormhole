package ccq

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/gorilla/mux"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const MAX_BODY_SIZE = 5 * 1024 * 1024

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
	logger           *zap.Logger
	env              common.Environment
	permissions      Permissions
	signerKey        *ecdsa.PrivateKey
	pendingResponses *PendingResponses
}

func (s *httpServer) handleQuery(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for all requests.
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Set CORS headers for the preflight request
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "PUT, POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Api-Key")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// There should be one and only one API key in the header.
	apiKeys, exists := r.Header["X-Api-Key"]
	if !exists || len(apiKeys) != 1 {
		s.logger.Debug("received a request with the wrong number of api keys", zap.Stringer("url", r.URL), zap.Int("numApiKeys", len(apiKeys)))
		http.Error(w, "api key is missing", http.StatusUnauthorized)
		return
	}
	apiKey := apiKeys[0]

	// Make sure the user is authorized before we go any farther.
	_, exists = s.permissions[strings.ToLower(apiKey)]
	if !exists {
		s.logger.Debug("invalid api key", zap.String("apiKey", apiKey))
		http.Error(w, "invalid api key", http.StatusForbidden)
		return
	}

	var q queryRequest
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MAX_BODY_SIZE)).Decode(&q)
	if err != nil {
		s.logger.Debug("failed to decode body", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	if status, err := validateRequest(s.logger, s.env, s.permissions, s.signerKey, apiKey, signedQueryRequest); err != nil {
		// Don't need to log here because the details were logged in the function.
		http.Error(w, err.Error(), status)
		return
	}

	m := gossipv1.GossipMessage{
		Message: &gossipv1.GossipMessage_SignedQueryRequest{
			SignedQueryRequest: signedQueryRequest,
		},
	}

	b, err := proto.Marshal(&m)
	if err != nil {
		s.logger.Error("failed to marshal gossip message", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pendingResponse := NewPendingResponse(signedQueryRequest)
	added := s.pendingResponses.Add(pendingResponse)
	if !added {
		http.Error(w, "Duplicate request", http.StatusBadRequest)
		return
	}

	err = s.topic.Publish(r.Context(), b)
	if err != nil {
		s.logger.Error("failed to publish gossip message", zap.Error(err))
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
			s.logger.Error("failed to marshal response", zap.Error(err))
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
			s.logger.Error("failed to encode response", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

	s.pendingResponses.Remove(pendingResponse)
}

func (s *httpServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("health check")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "ok")
}

func NewHTTPServer(addr string, t *pubsub.Topic, permissions Permissions, signerKey *ecdsa.PrivateKey, p *PendingResponses, logger *zap.Logger, env common.Environment) *http.Server {
	s := &httpServer{
		topic:            t,
		permissions:      permissions,
		signerKey:        signerKey,
		pendingResponses: p,
		logger:           logger,
		env:              env,
	}
	r := mux.NewRouter()
	r.HandleFunc("/v1/query", s.handleQuery).Methods("PUT", "POST", "OPTIONS")
	r.HandleFunc("/v1/health", s.handleHealth).Methods("GET")
	return &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
