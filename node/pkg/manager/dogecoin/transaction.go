// Package dogecoin implements Dogecoin transaction building and signing for the manager service.
package dogecoin

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// SighashAll is the sighash type for signing all inputs and outputs.
const SighashAll = txscript.SigHashAll

// UnsignedTransaction represents an unsigned Dogecoin transaction with metadata
// needed for signing.
type UnsignedTransaction struct {
	// Tx is the unsigned wire transaction.
	Tx *wire.MsgTx
	// RedeemScripts contains the redeem script for each input, indexed by input index.
	RedeemScripts [][]byte
}

// BuildUnsignedTransaction constructs an unsigned Dogecoin transaction from a UTXO unlock payload.
// The redeemScript is used for P2SH inputs and must be provided for sighash computation.
func BuildUnsignedTransaction(
	payload *vaa.UTXOUnlockPayload,
	redeemScript []byte,
) (*UnsignedTransaction, error) {
	if len(payload.Inputs) == 0 {
		return nil, fmt.Errorf("no inputs in payload")
	}
	if len(payload.Outputs) == 0 {
		return nil, fmt.Errorf("no outputs in payload")
	}

	tx := wire.NewMsgTx(wire.TxVersion)

	// Add inputs
	for _, input := range payload.Inputs {
		// Convert TransactionID to chainhash (reverse byte order for Bitcoin/Dogecoin)
		var txHash chainhash.Hash
		for i := 0; i < 32; i++ {
			txHash[i] = input.TransactionID[31-i]
		}

		outPoint := wire.NewOutPoint(&txHash, input.Vout)
		txIn := wire.NewTxIn(outPoint, nil, nil)
		tx.AddTxIn(txIn)
	}

	// Add outputs
	for i, output := range payload.Outputs {
		if output.Amount > math.MaxInt64 {
			return nil, fmt.Errorf("output %d amount %d exceeds max int64", i, output.Amount)
		}
		scriptPubKey, err := BuildScriptPubKey(output.AddressType, output.Address)
		if err != nil {
			return nil, fmt.Errorf("failed to build scriptPubKey for output %d: %w", i, err)
		}
		txOut := wire.NewTxOut(int64(output.Amount), scriptPubKey) // #nosec G115 -- validated above: output.Amount <= math.MaxInt64
		tx.AddTxOut(txOut)
	}

	// Store the redeem script for each input (same redeem script for all inputs from manager address)
	redeemScripts := make([][]byte, len(payload.Inputs))
	for i := range redeemScripts {
		redeemScripts[i] = redeemScript
	}

	return &UnsignedTransaction{
		Tx:            tx,
		RedeemScripts: redeemScripts,
	}, nil
}

// ComputeSighash computes the sighash for a specific input using the legacy sighash algorithm.
// This is used for P2SH multisig signing in Dogecoin.
func (ut *UnsignedTransaction) ComputeSighash(inputIndex int, hashType txscript.SigHashType) ([]byte, error) {
	if inputIndex < 0 || inputIndex >= len(ut.Tx.TxIn) {
		return nil, fmt.Errorf("input index %d out of range [0, %d)", inputIndex, len(ut.Tx.TxIn))
	}
	if inputIndex >= len(ut.RedeemScripts) {
		return nil, fmt.Errorf("no redeem script for input %d", inputIndex)
	}

	redeemScript := ut.RedeemScripts[inputIndex]

	// For legacy P2SH, we use the standard sighash computation
	// This creates a modified copy of the transaction for signing
	hash, err := txscript.CalcSignatureHash(redeemScript, hashType, ut.Tx, inputIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to compute sighash: %w", err)
	}

	return hash, nil
}

// ComputeAllSighashes computes the sighash for all inputs.
func (ut *UnsignedTransaction) ComputeAllSighashes(hashType txscript.SigHashType) ([][]byte, error) {
	sighashes := make([][]byte, len(ut.Tx.TxIn))
	for i := range ut.Tx.TxIn {
		hash, err := ut.ComputeSighash(i, hashType)
		if err != nil {
			return nil, fmt.Errorf("failed to compute sighash for input %d: %w", i, err)
		}
		sighashes[i] = hash
	}
	return sighashes, nil
}

// SerializeForBroadcast serializes the transaction for network broadcast.
// Note: This should only be called after all signatures have been applied.
func (ut *UnsignedTransaction) SerializeForBroadcast() ([]byte, error) {
	var buf bytes.Buffer
	if err := ut.Tx.Serialize(&buf); err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}
	return buf.Bytes(), nil
}

// TxHash returns the transaction hash (txid) in the standard Bitcoin/Dogecoin format.
func (ut *UnsignedTransaction) TxHash() chainhash.Hash {
	return ut.Tx.TxHash()
}

// InputCount returns the number of inputs in the transaction.
func (ut *UnsignedTransaction) InputCount() int {
	return len(ut.Tx.TxIn)
}

// OutputCount returns the number of outputs in the transaction.
func (ut *UnsignedTransaction) OutputCount() int {
	return len(ut.Tx.TxOut)
}

// DoubleSha256 computes the double SHA256 hash used in Bitcoin/Dogecoin.
func DoubleSha256(data []byte) []byte {
	first := sha256.Sum256(data)
	second := sha256.Sum256(first[:])
	return second[:]
}

// Hash160 computes RIPEMD160(SHA256(data)), used for address generation.
func Hash160(data []byte) []byte {
	return btcutil.Hash160(data)
}

// BuildP2SHAddress computes the P2SH address (script hash) for a redeem script.
// Returns the 20-byte script hash.
func BuildP2SHAddress(redeemScript []byte) []byte {
	return Hash160(redeemScript)
}

// ApplySignatureToInput applies a signature to a specific input for P2SH multisig.
// This builds the scriptSig: OP_0 <sig1> <sig2> ... <redeemScript>
func ApplySignatureToInput(
	tx *wire.MsgTx,
	inputIndex int,
	signatures [][]byte,
	redeemScript []byte,
) error {
	if inputIndex < 0 || inputIndex >= len(tx.TxIn) {
		return fmt.Errorf("input index %d out of range", inputIndex)
	}

	// Build the scriptSig for P2SH multisig
	// Format: OP_0 <sig1> <sig2> ... <sigM> <redeemScript>
	builder := txscript.NewScriptBuilder()

	// OP_0 for the off-by-one bug in CHECKMULTISIG
	builder.AddOp(txscript.OP_0)

	// Add each signature
	for _, sig := range signatures {
		builder.AddData(sig)
	}

	// Add the redeem script
	builder.AddData(redeemScript)

	scriptSig, err := builder.Script()
	if err != nil {
		return fmt.Errorf("failed to build scriptSig: %w", err)
	}

	tx.TxIn[inputIndex].SignatureScript = scriptSig
	return nil
}

// EncodeDERSignature ensures a signature is properly DER-encoded with sighash type appended.
// The input should be the raw ECDSA signature (r, s) values.
func EncodeDERSignature(r, s []byte, hashType txscript.SigHashType) []byte {
	// DER format: 0x30 [total-length] 0x02 [r-length] [r] 0x02 [s-length] [s] [sighash]

	// Remove leading zeros from r and s, but ensure they don't look negative
	r = canonicalizeInt(r)
	s = canonicalizeInt(s)

	// Calculate lengths
	rLen := len(r)
	sLen := len(s)
	totalLen := 4 + rLen + sLen // 0x02 + rLen + r + 0x02 + sLen + s

	// Build DER signature
	sig := make([]byte, 0, totalLen+3) // +3 for 0x30, totalLen, and hashType
	sig = append(sig, 0x30)            // DER sequence tag
	sig = append(sig, byte(totalLen))  // Total length
	sig = append(sig, 0x02)            // Integer tag for r
	sig = append(sig, byte(rLen))      // r length
	sig = append(sig, r...)            // r value
	sig = append(sig, 0x02)            // Integer tag for s
	sig = append(sig, byte(sLen))      // s length
	sig = append(sig, s...)            // s value
	sig = append(sig, byte(hashType))  // Sighash type

	return sig
}

// canonicalizeInt removes leading zeros and adds a zero byte if the high bit is set.
func canonicalizeInt(b []byte) []byte {
	// Remove leading zeros
	for len(b) > 1 && b[0] == 0 && b[1]&0x80 == 0 {
		b = b[1:]
	}
	// Add zero byte if high bit is set (to prevent negative interpretation)
	if len(b) > 0 && b[0]&0x80 != 0 {
		b = append([]byte{0}, b...)
	}
	return b
}

// VoutToBytes converts a vout index to little-endian bytes.
func VoutToBytes(vout uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, vout)
	return b
}
