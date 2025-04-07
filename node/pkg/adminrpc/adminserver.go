package adminrpc

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"
	"github.com/holiman/uint256"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/exp/slices"

	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/mr-tron/base58"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/certusone/wormhole/node/pkg/common"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
)

const maxResetReleaseTimerDays = 7
const ecdsaSignatureLength = 65

var (
	vaaInjectionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_vaa_injections_total",
			Help: "Total number of injected VAA queued for broadcast",
		})
)

type nodePrivilegedService struct {
	nodev1.UnimplementedNodePrivilegedServiceServer
	db              *db.Database
	injectC         chan<- *common.MessagePublication
	obsvReqSendC    chan<- *gossipv1.ObservationRequest
	logger          *zap.Logger
	signedInC       chan<- *gossipv1.SignedVAAWithQuorum
	governor        *governor.ChainGovernor
	evmConnector    connectors.Connector
	gsCache         sync.Map
	guardianSigner  guardiansigner.GuardianSigner
	guardianAddress ethcommon.Address
	rpcMap          map[string]string
	reobservers     interfaces.Reobservers
}

func NewPrivService(
	db *db.Database,
	injectC chan<- *common.MessagePublication,
	obsvReqSendC chan<- *gossipv1.ObservationRequest,
	logger *zap.Logger,
	signedInC chan<- *gossipv1.SignedVAAWithQuorum,
	governor *governor.ChainGovernor,
	evmConnector connectors.Connector,
	guardianSigner guardiansigner.GuardianSigner,
	guardianAddress ethcommon.Address,
	rpcMap map[string]string,
	reobservers interfaces.Reobservers,

) *nodePrivilegedService {
	return &nodePrivilegedService{
		db:              db,
		injectC:         injectC,
		obsvReqSendC:    obsvReqSendC,
		logger:          logger,
		signedInC:       signedInC,
		governor:        governor,
		evmConnector:    evmConnector,
		guardianSigner:  guardianSigner,
		guardianAddress: guardianAddress,
		rpcMap:          rpcMap,
		reobservers:     reobservers,
	}
}

// adminGuardianSetUpdateToVAA converts a nodev1.GuardianSetUpdate message to its canonical VAA representation.
// Returns an error if the data is invalid.
func adminGuardianSetUpdateToVAA(req *nodev1.GuardianSetUpdate, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	if len(req.Guardians) == 0 {
		return nil, errors.New("empty guardian set specified")
	}

	if len(req.Guardians) > common.MaxGuardianCount {
		return nil, fmt.Errorf("too many guardians - %d, maximum is %d", len(req.Guardians), common.MaxGuardianCount)
	}

	addrs := make([]ethcommon.Address, len(req.Guardians))
	for i, g := range req.Guardians {
		if !ethcommon.IsHexAddress(g.Pubkey) {
			return nil, fmt.Errorf("invalid pubkey format at index %d (%s)", i, g.Name)
		}

		ethAddr := ethcommon.HexToAddress(g.Pubkey)
		for j, pk := range addrs {
			if pk == ethAddr {
				return nil, fmt.Errorf("duplicate pubkey at index %d (duplicate of %d): %s", i, j, g.Name)
			}
		}

		addrs[i] = ethAddr
	}

	body, err := vaa.BodyGuardianSetUpdate{
		Keys:     addrs,
		NewIndex: guardianSetIndex + 1,
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

// adminContractUpgradeToVAA converts a nodev1.ContractUpgrade message to its canonical VAA representation.
// Returns an error if the data is invalid.
func adminContractUpgradeToVAA(req *nodev1.ContractUpgrade, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	b, err := hex.DecodeString(req.NewContract)
	if err != nil {
		return nil, errors.New("invalid new contract address encoding (expected hex)")
	}

	if len(b) != 32 {
		return nil, errors.New("invalid new_contract address")
	}

	if req.ChainId > math.MaxUint16 {
		return nil, errors.New("invalid chain_id")
	}

	newContractAddress := vaa.Address{}
	copy(newContractAddress[:], b)

	body, err := vaa.BodyContractUpgrade{
		ChainID:     vaa.ChainID(req.ChainId),
		NewContract: newContractAddress,
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

// tokenBridgeRegisterChain converts a nodev1.TokenBridgeRegisterChain message to its canonical VAA representation.
// Returns an error if the data is invalid.
func tokenBridgeRegisterChain(req *nodev1.BridgeRegisterChain, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	if req.ChainId > math.MaxUint16 {
		return nil, errors.New("invalid chain_id")
	}

	b, err := hex.DecodeString(req.EmitterAddress)
	if err != nil {
		return nil, errors.New("invalid emitter address encoding (expected hex)")
	}

	if len(b) != 32 {
		return nil, errors.New("invalid emitter address (expected 32 bytes)")
	}

	emitterAddress := vaa.Address{}
	copy(emitterAddress[:], b)

	body, err := vaa.BodyTokenBridgeRegisterChain{
		Module:         req.Module,
		ChainID:        vaa.ChainID(req.ChainId),
		EmitterAddress: emitterAddress,
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

// recoverChainId converts a nodev1.RecoverChainId message to its canonical VAA representation.
// Returns an error if the data is invalid.
func recoverChainId(req *nodev1.RecoverChainId, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	evm_chain_id_big := big.NewInt(0)
	evm_chain_id_big, ok := evm_chain_id_big.SetString(req.EvmChainId, 10)
	if !ok {
		return nil, errors.New("invalid evm_chain_id")
	}

	// uint256 has Bytes32 method for easier serialization
	evm_chain_id, overflow := uint256.FromBig(evm_chain_id_big)
	if overflow {
		return nil, errors.New("evm_chain_id overflow")
	}

	if req.NewChainId > math.MaxUint16 {
		return nil, errors.New("invalid new_chain_id")
	}

	body, err := vaa.BodyRecoverChainId{
		Module:     req.Module,
		EvmChainID: evm_chain_id,
		NewChainID: vaa.ChainID(req.NewChainId),
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

// accountantModifyBalance converts a nodev1.AccountantModifyBalance message to its canonical VAA representation.
// Returns an error if the data is invalid.
func accountantModifyBalance(req *nodev1.AccountantModifyBalance, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	if req.TargetChainId > math.MaxUint16 {
		return nil, errors.New("invalid target_chain_id")
	}
	if req.ChainId > math.MaxUint16 {
		return nil, errors.New("invalid chain_id")
	}
	if req.TokenChain > math.MaxUint16 {
		return nil, errors.New("invalid token_chain")
	}

	b, err := hex.DecodeString(req.TokenAddress)
	if err != nil {
		return nil, errors.New("invalid token address (expected hex)")
	}

	if len(b) != 32 {
		return nil, errors.New("invalid new token address (expected 32 bytes)")
	}

	if len(req.Reason) > vaa.AccountantModifyBalanceReasonLength {
		return nil, fmt.Errorf("the reason should not be larger than %d bytes", vaa.AccountantModifyBalanceReasonLength)
	}

	amount_big := big.NewInt(0)
	amount_big, ok := amount_big.SetString(req.Amount, 10)
	if !ok {
		return nil, errors.New("invalid amount")
	}

	// uint256 has Bytes32 method for easier serialization
	amount, overflow := uint256.FromBig(amount_big)
	if overflow {
		return nil, errors.New("amount overflow")
	}

	tokenAdress := vaa.Address{}
	copy(tokenAdress[:], b)

	body, err := vaa.BodyAccountantModifyBalance{
		Module:        req.Module,
		TargetChainID: vaa.ChainID(req.TargetChainId),

		Sequence:     req.Sequence,
		ChainId:      vaa.ChainID(req.ChainId),
		TokenChain:   vaa.ChainID(req.TokenChain),
		TokenAddress: tokenAdress,
		Kind:         uint8(req.Kind), // #nosec G115 -- The `ModificationKind` enum only has 3 values
		Amount:       amount,
		Reason:       req.Reason,
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

// tokenBridgeUpgradeContract converts a nodev1.TokenBridgeRegisterChain message to its canonical VAA representation.
// Returns an error if the data is invalid.
func tokenBridgeUpgradeContract(req *nodev1.BridgeUpgradeContract, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	if req.TargetChainId > math.MaxUint16 {
		return nil, errors.New("invalid target_chain_id")
	}

	b, err := hex.DecodeString(req.NewContract)
	if err != nil {
		return nil, errors.New("invalid new contract address (expected hex)")
	}

	if len(b) != 32 {
		return nil, errors.New("invalid new contract address (expected 32 bytes)")
	}

	newContract := vaa.Address{}
	copy(newContract[:], b)

	body, err := vaa.BodyTokenBridgeUpgradeContract{
		Module:        req.Module,
		TargetChainID: vaa.ChainID(req.TargetChainId),
		NewContract:   newContract,
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

// wormchainStoreCode converts a nodev1.WormchainStoreCode to its canonical VAA representation
// Returns an error if the data is invalid
func wormchainStoreCode(req *nodev1.WormchainStoreCode, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	// validate the length of the hex passed in
	b, err := hex.DecodeString(req.WasmHash)
	if err != nil {
		return nil, fmt.Errorf("invalid cosmwasm bytecode hash (expected hex): %w", err)
	}

	if len(b) != 32 {
		return nil, fmt.Errorf("invalid cosmwasm bytecode hash (expected 32 bytes but received %d bytes)", len(b))
	}

	wasmHash := [32]byte{}
	copy(wasmHash[:], b)

	body, err := vaa.BodyWormchainStoreCode{
		WasmHash: wasmHash,
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

// wormchainInstantiateContract converts a nodev1.WormchainInstantiateContract to its canonical VAA representation
// Returns an error if the data is invalid
func wormchainInstantiateContract(req *nodev1.WormchainInstantiateContract, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) { //nolint:unparam // error is always nil but kept to mirror function signature of other functions
	instantiationParams_hash := vaa.CreateInstatiateCosmwasmContractHash(req.CodeId, req.Label, []byte(req.InstantiationMsg))

	body, err := vaa.BodyWormchainInstantiateContract{
		InstantiationParamsHash: instantiationParams_hash,
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

// wormchainMigrateContract converts a nodev1.WormchainMigrateContract to its canonical VAA representation
func wormchainMigrateContract(req *nodev1.WormchainMigrateContract, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) { //nolint:unparam // error is always nil but kept to mirror function signature of other functions
	instantiationParams_hash := vaa.CreateMigrateCosmwasmContractHash(req.CodeId, req.Contract, []byte(req.InstantiationMsg))

	body, err := vaa.BodyWormchainMigrateContract{
		MigrationParamsHash: instantiationParams_hash,
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

func wormchainWasmInstantiateAllowlist(
	req *nodev1.WormchainWasmInstantiateAllowlist,
	timestamp time.Time,
	guardianSetIndex uint32,
	nonce uint32,
	sequence uint64,
) (*vaa.VAA, error) { //nolint:unparam // error is always nil but kept to mirror function signature of other functions
	decodedAddr, err := sdktypes.GetFromBech32(req.Contract, "wormhole")
	if err != nil {
		return nil, err
	}

	var action vaa.GovernanceAction
	if req.Action == nodev1.WormchainWasmInstantiateAllowlistAction_WORMCHAIN_WASM_INSTANTIATE_ALLOWLIST_ACTION_ADD {
		action = vaa.ActionAddWasmInstantiateAllowlist
	} else if req.Action == nodev1.WormchainWasmInstantiateAllowlistAction_WORMCHAIN_WASM_INSTANTIATE_ALLOWLIST_ACTION_DELETE {
		action = vaa.ActionDeleteWasmInstantiateAllowlist
	} else {
		return nil, fmt.Errorf("unrecognized wasm instantiate allowlist action")
	}

	var decodedAddr32 [32]byte
	copy(decodedAddr32[:], decodedAddr)

	body, err := vaa.BodyWormchainWasmAllowlistInstantiate{
		ContractAddr: decodedAddr32,
		CodeId:       req.CodeId,
	}.Serialize(action)

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

func gatewayScheduleUpgrade(
	req *nodev1.GatewayScheduleUpgrade,
	timestamp time.Time,
	guardianSetIndex uint32,
	nonce uint32,
	sequence uint64,
) (*vaa.VAA, error) { //nolint:unparam // error is always nil but kept to mirror function signature of other functions

	body, err := vaa.BodyGatewayScheduleUpgrade{
		Name:   req.Name,
		Height: req.Height,
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

func gatewayCancelUpgrade(
	timestamp time.Time,
	guardianSetIndex uint32,
	nonce uint32,
	sequence uint64,
) (*vaa.VAA, error) { //nolint:unparam // error is always nil but kept to mirror function signature of other functions

	body, err := vaa.EmptyPayloadVaa(vaa.GatewayModuleStr, vaa.ActionCancelUpgrade, vaa.ChainIDWormchain)

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

func gatewayIbcComposabilityMwSetContract(
	req *nodev1.GatewayIbcComposabilityMwSetContract,
	timestamp time.Time,
	guardianSetIndex uint32,
	nonce uint32,
	sequence uint64,
) (*vaa.VAA, error) {
	decodedAddr, err := sdktypes.GetFromBech32(req.Contract, "wormhole")
	if err != nil {
		return nil, err
	}

	var decodedAddr32 [32]byte
	copy(decodedAddr32[:], decodedAddr)

	body, err := vaa.BodyGatewayIbcComposabilityMwContract{
		ContractAddr: decodedAddr32,
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

// circleIntegrationUpdateWormholeFinality converts a nodev1.CircleIntegrationUpdateWormholeFinality to its canonical VAA representation
// Returns an error if the data is invalid
func circleIntegrationUpdateWormholeFinality(req *nodev1.CircleIntegrationUpdateWormholeFinality, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	if req.TargetChainId > math.MaxUint16 {
		return nil, fmt.Errorf("invalid target chain id, must be <= %d", math.MaxUint16)
	}
	if req.Finality > math.MaxUint8 {
		return nil, fmt.Errorf("invalid finality, must be <= %d", math.MaxUint8)
	}

	body, err := vaa.BodyCircleIntegrationUpdateWormholeFinality{
		TargetChainID: vaa.ChainID(req.TargetChainId),
		Finality:      uint8(req.Finality),
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

// circleIntegrationRegisterEmitterAndDomain converts a nodev1.CircleIntegrationRegisterEmitterAndDomain to its canonical VAA representation
// Returns an error if the data is invalid
func circleIntegrationRegisterEmitterAndDomain(req *nodev1.CircleIntegrationRegisterEmitterAndDomain, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	if req.TargetChainId > math.MaxUint16 {
		return nil, fmt.Errorf("invalid target chain id, must be <= %d", math.MaxUint16)
	}
	if req.ForeignEmitterChainId > math.MaxUint16 {
		return nil, fmt.Errorf("invalid foreign emitter chain id, must be <= %d", math.MaxUint16)
	}
	b, err := hex.DecodeString(req.ForeignEmitterAddress)
	if err != nil {
		return nil, errors.New("invalid foreign emitter address encoding (expected hex)")
	}

	if len(b) != 32 {
		return nil, errors.New("invalid foreign emitter address (expected 32 bytes)")
	}

	foreignEmitterAddress := vaa.Address{}
	copy(foreignEmitterAddress[:], b)

	body, err := vaa.BodyCircleIntegrationRegisterEmitterAndDomain{
		TargetChainID:         vaa.ChainID(req.TargetChainId),
		ForeignEmitterChainId: vaa.ChainID(req.ForeignEmitterChainId),
		ForeignEmitterAddress: foreignEmitterAddress,
		CircleDomain:          req.CircleDomain,
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

// circleIntegrationUpgradeContractImplementation converts a nodev1.CircleIntegrationUpgradeContractImplementation to its canonical VAA representation
// Returns an error if the data is invalid
func circleIntegrationUpgradeContractImplementation(req *nodev1.CircleIntegrationUpgradeContractImplementation, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	if req.TargetChainId > math.MaxUint16 {
		return nil, fmt.Errorf("invalid target chain id, must be <= %d", math.MaxUint16)
	}
	b, err := hex.DecodeString(req.NewImplementationAddress)
	if err != nil {
		return nil, errors.New("invalid new implementation address encoding (expected hex)")
	}

	if len(b) != 32 {
		return nil, errors.New("invalid new implementation address (expected 32 bytes)")
	}

	newImplementationAddress := vaa.Address{}
	copy(newImplementationAddress[:], b)

	body, err := vaa.BodyCircleIntegrationUpgradeContractImplementation{
		TargetChainID:            vaa.ChainID(req.TargetChainId),
		NewImplementationAddress: newImplementationAddress,
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

func ibcUpdateChannelChain(
	req *nodev1.IbcUpdateChannelChain,
	timestamp time.Time,
	guardianSetIndex uint32,
	nonce uint32,
	sequence uint64,
) (*vaa.VAA, error) {
	// validate parameters
	if req.TargetChainId > math.MaxUint16 {
		return nil, fmt.Errorf("invalid target chain id, must be <= %d", math.MaxUint16)
	}

	if req.ChainId > math.MaxUint16 {
		return nil, fmt.Errorf("invalid chain id, must be <= %d", math.MaxUint16)
	}

	if len(req.ChannelId) > 64 {
		return nil, fmt.Errorf("invalid channel ID length, must be <= 64")
	}
	channelId, err := vaa.LeftPadIbcChannelId(req.ChannelId)
	if err != nil {
		return nil, fmt.Errorf("failed to left pad channel id: %w", err)
	}

	var module string
	if req.Module == nodev1.IbcUpdateChannelChainModule_IBC_UPDATE_CHANNEL_CHAIN_MODULE_RECEIVER {
		module = vaa.IbcReceiverModuleStr
	} else if req.Module == nodev1.IbcUpdateChannelChainModule_IBC_UPDATE_CHANNEL_CHAIN_MODULE_TRANSLATOR {
		module = vaa.IbcTranslatorModuleStr
	} else {
		return nil, fmt.Errorf("unrecognized ibc update channel chain module")
	}

	body, err := vaa.BodyIbcUpdateChannelChain{
		TargetChainId: vaa.ChainID(req.TargetChainId),
		ChannelId:     channelId,
		ChainId:       vaa.ChainID(req.ChainId),
	}.Serialize(module)

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

// wormholeRelayerSetDefaultDeliveryProvider converts a nodev1.WormholeRelayerSetDefaultDeliveryProvider message to its canonical VAA representation.
// Returns an error if the data is invalid.
func wormholeRelayerSetDefaultDeliveryProvider(req *nodev1.WormholeRelayerSetDefaultDeliveryProvider, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	if req.ChainId > math.MaxUint16 {
		return nil, errors.New("invalid target_chain_id")
	}

	b, err := hex.DecodeString(req.NewDefaultDeliveryProviderAddress)
	if err != nil {
		return nil, errors.New("invalid new default delivery provider address (expected hex)")
	}

	if len(b) != 32 {
		return nil, errors.New("invalid new default delivery provider address (expected 32 bytes)")
	}

	NewDefaultDeliveryProviderAddress := vaa.Address{}
	copy(NewDefaultDeliveryProviderAddress[:], b)

	body, err := vaa.BodyWormholeRelayerSetDefaultDeliveryProvider{
		ChainID:                           vaa.ChainID(req.ChainId),
		NewDefaultDeliveryProviderAddress: NewDefaultDeliveryProviderAddress,
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

func evmCallToVaa(evmCall *nodev1.EvmCall, timestamp time.Time, guardianSetIndex, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	governanceContract := ethcommon.HexToAddress(evmCall.GovernanceContract)
	targetContract := ethcommon.HexToAddress(evmCall.TargetContract)

	payload, err := hex.DecodeString(evmCall.AbiEncodedCall)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ABI encoded call: %w", err)
	}
	if evmCall.ChainId > math.MaxUint16 {
		return nil, fmt.Errorf("chain id exceeds max uint16: %v", evmCall.ChainId)
	}

	body, err := vaa.BodyGeneralPurposeGovernanceEvm{
		ChainID:            vaa.ChainID(evmCall.ChainId),
		GovernanceContract: governanceContract,
		TargetContract:     targetContract,
		Payload:            payload,
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

func solanaCallToVaa(solanaCall *nodev1.SolanaCall, timestamp time.Time, guardianSetIndex, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	address, err := base58.Decode(solanaCall.GovernanceContract)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base58 governance contract address: %w", err)
	}
	if len(address) != 32 {
		return nil, errors.New("invalid governance contract address length (expected 32 bytes)")
	}

	var governanceContract [32]byte
	copy(governanceContract[:], address)

	instruction, err := hex.DecodeString(solanaCall.EncodedInstruction)
	if err != nil {
		return nil, fmt.Errorf("failed to decode instruction: %w", err)
	}
	if solanaCall.ChainId > math.MaxUint16 {
		return nil, fmt.Errorf("chain id exceeds max uint16: %v", solanaCall.ChainId)
	}

	body, err := vaa.BodyGeneralPurposeGovernanceSolana{
		ChainID:            vaa.ChainID(solanaCall.ChainId),
		GovernanceContract: governanceContract,
		Instruction:        instruction,
	}.Serialize()

	if err != nil {
		return nil, fmt.Errorf("failed to serialize governance body: %w", err)
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex, body)
	return v, nil
}

func GovMsgToVaa(message *nodev1.GovernanceMessage, currentSetIndex uint32, timestamp time.Time) (*vaa.VAA, error) {
	var (
		v   *vaa.VAA
		err error
	)

	switch payload := message.Payload.(type) {
	case *nodev1.GovernanceMessage_GuardianSet:
		v, err = adminGuardianSetUpdateToVAA(payload.GuardianSet, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_ContractUpgrade:
		v, err = adminContractUpgradeToVAA(payload.ContractUpgrade, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_BridgeRegisterChain:
		v, err = tokenBridgeRegisterChain(payload.BridgeRegisterChain, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_BridgeContractUpgrade:
		v, err = tokenBridgeUpgradeContract(payload.BridgeContractUpgrade, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_RecoverChainId:
		v, err = recoverChainId(payload.RecoverChainId, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_AccountantModifyBalance:
		v, err = accountantModifyBalance(payload.AccountantModifyBalance, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_WormchainStoreCode:
		v, err = wormchainStoreCode(payload.WormchainStoreCode, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_WormchainInstantiateContract:
		v, err = wormchainInstantiateContract(payload.WormchainInstantiateContract, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_WormchainMigrateContract:
		v, err = wormchainMigrateContract(payload.WormchainMigrateContract, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_WormchainWasmInstantiateAllowlist:
		v, err = wormchainWasmInstantiateAllowlist(payload.WormchainWasmInstantiateAllowlist, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_GatewayScheduleUpgrade:
		v, err = gatewayScheduleUpgrade(payload.GatewayScheduleUpgrade, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_GatewayCancelUpgrade:
		v, err = gatewayCancelUpgrade(timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_GatewayIbcComposabilityMwSetContract:
		v, err = gatewayIbcComposabilityMwSetContract(payload.GatewayIbcComposabilityMwSetContract, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_CircleIntegrationUpdateWormholeFinality:
		v, err = circleIntegrationUpdateWormholeFinality(payload.CircleIntegrationUpdateWormholeFinality, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_CircleIntegrationRegisterEmitterAndDomain:
		v, err = circleIntegrationRegisterEmitterAndDomain(payload.CircleIntegrationRegisterEmitterAndDomain, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_CircleIntegrationUpgradeContractImplementation:
		v, err = circleIntegrationUpgradeContractImplementation(payload.CircleIntegrationUpgradeContractImplementation, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_IbcUpdateChannelChain:
		v, err = ibcUpdateChannelChain(payload.IbcUpdateChannelChain, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_WormholeRelayerSetDefaultDeliveryProvider:
		v, err = wormholeRelayerSetDefaultDeliveryProvider(payload.WormholeRelayerSetDefaultDeliveryProvider, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_EvmCall:
		v, err = evmCallToVaa(payload.EvmCall, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_SolanaCall:
		v, err = solanaCallToVaa(payload.SolanaCall, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	default:
		err = fmt.Errorf("unsupported VAA type: %T", payload)
	}

	return v, err
}

func (s *nodePrivilegedService) InjectGovernanceVAA(ctx context.Context, req *nodev1.InjectGovernanceVAARequest) (*nodev1.InjectGovernanceVAAResponse, error) {
	s.logger.Info("governance VAA injected via admin socket", zap.String("request", req.String()))

	var (
		v   *vaa.VAA
		err error
	)

	timestamp := time.Unix(int64(req.Timestamp), 0)

	digests := make([][]byte, len(req.Messages))

	for i, message := range req.Messages {
		v, err = GovMsgToVaa(message, req.CurrentSetIndex, timestamp)

		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		// Generate digest of the unsigned VAA.
		digest := v.SigningDigest()

		s.logger.Info("governance VAA constructed",
			zap.Any("vaa", v),
			zap.String("digest", digest.String()),
		)

		vaaInjectionsTotal.Inc()

		s.injectC <- &common.MessagePublication{
			TxID:             ethcommon.Hash{}.Bytes(),
			Timestamp:        v.Timestamp,
			Nonce:            v.Nonce,
			Sequence:         v.Sequence,
			ConsistencyLevel: v.ConsistencyLevel,
			EmitterChain:     v.EmitterChain,
			EmitterAddress:   v.EmitterAddress,
			Payload:          v.Payload,
			Unreliable:       false,
		}

		digests[i] = digest.Bytes()
	}

	return &nodev1.InjectGovernanceVAAResponse{Digests: digests}, nil
}

// fetchMissing attempts to backfill a gap by fetching and storing missing signed VAAs from the network.
// Returns true if the gap was filled, false otherwise.
func (s *nodePrivilegedService) fetchMissing(
	ctx context.Context,
	nodes []string,
	c *http.Client,
	chain vaa.ChainID,
	addr string,
	seq uint64) (bool, error) {

	// shuffle the list of public RPC endpoints
	rand.Shuffle(len(nodes), func(i, j int) {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	})

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	for _, node := range nodes {
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(
			"%s/v1/signed_vaa/%d/%s/%d", node, chain, addr, seq), nil)
		if err != nil {
			return false, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.Do(req)
		if err != nil {
			s.logger.Warn("failed to fetch missing VAA",
				zap.String("node", node),
				zap.String("chain", chain.String()),
				zap.String("address", addr),
				zap.Uint64("sequence", seq),
				zap.Error(err),
			)
			continue
		}

		switch resp.StatusCode {
		case http.StatusNotFound:
			resp.Body.Close()
			continue
		case http.StatusOK:
			type getVaaResp struct {
				VaaBytes string `json:"vaaBytes"`
			}
			var respBody getVaaResp
			if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
				resp.Body.Close()
				s.logger.Warn("failed to decode VAA response",
					zap.String("node", node),
					zap.String("chain", chain.String()),
					zap.String("address", addr),
					zap.Uint64("sequence", seq),
					zap.Error(err),
				)
				continue
			}

			// base64 decode the VAA bytes
			vaaBytes, err := base64.StdEncoding.DecodeString(respBody.VaaBytes)
			if err != nil {
				resp.Body.Close()
				s.logger.Warn("failed to decode VAA body",
					zap.String("node", node),
					zap.String("chain", chain.String()),
					zap.String("address", addr),
					zap.Uint64("sequence", seq),
					zap.Error(err),
				)
				continue
			}

			s.logger.Info("backfilled VAA",
				zap.Uint16("chain", uint16(chain)),
				zap.String("address", addr),
				zap.Uint64("sequence", seq),
				zap.Int("numBytes", len(vaaBytes)),
			)

			// Inject into the gossip signed VAA receive path.
			// This has the same effect as if the VAA was received from the network
			// (verifying signature, storing in local DB...).
			s.signedInC <- &gossipv1.SignedVAAWithQuorum{
				Vaa: vaaBytes,
			}

			resp.Body.Close()
			return true, nil
		default:
			resp.Body.Close()
			return false, fmt.Errorf("unexpected response status: %d", resp.StatusCode)
		}
	}

	return false, nil
}

func (s *nodePrivilegedService) FindMissingMessages(ctx context.Context, req *nodev1.FindMissingMessagesRequest) (*nodev1.FindMissingMessagesResponse, error) {
	b, err := hex.DecodeString(req.EmitterAddress)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid emitter address encoding: %v", err)
	}
	if req.EmitterChain > math.MaxUint16 {
		return nil, status.Errorf(codes.InvalidArgument, "chain id exceeds max uint16: %v", req.EmitterChain)
	}
	emitterAddress := vaa.Address{}
	copy(emitterAddress[:], b)

	ids, first, last, err := s.db.FindEmitterSequenceGap(db.VAAID{
		EmitterChain:   vaa.ChainID(req.EmitterChain),
		EmitterAddress: emitterAddress,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database operation failed: %v", err)
	}

	if req.RpcBackfill {
		c := &http.Client{}
		unfilled := make([]uint64, 0, len(ids))
		for _, id := range ids {
			if ok, err := s.fetchMissing(ctx, req.BackfillNodes, c, vaa.ChainID(req.EmitterChain), emitterAddress.String(), id); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to backfill VAA: %v", err)
			} else if ok {
				continue
			}
			unfilled = append(unfilled, id)
		}
		ids = unfilled
	}

	resp := make([]string, len(ids))
	for i, v := range ids {
		resp[i] = fmt.Sprintf("%d/%s/%d", req.EmitterChain, emitterAddress, v)
	}
	return &nodev1.FindMissingMessagesResponse{
		MissingMessages: resp,
		FirstSequence:   first,
		LastSequence:    last,
	}, nil
}

func (s *nodePrivilegedService) SendObservationRequest(ctx context.Context, req *nodev1.SendObservationRequestRequest) (*nodev1.SendObservationRequestResponse, error) {
	if err := common.PostObservationRequest(s.obsvReqSendC, req.ObservationRequest); err != nil {
		return nil, err
	}

	s.logger.Info("sent observation request", zap.Any("request", req.ObservationRequest))
	return &nodev1.SendObservationRequestResponse{}, nil
}

func (s *nodePrivilegedService) ReobserveWithEndpoint(ctx context.Context, req *nodev1.ReobserveWithEndpointRequest) (*nodev1.ReobserveWithEndpointResponse, error) {
	if req.ChainId > math.MaxUint16 {
		return nil, status.Errorf(codes.Internal, "chain %d is not a valid uint16", req.ChainId)
	}

	watcher := s.reobservers[vaa.ChainID(req.ChainId)]
	if watcher == nil {
		return nil, status.Errorf(codes.Internal, "chain %d does not support reobservation by endpoint", req.ChainId)
	}

	numObservations, err := watcher.Reobserve(ctx, vaa.ChainID(req.ChainId), req.TxHash, req.Url)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "reobservation failed: %v", err)
	}

	return &nodev1.ReobserveWithEndpointResponse{NumObservations: numObservations}, nil
}

func (s *nodePrivilegedService) ChainGovernorStatus(ctx context.Context, req *nodev1.ChainGovernorStatusRequest) (*nodev1.ChainGovernorStatusResponse, error) {
	if s.governor == nil {
		return nil, fmt.Errorf("chain governor is not enabled")
	}

	return &nodev1.ChainGovernorStatusResponse{
		Response: s.governor.Status(),
	}, nil
}

func (s *nodePrivilegedService) ChainGovernorReload(ctx context.Context, req *nodev1.ChainGovernorReloadRequest) (*nodev1.ChainGovernorReloadResponse, error) {
	if s.governor == nil {
		return nil, fmt.Errorf("chain governor is not enabled")
	}

	resp, err := s.governor.Reload()
	if err != nil {
		return nil, err
	}

	return &nodev1.ChainGovernorReloadResponse{
		Response: resp,
	}, nil
}

func (s *nodePrivilegedService) ChainGovernorDropPendingVAA(ctx context.Context, req *nodev1.ChainGovernorDropPendingVAARequest) (*nodev1.ChainGovernorDropPendingVAAResponse, error) {
	if s.governor == nil {
		return nil, fmt.Errorf("chain governor is not enabled")
	}

	if len(req.VaaId) == 0 {
		return nil, fmt.Errorf("the VAA id must be specified as \"chainId/emitterAddress/seqNum\"")
	}

	resp, err := s.governor.DropPendingVAA(req.VaaId)
	if err != nil {
		return nil, err
	}

	return &nodev1.ChainGovernorDropPendingVAAResponse{
		Response: resp,
	}, nil
}

func (s *nodePrivilegedService) ChainGovernorReleasePendingVAA(ctx context.Context, req *nodev1.ChainGovernorReleasePendingVAARequest) (*nodev1.ChainGovernorReleasePendingVAAResponse, error) {
	if s.governor == nil {
		return nil, fmt.Errorf("chain governor is not enabled")
	}

	if len(req.VaaId) == 0 {
		return nil, fmt.Errorf("the VAA id must be specified as \"chainId/emitterAddress/seqNum\"")
	}

	resp, err := s.governor.ReleasePendingVAA(req.VaaId)
	if err != nil {
		return nil, err
	}

	return &nodev1.ChainGovernorReleasePendingVAAResponse{
		Response: resp,
	}, nil
}

func (s *nodePrivilegedService) ChainGovernorResetReleaseTimer(ctx context.Context, req *nodev1.ChainGovernorResetReleaseTimerRequest) (*nodev1.ChainGovernorResetReleaseTimerResponse, error) {
	if s.governor == nil {
		return nil, fmt.Errorf("chain governor is not enabled")
	}

	if len(req.VaaId) == 0 {
		return nil, fmt.Errorf("the VAA id must be specified as \"chainId/emitterAddress/seqNum\"")
	}

	if req.NumDays < 1 || req.NumDays > maxResetReleaseTimerDays {
		return nil, fmt.Errorf("the specified number of days falls outside the range of 1 to %d", maxResetReleaseTimerDays)
	}

	resp, err := s.governor.ResetReleaseTimer(req.VaaId, req.NumDays)
	if err != nil {
		return nil, err
	}

	return &nodev1.ChainGovernorResetReleaseTimerResponse{
		Response: resp,
	}, nil
}

func (s *nodePrivilegedService) PurgePythNetVaas(ctx context.Context, req *nodev1.PurgePythNetVaasRequest) (*nodev1.PurgePythNetVaasResponse, error) {
	prefix := db.VAAID{EmitterChain: vaa.ChainIDPythNet}
	oldestTime := time.Now().Add(-time.Hour * 24 * time.Duration(req.DaysOld)) // #nosec G115 -- This conversion is safe indefinitely
	resp, err := s.db.PurgeVaas(prefix, oldestTime, req.LogOnly)
	if err != nil {
		return nil, err
	}

	return &nodev1.PurgePythNetVaasResponse{
		Response: resp,
	}, nil
}

func (s *nodePrivilegedService) SignExistingVAA(ctx context.Context, req *nodev1.SignExistingVAARequest) (*nodev1.SignExistingVAAResponse, error) {
	v, err := vaa.Unmarshal(req.Vaa)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal VAA: %w", err)
	}

	if req.NewGuardianSetIndex <= v.GuardianSetIndex {
		return nil, errors.New("new guardian set index must be higher than provided VAA")
	}

	if s.evmConnector == nil {
		return nil, errors.New("the node needs to have an Ethereum connection configured to sign existing VAAs")
	}

	var gs *common.GuardianSet
	if cachedGs, exists := s.gsCache.Load(v.GuardianSetIndex); exists {
		var ok bool
		gs, ok = cachedGs.(*common.GuardianSet)
		if !ok {
			return nil, fmt.Errorf("internal error")
		}
	} else {
		evmGs, err := s.evmConnector.GetGuardianSet(ctx, v.GuardianSetIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to load guardian set [%d]: %w", v.GuardianSetIndex, err)
		}
		gs = &common.GuardianSet{
			Keys:  evmGs.Keys,
			Index: v.GuardianSetIndex,
		}
		s.gsCache.Store(v.GuardianSetIndex, gs)
	}

	if slices.Index(gs.Keys, s.guardianAddress) != -1 {
		return nil, fmt.Errorf("local guardian is already on the old set")
	}

	// Verify VAA
	err = v.Verify(gs.Keys)
	if err != nil {
		return nil, fmt.Errorf("failed to verify existing VAA: %w", err)
	}

	if len(req.NewGuardianAddrs) > 255 {
		return nil, errors.New("new guardian set has too many guardians")
	}
	newGS := make([]ethcommon.Address, len(req.NewGuardianAddrs))
	for i, guardianString := range req.NewGuardianAddrs {
		guardianAddress := ethcommon.HexToAddress(guardianString)
		newGS[i] = guardianAddress
	}

	// Make sure there are no duplicates. Compact needs to take a sorted slice to remove all duplicates.
	newGSSorted := slices.Clone(newGS)
	slices.SortFunc(newGSSorted, func(a, b ethcommon.Address) int {
		return bytes.Compare(a[:], b[:])
	})
	newGsLen := len(newGSSorted)
	if len(slices.Compact(newGSSorted)) != newGsLen {
		return nil, fmt.Errorf("duplicate guardians in the guardian set")
	}

	localGuardianIndex := slices.Index(newGS, s.guardianAddress)
	if localGuardianIndex == -1 {
		return nil, fmt.Errorf("local guardian is not a member of the new guardian set")
	}

	newVAA := &vaa.VAA{
		Version: v.Version,
		// Set the new guardian set index
		GuardianSetIndex: req.NewGuardianSetIndex,
		// Signatures will be repopulated
		Signatures:       nil,
		Timestamp:        v.Timestamp,
		Nonce:            v.Nonce,
		Sequence:         v.Sequence,
		ConsistencyLevel: v.ConsistencyLevel,
		EmitterChain:     v.EmitterChain,
		EmitterAddress:   v.EmitterAddress,
		Payload:          v.Payload,
	}

	// Copy original VAA signatures
	for _, sig := range v.Signatures {
		signerAddress := gs.Keys[sig.Index]
		newIndex := slices.Index(newGS, signerAddress)
		// Guardian is not part of the new set
		if newIndex == -1 {
			continue
		}
		newVAA.Signatures = append(newVAA.Signatures, &vaa.Signature{
			Index:     uint8(newIndex), // #nosec G115 -- The length of newGS is constrained to a uint8 above
			Signature: sig.Signature,
		})
	}

	// Add our own signature only if the new guardian set would reach quorum
	if vaa.CalculateQuorum(len(newGS)) > len(newVAA.Signatures)+1 {
		return nil, errors.New("cannot reach quorum on new guardian set with the local signature")
	}

	// Add local signature
	sig, err := s.guardianSigner.Sign(ctx, v.SigningDigest().Bytes())
	if err != nil {
		panic(err)
	}

	signature := [ecdsaSignatureLength]byte{}
	copy(signature[:], sig)

	newVAA.Signatures = append(v.Signatures, &vaa.Signature{
		Index:     uint8(localGuardianIndex), // #nosec G115 -- The length of newGS is constrained to a uint8 above
		Signature: signature,
	})

	// Sort VAA signatures by guardian ID
	slices.SortFunc(newVAA.Signatures, func(a, b *vaa.Signature) int {
		if a.Index < b.Index {
			return -1
		} else if a.Index > b.Index {
			return 1
		}
		return 0
	})

	newVAABytes, err := newVAA.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal new VAA: %w", err)
	}

	return &nodev1.SignExistingVAAResponse{Vaa: newVAABytes}, nil
}

func (s *nodePrivilegedService) DumpRPCs(ctx context.Context, req *nodev1.DumpRPCsRequest) (*nodev1.DumpRPCsResponse, error) {
	return &nodev1.DumpRPCsResponse{
		Response: s.rpcMap,
	}, nil
}

func (s *nodePrivilegedService) GetAndObserveMissingVAAs(ctx context.Context, req *nodev1.GetAndObserveMissingVAAsRequest) (*nodev1.GetAndObserveMissingVAAsResponse, error) {
	// Get URL and API key from the command line
	url := req.GetUrl()
	apiKey := req.GetApiKey()

	// Create the body of the request
	jsonBody := []byte(`{"apiKey": "` + apiKey + `"}`)
	jsonBodyReader := bytes.NewReader(jsonBody)

	// Create the actual request
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, url, jsonBodyReader)
	if err != nil {
		fmt.Printf("GetAndObserveMissingVAAs: could not create request: %s\n", err)
		return nil, err
	}

	httpRequest.Header.Set("Content-Type", "application/json")

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	// Call the cloud function to get the missing VAAs
	results, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("GetAndObserveMissingVAAs: error making http request: %s\n", err)
		return nil, err
	}

	defer results.Body.Close()

	// Collect the results
	resBody, err := io.ReadAll(results.Body)
	if err != nil {
		fmt.Printf("GetAndObserveMissingVAAs: could not read response body: %s\n", err)
		return nil, err
	}
	fmt.Printf("client: response body: %s\n", resBody)
	type MissingVAA struct {
		Chain  int    `json:"chain"`
		VaaKey string `json:"vaaKey"`
		Txhash string `json:"txhash"`
	}
	var missingVAAs []MissingVAA
	err = json.Unmarshal(resBody, &missingVAAs)
	if err != nil {
		fmt.Printf("GetAndObserveMissingVAAs: could not unmarshal response body: %s\n", err)
		return nil, err
	}

	MAX_VAAS_TO_PROCESS := 25
	// Only do a max of 25 at a time so as to not overload the node
	numVaas := len(missingVAAs)
	processingLen := numVaas
	if processingLen > MAX_VAAS_TO_PROCESS {
		processingLen = MAX_VAAS_TO_PROCESS
	}

	// Start injecting the VAAs
	obsCounter := 0
	errCounter := 0
	errMsgs := "Messages: "
	for i := 0; i < processingLen; i++ {
		missingVAA := missingVAAs[i]
		// First check to see if this VAA has already been signed
		// Convert vaaKey to VAAID
		splits := strings.Split(missingVAA.VaaKey, "/")
		chainID, err := strconv.Atoi(splits[0])
		if err != nil {
			errMsgs += fmt.Sprintf("\nerror converting chainID [%s] to int", missingVAA.VaaKey)
			errCounter++
			continue
		}
		if chainID > math.MaxUint16 {
			errMsgs += fmt.Sprintf("\nchainID [%d] not a valid uint16", chainID)
			errCounter++
			continue
		}
		sequence, err := strconv.ParseUint(splits[2], 10, 64)
		if err != nil {
			errMsgs += fmt.Sprintf("\nerror converting sequence %s to uint64", splits[2])
			errCounter++
			continue
		}
		vaaKey := db.VAAID{EmitterChain: vaa.ChainID(chainID), EmitterAddress: vaa.Address([]byte(splits[1])), Sequence: sequence} // #nosec G115 -- This chainId conversion is verified above
		hasVaa, err := s.db.HasVAA(vaaKey)
		if err != nil || hasVaa {
			errMsgs += fmt.Sprintf("\nerror checking for VAA %s", missingVAA.VaaKey)
			errCounter++
			continue
		}
		var obsvReq gossipv1.ObservationRequest
		if missingVAA.Chain > math.MaxUint16 {
			errMsgs += fmt.Sprintf("\nmissing VAA chainID [%d] not a valid uint16", missingVAA.Chain)
			errCounter++
			continue
		}
		obsvReq.ChainId = uint32(missingVAA.Chain) // #nosec G115 -- This conversion is checked above
		obsvReq.TxHash, err = hex.DecodeString(strings.TrimPrefix(missingVAA.Txhash, "0x"))
		if err != nil {
			obsvReq.TxHash, err = base58.Decode(missingVAA.Txhash)
			if err != nil {
				errMsgs += "Invalid transaction hash (neither hex nor base58)"
				errCounter++
				continue
			}
		}
		errMsgs += fmt.Sprintf("\nAttempting to observe %s", missingVAA.Txhash)
		// Call the following function to send the observation request
		if err := common.PostObservationRequest(s.obsvReqSendC, &obsvReq); err != nil {
			errMsgs += fmt.Sprintf("\nPostObservationRequest error %s", err.Error())
			errCounter++
			continue
		}
		obsCounter++
	}
	response := "There were no missing VAAs to recover."
	if processingLen > 0 {
		response = fmt.Sprintf("Successfully injected %d of %d VAAs. %d errors were encountered.", obsCounter, processingLen, errCounter)
		if numVaas > MAX_VAAS_TO_PROCESS {
			response += fmt.Sprintf("\nOnly %d of the %d missing VAAs were processed.  Run the command again to process more.", MAX_VAAS_TO_PROCESS, numVaas)
		}
	}
	response += "\n" + errMsgs
	return &nodev1.GetAndObserveMissingVAAsResponse{
		Response: response,
	}, nil
}
