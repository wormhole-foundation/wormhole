package guardiand

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"golang.org/x/exp/slices"

	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/node/pkg/publicrpc"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/certusone/wormhole/node/pkg/common"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type nodePrivilegedService struct {
	nodev1.UnimplementedNodePrivilegedServiceServer
	db              *db.Database
	injectC         chan<- *vaa.VAA
	obsvReqSendC    chan<- *gossipv1.ObservationRequest
	logger          *zap.Logger
	signedInC       chan<- *gossipv1.SignedVAAWithQuorum
	governor        *governor.ChainGovernor
	evmConnector    connectors.Connector
	gsCache         sync.Map
	gk              *ecdsa.PrivateKey
	guardianAddress ethcommon.Address
	testnetMode     bool
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

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyGuardianSetUpdate{
			Keys:     addrs,
			NewIndex: guardianSetIndex + 1,
		}.Serialize())

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

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyContractUpgrade{
			ChainID:     vaa.ChainID(req.ChainId),
			NewContract: newContractAddress,
		}.Serialize())

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

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyTokenBridgeRegisterChain{
			Module:         req.Module,
			ChainID:        vaa.ChainID(req.ChainId),
			EmitterAddress: emitterAddress,
		}.Serialize())

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

	if len(req.Reason) > 32 {
		return nil, errors.New("the reason should not be larger than 32 bytes")
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

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyAccountantModifyBalance{
			Module:        req.Module,
			TargetChainID: vaa.ChainID(req.TargetChainId),

			Sequence:     req.Sequence,
			ChainId:      vaa.ChainID(req.ChainId),
			TokenChain:   vaa.ChainID(req.TokenChain),
			TokenAddress: tokenAdress,
			Kind:         uint8(req.Kind),
			Amount:       amount,
			Reason:       req.Reason,
		}.Serialize())

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

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyTokenBridgeUpgradeContract{
			Module:        req.Module,
			TargetChainID: vaa.ChainID(req.TargetChainId),
			NewContract:   newContract,
		}.Serialize())

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

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyWormchainStoreCode{
			WasmHash: wasmHash,
		}.Serialize())

	return v, nil
}

// wormchainInstantiateContract converts a nodev1.WormchainInstantiateContract to its canonical VAA representation
// Returns an error if the data is invalid
func wormchainInstantiateContract(req *nodev1.WormchainInstantiateContract, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	instantiationParams_hash := vaa.CreateInstatiateCosmwasmContractHash(req.CodeId, req.Label, []byte(req.InstantiationMsg))

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyWormchainInstantiateContract{
			InstantiationParamsHash: instantiationParams_hash,
		}.Serialize())

	return v, nil
}

// wormchainMigrateContract converts a nodev1.WormchainMigrateContract to its canonical VAA representation
func wormchainMigrateContract(req *nodev1.WormchainMigrateContract, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	instantiationParams_hash := vaa.CreateMigrateCosmwasmContractHash(req.CodeId, req.Contract, []byte(req.InstantiationMsg))

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyWormchainMigrateContract{
			MigrationParamsHash: instantiationParams_hash,
		}.Serialize())

	return v, nil
}

<<<<<<< HEAD
<<<<<<< HEAD
// circleIntegrationUpdateWormholeFinality converts a nodev1.CircleIntegrationUpdateWormholeFinality to its canonical VAA representation
// Returns an error if the data is invalid
func circleIntegrationUpdateWormholeFinality(req *nodev1.CircleIntegrationUpdateWormholeFinality, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	if req.TargetChainId > math.MaxUint16 {
		return nil, fmt.Errorf("invalid target chain id, must be <= %d", math.MaxUint16)
	}
=======
<<<<<<< HEAD
=======
// circleIntegrationUpdateWormholeFinality converts a nodev1.CircleIntegrationUpdateWormholeFinality to its canonical VAA representation
// Returns an error if the data is invalid
func circleIntegrationUpdateWormholeFinality(req *nodev1.CircleIntegrationUpdateWormholeFinality, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
>>>>>>> 6173890e (Adding wormhole relayer governance VAA injection methods)
	if req.Finality > math.MaxUint8 {
		return nil, fmt.Errorf("invalid finality, must be <= %d", math.MaxUint8)
	}
	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyCircleIntegrationUpdateWormholeFinality{
<<<<<<< HEAD
			TargetChainID: vaa.ChainID(req.TargetChainId),
			Finality:      uint8(req.Finality),
=======
			Finality: uint8(req.Finality),
>>>>>>> 6173890e (Adding wormhole relayer governance VAA injection methods)
		}.Serialize())

	return v, nil
}

// circleIntegrationRegisterEmitterAndDomain converts a nodev1.CircleIntegrationRegisterEmitterAndDomain to its canonical VAA representation
// Returns an error if the data is invalid
func circleIntegrationRegisterEmitterAndDomain(req *nodev1.CircleIntegrationRegisterEmitterAndDomain, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
<<<<<<< HEAD
	if req.TargetChainId > math.MaxUint16 {
		return nil, fmt.Errorf("invalid target chain id, must be <= %d", math.MaxUint16)
	}
=======
>>>>>>> 6173890e (Adding wormhole relayer governance VAA injection methods)
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

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyCircleIntegrationRegisterEmitterAndDomain{
<<<<<<< HEAD
			TargetChainID:         vaa.ChainID(req.TargetChainId),
=======
>>>>>>> 6173890e (Adding wormhole relayer governance VAA injection methods)
			ForeignEmitterChainId: vaa.ChainID(req.ForeignEmitterChainId),
			ForeignEmitterAddress: foreignEmitterAddress,
			CircleDomain:          req.CircleDomain,
		}.Serialize())

	return v, nil
}

// circleIntegrationUpgradeContractImplementation converts a nodev1.CircleIntegrationUpgradeContractImplementation to its canonical VAA representation
// Returns an error if the data is invalid
func circleIntegrationUpgradeContractImplementation(req *nodev1.CircleIntegrationUpgradeContractImplementation, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
<<<<<<< HEAD
	if req.TargetChainId > math.MaxUint16 {
		return nil, fmt.Errorf("invalid target chain id, must be <= %d", math.MaxUint16)
	}
=======
>>>>>>> 6173890e (Adding wormhole relayer governance VAA injection methods)
	b, err := hex.DecodeString(req.NewImplementationAddress)
	if err != nil {
		return nil, errors.New("invalid new implementation address encoding (expected hex)")
	}

	if len(b) != 32 {
		return nil, errors.New("invalid new implementation address (expected 32 bytes)")
	}

	newImplementationAddress := vaa.Address{}
	copy(newImplementationAddress[:], b)

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyCircleIntegrationUpgradeContractImplementation{
<<<<<<< HEAD
			TargetChainID:            vaa.ChainID(req.TargetChainId),
=======
>>>>>>> 6173890e (Adding wormhole relayer governance VAA injection methods)
			NewImplementationAddress: newImplementationAddress,
		}.Serialize())

	return v, nil
}

<<<<<<< HEAD
=======
=======
>>>>>>> a4732803 (Fix compile error)
// wormholeRelayerSetDefaultRelayProvider converts a nodev1.WormholeRelayerSetDefaultRelayProvider message to its canonical VAA representation.
// Returns an error if the data is invalid.
func wormholeRelayerSetDefaultRelayProvider(req *nodev1.WormholeRelayerSetDefaultRelayProvider, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	if req.ChainId > math.MaxUint16 {
		return nil, errors.New("invalid target_chain_id")
	}

	b, err := hex.DecodeString(req.NewDefaultRelayProviderAddress)
	if err != nil {
		return nil, errors.New("invalid new default relay provider address (expected hex)")
	}

	if len(b) != 32 {
		return nil, errors.New("invalid new default relay provider address (expected 32 bytes)")
	}

	newDefaultRelayProviderAddress := vaa.Address{}
	copy(newDefaultRelayProviderAddress[:], b)

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyWormholeRelayerSetDefaultRelayProvider{
			ChainID: vaa.ChainID(req.ChainId),
			NewDefaultRelayProviderAddress: newDefaultRelayProviderAddress,
		}.Serialize())

	return v, nil
}

<<<<<<< HEAD
>>>>>>> 8c709813 (Adding wormhole relayer governance VAA injection methods)
>>>>>>> 6173890e (Adding wormhole relayer governance VAA injection methods)
=======
>>>>>>> a4732803 (Fix compile error)
func (s *nodePrivilegedService) InjectGovernanceVAA(ctx context.Context, req *nodev1.InjectGovernanceVAARequest) (*nodev1.InjectGovernanceVAAResponse, error) {
	s.logger.Info("governance VAA injected via admin socket", zap.String("request", req.String()))

	var (
		v   *vaa.VAA
		err error
	)

	timestamp := time.Unix(int64(req.Timestamp), 0)

	digests := make([][]byte, len(req.Messages))

	for i, message := range req.Messages {
		switch payload := message.Payload.(type) {
		case *nodev1.GovernanceMessage_GuardianSet:
			v, err = adminGuardianSetUpdateToVAA(payload.GuardianSet, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_ContractUpgrade:
			v, err = adminContractUpgradeToVAA(payload.ContractUpgrade, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_BridgeRegisterChain:
			v, err = tokenBridgeRegisterChain(payload.BridgeRegisterChain, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_BridgeContractUpgrade:
			v, err = tokenBridgeUpgradeContract(payload.BridgeContractUpgrade, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_AccountantModifyBalance:
			v, err = accountantModifyBalance(payload.AccountantModifyBalance, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_WormchainStoreCode:
			v, err = wormchainStoreCode(payload.WormchainStoreCode, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_WormchainInstantiateContract:
			v, err = wormchainInstantiateContract(payload.WormchainInstantiateContract, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_WormchainMigrateContract:
			v, err = wormchainMigrateContract(payload.WormchainMigrateContract, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
<<<<<<< HEAD
		case *nodev1.GovernanceMessage_CircleIntegrationUpdateWormholeFinality:
			v, err = circleIntegrationUpdateWormholeFinality(payload.CircleIntegrationUpdateWormholeFinality, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_CircleIntegrationRegisterEmitterAndDomain:
			v, err = circleIntegrationRegisterEmitterAndDomain(payload.CircleIntegrationRegisterEmitterAndDomain, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_CircleIntegrationUpgradeContractImplementation:
			v, err = circleIntegrationUpgradeContractImplementation(payload.CircleIntegrationUpgradeContractImplementation, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
=======
		case *nodev1.GovernanceMessage_WormholeRelayerSetDefaultRelayProvider:
			v, err = wormholeRelayerSetDefaultRelayProvider(payload.WormholeRelayerSetDefaultRelayProvider, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
>>>>>>> 6173890e (Adding wormhole relayer governance VAA injection methods)
		default:
			panic(fmt.Sprintf("unsupported VAA type: %T", payload))
		}
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		// Generate digest of the unsigned VAA.
		digest := v.SigningDigest()

		s.logger.Info("governance VAA constructed",
			zap.Any("vaa", v),
			zap.String("digest", digest.String()),
		)

		s.injectC <- v

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
			// (verifying signature, publishing to BigTable, storing in local DB...).
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

func adminServiceRunnable(
	logger *zap.Logger,
	socketPath string,
	injectC chan<- *vaa.VAA,
	signedInC chan<- *gossipv1.SignedVAAWithQuorum,
	obsvReqSendC chan<- *gossipv1.ObservationRequest,
	db *db.Database,
	gst *common.GuardianSetState,
	gov *governor.ChainGovernor,
	gk *ecdsa.PrivateKey,
	ethRpc *string,
	ethContract *string,
	testnetMode bool,
) (supervisor.Runnable, error) {
	// Delete existing UNIX socket, if present.
	fi, err := os.Stat(socketPath)
	if err == nil {
		fmode := fi.Mode()
		if fmode&os.ModeType == os.ModeSocket {
			err = os.Remove(socketPath)
			if err != nil {
				return nil, fmt.Errorf("failed to remove existing socket at %s: %w", socketPath, err)
			}
		} else {
			return nil, fmt.Errorf("%s is not a UNIX socket", socketPath)
		}
	}

	// Create a new UNIX socket and listen to it.

	// The socket is created with the default umask. We set a restrictive umask in setRestrictiveUmask
	// to ensure that any files we create are only readable by the user - this is much harder to mess up.
	// The umask avoids a race condition between file creation and chmod.

	laddr, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("invalid listen address: %v", err)
	}
	l, err := net.ListenUnix("unix", laddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", socketPath, err)
	}

	logger.Info("admin server listening on", zap.String("path", socketPath))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var evmConnector connectors.Connector
	if ethRPC != nil && ethContract != nil {
		contract := ethcommon.HexToAddress(*ethContract)
		evmConnector, err = connectors.NewEthereumConnector(ctx, "eth", *ethRpc, contract, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to connecto to ethereum")
		}
	}

	nodeService := &nodePrivilegedService{
		db:              db,
		injectC:         injectC,
		obsvReqSendC:    obsvReqSendC,
		logger:          logger.Named("adminservice"),
		signedInC:       signedInC,
		governor:        gov,
		gk:              gk,
		guardianAddress: ethcrypto.PubkeyToAddress(gk.PublicKey),
		evmConnector:    evmConnector,
		testnetMode:     testnetMode,
	}

	publicrpcService := publicrpc.NewPublicrpcServer(logger, db, gst, gov)

	grpcServer := common.NewInstrumentedGRPCServer(logger, common.GrpcLogDetailMinimal)
	nodev1.RegisterNodePrivilegedServiceServer(grpcServer, nodeService)
	publicrpcv1.RegisterPublicRPCServiceServer(grpcServer, publicrpcService)
	return supervisor.GRPCServer(grpcServer, l, false), nil
}

func (s *nodePrivilegedService) SendObservationRequest(ctx context.Context, req *nodev1.SendObservationRequestRequest) (*nodev1.SendObservationRequestResponse, error) {
	if err := common.PostObservationRequest(s.obsvReqSendC, req.ObservationRequest); err != nil {
		return nil, err
	}

	s.logger.Info("sent observation request", zap.Any("request", req.ObservationRequest))
	return &nodev1.SendObservationRequestResponse{}, nil
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

	resp, err := s.governor.ResetReleaseTimer(req.VaaId)
	if err != nil {
		return nil, err
	}

	return &nodev1.ChainGovernorResetReleaseTimerResponse{
		Response: resp,
	}, nil
}

func (s *nodePrivilegedService) PurgePythNetVaas(ctx context.Context, req *nodev1.PurgePythNetVaasRequest) (*nodev1.PurgePythNetVaasResponse, error) {
	prefix := db.VAAID{EmitterChain: vaa.ChainIDPythNet}
	oldestTime := time.Now().Add(-time.Hour * 24 * time.Duration(req.DaysOld))
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
		gs = cachedGs.(*common.GuardianSet)
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
	slices.SortFunc(newGSSorted, func(a, b ethcommon.Address) bool {
		return bytes.Compare(a[:], b[:]) < 0
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
			Index:     uint8(newIndex),
			Signature: sig.Signature,
		})
	}

	// Add our own signature only if the new guardian set would reach quorum
	if vaa.CalculateQuorum(len(newGS)) > len(newVAA.Signatures)+1 {
		return nil, errors.New("cannot reach quorum on new guardian set with the local signature")
	}

	// Add local signature
	newVAA.AddSignature(s.gk, uint8(localGuardianIndex))

	// Sort VAA signatures by guardian ID
	slices.SortFunc(newVAA.Signatures, func(a, b *vaa.Signature) bool {
		return a.Index < b.Index
	})

	newVAABytes, err := newVAA.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal new VAA: %w", err)
	}

	return &nodev1.SignExistingVAAResponse{Vaa: newVAABytes}, nil
}

func (s *nodePrivilegedService) DumpRPCs(ctx context.Context, req *nodev1.DumpRPCsRequest) (*nodev1.DumpRPCsResponse, error) {
	rpcMap := make(map[string]string)

	rpcMap["acalaRPC"] = *acalaRPC
	rpcMap["algorandIndexerRPC"] = *algorandIndexerRPC
	rpcMap["algorandAlgodRPC"] = *algorandAlgodRPC
	rpcMap["aptosRPC"] = *aptosRPC
	rpcMap["arbitrumRPC"] = *arbitrumRPC
	rpcMap["auroraRPC"] = *auroraRPC
	rpcMap["avalancheRPC"] = *avalancheRPC
	rpcMap["baseRPC"] = *baseRPC
	rpcMap["bscRPC"] = *bscRPC
	rpcMap["celoRPC"] = *celoRPC
	rpcMap["ethRPC"] = *ethRPC
	rpcMap["fantomRPC"] = *fantomRPC
	rpcMap["ibcLCD"] = *ibcLCD
	rpcMap["ibcWS"] = *ibcWS
	rpcMap["karuraRPC"] = *karuraRPC
	rpcMap["klaytnRPC"] = *klaytnRPC
	rpcMap["moonbeamRPC"] = *moonbeamRPC
	rpcMap["nearRPC"] = *nearRPC
	rpcMap["neonRPC"] = *neonRPC
	rpcMap["oasisRPC"] = *oasisRPC
	rpcMap["optimismRPC"] = *optimismRPC
	rpcMap["polygonRPC"] = *polygonRPC
	rpcMap["pythnetRPC"] = *pythnetRPC
	rpcMap["pythnetWS"] = *pythnetWS
	rpcMap["sei"] = "IBC"
	if s.testnetMode {
		rpcMap["sepoliaRPC"] = *sepoliaRPC
	}
	rpcMap["solanaRPC"] = *solanaRPC
	rpcMap["suiRPC"] = *suiRPC
	rpcMap["terraWS"] = *terraWS
	rpcMap["terraLCD"] = *terraLCD
	rpcMap["terra2WS"] = *terra2WS
	rpcMap["terra2LCD"] = *terra2LCD
	rpcMap["wormchainWS"] = *wormchainWS
	rpcMap["wormchainLCD"] = *wormchainLCD
	rpcMap["xplaWS"] = *xplaWS
	rpcMap["xplaLCD"] = *xplaLCD

	return &nodev1.DumpRPCsResponse{
		Response: rpcMap,
	}, nil
}
