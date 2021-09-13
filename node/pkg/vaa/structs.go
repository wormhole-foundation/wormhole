package vaa

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type (
	// VAA is a verifiable action approval of the Wormhole protocol
	VAA struct {
		// Version of the VAA schema
		Version uint8
		// GuardianSetIndex is the index of the guardian set that signed this VAA
		GuardianSetIndex uint32
		// SignatureData is the signature of the guardian set
		Signatures []*Signature

		// Timestamp when the VAA was created
		Timestamp time.Time
		// Nonce of the VAA
		Nonce uint32
		// Sequence of the VAA
		Sequence uint64
		/// ConsistencyLevel of the VAA
		ConsistencyLevel uint8
		// EmitterChain the VAA was emitted on
		EmitterChain ChainID
		// EmitterAddress of the contract that emitted the Message
		EmitterAddress Address
		// Payload of the message
		Payload []byte
	}

	// ChainID of a Wormhole chain
	ChainID uint16
	// Action of a VAA
	Action uint8

	// Address is a Wormhole protocol address, it contains the native chain's address. If the address data type of a
	// chain is < 32bytes the value is zero-padded on the left.
	Address [32]byte

	// Signature of a single guardian
	Signature struct {
		// Index of the validator
		Index uint8
		// Signature data
		Signature SignatureData
	}

	SignatureData [65]byte
)

func (a Address) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, a)), nil
}

func (a Address) String() string {
	return hex.EncodeToString(a[:])
}

func (a SignatureData) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, a)), nil
}

func (a SignatureData) String() string {
	return hex.EncodeToString(a[:])
}

func (c ChainID) String() string {
	switch c {
	case ChainIDSolana:
		return "solana"
	case ChainIDEthereum:
		return "ethereum"
	case ChainIDTerra:
		return "terra"
	case ChainIDBSC:
		return "bsc"
	default:
		return fmt.Sprintf("unknown chain ID: %d", c)
	}
}

const (
	// ChainIDSolana is the ChainID of Solana
	ChainIDSolana ChainID = 1
	// ChainIDEthereum is the ChainID of Ethereum
	ChainIDEthereum ChainID = 2
	// ChainIDTerra is the ChainID of Terra
	ChainIDTerra ChainID = 3
	// ChainIDBSC is the ChainID of Binance Smart Chain
	ChainIDBSC ChainID = 4

	minVAALength        = 1 + 4 + 52 + 4 + 1 + 1
	SupportedVAAVersion = 0x01
)

// Unmarshal deserializes the binary representation of a VAA
func Unmarshal(data []byte) (*VAA, error) {
	if len(data) < minVAALength {
		return nil, fmt.Errorf("VAA is too short")
	}
	v := &VAA{}

	v.Version = data[0]
	if v.Version != SupportedVAAVersion {
		return nil, fmt.Errorf("unsupported VAA version: %d", v.Version)
	}

	reader := bytes.NewReader(data[1:])

	if err := binary.Read(reader, binary.BigEndian, &v.GuardianSetIndex); err != nil {
		return nil, fmt.Errorf("failed to read guardian set index: %w", err)
	}

	lenSignatures, er := reader.ReadByte()
	if er != nil {
		return nil, fmt.Errorf("failed to read signature length")
	}

	v.Signatures = make([]*Signature, lenSignatures)
	for i := 0; i < int(lenSignatures); i++ {
		index, err := reader.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("failed to read validator index [%d]", i)
		}

		signature := [65]byte{}
		if n, err := reader.Read(signature[:]); err != nil || n != 65 {
			return nil, fmt.Errorf("failed to read signature [%d]: %w", i, err)
		}

		v.Signatures[i] = &Signature{
			Index:     index,
			Signature: signature,
		}
	}

	unixSeconds := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &unixSeconds); err != nil {
		return nil, fmt.Errorf("failed to read timestamp: %w", err)
	}
	v.Timestamp = time.Unix(int64(unixSeconds), 0)

	if err := binary.Read(reader, binary.BigEndian, &v.Nonce); err != nil {
		return nil, fmt.Errorf("failed to read nonce: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &v.EmitterChain); err != nil {
		return nil, fmt.Errorf("failed to read emitter chain: %w", err)
	}

	emitterAddress := Address{}
	if n, err := reader.Read(emitterAddress[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read emitter address [%d]: %w", n, err)
	}
	v.EmitterAddress = emitterAddress

	if err := binary.Read(reader, binary.BigEndian, &v.Sequence); err != nil {
		return nil, fmt.Errorf("failed to read sequence: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &v.ConsistencyLevel); err != nil {
		return nil, fmt.Errorf("failed to read commitment: %w", err)
	}

	payload := make([]byte, 1000)
	n, err := reader.Read(payload)
	if err != nil || n == 0 {
		return nil, fmt.Errorf("failed to read payload [%d]: %w", n, err)
	}
	v.Payload = payload[:n]

	return v, nil
}

// signingBody returns the binary representation of the data that is relevant for signing and verifying the VAA
func (v *VAA) signingBody() ([]byte, error) {
	return v.serializeBody()
}

// SigningMsg returns the hash of the signing body. This is used for signature generation and verification
func (v *VAA) SigningMsg() (common.Hash, error) {
	body, err := v.signingBody()
	if err != nil {
		// Should never happen on a successfully parsed VAA
		return common.Hash{}, fmt.Errorf("failed to serialize signing body: %w", err)
	}

	// In order to save space in the solana signature verification instruction, we hash twice so we only need to pass in
	// the first hash (32 bytes) vs the full body data.
	hash := crypto.Keccak256Hash(crypto.Keccak256Hash(body).Bytes())
	return hash, nil
}

// VerifySignatures verifies the signature of the VAA given the signer addresses.
// Returns true if the signatures were verified successfully.
func (v *VAA) VerifySignatures(addresses []common.Address) bool {
	if len(addresses) < len(v.Signatures) {
		return false
	}

	h, err := v.SigningMsg()
	if err != nil {
		return false
	}

	for _, sig := range v.Signatures {
		if int(sig.Index) >= len(addresses) {
			return false
		}

		pubKey, err := crypto.Ecrecover(h.Bytes(), sig.Signature[:])
		if err != nil {
			return false
		}
		addr := common.BytesToAddress(crypto.Keccak256(pubKey[1:])[12:])

		if addr != addresses[sig.Index] {
			return false
		}
	}

	return true
}

// Marshal returns the binary representation of the VAA
func (v *VAA) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	MustWrite(buf, binary.BigEndian, v.Version)
	MustWrite(buf, binary.BigEndian, v.GuardianSetIndex)

	// Write signatures
	MustWrite(buf, binary.BigEndian, uint8(len(v.Signatures)))
	for _, sig := range v.Signatures {
		MustWrite(buf, binary.BigEndian, sig.Index)
		buf.Write(sig.Signature[:])
	}

	// Write Body
	body, err := v.serializeBody()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize body: %w", err)
	}
	buf.Write(body)

	return buf.Bytes(), nil
}

// MessageID returns a human-readable emitter_chain/emitter_address/sequence tuple.
func (v *VAA) MessageID() string {
	return fmt.Sprintf("%d/%s/%d", v.EmitterChain, v.EmitterAddress, v.Sequence)
}

func (v *VAA) serializeBody() ([]byte, error) {
	buf := new(bytes.Buffer)
	MustWrite(buf, binary.BigEndian, uint32(v.Timestamp.Unix()))
	MustWrite(buf, binary.BigEndian, v.Nonce)
	MustWrite(buf, binary.BigEndian, v.EmitterChain)
	buf.Write(v.EmitterAddress[:])
	MustWrite(buf, binary.BigEndian, v.Sequence)
	MustWrite(buf, binary.BigEndian, v.ConsistencyLevel)
	buf.Write(v.Payload)

	return buf.Bytes(), nil
}

func (v *VAA) AddSignature(key *ecdsa.PrivateKey, index uint8) {
	data, err := v.SigningMsg()
	if err != nil {
		panic(err)
	}
	sig, err := crypto.Sign(data.Bytes(), key)
	if err != nil {
		panic(err)
	}
	sigData := [65]byte{}
	copy(sigData[:], sig)

	v.Signatures = append(v.Signatures, &Signature{
		Index:     index,
		Signature: sigData,
	})
}

// MustWrite calls binary.Write and panics on errors
func MustWrite(w io.Writer, order binary.ByteOrder, data interface{}) {
	if err := binary.Write(w, order, data); err != nil {
		panic(fmt.Errorf("failed to write binary data: %v", data).Error())
	}
}
