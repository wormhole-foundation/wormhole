// Package dogecoin implements Dogecoin transaction building and signing for the manager service.
// Dogecoin uses the same transaction format as Bitcoin, so we leverage btcsuite libraries.
package dogecoin

import (
	"encoding/binary"
	"fmt"

	"github.com/btcsuite/btcd/txscript"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// BuildRedeemScript builds the redeem script as specified in the whitepaper.
// Format:
//
//	<emitter_chain>     (2 bytes, u16 BE)
//	<emitter_contract>  (32 bytes)
//	OP_2DROP
//	<recipient_address> (32 bytes)
//	OP_DROP
//	OP_M <pubkeys...> OP_N OP_CHECKMULTISIG
func BuildRedeemScript(
	emitterChain vaa.ChainID,
	emitterContract vaa.Address,
	recipientAddress [32]byte,
	m uint8,
	pubkeys [][]byte,
) ([]byte, error) {
	if len(pubkeys) > 15 {
		return nil, fmt.Errorf("too many pubkeys: %d (max 15 for standard multisig)", len(pubkeys))
	}
	n := uint8(len(pubkeys)) // #nosec G115 -- validated above: len(pubkeys) <= 15
	if m < 1 || m > n {
		return nil, fmt.Errorf("invalid m-of-n: m=%d, n=%d", m, n)
	}

	builder := txscript.NewScriptBuilder()

	// Push emitter chain (2 bytes, big-endian)
	chainBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(chainBytes, uint16(emitterChain))
	builder.AddData(chainBytes)

	// Push emitter contract (32 bytes)
	builder.AddData(emitterContract[:])

	// OP_2DROP - drops the top two stack items
	builder.AddOp(txscript.OP_2DROP)

	// Push recipient address (32 bytes)
	builder.AddData(recipientAddress[:])

	// OP_DROP - drops the top stack item
	builder.AddOp(txscript.OP_DROP)

	// OP_M (threshold)
	builder.AddInt64(int64(m))

	// Push each pubkey (must be 33 bytes compressed)
	for i, pk := range pubkeys {
		if len(pk) != 33 {
			return nil, fmt.Errorf("pubkey %d has invalid length %d (expected 33 for compressed)", i, len(pk))
		}
		builder.AddData(pk)
	}

	// OP_N (total pubkeys)
	builder.AddInt64(int64(n))

	// OP_CHECKMULTISIG
	builder.AddOp(txscript.OP_CHECKMULTISIG)

	return builder.Script()
}

// BuildP2PKHScriptPubKey builds a P2PKH scriptPubKey for a 20-byte pubkey hash.
// Format: OP_DUP OP_HASH160 <20-byte hash> OP_EQUALVERIFY OP_CHECKSIG
func BuildP2PKHScriptPubKey(pubkeyHash []byte) ([]byte, error) {
	if len(pubkeyHash) != 20 {
		return nil, fmt.Errorf("pubkey hash must be 20 bytes, got %d", len(pubkeyHash))
	}
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_DUP)
	builder.AddOp(txscript.OP_HASH160)
	builder.AddData(pubkeyHash)
	builder.AddOp(txscript.OP_EQUALVERIFY)
	builder.AddOp(txscript.OP_CHECKSIG)
	return builder.Script()
}

// BuildP2SHScriptPubKey builds a P2SH scriptPubKey for a 20-byte script hash.
// Format: OP_HASH160 <20-byte hash> OP_EQUAL
func BuildP2SHScriptPubKey(scriptHash []byte) ([]byte, error) {
	if len(scriptHash) != 20 {
		return nil, fmt.Errorf("script hash must be 20 bytes, got %d", len(scriptHash))
	}
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_HASH160)
	builder.AddData(scriptHash)
	builder.AddOp(txscript.OP_EQUAL)
	return builder.Script()
}

// BuildScriptPubKey builds a scriptPubKey for the given address type and address bytes.
func BuildScriptPubKey(addrType vaa.UTXOAddressType, address []byte) ([]byte, error) {
	switch addrType {
	case vaa.UTXOAddressTypeP2PKH:
		return BuildP2PKHScriptPubKey(address)
	case vaa.UTXOAddressTypeP2SH:
		return BuildP2SHScriptPubKey(address)
	default:
		return nil, fmt.Errorf("unsupported address type: %d", addrType)
	}
}
