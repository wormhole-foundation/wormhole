package ccq

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	eth_common "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethRpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/gagliardetto/solana-go"
)

func FetchCurrentGuardianSet(ctx context.Context, rpcUrl, coreAddr string) (*common.GuardianSet, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	ethContract := eth_common.HexToAddress(coreAddr)
	rawClient, err := ethRpc.DialContext(ctx, rpcUrl)
	if err != nil {
		return nil, errors.New("failed to connect to ethereum")
	}
	client := ethClient.NewClient(rawClient)
	caller, err := ethAbi.NewAbiCaller(ethContract, client)
	if err != nil {
		return nil, errors.New("failed to create caller")
	}
	currentIndex, err := caller.GetCurrentGuardianSetIndex(&ethBind.CallOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("error requesting current guardian set index: %w", err)
	}
	gs, err := caller.GetGuardianSet(&ethBind.CallOpts{Context: ctx}, currentIndex)
	if err != nil {
		return nil, fmt.Errorf("error requesting current guardian set value: %w", err)
	}
	return &common.GuardianSet{
		Keys:  gs.Keys,
		Index: currentIndex,
	}, nil
}

// validateRequest verifies that this API key is allowed to do all of the calls in this request. In the case of an error, it returns the HTTP status.
func validateRequest(logger *zap.Logger, env common.Environment, perms *Permissions, signerKey *ecdsa.PrivateKey, apiKey string, qr *gossipv1.SignedQueryRequest) (int, *query.QueryRequest, error) {
	permsForUser, exists := perms.GetUserEntry(apiKey)
	if !exists {
		logger.Debug("invalid api key", zap.String("apiKey", apiKey))
		invalidQueryRequestReceived.WithLabelValues("invalid_api_key").Inc()
		return http.StatusForbidden, nil, errors.New("invalid api key")
	}

	// TODO: Should we verify the signatures?

	if len(qr.Signature) == 0 {
		if !permsForUser.allowUnsigned || signerKey == nil {
			logger.Debug("request not signed and unsigned requests not supported for this user",
				zap.String("userName", permsForUser.userName),
				zap.Bool("allowUnsigned", permsForUser.allowUnsigned),
				zap.Bool("signerKeyConfigured", signerKey != nil),
			)
			invalidQueryRequestReceived.WithLabelValues("request_not_signed").Inc()
			return http.StatusBadRequest, nil, errors.New("request not signed")
		}

		// Sign the request using our key.
		var err error
		digest := query.QueryRequestDigest(env, qr.QueryRequest)
		qr.Signature, err = ethCrypto.Sign(digest.Bytes(), signerKey)
		if err != nil {
			logger.Debug("failed to sign request", zap.String("userName", permsForUser.userName), zap.Error(err))
			invalidQueryRequestReceived.WithLabelValues("failed_to_sign_request").Inc()
			return http.StatusInternalServerError, nil, fmt.Errorf("failed to sign request: %w", err)
		}
	}

	var queryRequest query.QueryRequest
	err := queryRequest.Unmarshal(qr.QueryRequest)
	if err != nil {
		logger.Debug("failed to unmarshal request", zap.String("userName", permsForUser.userName), zap.Error(err))
		invalidQueryRequestReceived.WithLabelValues("failed_to_unmarshal_request").Inc()
		return http.StatusBadRequest, nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	// Make sure the overall query request is sane.
	if err := queryRequest.Validate(); err != nil {
		logger.Debug("failed to validate request", zap.String("userName", permsForUser.userName), zap.Error(err))
		invalidQueryRequestReceived.WithLabelValues("failed_to_validate_request").Inc()
		return http.StatusBadRequest, nil, fmt.Errorf("failed to validate request: %w", err)
	}

	// Make sure they are allowed to make all of the calls that they are asking for.
	for _, pcq := range queryRequest.PerChainQueries {
		var status int
		var err error
		switch q := pcq.Query.(type) {
		case *query.EthCallQueryRequest:
			status, err = validateCallData(logger, permsForUser, "ethCall", pcq.ChainId, q.CallData)
		case *query.EthCallByTimestampQueryRequest:
			status, err = validateCallData(logger, permsForUser, "ethCallByTimestamp", pcq.ChainId, q.CallData)
		case *query.EthCallWithFinalityQueryRequest:
			status, err = validateCallData(logger, permsForUser, "ethCallWithFinality", pcq.ChainId, q.CallData)
		case *query.SolanaAccountQueryRequest:
			status, err = validateSolanaAccountQuery(logger, permsForUser, "solAccount", pcq.ChainId, q)
		case *query.SolanaPdaQueryRequest:
			status, err = validateSolanaPdaQuery(logger, permsForUser, "solPDA", pcq.ChainId, q)
		default:
			logger.Debug("unsupported query type", zap.String("userName", permsForUser.userName), zap.Any("type", pcq.Query))
			invalidQueryRequestReceived.WithLabelValues("unsupported_query_type").Inc()
			return http.StatusBadRequest, nil, errors.New("unsupported query type")
		}

		if err != nil {
			// Metric is pegged below.
			return status, nil, err
		}
	}

	logger.Debug("submitting query request", zap.String("userName", permsForUser.userName))
	return http.StatusOK, &queryRequest, nil
}

// validateCallData performs verification on all of the call data objects in a query.
func validateCallData(logger *zap.Logger, permsForUser *permissionEntry, callTag string, chainId vaa.ChainID, callData []*query.EthCallData) (int, error) {
	for _, cd := range callData {
		contractAddress, err := vaa.BytesToAddress(cd.To)
		if err != nil {
			logger.Debug("failed to parse contract address", zap.String("userName", permsForUser.userName), zap.String("contract", hex.EncodeToString(cd.To)), zap.Error(err))
			invalidQueryRequestReceived.WithLabelValues("invalid_contract_address").Inc()
			return http.StatusBadRequest, fmt.Errorf("failed to parse contract address: %w", err)
		}
		if len(cd.Data) < ETH_CALL_SIG_LENGTH {
			logger.Debug("eth call data must be at least four bytes", zap.String("userName", permsForUser.userName), zap.String("data", hex.EncodeToString(cd.Data)))
			invalidQueryRequestReceived.WithLabelValues("bad_call_data").Inc()
			return http.StatusBadRequest, errors.New("eth call data must be at least four bytes")
		}
		if !permsForUser.allowAnything {
			call := hex.EncodeToString(cd.Data[0:ETH_CALL_SIG_LENGTH])
			callKey := fmt.Sprintf("%s:%d:%s:%s", callTag, chainId, contractAddress, call)
			if _, exists := permsForUser.allowedCalls[callKey]; !exists {
				// The call data doesn't exist including the contract address. See if it's covered by a wildcard.
				wildCardCallKey := fmt.Sprintf("%s:%d:*:%s", callTag, chainId, call)
				if _, exists := permsForUser.allowedCalls[wildCardCallKey]; !exists {
					logger.Debug("requested call not authorized", zap.String("userName", permsForUser.userName), zap.String("callKey", callKey))
					invalidQueryRequestReceived.WithLabelValues("call_not_authorized").Inc()
					return http.StatusBadRequest, fmt.Errorf(`call "%s" not authorized`, callKey)
				}
			}
		}

		totalRequestedCallsByChain.WithLabelValues(chainId.String()).Inc()
	}

	return http.StatusOK, nil
}

// validateSolanaAccountQuery performs verification on a Solana sol_account query.
func validateSolanaAccountQuery(logger *zap.Logger, permsForUser *permissionEntry, callTag string, chainId vaa.ChainID, q *query.SolanaAccountQueryRequest) (int, error) {
	if !permsForUser.allowAnything {
		for _, acct := range q.Accounts {
			callKey := fmt.Sprintf("%s:%d:%s", callTag, chainId, solana.PublicKey(acct).String())
			if _, exists := permsForUser.allowedCalls[callKey]; !exists {
				logger.Debug("requested call not authorized", zap.String("userName", permsForUser.userName), zap.String("callKey", callKey))
				invalidQueryRequestReceived.WithLabelValues("call_not_authorized").Inc()
				return http.StatusForbidden, fmt.Errorf(`call "%s" not authorized`, callKey)
			}

			totalRequestedCallsByChain.WithLabelValues(chainId.String()).Inc()
		}
	}

	return http.StatusOK, nil
}

// validateSolanaPdaQuery performs verification on a Solana sol_account query.
func validateSolanaPdaQuery(logger *zap.Logger, permsForUser *permissionEntry, callTag string, chainId vaa.ChainID, q *query.SolanaPdaQueryRequest) (int, error) {
	if !permsForUser.allowAnything {
		for _, acct := range q.PDAs {
			callKey := fmt.Sprintf("%s:%d:%s", callTag, chainId, solana.PublicKey(acct.ProgramAddress).String())
			if _, exists := permsForUser.allowedCalls[callKey]; !exists {
				logger.Debug("requested call not authorized", zap.String("userName", permsForUser.userName), zap.String("callKey", callKey))
				invalidQueryRequestReceived.WithLabelValues("call_not_authorized").Inc()
				return http.StatusForbidden, fmt.Errorf(`call "%s" not authorized`, callKey)
			}

			totalRequestedCallsByChain.WithLabelValues(chainId.String()).Inc()
		}
	}

	return http.StatusOK, nil
}
