// Package manager implements the Manager Service for the guardian node.
// The Manager Service subscribes to incoming VAAs and processes them
// according to manager requirements.
package manager

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/txscript"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	"github.com/certusone/wormhole/node/pkg/manager/dogecoin"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// ManagerSignature represents signatures produced by a single manager signer for a VAA.
// Multiple ManagerSignatures from different signers are aggregated to form a complete
// multisig transaction.
type ManagerSignature struct {
	// VAAHash is the hash of the VAA that triggered this signing.
	VAAHash []byte
	// VAAID is the VAA ID in format "{chain}/{emitter}/{sequence}".
	VAAID string
	// DestinationChain is the target chain (e.g., Dogecoin).
	DestinationChain vaa.ChainID
	// ManagerSetIndex is the delegated manager set index from the payload.
	ManagerSetIndex uint32
	// SignerIndex is this signer's index within the manager set.
	SignerIndex uint8
	// InputSignatures contains one signature per input UTXO, in order.
	// Each signature is in the format expected by the destination chain (DER-encoded for Dogecoin).
	InputSignatures [][]byte
}

// AggregatedTransaction holds signatures from multiple signers for a single VAA.
// It is used to collect signatures until we have M-of-N required for broadcast.
type AggregatedTransaction struct {
	// VAAHash is the hash of the VAA that triggered this signing.
	VAAHash []byte
	// VAAID is the VAA ID in format "{chain}/{emitter}/{sequence}".
	VAAID string
	// DestinationChain is the target chain (e.g., Dogecoin).
	DestinationChain vaa.ChainID
	// ManagerSetIndex is the delegated manager set index from the payload.
	ManagerSetIndex uint32
	// Required is the M value (number of signatures needed).
	Required uint8
	// Total is the N value (total number of possible signers).
	Total uint8
	// Signatures maps signer index to their signatures.
	// Each entry contains the per-input signatures from that signer.
	Signatures map[uint8][][]byte
}

// emitterEntry represents a known manager emitter.
type emitterEntry struct {
	chainId vaa.ChainID
	addr    []byte
}

// ManagerSetConfig holds the manager set configuration for a specific chain.
// This includes the M-of-N multisig parameters and the public keys of all signers.
type ManagerSetConfig struct {
	// Index is the manager set index (for governance tracking)
	Index uint32
	// M is the number of signatures required (threshold)
	M uint8
	// N is the total number of signers
	N uint8
	// PublicKeys are the compressed secp256k1 public keys of the manager signers
	PublicKeys [][]byte
	// IsSigner indicates whether this node is part of the manager set
	IsSigner bool
	// SignerIndex is this node's index within the manager set (0-based)
	// Only valid if IsSigner is true
	SignerIndex uint8
}

// ManagerService manages manager-related processing of VAAs.
type ManagerService struct {
	ctx      context.Context
	logger   *zap.Logger
	vaaC     <-chan *vaa.VAA
	env      common.Environment
	emitters []emitterEntry
	signers  map[vaa.ChainID]guardiansigner.GuardianSigner
	// signerPubKeys stores the compressed secp256k1 public keys for each chain's signer.
	signerPubKeys map[vaa.ChainID][]byte
	// managerSets is a map of chain ID -> manager set index -> manager set config.
	// This allows tracking multiple manager sets per chain (for governance transitions).
	managerSets map[vaa.ChainID]map[uint32]*ManagerSetConfig
	// gossipSendC is the channel for broadcasting signed manager transactions to the gossip network.
	gossipSendC chan<- []byte
	// incomingTxC receives signed manager transactions from other manager nodes via gossip.
	incomingTxC <-chan *gossipv1.SignedManagerTransaction
	// pendingTxMu protects access to pendingTx.
	pendingTxMu sync.RWMutex
	// pendingTx stores aggregated transactions indexed by VAA hash (hex encoded).
	pendingTx map[string]*AggregatedTransaction
}

// NewManagerService creates a new ManagerService instance.
func NewManagerService(
	ctx context.Context,
	logger *zap.Logger,
	vaaC <-chan *vaa.VAA,
	env common.Environment,
	signers map[vaa.ChainID]guardiansigner.GuardianSigner,
	gossipSendC chan<- []byte,
	incomingTxC <-chan *gossipv1.SignedManagerTransaction,
) *ManagerService {
	// Select the appropriate emitter list based on environment
	var emitters []emitterEntry
	//nolint:exhaustive // MainNet, GoTest, and AccountantMock intentionally fall through to default
	switch env {
	case common.UnsafeDevNet:
		emitters = parseEmitters(sdk.KnownDevnetManagerEmitters)
	case common.TestNet:
		emitters = parseEmitters(sdk.KnownTestnetManagerEmitters)
	// TODO: Add mainnet emitter list when available
	// case common.MainNet:
	// 	emitters = parseEmitters(sdk.KnownManagerEmitters)
	default:
		emitters = []emitterEntry{}
	}

	// Initialize manager sets map for each chain that has a signer configured
	managerSets := make(map[vaa.ChainID]map[uint32]*ManagerSetConfig)
	for chainID := range signers {
		managerSets[chainID] = make(map[uint32]*ManagerSetConfig)
	}

	// Compute compressed public keys for each signer
	signerPubKeys := make(map[vaa.ChainID][]byte)
	for chainID, signer := range signers {
		pubKey := signer.PublicKey(ctx)
		signerPubKeys[chainID] = compressPublicKey(&pubKey)
	}

	// Load default manager sets based on environment
	if env == common.UnsafeDevNet {
		// Load the devnet manager set for Dogecoin
		if _, ok := signers[vaa.ChainIDDogecoin]; ok {
			devnetSet := loadDevnetManagerSet(ctx, signers[vaa.ChainIDDogecoin])
			managerSets[vaa.ChainIDDogecoin][devnetSet.Index] = devnetSet
		}
	}

	return &ManagerService{
		ctx:           ctx,
		logger:        logger.With(zap.String("component", "manager")),
		vaaC:          vaaC,
		env:           env,
		emitters:      emitters,
		signers:       signers,
		signerPubKeys: signerPubKeys,
		managerSets:   managerSets,
		gossipSendC:   gossipSendC,
		incomingTxC:   incomingTxC,
		pendingTx:     make(map[string]*AggregatedTransaction),
	}
}

// loadDevnetManagerSet creates a ManagerSetConfig from the SDK's KnownDevnetManagerSet.
// It also determines the signer's index within the set by comparing public keys.
func loadDevnetManagerSet(ctx context.Context, signer guardiansigner.GuardianSigner) *ManagerSetConfig {
	sdkSet := sdk.KnownDevnetManagerSet

	// Convert public keys from [33]byte to []byte
	pubKeys := make([][]byte, len(sdkSet.PublicKeys))
	for i, pk := range sdkSet.PublicKeys {
		pubKeys[i] = pk[:]
	}

	// Determine signer index by matching public key
	var isSigner bool
	var signerIndex uint8
	if signer != nil && ctx != nil {
		signerPubKey := signer.PublicKey(ctx)
		signerCompressed := compressPublicKey(&signerPubKey)
		for i, pk := range pubKeys {
			if bytes.Equal(signerCompressed, pk) {
				isSigner = true
				signerIndex = uint8(i) // #nosec G115 -- pubKeys length is bounded by PublicKeysLen (uint8) in governance
				break
			}
		}
	}

	return &ManagerSetConfig{
		Index:       0, // Initial set index
		M:           sdkSet.M,
		N:           sdkSet.N,
		PublicKeys:  pubKeys,
		IsSigner:    isSigner,
		SignerIndex: signerIndex,
	}
}

// parseEmitters converts the SDK emitter format to internal emitterEntry slice.
func parseEmitters(sdkEmitters []struct {
	ChainId vaa.ChainID
	Addr    string
}) []emitterEntry {
	result := make([]emitterEntry, 0, len(sdkEmitters))
	for _, e := range sdkEmitters {
		addr, err := hex.DecodeString(e.Addr)
		if err != nil {
			panic("invalid emitter address: " + e.Addr)
		}
		result = append(result, emitterEntry{
			chainId: e.ChainId,
			addr:    addr,
		})
	}
	return result
}

// Run starts the manager service and begins processing incoming VAAs.
func (c *ManagerService) Run(ctx context.Context) error {
	c.logger.Info("manager service enabled",
		zap.String("environment", string(c.env)),
		zap.Int("known_emitters", len(c.emitters)),
		zap.Int("signers", len(c.signers)),
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case v := <-c.vaaC:
			c.handleVAA(v)
		case tx := <-c.incomingTxC:
			c.handleIncomingTransaction(tx)
		}
	}
}

// handleVAA processes an incoming signed VAA.
func (c *ManagerService) handleVAA(v *vaa.VAA) {
	// SECURITY: this channel should only be pushed to by a process that has verified the signatures on the VAA to belong to the current guardian set

	// Check if this VAA is from a known manager emitter
	if !c.isKnownEmitter(v.EmitterChain, v.EmitterAddress) {
		c.logger.Debug("skipping VAA from unknown emitter",
			zap.String("message_id", v.MessageID()),
			zap.Stringer("emitter_chain", v.EmitterChain),
			zap.String("emitter_address", v.EmitterAddress.String()),
		)
		return
	}

	// VAA is from a known manager emitter, process it
	c.logger.Info("received VAA from manager emitter",
		zap.String("message_id", v.MessageID()),
		zap.Stringer("emitter_chain", v.EmitterChain),
		zap.Uint64("sequence", v.Sequence),
	)

	// Parse the UTXO unlock payload
	payload, err := vaa.DeserializeUTXOUnlockPayload(v.Payload)
	if err != nil {
		c.logger.Error("failed to parse UTXO unlock payload",
			zap.String("message_id", v.MessageID()),
			zap.Error(err),
		)
		return
	}

	c.logger.Info("parsed UTXO unlock payload",
		zap.String("message_id", v.MessageID()),
		zap.Stringer("destination_chain", payload.DestinationChain),
		zap.Uint32("manager_set_index", payload.DelegatedManagerSetIndex),
		zap.Int("num_inputs", len(payload.Inputs)),
		zap.Int("num_outputs", len(payload.Outputs)),
	)

	// Check if we have a signer for the destination chain
	signer, ok := c.signers[payload.DestinationChain]
	if !ok {
		c.logger.Warn("no signer configured for destination chain",
			zap.String("message_id", v.MessageID()),
			zap.Stringer("destination_chain", payload.DestinationChain),
		)
		return
	}

	// Sign the transaction inputs
	sig, err := c.signTransaction(v, payload, signer)
	if err != nil {
		c.logger.Error("failed to sign transaction",
			zap.String("message_id", v.MessageID()),
			zap.Error(err),
		)
		return
	}

	c.logger.Info("signed manager transaction",
		zap.String("message_id", v.MessageID()),
		zap.Stringer("destination_chain", sig.DestinationChain),
		zap.Int("num_signatures", len(sig.InputSignatures)),
	)

	// Broadcast the signature to other manager service instances
	if c.gossipSendC != nil {
		c.broadcastSignature(sig)
	}

	// Store the signature for aggregation
	c.storeSignature(sig)
}

// handleIncomingTransaction processes a signed manager transaction received from another manager node.
func (c *ManagerService) handleIncomingTransaction(tx *gossipv1.SignedManagerTransaction) {
	c.logger.Debug("received signed manager transaction from peer",
		zap.String("vaa_id", tx.VaaId),
		zap.Uint32("destination_chain", tx.DestinationChain),
		zap.Uint32("signer_index", tx.SignerIndex),
		zap.Int("num_signatures", len(tx.Signatures)),
	)

	destChain := vaa.ChainID(tx.DestinationChain) // #nosec G115 -- ChainID is uint16, protobuf uses uint32 for wire compatibility

	// Validate the signer is in the manager set
	chainSets, ok := c.managerSets[destChain]
	if !ok {
		c.logger.Warn("received transaction for unconfigured chain",
			zap.Stringer("chain", destChain),
		)
		return
	}

	managerSet, ok := chainSets[tx.ManagerSetIndex]
	if !ok {
		c.logger.Warn("received transaction for unknown manager set",
			zap.Uint32("index", tx.ManagerSetIndex),
		)
		return
	}

	if tx.SignerIndex >= uint32(managerSet.N) {
		c.logger.Warn("received transaction with invalid signer index",
			zap.Uint32("signer_index", tx.SignerIndex),
			zap.Uint8("max_signers", managerSet.N),
		)
		return
	}

	// Convert to ManagerSignature and store
	sig := &ManagerSignature{
		VAAHash:          tx.VaaHash,
		VAAID:            tx.VaaId,
		DestinationChain: destChain,
		ManagerSetIndex:  tx.ManagerSetIndex,
		SignerIndex:      uint8(tx.SignerIndex), // #nosec G115 -- validated above: tx.SignerIndex < managerSet.N (uint8)
		InputSignatures:  tx.Signatures,
	}

	c.storeSignature(sig)
}

// isKnownEmitter checks if the given chain and emitter address match a known manager emitter.
func (c *ManagerService) isKnownEmitter(chain vaa.ChainID, addr vaa.Address) bool {
	for _, e := range c.emitters {
		if e.chainId == chain && bytes.Equal(e.addr, addr.Bytes()) {
			return true
		}
	}
	return false
}

// broadcastSignature broadcasts a signed manager transaction to the gossip network.
func (c *ManagerService) broadcastSignature(sig *ManagerSignature) {
	msg := &gossipv1.SignedManagerTransaction{
		VaaHash:          sig.VAAHash,
		VaaId:            sig.VAAID,
		DestinationChain: uint32(sig.DestinationChain),
		ManagerSetIndex:  sig.ManagerSetIndex,
		SignerIndex:      uint32(sig.SignerIndex),
		Signatures:       sig.InputSignatures,
	}

	envelope := &gossipv1.GossipMessage{
		Message: &gossipv1.GossipMessage_SignedManagerTransaction{
			SignedManagerTransaction: msg,
		},
	}

	b, err := proto.Marshal(envelope)
	if err != nil {
		c.logger.Error("failed to marshal signed manager transaction", zap.Error(err))
		return
	}

	select {
	case c.gossipSendC <- b:
		c.logger.Debug("broadcast signed manager transaction",
			zap.String("vaa_id", sig.VAAID),
			zap.Uint8("signer_index", sig.SignerIndex),
		)
	default:
		c.logger.Warn("gossip send channel full, dropping signed manager transaction",
			zap.String("vaa_id", sig.VAAID),
		)
	}
}

// storeSignature stores a signature from a manager signer for aggregation.
// Once M-of-N signatures are collected, the transaction can be broadcast.
func (c *ManagerService) storeSignature(sig *ManagerSignature) {
	hashHex := hex.EncodeToString(sig.VAAHash)

	c.pendingTxMu.Lock()
	defer c.pendingTxMu.Unlock()

	// Get or create the aggregated transaction entry
	aggTx, exists := c.pendingTx[hashHex]
	if !exists {
		// Look up the manager set to get M and N values
		chainSets, ok := c.managerSets[sig.DestinationChain]
		if !ok {
			c.logger.Error("no manager sets configured for chain",
				zap.Stringer("chain", sig.DestinationChain),
			)
			return
		}

		managerSet, ok := chainSets[sig.ManagerSetIndex]
		if !ok {
			c.logger.Error("manager set not found",
				zap.Uint32("index", sig.ManagerSetIndex),
				zap.Stringer("chain", sig.DestinationChain),
			)
			return
		}

		aggTx = &AggregatedTransaction{
			VAAHash:          sig.VAAHash,
			VAAID:            sig.VAAID,
			DestinationChain: sig.DestinationChain,
			ManagerSetIndex:  sig.ManagerSetIndex,
			Required:         managerSet.M,
			Total:            managerSet.N,
			Signatures:       make(map[uint8][][]byte),
		}
		c.pendingTx[hashHex] = aggTx
	}

	// Store the signatures from this signer
	if _, alreadyHave := aggTx.Signatures[sig.SignerIndex]; alreadyHave {
		c.logger.Debug("already have signature from signer",
			zap.String("vaa_id", sig.VAAID),
			zap.Uint8("signer_index", sig.SignerIndex),
		)
		return
	}

	aggTx.Signatures[sig.SignerIndex] = sig.InputSignatures

	c.logger.Info("stored manager signature",
		zap.String("vaa_id", sig.VAAID),
		zap.Uint8("signer_index", sig.SignerIndex),
		zap.Int("collected", len(aggTx.Signatures)),
		zap.Uint8("required", aggTx.Required),
	)

	// Check if we have enough signatures
	// #nosec G115 -- Signatures map is bounded by N (uint8)
	if uint8(len(aggTx.Signatures)) >= aggTx.Required {
		c.logger.Info("collected enough signatures for multisig",
			zap.String("vaa_id", aggTx.VAAID),
			zap.Int("collected", len(aggTx.Signatures)),
			zap.Uint8("required", aggTx.Required),
		)
		// TODO: Notify that transaction is ready for broadcast
	}
}

// signTransaction signs a UTXO unlock transaction using chain-specific logic.
// It computes the sighash for each input and signs with the guardian signer.
func (c *ManagerService) signTransaction(
	v *vaa.VAA,
	payload *vaa.UTXOUnlockPayload,
	signer guardiansigner.GuardianSigner,
) (*ManagerSignature, error) {
	switch payload.DestinationChain {
	case vaa.ChainIDDogecoin:
		return c.signDogecoinTransaction(v, payload, signer)
	default:
		return nil, fmt.Errorf("unsupported destination chain: %s", payload.DestinationChain)
	}
}

// signDogecoinTransaction signs a Dogecoin transaction for the given UTXO unlock payload.
// This constructs the transaction, computes the sighash for each input, and signs.
func (c *ManagerService) signDogecoinTransaction(
	v *vaa.VAA,
	payload *vaa.UTXOUnlockPayload,
	signer guardiansigner.GuardianSigner,
) (*ManagerSignature, error) {
	// Get the manager sets for Dogecoin
	chainSets, ok := c.managerSets[vaa.ChainIDDogecoin]
	if !ok {
		return nil, fmt.Errorf("no manager sets configured for Dogecoin")
	}

	// Look up the specific manager set by index from the payload
	managerSet, ok := chainSets[payload.DelegatedManagerSetIndex]
	if !ok {
		return nil, fmt.Errorf("manager set index %d not found for Dogecoin", payload.DelegatedManagerSetIndex)
	}

	// Verify this node is part of the manager set
	if !managerSet.IsSigner {
		return nil, fmt.Errorf("this node is not part of the Dogecoin manager set (index %d)", payload.DelegatedManagerSetIndex)
	}

	// Build redeem scripts for each input
	// Each input has its own recipient address (from when funds were locked)
	redeemScripts := make([][]byte, len(payload.Inputs))
	for i, input := range payload.Inputs {
		redeemScript, err := dogecoin.BuildRedeemScript(
			v.EmitterChain,
			v.EmitterAddress,
			input.OriginalRecipientAddress,
			managerSet.M,
			managerSet.PublicKeys,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to build redeem script for input %d: %w", i, err)
		}
		redeemScripts[i] = redeemScript
	}

	// For transaction building, we use the first input's redeem script
	// (they should all produce the same P2SH address if from the same manager address)
	unsignedTx, err := dogecoin.BuildUnsignedTransaction(payload, redeemScripts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to build unsigned transaction: %w", err)
	}

	// Override the redeem scripts with the per-input scripts
	unsignedTx.RedeemScripts = redeemScripts

	// Sign each input
	inputSignatures := make([][]byte, len(payload.Inputs))
	for i := range payload.Inputs {
		// Compute sighash for this input
		sighash, err := unsignedTx.ComputeSighash(i, txscript.SigHashAll)
		if err != nil {
			return nil, fmt.Errorf("failed to compute sighash for input %d: %w", i, err)
		}

		// Sign the sighash using the guardian signer
		// Note: GuardianSigner.Sign expects a hash and returns a signature
		sig, err := signer.Sign(c.ctx, sighash)
		if err != nil {
			return nil, fmt.Errorf("failed to sign input %d: %w", i, err)
		}

		// The guardian signer returns an Ethereum-style signature (r || s || v, 65 bytes)
		// We need to convert it to DER format for Bitcoin/Dogecoin
		derSig, err := convertEthSigToDER(sig, txscript.SigHashAll)
		if err != nil {
			return nil, fmt.Errorf("failed to convert signature for input %d: %w", i, err)
		}

		inputSignatures[i] = derSig
	}

	return &ManagerSignature{
		VAAHash:          v.SigningDigest().Bytes(),
		VAAID:            v.MessageID(),
		DestinationChain: payload.DestinationChain,
		ManagerSetIndex:  payload.DelegatedManagerSetIndex,
		SignerIndex:      uint8(managerSet.SignerIndex),
		InputSignatures:  inputSignatures,
	}, nil
}

// convertEthSigToDER converts an Ethereum-style signature (r || s || v, 65 bytes) to DER format.
// The sighash type byte is appended to the DER signature.
// This also performs low-S normalization required by Bitcoin/Dogecoin (BIP-62).
func convertEthSigToDER(ethSig []byte, hashType txscript.SigHashType) ([]byte, error) {
	if len(ethSig) != 65 {
		return nil, fmt.Errorf("invalid Ethereum signature length: expected 65, got %d", len(ethSig))
	}

	// Extract r and s from the Ethereum signature
	r := new(btcec.ModNScalar)
	if overflow := r.SetByteSlice(ethSig[0:32]); overflow {
		return nil, fmt.Errorf("r value overflows curve order")
	}

	s := new(btcec.ModNScalar)
	if overflow := s.SetByteSlice(ethSig[32:64]); overflow {
		return nil, fmt.Errorf("s value overflows curve order")
	}

	// Low-S normalization: if s > N/2, replace with N - s (BIP-62)
	// This is required for Bitcoin/Dogecoin transaction validity
	if s.IsOverHalfOrder() {
		s.Negate()
	}

	// Convert back to bytes for DER encoding
	var rBytes, sBytes [32]byte
	r.PutBytes(&rBytes)
	s.PutBytes(&sBytes)

	// Encode as DER with sighash type
	return dogecoin.EncodeDERSignature(rBytes[:], sBytes[:], hashType), nil
}

// compressPublicKey converts an ECDSA public key to compressed secp256k1 format (33 bytes).
func compressPublicKey(pubKey *ecdsa.PublicKey) []byte {
	btcPubKey, err := btcec.ParsePubKey(append([]byte{0x04}, append(pubKey.X.Bytes(), pubKey.Y.Bytes()...)...))
	if err != nil {
		// This should not happen with a valid ECDSA key
		return nil
	}
	return btcPubKey.SerializeCompressed()
}

// GetPendingTransactionByHash returns the aggregated transaction for a given VAA hash.
// Returns nil if no transaction exists for the hash.
func (c *ManagerService) GetPendingTransactionByHash(hashHex string) *AggregatedTransaction {
	c.pendingTxMu.RLock()
	defer c.pendingTxMu.RUnlock()
	return c.pendingTx[hashHex]
}

// GetPendingTransactionByID searches for an aggregated transaction by VAA ID.
// Returns nil if no transaction exists with the given ID.
// This is less efficient than GetPendingTransactionByHash as it requires iteration.
func (c *ManagerService) GetPendingTransactionByID(vaaID string) *AggregatedTransaction {
	c.pendingTxMu.RLock()
	defer c.pendingTxMu.RUnlock()

	for _, tx := range c.pendingTx {
		if tx.VAAID == vaaID {
			return tx
		}
	}
	return nil
}

// IsComplete returns true if this aggregated transaction has enough signatures.
func (a *AggregatedTransaction) IsComplete() bool {
	return uint8(len(a.Signatures)) >= a.Required // #nosec G115 -- Signatures map is bounded by N (uint8)
}

// GetFeatureString returns the feature flag string for heartbeat messages.
// Format: "manager:CHAIN_ID/COMPRESSED_PUBKEY_HEX" for single chain
// or "manager:CHAIN_ID1/PUBKEY1|CHAIN_ID2/PUBKEY2" for multiple chains.
func (c *ManagerService) GetFeatureString() string {
	if len(c.signerPubKeys) == 0 {
		return ""
	}

	// Collect chain IDs and sort them for consistent output
	chainIDs := make([]int, 0, len(c.signerPubKeys))
	for chainID := range c.signerPubKeys {
		chainIDs = append(chainIDs, int(chainID))
	}
	sort.Ints(chainIDs)

	// Build the feature string with chain ID and public key
	parts := make([]string, 0, len(chainIDs))
	for _, id := range chainIDs {
		pubKey := c.signerPubKeys[vaa.ChainID(id)] // #nosec G115 -- id was converted from ChainID (uint16) above
		parts = append(parts, fmt.Sprintf("%d/%s", id, hex.EncodeToString(pubKey)))
	}

	return "manager:" + strings.Join(parts, "|")
}
