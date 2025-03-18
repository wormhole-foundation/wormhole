package ccq

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
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
	permissions      *Permissions
	signerKey        *ecdsa.PrivateKey
	pendingResponses *PendingResponses
	loggingMap       *LoggingMap
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

	start := time.Now()
	allQueryRequestsReceived.Inc()

	// Decode the body first. This is because the library seems to hang if we receive a large body and return without decoding it.
	// This could be a slight waste of resources, but should not be a DoS risk because we cap the max body size.

	var q queryRequest
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MAX_BODY_SIZE)).Decode(&q)
	if err != nil {
		s.logger.Error("failed to decode body", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		invalidQueryRequestReceived.WithLabelValues("failed_to_decode_body").Inc()
		return
	}

	// There should be one and only one API key in the header.
	apiKeys, exists := r.Header["X-Api-Key"]
	if !exists || len(apiKeys) != 1 {
		s.logger.Error("received a request with the wrong number of api keys", zap.Stringer("url", r.URL), zap.Int("numApiKeys", len(apiKeys)))
		http.Error(w, "api key is missing", http.StatusUnauthorized)
		invalidQueryRequestReceived.WithLabelValues("missing_api_key").Inc()
		return
	}
	apiKey := strings.ToLower(apiKeys[0])

	// Make sure the user is authorized before we go any farther.
	permEntry, exists := s.permissions.GetUserEntry(apiKey)
	if !exists {
		s.logger.Error("invalid api key", zap.String("apiKey", apiKey))
		http.Error(w, "invalid api key", http.StatusForbidden)
		invalidQueryRequestReceived.WithLabelValues("invalid_api_key").Inc()
		return
	}

	if permEntry.rateLimiter != nil && !permEntry.rateLimiter.Allow() {
		s.logger.Debug("denying request due to rate limit", zap.String("userId", permEntry.userName))
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		rateLimitExceededByUser.WithLabelValues(permEntry.userName).Inc()
		return
	}

	totalRequestsByUser.WithLabelValues(permEntry.userName).Inc()

	queryRequestBytes, err := hex.DecodeString(q.Bytes)
	if err != nil {
		s.logger.Error("failed to decode request bytes", zap.String("userId", permEntry.userName), zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		invalidQueryRequestReceived.WithLabelValues("failed_to_decode_request").Inc()
		invalidRequestsByUser.WithLabelValues(permEntry.userName).Inc()
		return
	}

	signature, err := hex.DecodeString(q.Signature)
	if err != nil {
		s.logger.Error("failed to decode signature bytes", zap.String("userId", permEntry.userName), zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		invalidQueryRequestReceived.WithLabelValues("failed_to_decode_signature").Inc()
		invalidRequestsByUser.WithLabelValues(permEntry.userName).Inc()
		return
	}

	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    signature,
	}

	status, queryReq, err := validateRequest(s.logger, s.env, s.permissions, s.signerKey, apiKey, signedQueryRequest)
	if err != nil {
		s.logger.Error("failed to validate request", zap.String("userId", permEntry.userName), zap.String("requestId", hex.EncodeToString(signedQueryRequest.Signature)), zap.Int("status", status), zap.Error(err))
		http.Error(w, err.Error(), status)
		// Error specific metric has already been pegged.
		invalidRequestsByUser.WithLabelValues(permEntry.userName).Inc()
		return
	}

	requestId := hex.EncodeToString(signedQueryRequest.Signature)
	s.logger.Info("received request from client", zap.String("userId", permEntry.userName), zap.String("requestId", requestId))

	m := gossipv1.GossipMessage{
		Message: &gossipv1.GossipMessage_SignedQueryRequest{
			SignedQueryRequest: signedQueryRequest,
		},
	}

	b, err := proto.Marshal(&m)
	if err != nil {
		s.logger.Error("failed to marshal gossip message", zap.String("userId", permEntry.userName), zap.String("requestId", requestId), zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		invalidQueryRequestReceived.WithLabelValues("failed_to_marshal_gossip_msg").Inc()
		invalidRequestsByUser.WithLabelValues(permEntry.userName).Inc()
		return
	}

	pendingResponse := NewPendingResponse(signedQueryRequest, permEntry.userName, queryReq)
	added := s.pendingResponses.Add(pendingResponse)
	if !added {
		s.logger.Info("duplicate request", zap.String("userId", permEntry.userName), zap.String("requestId", requestId))
		http.Error(w, "Duplicate request", http.StatusBadRequest)
		invalidQueryRequestReceived.WithLabelValues("duplicate_request").Inc()
		invalidRequestsByUser.WithLabelValues(permEntry.userName).Inc()
		return
	}

	if permEntry.logResponses {
		s.loggingMap.AddRequest(requestId)
	}

	s.logger.Info("posting request to gossip", zap.String("userId", permEntry.userName), zap.String("requestId", requestId))
	err = s.topic.Publish(r.Context(), b)
	if err != nil {
		s.logger.Error("failed to publish gossip message", zap.String("userId", permEntry.userName), zap.String("requestId", requestId), zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		invalidQueryRequestReceived.WithLabelValues("failed_to_publish_gossip_msg").Inc()
		invalidRequestsByUser.WithLabelValues(permEntry.userName).Inc()
		s.pendingResponses.Remove(pendingResponse)
		return
	}

	// Wait for the response or timeout
outer:
	select {
	case <-time.After(query.RequestTimeout + 5*time.Second):
		maxMatchingResponses, outstandingResponses, quorum := pendingResponse.getStats()
		s.logger.Info("publishing time out to client",
			zap.String("userId", permEntry.userName),
			zap.String("requestId", requestId),
			zap.Int("maxMatchingResponses", maxMatchingResponses),
			zap.Int("outstandingResponses", outstandingResponses),
			zap.Int("quorum", quorum),
		)
		http.Error(w, "Timed out waiting for response", http.StatusGatewayTimeout)
		queryTimeoutsByUser.WithLabelValues(permEntry.userName).Inc()
		failedQueriesByUser.WithLabelValues(permEntry.userName).Inc()
	case res := <-pendingResponse.ch:
		s.logger.Info("publishing response to client", zap.String("userId", permEntry.userName), zap.String("requestId", requestId))
		resBytes, err := res.Response.Marshal()
		if err != nil {
			s.logger.Error("failed to marshal response", zap.String("userId", permEntry.userName), zap.String("requestId", requestId), zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			invalidQueryRequestReceived.WithLabelValues("failed_to_marshal_response").Inc()
			failedQueriesByUser.WithLabelValues(permEntry.userName).Inc()
			break
		}
		// Signature indices must be ascending for on-chain verification
		sort.Slice(res.Signatures, func(i, j int) bool {
			return res.Signatures[i].Index < res.Signatures[j].Index
		})
		signatures := make([]string, 0, len(res.Signatures))
		for _, sig := range res.Signatures {
			if sig.Index > math.MaxUint8 {
				s.logger.Error("Signature index out of bounds", zap.Int("sig.Index", sig.Index))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				invalidQueryRequestReceived.WithLabelValues("failed_to_marshal_response").Inc()
				failedQueriesByUser.WithLabelValues(permEntry.userName).Inc()
				break outer
			}
			// ECDSA signature + a byte for the index of the guardian in the guardian set
			signature := fmt.Sprintf("%s%02x", sig.Signature, uint8(sig.Index)) // #nosec G115 -- This is validated above
			signatures = append(signatures, signature)
		}
		w.Header().Add("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(&queryResponse{
			Signatures: signatures,
			Bytes:      hex.EncodeToString(resBytes),
		})
		if err != nil {
			s.logger.Error("failed to encode response", zap.String("userId", permEntry.userName), zap.String("requestId", requestId), zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			invalidQueryRequestReceived.WithLabelValues("failed_to_encode_response").Inc()
			failedQueriesByUser.WithLabelValues(permEntry.userName).Inc()
			break
		}
		successfulQueriesByUser.WithLabelValues(permEntry.userName).Inc()
	case errEntry := <-pendingResponse.errCh:
		s.logger.Info("publishing error response to client", zap.String("userId", permEntry.userName), zap.String("requestId", requestId), zap.Int("status", errEntry.status), zap.Error(errEntry.err))
		http.Error(w, errEntry.err.Error(), errEntry.status)
		// Metrics have already been pegged.
		break
	}

	totalQueryTime.Observe(float64(time.Since(start).Milliseconds()))
	validQueryRequestsReceived.Inc()
	s.pendingResponses.Remove(pendingResponse)
}

func NewHTTPServer(addr string, t *pubsub.Topic, permissions *Permissions, signerKey *ecdsa.PrivateKey, p *PendingResponses, logger *zap.Logger, env common.Environment, loggingMap *LoggingMap) *http.Server {
	s := &httpServer{
		topic:            t,
		permissions:      permissions,
		signerKey:        signerKey,
		pendingResponses: p,
		logger:           logger,
		env:              env,
		loggingMap:       loggingMap,
	}
	r := mux.NewRouter()
	r.HandleFunc("/v1/query", s.handleQuery).Methods("PUT", "POST", "OPTIONS")
	return &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
