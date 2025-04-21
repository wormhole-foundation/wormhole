package ccq

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/certusone/wormhole/node/pkg/query/queryratelimit"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// MAX_BODY_SIZE caps request body size to prevent DoS attacks.
// Realistic queries are ~1-50KB raw (~2-125KB encoded).
// 512KB provides comfortable headroom while preventing abuse.
const MAX_BODY_SIZE = 512 * 1024

const (
	SignatureFormatRaw    = "raw"
	SignatureFormatEIP191 = "eip191"
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
	logger           *zap.Logger
	env              common.Environment
	signerKey        *ecdsa.PrivateKey
	pendingResponses *PendingResponses
	loggingMap       *LoggingMap

	policyProvider *queryratelimit.PolicyProvider
	limitEnforcer  *queryratelimit.Enforcer
}

func (s *httpServer) handleQuery(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for all requests.
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Set CORS headers for the preflight request
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "PUT, POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Signature-Format")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	start := time.Now()
	allQueryRequestsReceived.Inc()

	// Decode the body first. We need the query request bytes to compute the digest
	// for signature verification, so we cannot validate the signature before reading
	// the body. However, we aggressively cap MAX_BODY_SIZE (256KB) to limit resource
	// consumption from invalid/malicious requests. Signature validation happens
	// immediately after, before expensive unmarshal and validation operations.

	var q queryRequest
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MAX_BODY_SIZE)).Decode(&q)
	if err != nil {
		s.logger.Error("failed to decode body", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		invalidQueryRequestReceived.WithLabelValues("failed_to_decode_body").Inc()
		return
	}

	queryRequestBytes, err := hex.DecodeString(q.Bytes)
	if err != nil {
		s.logger.Error("failed to decode request bytes", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		invalidQueryRequestReceived.WithLabelValues("failed_to_decode_request").Inc()
		return
	}

	signature, err := hex.DecodeString(q.Signature)
	if err != nil {
		s.logger.Error("failed to decode signature bytes", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		invalidQueryRequestReceived.WithLabelValues("failed_to_decode_signature").Inc()
		return
	}

	// Basic signature format validation before expensive operations
	if len(signature) != 65 {
		s.logger.Error("invalid signature length", zap.Int("length", len(signature)))
		http.Error(w, fmt.Sprintf("signature must be 65 bytes, got %d", len(signature)), http.StatusBadRequest)
		invalidQueryRequestReceived.WithLabelValues("invalid_signature_length").Inc()
		return
	}

	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    signature,
	}

	// Check X-Signature-Format header early to fail fast on invalid header
	sigFormat := r.Header.Get("X-Signature-Format")
	if sigFormat == "" {
		sigFormat = SignatureFormatRaw
	} else if sigFormat != SignatureFormatRaw && sigFormat != SignatureFormatEIP191 {
		http.Error(w, "invalid X-Signature-Format value. Use 'eip191' or 'raw'", http.StatusBadRequest)
		invalidQueryRequestReceived.WithLabelValues("invalid_signature_format_header").Inc()
		return
	}

	var userIdentifier string // For logging and metrics
	var queryReq *query.QueryRequest

	// Recover signer address from signature prior to expensive unmarshal/validate
	digest := query.QueryRequestDigest(s.env, signedQueryRequest.QueryRequest)
	var signerAddr eth_common.Address
	switch sigFormat {
	case SignatureFormatEIP191:
		signerAddr, err = query.RecoverPrefixedSigner(digest.Bytes(), signedQueryRequest.Signature)
	case SignatureFormatRaw:
		signerAddr, err = query.RecoverQueryRequestSigner(digest.Bytes(), signedQueryRequest.Signature)
	}

	if err != nil {
		s.logger.Error("failed to recover signer from signature", zap.Error(err))
		http.Error(w, "invalid signature", http.StatusBadRequest)
		invalidQueryRequestReceived.WithLabelValues("failed_to_recover_signer").Inc()
		return
	}

	// Basic validation of query request structure (after signature verified)
	var qr query.QueryRequest
	err = qr.Unmarshal(signedQueryRequest.QueryRequest)
	if err != nil {
		s.logger.Error("failed to unmarshal request", zap.Error(err))
		http.Error(w, "failed to unmarshal request", http.StatusBadRequest)
		invalidQueryRequestReceived.WithLabelValues("failed_to_unmarshal_request").Inc()
		return
	}

	if err := qr.Validate(); err != nil {
		s.logger.Error("invalid query request", zap.Error(err))
		http.Error(w, "invalid query request", http.StatusBadRequest)
		invalidQueryRequestReceived.WithLabelValues("failed_to_validate_request").Inc()
		return
	}

	// Determine rate limit key: use staker address from signed payload if provided, otherwise signer
	var rateLimitKey eth_common.Address
	if qr.StakerAddress != nil {
		rateLimitKey = *qr.StakerAddress
		userIdentifier = "delegated:" + signerAddr.Hex() + "->staker:" + rateLimitKey.Hex()
		s.logger.Debug("delegated query", zap.String("signer", signerAddr.Hex()), zap.String("staker", rateLimitKey.Hex()))
		delegatedQueriesReceived.Inc()
	} else {
		rateLimitKey = signerAddr
		userIdentifier = "signer:" + signerAddr.Hex()
		s.logger.Debug("self-staking query", zap.String("signer", signerAddr.Hex()))
		selfStakingQueriesReceived.Inc()
	}

	// Track total requests by user
	totalRequestsByUser.WithLabelValues(userIdentifier).Inc()

	// If staking-based rate limiting is enabled, enforce it here
	if s.policyProvider != nil && s.limitEnforcer != nil {
		// Determine staker address (same as rateLimitKey above)
		stakerAddr := rateLimitKey

		// Fetch staking policy
		policy, err := s.policyProvider.GetPolicy(r.Context(), signerAddr, stakerAddr)
		if err != nil {
			s.logger.Error("failed to fetch staking policy",
				zap.String("signer", signerAddr.Hex()),
				zap.String("staker", stakerAddr.Hex()),
				zap.Error(err))
			http.Error(w, "failed to verify staking eligibility", http.StatusInternalServerError)
			invalidQueryRequestReceived.WithLabelValues("failed_to_fetch_policy").Inc()
			queryratelimit.StakingPolicyRejections.WithLabelValues("failed_to_fetch_policy").Inc()
			invalidRequestsByUser.WithLabelValues(userIdentifier).Inc()
			return
		}

		// Check if user has any limits (i.e., has stake)
		if len(policy.Limits.Types) == 0 {
			s.logger.Info("requestor has insufficient stake",
				zap.String("signer", signerAddr.Hex()),
				zap.String("staker", stakerAddr.Hex()))

			// Provide more specific error message for delegation scenarios
			var errorMsg string
			if signerAddr != stakerAddr {
				errorMsg = fmt.Sprintf("insufficient stake for CCQ access: signer %s is not authorized to use staker %s's rate limits (or staker has no stake)",
					signerAddr.Hex(), stakerAddr.Hex())
			} else {
				errorMsg = fmt.Sprintf("insufficient stake for CCQ access: address %s has no stake or is below minimum threshold", signerAddr.Hex())
			}

			http.Error(w, errorMsg, http.StatusForbidden)
			invalidQueryRequestReceived.WithLabelValues("insufficient_stake").Inc()
			queryratelimit.StakingPolicyRejections.WithLabelValues("insufficient_stake").Inc()
			invalidRequestsByUser.WithLabelValues(userIdentifier).Inc()
			return
		}

		// Build action for rate limit enforcement
		action := &queryratelimit.Action{
			Key:   stakerAddr,
			Time:  time.Now(),
			Types: make(map[uint8]int),
		}

		for _, pcq := range qr.PerChainQueries {
			action.Types[uint8(pcq.Query.Type())] += 1
		}

		// Enforce rate limits
		limitResult, err := s.limitEnforcer.EnforcePolicy(r.Context(), policy, action)
		if err != nil {
			s.logger.Error("failed to enforce rate limit",
				zap.String("signer", signerAddr.Hex()),
				zap.String("staker", stakerAddr.Hex()),
				zap.Error(err))
			http.Error(w, "failed to enforce rate limit", http.StatusInternalServerError)
			invalidQueryRequestReceived.WithLabelValues("failed_to_enforce_rate_limit").Inc()
			queryratelimit.StakingPolicyRejections.WithLabelValues("failed_to_enforce_rate_limit").Inc()
			invalidRequestsByUser.WithLabelValues(userIdentifier).Inc()
			return
		}

		if !limitResult.Allowed {
			s.logger.Info("rate limit exceeded",
				zap.String("signer", signerAddr.Hex()),
				zap.String("staker", stakerAddr.Hex()),
				zap.Any("exceededTypes", limitResult.ExceededTypes))
			http.Error(w, fmt.Sprintf("rate limit exceeded for query types: %v", limitResult.ExceededTypes), http.StatusTooManyRequests)
			invalidQueryRequestReceived.WithLabelValues("rate_limit_exceeded").Inc()
			queryratelimit.StakingPolicyRejections.WithLabelValues("rate_limit_exceeded").Inc()
			rateLimitExceededByUser.WithLabelValues(userIdentifier).Inc()
			invalidRequestsByUser.WithLabelValues(userIdentifier).Inc()
			return
		}

		s.logger.Debug("rate limit check passed",
			zap.String("signer", signerAddr.Hex()),
			zap.String("staker", stakerAddr.Hex()))
	}

	queryReq = &qr

	requestId := hex.EncodeToString(signedQueryRequest.Signature)
	s.logger.Info("received request from client", zap.String("userId", userIdentifier), zap.String("requestId", requestId))

	m := gossipv1.GossipMessage{
		Message: &gossipv1.GossipMessage_SignedQueryRequest{
			SignedQueryRequest: signedQueryRequest,
		},
	}

	b, err := proto.Marshal(&m)
	if err != nil {
		s.logger.Error("failed to marshal gossip message", zap.String("userId", userIdentifier), zap.String("requestId", requestId), zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		invalidQueryRequestReceived.WithLabelValues("failed_to_marshal_gossip_msg").Inc()
		invalidRequestsByUser.WithLabelValues(userIdentifier).Inc()
		return
	}

	pendingResponse := NewPendingResponse(signedQueryRequest, userIdentifier, queryReq)
	added := s.pendingResponses.Add(pendingResponse)
	if !added {
		s.logger.Info("duplicate request", zap.String("userId", userIdentifier), zap.String("requestId", requestId))
		http.Error(w, "Duplicate request", http.StatusBadRequest)
		invalidQueryRequestReceived.WithLabelValues("duplicate_request").Inc()
		invalidRequestsByUser.WithLabelValues(userIdentifier).Inc()
		return
	}

	s.loggingMap.AddRequest(requestId)

	s.logger.Info("posting request to gossip", zap.String("userId", userIdentifier), zap.String("requestId", requestId))
	err = s.topic.Publish(r.Context(), b)
	if err != nil {
		s.logger.Error("failed to publish gossip message", zap.String("userId", userIdentifier), zap.String("requestId", requestId), zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		invalidQueryRequestReceived.WithLabelValues("failed_to_publish_gossip_msg").Inc()
		invalidRequestsByUser.WithLabelValues(userIdentifier).Inc()
		s.pendingResponses.Remove(pendingResponse)
		return
	}

	// Wait for the response or timeout
	var querySucceeded bool
outer:
	select {
	case <-time.After(query.RequestTimeout + 5*time.Second):
		maxMatchingResponses, outstandingResponses, quorum := pendingResponse.getStats()
		s.logger.Info("publishing time out to client",
			zap.String("userId", userIdentifier),
			zap.String("requestId", requestId),
			zap.Int("maxMatchingResponses", maxMatchingResponses),
			zap.Int("outstandingResponses", outstandingResponses),
			zap.Int("quorum", quorum),
		)
		queryTimeoutsByUser.WithLabelValues(pendingResponse.userName).Inc()
		http.Error(w, "Timed out waiting for response", http.StatusGatewayTimeout)
	case res := <-pendingResponse.ch:
		s.logger.Info("publishing response to client", zap.String("userId", userIdentifier), zap.String("requestId", requestId))
		resBytes, respMarshalErr := res.Response.Marshal()
		if respMarshalErr != nil {
			s.logger.Error("failed to marshal response", zap.String("userId", userIdentifier), zap.String("requestId", requestId), zap.Error(respMarshalErr))
			http.Error(w, respMarshalErr.Error(), http.StatusInternalServerError)
			invalidQueryRequestReceived.WithLabelValues("failed_to_marshal_response").Inc()
			invalidRequestsByUser.WithLabelValues(userIdentifier).Inc()
			break
		}
		// Signature indices must be ascending for on-chain verification
		sort.Slice(res.Signatures, func(i, j int) bool {
			return res.Signatures[i].Index < res.Signatures[j].Index
		})
		signatures := make([]string, 0, len(res.Signatures))
		for _, sig := range res.Signatures {
			if sig.Index > math.MaxUint8 {
				boundsErr := "Signature index out of bounds"
				s.logger.Error(boundsErr, zap.Int("sig.Index", sig.Index))
				http.Error(w, boundsErr, http.StatusInternalServerError)
				invalidQueryRequestReceived.WithLabelValues("failed_to_marshal_response").Inc()
				invalidRequestsByUser.WithLabelValues(userIdentifier).Inc()
				break outer
			}
			// ECDSA signature + a byte for the index of the guardian in the guardian set
			signature := fmt.Sprintf("%s%02x", sig.Signature, uint8(sig.Index)) // #nosec G115 -- This is validated above
			signatures = append(signatures, signature)
		}
		w.Header().Add("Content-Type", "application/json")
		encodeErr := json.NewEncoder(w).Encode(&queryResponse{
			Signatures: signatures,
			Bytes:      hex.EncodeToString(resBytes),
		})
		if encodeErr != nil {
			s.logger.Error("failed to encode response", zap.String("userId", userIdentifier), zap.String("requestId", requestId), zap.Error(encodeErr))
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			invalidQueryRequestReceived.WithLabelValues("failed_to_encode_response").Inc()
			invalidRequestsByUser.WithLabelValues(userIdentifier).Inc()
			break
		}
		querySucceeded = true
	case errEntry := <-pendingResponse.errCh:
		s.logger.Info("publishing error response to client", zap.String("userId", userIdentifier), zap.String("requestId", requestId), zap.Int("status", errEntry.status), zap.Error(errEntry.err))
		http.Error(w, errEntry.err.Error(), errEntry.status)
		// Metrics have already been pegged.
		break
	}

	totalQueryTime.Observe(float64(time.Since(start).Milliseconds()))
	if querySucceeded {
		successfulQueriesByUser.WithLabelValues(pendingResponse.userName).Inc()
		validQueryRequestsReceived.Inc()
	}
	s.pendingResponses.Remove(pendingResponse)
}

func NewHTTPServer(addr string, t *pubsub.Topic, signerKey *ecdsa.PrivateKey, p *PendingResponses, logger *zap.Logger, env common.Environment, loggingMap *LoggingMap, policyProvider *queryratelimit.PolicyProvider, limitEnforcer *queryratelimit.Enforcer) *http.Server {
	s := &httpServer{
		topic:            t,
		signerKey:        signerKey,
		policyProvider:   policyProvider,
		limitEnforcer:    limitEnforcer,
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
