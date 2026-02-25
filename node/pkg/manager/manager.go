// Package manager implements the Manager Service for the guardian node.
// The Manager Service subscribes to incoming VAAs and processes them
// according to manager requirements.
package manager

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/txscript"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	"github.com/certusone/wormhole/node/pkg/manager/dogecoin"
	"github.com/certusone/wormhole/node/pkg/manager/xrpl"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
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
	// managerTxSendC is the channel for sending manager transactions to be signed and broadcast via gossip.
	managerTxSendC chan<- *gossipv1.ManagerTransaction
	// managerTxRecvC receives verified manager transactions from other manager nodes via gossip.
	managerTxRecvC <-chan *gossipv1.ManagerTransaction
	// db is the database for persistent storage of aggregated transactions.
	db *db.ManagerDB
	// pendingTxMu protects access to database operations for aggregated transactions.
	pendingTxMu sync.RWMutex
	// xrplSequencer is the known XRPL sequencer emitter address.
	xrplSequencer *emitterEntry
	// reader is used to dynamically fetch manager sets from the DelegatedManagerSet contract.
	reader *ManagerSetReader
}

// NewManagerService creates a new ManagerService instance.
// The delegatedManagerSetRPC parameter is the Ethereum RPC URL (ethRPC) for fetching manager sets
// from the DelegatedManagerSet contract. It can be empty for DevNet.
func NewManagerService(
	ctx context.Context,
	logger *zap.Logger,
	vaaC <-chan *vaa.VAA,
	env common.Environment,
	signers map[vaa.ChainID]guardiansigner.GuardianSigner,
	managerTxSendC chan<- *gossipv1.ManagerTransaction,
	managerTxRecvC <-chan *gossipv1.ManagerTransaction,
	database *db.Database,
	delegatedManagerSetRPC string,
) (*ManagerService, error) {
	// Select the appropriate emitter and sequencer lists based on environment
	var emitters []emitterEntry
	var xrplSequencer *emitterEntry
	//nolint:exhaustive // GoTest, and AccountantMock intentionally fall through to default
	switch env {
	case common.UnsafeDevNet:
		emitters = parseEmitters(sdk.KnownDevnetManagerEmitters)
		xrplSequencer = parseSequencer(sdk.KnownDevnetXRPLSequencer)
	case common.TestNet:
		emitters = parseEmitters(sdk.KnownTestnetManagerEmitters)
		xrplSequencer = parseSequencer(sdk.KnownTestnetXRPLSequencer)
	case common.MainNet:
		emitters = parseEmitters(sdk.KnownManagerEmitters)
		xrplSequencer = parseSequencer(sdk.KnownXRPLSequencer)
	default:
		emitters = []emitterEntry{}
	}

	// Compute compressed public keys for each signer
	signerPubKeys := make(map[vaa.ChainID][]byte)
	for chainID, signer := range signers {
		pubKey := signer.PublicKey(ctx)
		signerPubKeys[chainID] = compressPublicKey(&pubKey)
	}

	// Create reader for dynamic manager set loading
	if delegatedManagerSetRPC == "" {
		return nil, fmt.Errorf("delegatedManagerSetRPC is required")
	}
	reader, err := NewManagerSetReader(logger, env, delegatedManagerSetRPC)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager set reader: %w", err)
	}

	return &ManagerService{
		ctx:            ctx,
		logger:         logger.With(zap.String("component", "manager")),
		vaaC:           vaaC,
		env:            env,
		emitters:       emitters,
		xrplSequencer:  xrplSequencer,
		signers:        signers,
		signerPubKeys:  signerPubKeys,
		managerTxSendC: managerTxSendC,
		managerTxRecvC: managerTxRecvC,
		db:             db.NewManagerDB(database.Conn()),
		reader:         reader,
	}, nil
}

// parseSequencer converts a single SDK sequencer entry to an internal emitterEntry.
// Returns nil if the entry has an empty address (e.g. mainnet before configuration).
func parseSequencer(sdkEntry struct {
	ChainId vaa.ChainID
	Addr    string
}) *emitterEntry {
	if sdkEntry.Addr == "" {
		return nil
	}
	addr, err := hex.DecodeString(sdkEntry.Addr)
	if err != nil {
		panic("invalid sequencer address: " + sdkEntry.Addr)
	}
	return &emitterEntry{
		chainId: sdkEntry.ChainId,
		addr:    addr,
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

// getManagerSet retrieves a manager set by chain ID and index.
func (c *ManagerService) getManagerSet(ctx context.Context, chainID vaa.ChainID, index uint32) (*ManagerSetConfig, error) {
	signer := c.signers[chainID]
	return c.reader.GetManagerSet(ctx, chainID, index, signer)
}

// Run starts the manager service and begins processing incoming VAAs.
func (c *ManagerService) Run(ctx context.Context) error {
	c.logger.Info("manager service enabled",
		zap.String("environment", string(c.env)),
		zap.Int("known_emitters", len(c.emitters)),
		zap.Bool("xrpl_sequencer", c.xrplSequencer != nil),
		zap.Int("signers", len(c.signers)),
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case v := <-c.vaaC:
			c.handleVAA(v)
		case tx := <-c.managerTxRecvC:
			c.handleIncomingTransaction(tx)
		}
	}
}

// handleVAA processes an incoming signed VAA.
func (c *ManagerService) handleVAA(v *vaa.VAA) {
	// SECURITY: this channel should only be pushed to by a process that has verified the signatures on the VAA to belong to the current guardian set

	// SECURITY: Validate that this VAA is from an authorized emitter for its payload type.
	if !c.validateEmitter(v) {
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

	// Detect payload type by 4-byte prefix and dispatch to chain-specific handler
	prefix := vaa.GetPayloadPrefix(v.Payload)
	switch prefix {
	case vaa.UTXOPayloadPrefix:
		c.handleUTXOPayload(v)
	case vaa.XRPLPayloadPrefix:
		c.handleXRPLPayload(v)
	case vaa.XRPLTicketRefillPrefix:
		c.handleXRPLTicketRefill(v)
	default:
		c.logger.Warn("unknown payload prefix",
			zap.String("message_id", v.MessageID()),
			zap.String("prefix", string(prefix[:])),
		)
	}
}

// handleIncomingTransaction processes a verified manager transaction received from another manager node.
// The p2p layer has already verified the guardian signature before passing this to us.
//
// SECURITY: this method aggregates manager signatures without verification, allowing for:
//
// 1. Invalid signatures stored permanently
//
// 2. Signer index spoofing (claim another signer's slot)
//
// 3. Metadata poisoning (first gossip message sets threshold, etc.)
//
// A suggested mitigation would be to make the following changes to this component without affecting gossip.
//
// # Approach
//
// Add SigHashes and PendingSignatures fields to AggregatedTransaction in the DB.
//
// - SigHashes [][]byte - Set when the VAA is processed locally. These are the hashes that are signed by the signer.
//
// - PendingSignatures map[uint8][][]byte - Unverified signatures stored before sighashes are known.
//
// - Signatures map[uint8][][]byte - Verified signatures (existing field, now only contains verified sigs).
//
// # Flow
//
// In handleIncomingTransaction, if hashes exist, verify the signatures before storing them, otherwise store them in PendingSignatures.
//
// In handleVAA, store the computed hashes and verify any pending signatures, dropping arrays that contain any invalid ones.
// Only handleVAA should populate the metadata.
//
// This logic can likely be handled internally in storeSignature.
func (c *ManagerService) handleIncomingTransaction(tx *gossipv1.ManagerTransaction) {
	c.logger.Debug("received signed manager transaction from peer",
		zap.String("vaa_id", tx.VaaId),
		zap.Uint32("destination_chain", tx.DestinationChain),
		zap.Uint32("signer_index", tx.SignerIndex),
		zap.Int("num_signatures", len(tx.Signatures)),
	)

	destChain := vaa.ChainID(tx.DestinationChain) // #nosec G115 -- ChainID is uint16, protobuf uses uint32 for wire compatibility

	// Validate the signer is in the manager set (fetch dynamically if needed)
	managerSet, err := c.getManagerSet(c.ctx, destChain, tx.ManagerSetIndex)
	if err != nil {
		c.logger.Warn("failed to get manager set for incoming transaction",
			zap.Stringer("chain", destChain),
			zap.Uint32("index", tx.ManagerSetIndex),
			zap.Error(err),
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

// validateEmitter checks if the VAA is from an authorized emitter for its payload type.
//
// For XRPL payloads, it checks against the known sequencer addresses.
//
// SECURITY: This is critical for XRPL as the sequencer contract controls all accounts!
//
// For UTXO payloads (e.g. Dogecoin), it checks against the known manager emitters list.
//
// SECURITY: This is a defense-in-depth / DoS prevention check for UTX0 done to intentionally limit
// the scope of unknown emitters incidentally triggering this signing logic.
// In the future, this could be moved to a permissionless on-chain registration.
// Note that for the `UTX0` case, redeem scripts are tied to a particular emitter.
func (c *ManagerService) validateEmitter(v *vaa.VAA) bool {
	prefix := vaa.GetPayloadPrefix(v.Payload)
	switch prefix {
	case vaa.XRPLPayloadPrefix:
		return c.isXRPLSequencer(v.EmitterChain, v.EmitterAddress)
	case vaa.XRPLTicketRefillPrefix:
		return c.isXRPLSequencer(v.EmitterChain, v.EmitterAddress)
	case vaa.UTXOPayloadPrefix:
		return c.isKnownEmitter(v.EmitterChain, v.EmitterAddress)
	default:
		return false
	}
}

// isXRPLSequencer checks if the given chain and address match the known XRPL sequencer.
func (c *ManagerService) isXRPLSequencer(chain vaa.ChainID, addr vaa.Address) bool {
	if c.xrplSequencer == nil {
		return false
	}
	return c.xrplSequencer.chainId == chain && bytes.Equal(c.xrplSequencer.addr, addr.Bytes())
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

// broadcastSignature broadcasts a manager transaction to the gossip network.
// The p2p layer will sign the message with the guardian key before publishing.
func (c *ManagerService) broadcastSignature(sig *ManagerSignature) {
	msg := &gossipv1.ManagerTransaction{
		VaaHash:          sig.VAAHash,
		VaaId:            sig.VAAID,
		DestinationChain: uint32(sig.DestinationChain),
		ManagerSetIndex:  sig.ManagerSetIndex,
		SignerIndex:      uint32(sig.SignerIndex),
		Signatures:       sig.InputSignatures,
		SentTimestamp:    time.Now().Unix(),
	}

	select {
	case c.managerTxSendC <- msg:
		c.logger.Debug("broadcast manager transaction",
			zap.String("vaa_id", sig.VAAID),
			zap.Uint8("signer_index", sig.SignerIndex),
		)
	default:
		c.logger.Warn("gossip send channel full, dropping manager transaction",
			zap.String("vaa_id", sig.VAAID),
		)
	}
}

// storeSignature stores a signature from a manager signer for aggregation.
// Once M-of-N signatures are collected, the transaction can be broadcast.
// Signatures are persisted to BadgerDB for durability across restarts.
func (c *ManagerService) storeSignature(sig *ManagerSignature) {
	hashHex := hex.EncodeToString(sig.VAAHash)

	c.pendingTxMu.Lock()
	defer c.pendingTxMu.Unlock()

	// Try to get existing aggregated transaction from database
	aggTx, err := c.db.GetAggregatedTransaction(hashHex)
	if err != nil && !errors.Is(err, db.ErrManagerSigNotFound) {
		c.logger.Error("failed to get aggregated transaction from database",
			zap.String("vaa_hash", hashHex),
			zap.Error(err),
		)
		return
	}

	if aggTx == nil {
		// Look up the manager set to get M and N values (fetch dynamically if needed)
		managerSet, err := c.getManagerSet(c.ctx, sig.DestinationChain, sig.ManagerSetIndex)
		if err != nil {
			c.logger.Error("failed to get manager set for signature storage",
				zap.Stringer("chain", sig.DestinationChain),
				zap.Uint32("index", sig.ManagerSetIndex),
				zap.Error(err),
			)
			return
		}

		aggTx = &db.AggregatedTransaction{
			VAAHash:          sig.VAAHash,
			VAAID:            sig.VAAID,
			DestinationChain: sig.DestinationChain,
			ManagerSetIndex:  sig.ManagerSetIndex,
			Required:         managerSet.M,
			Total:            managerSet.N,
			Signatures:       make(map[uint8][][]byte),
		}
	}

	// Check if we already have this signature
	if _, alreadyHave := aggTx.Signatures[sig.SignerIndex]; alreadyHave {
		c.logger.Debug("already have signature from signer",
			zap.String("vaa_id", sig.VAAID),
			zap.Uint8("signer_index", sig.SignerIndex),
		)
		return
	}

	// Add the new signature
	aggTx.Signatures[sig.SignerIndex] = sig.InputSignatures

	// Persist to database
	if err := c.db.StoreAggregatedTransaction(hashHex, aggTx); err != nil {
		c.logger.Error("failed to store aggregated transaction to database",
			zap.String("vaa_hash", hashHex),
			zap.Error(err),
		)
		return
	}

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
	}
}

// handleUTXOPayload processes a UTXO unlock payload from a VAA.
func (c *ManagerService) handleUTXOPayload(v *vaa.VAA) {
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
	sig, err := c.signUTXOTransaction(v, payload, signer)
	if err != nil {
		c.logger.Error("failed to sign UTXO transaction",
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

	if c.managerTxSendC != nil {
		c.broadcastSignature(sig)
	}
	c.storeSignature(sig)
}

// handleXRPLPayload processes an XRPL release payload from a VAA.
func (c *ManagerService) handleXRPLPayload(v *vaa.VAA) {
	payload, err := vaa.DeserializeXRPLReleasePayload(v.Payload)
	if err != nil {
		c.logger.Error("failed to parse XRPL release payload",
			zap.String("message_id", v.MessageID()),
			zap.Error(err),
		)
		return
	}

	c.logger.Info("parsed XRPL release payload",
		zap.String("message_id", v.MessageID()),
		zap.Uint64("ticket_id", payload.TicketID),
		zap.Uint64("amount", payload.Amount),
		zap.Uint8("token_type", uint8(payload.Token.Type)),
	)

	// Check if we have an XRPL signer
	signer, ok := c.signers[vaa.ChainIDXRPL]
	if !ok {
		c.logger.Warn("no signer configured for XRPL",
			zap.String("message_id", v.MessageID()),
		)
		return
	}

	// Sign the XRPL transaction
	sig, err := c.signXRPLTransaction(v, payload, signer)
	if err != nil {
		c.logger.Error("failed to sign XRPL transaction",
			zap.String("message_id", v.MessageID()),
			zap.Error(err),
		)
		return
	}

	c.logger.Info("signed XRPL manager transaction",
		zap.String("message_id", v.MessageID()),
		zap.Stringer("destination_chain", sig.DestinationChain),
	)

	if c.managerTxSendC != nil {
		c.broadcastSignature(sig)
	}
	c.storeSignature(sig)
}

// handleXRPLTicketRefill processes an XRPL ticket refill payload from a VAA.
func (c *ManagerService) handleXRPLTicketRefill(v *vaa.VAA) {
	payload, err := vaa.DeserializeXRPLTicketRefillPayload(v.Payload)
	if err != nil {
		c.logger.Error("failed to parse XRPL ticket refill payload",
			zap.String("message_id", v.MessageID()),
			zap.Error(err),
		)
		return
	}

	c.logger.Info("parsed XRPL ticket refill payload",
		zap.String("message_id", v.MessageID()),
		zap.Uint64("use_ticket", payload.UseTicket),
		zap.Uint64("request_count", payload.RequestCount),
	)

	// Check if we have an XRPL signer
	signer, ok := c.signers[vaa.ChainIDXRPL]
	if !ok {
		c.logger.Warn("no signer configured for XRPL",
			zap.String("message_id", v.MessageID()),
		)
		return
	}

	// Sign the XRPL TicketCreate transaction
	sig, err := c.signXRPLTicketRefillTransaction(v, payload, signer)
	if err != nil {
		c.logger.Error("failed to sign XRPL ticket refill transaction",
			zap.String("message_id", v.MessageID()),
			zap.Error(err),
		)
		return
	}

	c.logger.Info("signed XRPL ticket refill transaction",
		zap.String("message_id", v.MessageID()),
		zap.Stringer("destination_chain", sig.DestinationChain),
	)

	if c.managerTxSendC != nil {
		c.broadcastSignature(sig)
	}
	c.storeSignature(sig)
}

// signXRPLTicketRefillTransaction signs an XRPL TicketCreate transaction for the given ticket refill payload.
func (c *ManagerService) signXRPLTicketRefillTransaction(
	v *vaa.VAA,
	payload *vaa.XRPLTicketRefillPayload,
	signer guardiansigner.GuardianSigner,
) (*ManagerSignature, error) {
	// Get the current manager set for XRPL (XRFL payload doesn't embed a manager set index)
	managerSet, err := c.getCurrentManagerSet(c.ctx, vaa.ChainIDXRPL)
	if err != nil {
		return nil, fmt.Errorf("failed to get current manager set for XRPL: %w", err)
	}

	// Verify this node is part of the manager set
	if !managerSet.IsSigner {
		return nil, fmt.Errorf("this node is not part of the XRPL manager set")
	}

	// Build the XRPL TicketCreate transaction
	flatTx, err := xrpl.BuildTicketCreateTransaction(payload, managerSet.M)
	if err != nil {
		return nil, fmt.Errorf("failed to build XRPL ticket create transaction: %w", err)
	}

	// Derive this signer's XRPL address from compressed public key
	signerPubKey := c.signerPubKeys[vaa.ChainIDXRPL]
	signerAddress, err := xrpl.CompressedPubKeyToAddress(signerPubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive signer XRPL address: %w", err)
	}

	// Compute the multisign hash for this signer
	hash, err := xrpl.ComputeMultisignHash(flatTx, signerAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to compute multisign hash: %w", err)
	}

	// Sign the hash using the guardian signer
	ethSig, err := signer.Sign(c.ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign XRPL ticket refill transaction: %w", err)
	}

	// Convert Ethereum-style signature to XRPL DER format (without sighash type byte)
	derSig, err := convertEthSigToXRPLDER(ethSig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signature to XRPL DER: %w", err)
	}

	return &ManagerSignature{
		VAAHash:          v.SigningDigest().Bytes(),
		VAAID:            v.MessageID(),
		DestinationChain: vaa.ChainIDXRPL,
		ManagerSetIndex:  managerSet.Index,
		SignerIndex:      managerSet.SignerIndex,
		InputSignatures:  [][]byte{derSig},
	}, nil
}

// signUTXOTransaction signs a UTXO unlock transaction using chain-specific logic.
// It computes the sighash for each input and signs with the guardian signer.
func (c *ManagerService) signUTXOTransaction(
	v *vaa.VAA,
	payload *vaa.UTXOUnlockPayload,
	signer guardiansigner.GuardianSigner,
) (*ManagerSignature, error) {
	switch payload.DestinationChain {
	case vaa.ChainIDDogecoin:
		return c.signDogecoinTransaction(v, payload, signer)
	default:
		return nil, fmt.Errorf("unsupported UTXO destination chain: %s", payload.DestinationChain)
	}
}

// signDogecoinTransaction signs a Dogecoin transaction for the given UTXO unlock payload.
// This constructs the transaction, computes the sighash for each input, and signs.
func (c *ManagerService) signDogecoinTransaction(
	v *vaa.VAA,
	payload *vaa.UTXOUnlockPayload,
	signer guardiansigner.GuardianSigner,
) (*ManagerSignature, error) {
	// Look up the specific manager set by index from the payload (fetch dynamically if needed)
	managerSet, err := c.getManagerSet(c.ctx, vaa.ChainIDDogecoin, payload.DelegatedManagerSetIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to get manager set for Dogecoin: %w", err)
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

	if len(redeemScripts) < 1 {
		return nil, fmt.Errorf("invalid redeemScripts length, must have at least 1")
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

// normalizeEthSig extracts r and s from a 65-byte Ethereum-style signature (r || s || v)
// and performs low-S normalization (if s > N/2, replace with N - s).
// Low-S is required by both Bitcoin/Dogecoin (BIP-62) and XRPL canonical signatures.
func normalizeEthSig(ethSig []byte) (rBytes, sBytes []byte, err error) {
	if len(ethSig) != 65 {
		return nil, nil, fmt.Errorf("invalid Ethereum signature length: expected 65, got %d", len(ethSig))
	}

	r := new(btcec.ModNScalar)
	if overflow := r.SetByteSlice(ethSig[0:32]); overflow {
		return nil, nil, fmt.Errorf("r value overflows curve order")
	}

	s := new(btcec.ModNScalar)
	if overflow := s.SetByteSlice(ethSig[32:64]); overflow {
		return nil, nil, fmt.Errorf("s value overflows curve order")
	}

	if s.IsOverHalfOrder() {
		s.Negate()
	}

	var rBuf, sBuf [32]byte
	r.PutBytes(&rBuf)
	s.PutBytes(&sBuf)
	return rBuf[:], sBuf[:], nil
}

// convertEthSigToDER converts an Ethereum-style signature to DER format
// with an appended sighash type byte for Bitcoin/Dogecoin.
func convertEthSigToDER(ethSig []byte, hashType txscript.SigHashType) ([]byte, error) {
	r, s, err := normalizeEthSig(ethSig)
	if err != nil {
		return nil, err
	}
	return dogecoin.EncodeDERSignature(r, s, hashType), nil
}

// signXRPLTransaction signs an XRPL Payment transaction for the given XRPL release payload.
func (c *ManagerService) signXRPLTransaction(
	v *vaa.VAA,
	payload *vaa.XRPLReleasePayload,
	signer guardiansigner.GuardianSigner,
) (*ManagerSignature, error) {
	// Get the current manager set for XRPL (XREL payload doesn't embed a manager set index)
	managerSet, err := c.getCurrentManagerSet(c.ctx, vaa.ChainIDXRPL)
	if err != nil {
		return nil, fmt.Errorf("failed to get current manager set for XRPL: %w", err)
	}

	// Verify this node is part of the manager set
	if !managerSet.IsSigner {
		return nil, fmt.Errorf("this node is not part of the XRPL manager set")
	}

	// Build the XRPL Payment transaction
	flatTx, err := xrpl.BuildPaymentTransaction(payload, managerSet.M)
	if err != nil {
		return nil, fmt.Errorf("failed to build XRPL payment transaction: %w", err)
	}

	// Derive this signer's XRPL address from compressed public key
	signerPubKey := c.signerPubKeys[vaa.ChainIDXRPL]
	signerAddress, err := xrpl.CompressedPubKeyToAddress(signerPubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive signer XRPL address: %w", err)
	}

	// Compute the multisign hash for this signer
	hash, err := xrpl.ComputeMultisignHash(flatTx, signerAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to compute multisign hash: %w", err)
	}

	// Sign the hash using the guardian signer
	ethSig, err := signer.Sign(c.ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign XRPL transaction: %w", err)
	}

	// Convert Ethereum-style signature to XRPL DER format (without sighash type byte)
	derSig, err := convertEthSigToXRPLDER(ethSig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signature to XRPL DER: %w", err)
	}

	return &ManagerSignature{
		VAAHash:          v.SigningDigest().Bytes(),
		VAAID:            v.MessageID(),
		DestinationChain: vaa.ChainIDXRPL,
		ManagerSetIndex:  managerSet.Index,
		SignerIndex:      managerSet.SignerIndex,
		InputSignatures:  [][]byte{derSig},
	}, nil
}

// convertEthSigToXRPLDER converts an Ethereum-style signature to DER format
// without a sighash type byte for XRPL.
func convertEthSigToXRPLDER(ethSig []byte) ([]byte, error) {
	r, s, err := normalizeEthSig(ethSig)
	if err != nil {
		return nil, err
	}
	return xrpl.EncodeDERSignature(r, s), nil
}

// getCurrentManagerSet retrieves the current manager set for a chain by first looking up
// the current index from the contract.
func (c *ManagerService) getCurrentManagerSet(ctx context.Context, chainID vaa.ChainID) (*ManagerSetConfig, error) {
	signer := c.signers[chainID]
	return c.reader.GetCurrentManagerSet(ctx, chainID, signer)
}

// compressPublicKey converts an ECDSA public key to compressed secp256k1 format (33 bytes).
func compressPublicKey(pubKey *ecdsa.PublicKey) []byte {
	// Use FillBytes to zero-pad coordinates to exactly 32 bytes each.
	// big.Int.Bytes() strips leading zeros, which would produce a short
	// buffer that btcec.ParsePubKey rejects.
	var buf [65]byte
	buf[0] = 0x04
	pubKey.X.FillBytes(buf[1:33])
	pubKey.Y.FillBytes(buf[33:65])
	btcPubKey, err := btcec.ParsePubKey(buf[:])
	if err != nil {
		// This should not happen with a valid ECDSA key
		return nil
	}
	return btcPubKey.SerializeCompressed()
}

// GetPendingTransactionByHash returns the aggregated transaction for a given VAA hash.
// Returns nil if no transaction exists for the hash.
func (c *ManagerService) GetPendingTransactionByHash(hashHex string) *db.AggregatedTransaction {
	c.pendingTxMu.RLock()
	defer c.pendingTxMu.RUnlock()

	tx, err := c.db.GetAggregatedTransaction(hashHex)
	if err != nil {
		if !errors.Is(err, db.ErrManagerSigNotFound) {
			c.logger.Error("failed to get aggregated transaction from database",
				zap.String("vaa_hash", hashHex),
				zap.Error(err),
			)
		}
		return nil
	}
	return tx
}

// GetPendingTransactionByID retrieves an aggregated transaction by VAA ID.
// Returns nil if no transaction exists with the given ID.
// This uses an index for O(1) lookup.
func (c *ManagerService) GetPendingTransactionByID(vaaID string) *db.AggregatedTransaction {
	c.pendingTxMu.RLock()
	defer c.pendingTxMu.RUnlock()

	tx, err := c.db.GetAggregatedTransactionByVAAID(vaaID)
	if err != nil {
		if !errors.Is(err, db.ErrManagerSigNotFound) {
			c.logger.Error("failed to get aggregated transaction by VAA ID from database",
				zap.String("vaa_id", vaaID),
				zap.Error(err),
			)
		}
		return nil
	}
	return tx
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
