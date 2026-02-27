package ccq

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	eth_common "github.com/ethereum/go-ethereum/common"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
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
