package ccq

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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
)

func FetchCurrentGuardianSet(rpcUrl, coreAddr string) (*common.GuardianSet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	ethContract := eth_common.HexToAddress(coreAddr)
	rawClient, err := ethRpc.DialContext(ctx, rpcUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ethereum")
	}
	client := ethClient.NewClient(rawClient)
	caller, err := ethAbi.NewAbiCaller(ethContract, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create caller")
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

type Config struct {
	Permissions []User `json:"Permissions"`
}

type User struct {
	UserName      string        `json:"userName"`
	ApiKey        string        `json:"apiKey"`
	AllowUnsigned bool          `json:"allowUnsigned"`
	AllowedCalls  []AllowedCall `json:"allowedCalls"`
}

type AllowedCall struct {
	EthCall             *EthCall             `json:"ethCall"`
	EthCallByTimestamp  *EthCallByTimestamp  `json:"ethCallByTimestamp"`
	EthCallWithFinality *EthCallWithFinality `json:"ethCallWithFinality"`
}

type EthCall struct {
	Chain           int    `json:"chain"`
	ContractAddress string `json:"contractAddress"`
	Call            string `json:"call"`
}

type EthCallByTimestamp struct {
	Chain           int    `json:"chain"`
	ContractAddress string `json:"contractAddress"`
	Call            string `json:"call"`
}

type EthCallWithFinality struct {
	Chain           int    `json:"chain"`
	ContractAddress string `json:"contractAddress"`
	Call            string `json:"call"`
}

type Permissions map[string]*permissionEntry

type permissionEntry struct {
	userName      string
	apiKey        string
	allowUnsigned bool
	allowedCalls  allowedCallsForUser // Key is something like "ethCall:2:000000000000000000000000b4fbf271143f4fbf7b91a5ded31805e42b2208d6:06fdde03"
}

type allowedCallsForUser map[string]struct{}

const ETH_CALL_SIG_LENGTH = 4

// parseConfigFile parses the permissions config file into a map keyed by API key.
func parseConfigFile(fileName string) (Permissions, error) {
	jsonFile, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf(`failed to open permissions file "%s": %w`, fileName, err)
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, fmt.Errorf(`failed to read permissions file "%s": %w`, fileName, err)
	}

	retVal, err := parseConfig(byteValue)
	if err != nil {
		return retVal, fmt.Errorf(`failed to parse permissions file "%s": %w`, fileName, err)
	}

	return retVal, err
}

// parseConfig parses the permissions config from a buffer into a map keyed by API key.
func parseConfig(byteValue []byte) (Permissions, error) {
	var config Config
	if err := json.Unmarshal(byteValue, &config); err != nil {
		return nil, fmt.Errorf(`failed to unmarshal json: %w`, err)
	}

	ret := make(Permissions)
	userNames := map[string]struct{}{}
	for _, user := range config.Permissions {
		// Since we log user names in all our error messages, make sure they are unique.
		if _, exists := userNames[user.UserName]; exists {
			return nil, fmt.Errorf(`UserName "%s" is a duplicate`, user.UserName)
		}
		userNames[user.UserName] = struct{}{}

		apiKey := strings.ToLower(user.ApiKey)
		if _, exists := ret[apiKey]; exists {
			return nil, fmt.Errorf(`API key "%s" is a duplicate`, apiKey)
		}

		// Build the list of allowed calls for this API key.
		allowedCalls := make(allowedCallsForUser)
		for _, ac := range user.AllowedCalls {
			var chain int
			var callType, contractAddressStr, callStr string
			// var contractAddressStr string
			if ac.EthCall != nil {
				callType = "ethCall"
				chain = ac.EthCall.Chain
				contractAddressStr = ac.EthCall.ContractAddress
				callStr = ac.EthCall.Call
			} else if ac.EthCallByTimestamp != nil {
				callType = "ethCallByTimestamp"
				chain = ac.EthCallByTimestamp.Chain
				contractAddressStr = ac.EthCallByTimestamp.ContractAddress
				callStr = ac.EthCallByTimestamp.Call
			} else if ac.EthCallWithFinality != nil {
				callType = "ethCallWithFinality"
				chain = ac.EthCallWithFinality.Chain
				contractAddressStr = ac.EthCallWithFinality.ContractAddress
				callStr = ac.EthCallWithFinality.Call
			} else {
				return nil, fmt.Errorf(`unsupported call type for user "%s", must be "ethCall", "ethCallByTimestamp" or "ethCallWithFinality"`, user.UserName)
			}

			// Convert the contract address into a standard format like "000000000000000000000000b4fbf271143f4fbf7b91a5ded31805e42b2208d6".
			contractAddress, err := vaa.StringToAddress(contractAddressStr)
			if err != nil {
				return nil, fmt.Errorf(`invalid contract address "%s" for user "%s"`, contractAddressStr, user.UserName)
			}

			// The call should be the ABI four byte hex hash of the function signature. Parse it into a standard form of "06fdde03".
			call, err := hex.DecodeString(strings.TrimPrefix(callStr, "0x"))
			if err != nil {
				return nil, fmt.Errorf(`invalid eth call "%s" for user "%s"`, callStr, user.UserName)
			}
			if len(call) != ETH_CALL_SIG_LENGTH {
				return nil, fmt.Errorf(`eth call "%s" for user "%s" has an invalid length, must be %d bytes`, callStr, user.UserName, ETH_CALL_SIG_LENGTH)
			}

			// The permission key is the chain, contract address and call formatted as a colon separated string.
			callKey := fmt.Sprintf("%s:%d:%s:%s", callType, chain, contractAddress, hex.EncodeToString(call))

			if _, exists := allowedCalls[callKey]; exists {
				return nil, fmt.Errorf(`"%s" is a duplicate allowed call for user "%s"`, callKey, user.UserName)
			}

			allowedCalls[callKey] = struct{}{}
		}

		pe := &permissionEntry{
			userName:      user.UserName,
			apiKey:        apiKey,
			allowUnsigned: user.AllowUnsigned,
			allowedCalls:  allowedCalls,
		}

		ret[apiKey] = pe
	}

	return ret, nil
}

// validateRequest verifies that this API key is allowed to do all of the calls in this request. In the case of an error, it returns the HTTP status.
func validateRequest(logger *zap.Logger, env common.Environment, perms Permissions, signerKey *ecdsa.PrivateKey, apiKey string, qr *gossipv1.SignedQueryRequest) (int, error) {
	permsForUser, exists := perms[apiKey]
	if !exists {
		logger.Debug("invalid api key", zap.String("apiKey", apiKey))
		invalidQueryRequestReceived.WithLabelValues("invalid_api_key").Inc()
		return http.StatusForbidden, fmt.Errorf("invalid api key")
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
			return http.StatusBadRequest, fmt.Errorf("request not signed")
		}

		// Sign the request using our key.
		var err error
		digest := query.QueryRequestDigest(env, qr.QueryRequest)
		qr.Signature, err = ethCrypto.Sign(digest.Bytes(), signerKey)
		if err != nil {
			logger.Debug("failed to sign request", zap.String("userName", permsForUser.userName), zap.Error(err))
			invalidQueryRequestReceived.WithLabelValues("failed_to_sign_request").Inc()
			return http.StatusInternalServerError, fmt.Errorf("failed to sign request: %w", err)
		}
	}

	var queryRequest query.QueryRequest
	err := queryRequest.Unmarshal(qr.QueryRequest)
	if err != nil {
		logger.Debug("failed to unmarshal request", zap.String("userName", permsForUser.userName), zap.Error(err))
		invalidQueryRequestReceived.WithLabelValues("failed_to_unmarshal_request").Inc()
		return http.StatusInternalServerError, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	// Make sure the overall query request is sane.
	if err := queryRequest.Validate(); err != nil {
		logger.Debug("failed to validate request", zap.String("userName", permsForUser.userName), zap.Error(err))
		invalidQueryRequestReceived.WithLabelValues("failed_to_validate_request").Inc()
		return http.StatusBadRequest, fmt.Errorf("failed to validate request: %w", err)
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
		default:
			logger.Debug("unsupported query type", zap.String("userName", permsForUser.userName), zap.Any("type", pcq.Query))
			invalidQueryRequestReceived.WithLabelValues("unsupported_query_type").Inc()
			return http.StatusBadRequest, fmt.Errorf("unsupported query type")
		}

		if err != nil {
			// Metric is pegged below.
			return status, err
		}
	}

	logger.Debug("submitting query request", zap.String("userName", permsForUser.userName))
	return http.StatusOK, nil
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
			return http.StatusBadRequest, fmt.Errorf("eth call data must be at least four bytes")
		}
		call := hex.EncodeToString(cd.Data[0:ETH_CALL_SIG_LENGTH])
		callKey := fmt.Sprintf("%s:%d:%s:%s", callTag, chainId, contractAddress, call)
		if _, exists := permsForUser.allowedCalls[callKey]; !exists {
			logger.Debug("requested call not authorized", zap.String("userName", permsForUser.userName), zap.String("callKey", callKey))
			invalidQueryRequestReceived.WithLabelValues("call_not_authorized").Inc()
			return http.StatusBadRequest, fmt.Errorf(`call "%s" not authorized`, callKey)
		}

		totalRequestedCallsByChain.WithLabelValues(chainId.String()).Inc()
	}

	return http.StatusOK, nil
}
