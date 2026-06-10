// Package der contains shared DER signature encoding constants for manager chains.
package der

const (
	// SequenceTag is the ASN.1 SEQUENCE tag.
	SequenceTag = 0x30
	// IntegerTag is the ASN.1 INTEGER tag.
	IntegerTag = 0x02
	// FixedOverhead is the DER body overhead for two INTEGER values: integer
	// tag + length byte for r, plus integer tag + length byte for s.
	FixedOverhead = 4
	// SequenceHeaderLen is the SEQUENCE tag byte plus the total-length byte.
	SequenceHeaderLen = 2

	// Secp256k1OrderLen is the byte length of the secp256k1 curve order (256 bits).
	// Source: SEC 2 section 2.4.1, https://www.secg.org/sec2-v2.pdf
	Secp256k1OrderLen = 32
	// Secp256k1MaxIntEncodedLen is the maximum DER-encoded integer component length
	// for a secp256k1 signature: the curve order (32 bytes) plus one leading zero
	// byte that canonicalization may prepend when the high bit is set.
	Secp256k1MaxIntEncodedLen = Secp256k1OrderLen + 1
	// MaxTotalLen is the maximum DER SEQUENCE body length for a secp256k1
	// signature: fixed overhead (two tag+length pairs; 4 bytes) plus both
	// integer components at their max encoded length.
	MaxTotalLen = FixedOverhead + 2*Secp256k1MaxIntEncodedLen
)
