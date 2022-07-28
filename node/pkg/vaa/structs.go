package vaa

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"strings"
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

	TransferPayloadHdr struct {
		Type          uint8
		Amount        *big.Int
		OriginAddress Address
		OriginChain   ChainID
		TargetAddress Address
		TargetChain   ChainID
	}
)

func (a Address) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, a)), nil
}

func (a Address) String() string {
	return hex.EncodeToString(a[:])
}

func (a Address) Bytes() []byte {
	return a[:]
}

func (a SignatureData) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, a)), nil
}

func (a SignatureData) String() string {
	return hex.EncodeToString(a[:])
}

func (c ChainID) String() string {
	switch c {
	case ChainIDUnset:
		return "unset"
	case ChainIDSolana:
		return "solana"
	case ChainIDEthereum:
		return "ethereum"
	case ChainIDTerra:
		return "terra"
	case ChainIDBSC:
		return "bsc"
	case ChainIDPolygon:
		return "polygon"
	case ChainIDAvalanche:
		return "avalanche"
	case ChainIDOasis:
		return "oasis"
	case ChainIDAurora:
		return "aurora"
	case ChainIDFantom:
		return "fantom"
	case ChainIDAlgorand:
		return "algorand"
	case ChainIDEthereumRopsten:
		return "ethereum-ropsten"
	case ChainIDKarura:
		return "karura"
	case ChainIDAcala:
		return "acala"
	case ChainIDKlaytn:
		return "klaytn"
	case ChainIDCelo:
		return "celo"
	case ChainIDMoonbeam:
		return "moonbeam"
	case ChainIDNeon:
		return "neon"
	case ChainIDTerra2:
		return "terra2"
	case ChainIDInjective:
		return "injective"
	case ChainIDPythNet:
		return "pythnet"
	default:
		return fmt.Sprintf("unknown chain ID: %d", c)
	}
}

func ChainIDFromString(s string) (ChainID, error) {
	s = strings.ToLower(s)

	switch s {
	case "solana":
		return ChainIDSolana, nil
	case "ethereum":
		return ChainIDEthereum, nil
	case "terra":
		return ChainIDTerra, nil
	case "bsc":
		return ChainIDBSC, nil
	case "polygon":
		return ChainIDPolygon, nil
	case "avalanche":
		return ChainIDAvalanche, nil
	case "oasis":
		return ChainIDOasis, nil
	case "aurora":
		return ChainIDAurora, nil
	case "fantom":
		return ChainIDFantom, nil
	case "algorand":
		return ChainIDAlgorand, nil
	case "ethereum-ropsten":
		return ChainIDEthereumRopsten, nil
	case "karura":
		return ChainIDKarura, nil
	case "acala":
		return ChainIDAcala, nil
	case "klaytn":
		return ChainIDKlaytn, nil
	case "celo":
		return ChainIDCelo, nil
	case "moonbeam":
		return ChainIDMoonbeam, nil
	case "neon":
		return ChainIDNeon, nil
	case "terra2":
		return ChainIDTerra2, nil
	case "injective":
		return ChainIDInjective, nil
	case "pythnet":
		return ChainIDPythNet, nil
	default:
		return ChainIDUnset, fmt.Errorf("unknown chain ID: %s", s)
	}
}

const (
	ChainIDUnset ChainID = 0
	// ChainIDSolana is the ChainID of Solana
	ChainIDSolana ChainID = 1
	// ChainIDEthereum is the ChainID of Ethereum
	ChainIDEthereum ChainID = 2
	// ChainIDTerra is the ChainID of Terra
	ChainIDTerra ChainID = 3
	// ChainIDBSC is the ChainID of Binance Smart Chain
	ChainIDBSC ChainID = 4
	// ChainIDPolygon is the ChainID of Polygon
	ChainIDPolygon ChainID = 5
	// ChainIDAvalanche is the ChainID of Avalanche
	ChainIDAvalanche ChainID = 6
	// ChainIDOasis is the ChainID of Oasis
	ChainIDOasis ChainID = 7
	// ChainIDAlgorand is the ChainID of Algorand
	ChainIDAlgorand ChainID = 8
	// ChainIDAurora is the ChainID of Aurora
	ChainIDAurora ChainID = 9
	// ChainIDFantom is the ChainID of Fantom
	ChainIDFantom ChainID = 10
	// ChainIDKarura is the ChainID of Karura
	ChainIDKarura ChainID = 11
	// ChainIDAcala is the ChainID of Acala
	ChainIDAcala ChainID = 12
	// ChainIDKlaytn is the ChainID of Klaytn
	ChainIDKlaytn ChainID = 13
	// ChainIDCelo is the ChainID of Celo
	ChainIDCelo ChainID = 14
	// ChainIDMoonbeam is the ChainID of Moonbeam
	ChainIDMoonbeam ChainID = 16
	// ChainIDNeon is the ChainID of Neon
	ChainIDNeon ChainID = 17
	// ChainIDTerra2 is the ChainID of Terra 2
	ChainIDTerra2 ChainID = 18
	// ChainIDInjective is the ChainID of Injective
	ChainIDInjective ChainID = 19
	// ChainIDPythNet is the ChainID of PythNet
	ChainIDPythNet ChainID = 26

	// ChainIDEthereumRopsten is the ChainID of Ethereum Ropsten
	ChainIDEthereumRopsten ChainID = 10001

	// Minimum VAA size is derrived from the following assumptions:
	//  HEADER
	//  - Supported VAA Version (1 byte)
	//  - Guardian Set Index (4 bytes)
	//  - Length of Signatures (1 byte) <== assume no signatures
	//  - Actual Signatures (0 bytes)
	//  BODY
	//  - timestamp (4 bytes)
	//  - nonce (4 bytes)
	//  - emitter chain (2 bytes)
	//  - emitter address (32 bytes)
	//  - sequence (8 bytes)
	//  - consistency level (1 byte)
	//  - payload (0 bytes)
	//
	// From Above: 1 + 4 + 1 + 0 + 4 + 4 + 2 + 32 + 8  + 1 + 0 // Equals 57
	//
	// More details here: https://docs.wormholenetwork.com/wormhole/vaas
	minVAALength        = 57
	SupportedVAAVersion = 0x01

	InternalTruncatedPayloadSafetyLimit = 1000
)

// Unmarshal deserializes the binary representation of a VAA
//
// WARNING: Unmarshall will truncate payloads at 1000 bytes, this is done mainly to avoid denial of service
//   - If you need to access the full payload, consider parsing VAA from Bytes instead of Unmarshal
//
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

	payload := make([]byte, InternalTruncatedPayloadSafetyLimit)
	n, err := reader.Read(payload)
	if err != nil || n == 0 {
		return nil, fmt.Errorf("failed to read payload [%d]: %w", n, err)
	}

	v.Payload = payload[:n]

	return v, nil
}

// signingBody returns the binary representation of the data that is relevant for signing and verifying the VAA
func (v *VAA) signingBody() []byte {
	return v.serializeBody()
}

// SigningMsg returns the hash of the signing body. This is used for signature generation and verification
func (v *VAA) SigningMsg() common.Hash {
	// In order to save space in the solana signature verification instruction, we hash twice so we only need to pass in
	// the first hash (32 bytes) vs the full body data.
	hash := crypto.Keccak256Hash(crypto.Keccak256Hash(v.signingBody()).Bytes())
	return hash
}

// VerifySignatures verifies the signature of the VAA given the signer addresses.
// Returns true if the signatures were verified successfully.
func (v *VAA) VerifySignatures(addresses []common.Address) bool {
	if len(addresses) < len(v.Signatures) {
		return false
	}

	h := v.SigningMsg()

	last_index := -1
	signing_addresses := []common.Address{}

	for _, sig := range v.Signatures {
		if int(sig.Index) >= len(addresses) {
			return false
		}

		// Ensure increasing indexes
		if int(sig.Index) <= last_index {
			return false
		}
		last_index = int(sig.Index)

		// Get pubKey to determine who signers address
		pubKey, err := crypto.Ecrecover(h.Bytes(), sig.Signature[:])
		if err != nil {
			return false
		}
		addr := common.BytesToAddress(crypto.Keccak256(pubKey[1:])[12:])

		// Ensure this signer is at the correct positional index
		if addr != addresses[sig.Index] {
			return false
		}

		// Ensure we never see the same signer twice
		for _, signing_address := range signing_addresses {
			if signing_address == addr {
				return false
			}
		}
		signing_addresses = append(signing_addresses, addr)
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
	buf.Write(v.serializeBody())

	return buf.Bytes(), nil
}

// MessageID returns a human-readable emitter_chain/emitter_address/sequence tuple.
func (v *VAA) MessageID() string {
	return fmt.Sprintf("%d/%s/%d", v.EmitterChain, v.EmitterAddress, v.Sequence)
}

// HexDigest returns the hex-encoded digest.
func (v *VAA) HexDigest() string {
	return hex.EncodeToString(v.SigningMsg().Bytes())
}

func (v *VAA) serializeBody() []byte {
	buf := new(bytes.Buffer)
	MustWrite(buf, binary.BigEndian, uint32(v.Timestamp.Unix()))
	MustWrite(buf, binary.BigEndian, v.Nonce)
	MustWrite(buf, binary.BigEndian, v.EmitterChain)
	buf.Write(v.EmitterAddress[:])
	MustWrite(buf, binary.BigEndian, v.Sequence)
	MustWrite(buf, binary.BigEndian, v.ConsistencyLevel)
	buf.Write(v.Payload)

	return buf.Bytes()
}

func (v *VAA) AddSignature(key *ecdsa.PrivateKey, index uint8) {
	sig, err := crypto.Sign(v.SigningMsg().Bytes(), key)
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

// NOTE: This function assumes that the caller has verified that the VAA is from the token bridge.
func IsTransfer(payload []byte) bool {
	return (len(payload) > 0) && ((payload[0] == 1) || (payload[0] == 3))
}

func DecodeTransferPayloadHdr(payload []byte) (*TransferPayloadHdr, error) {
	if !IsTransfer(payload) {
		return nil, fmt.Errorf("unsupported payload type")
	}

	if len(payload) < 101 {
		return nil, fmt.Errorf("buffer too short")
	}

	p := &TransferPayloadHdr{}

	// Payload type: payload[0]
	p.Type = uint8(payload[0])

	// Amount: payload[1] for 32
	p.Amount = new(big.Int)
	p.Amount.SetBytes(payload[1:33])

	reader := bytes.NewReader(payload[33:])

	// Origin address: payload[33] for 32
	err := binary.Read(reader, binary.BigEndian, &p.OriginAddress)
	if err != nil {
		return nil, err
	}

	// Origin chain ID: payload[65] for 2
	err = binary.Read(reader, binary.BigEndian, &p.OriginChain)
	if err != nil {
		return nil, err
	}

	// Target address: payload[67] for 32
	err = binary.Read(reader, binary.BigEndian, &p.TargetAddress)
	if err != nil {
		return nil, err
	}

	// Target chain ID: payload[99] for 2
	err = binary.Read(reader, binary.BigEndian, &p.TargetChain)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// GetEmitterChain implements the processor.Observation interface for *VAA.
func (v *VAA) GetEmitterChain() ChainID {
	return v.EmitterChain
}

// MustWrite calls binary.Write and panics on errors
func MustWrite(w io.Writer, order binary.ByteOrder, data interface{}) {
	if err := binary.Write(w, order, data); err != nil {
		panic(fmt.Errorf("failed to write binary data: %v", data).Error())
	}
}

// StringToAddress converts a hex-encoded address into a vaa.Address
func StringToAddress(value string) (Address, error) {
	var address Address

	// Make sure we have enough to decode
	if len(value) < 2 {
		return address, fmt.Errorf("value must be at least 1 byte")
	}

	// Trim any preceding "0x" to the address
	value = strings.TrimPrefix(value, "0x")

	// Decode the string from hex to binary
	res, err := hex.DecodeString(value)
	if err != nil {
		return address, err
	}

	// Make sure we don't have too many bytes
	if len(res) > 32 {
		return address, fmt.Errorf("value must be no more than 32 bytes")
	}
	copy(address[32-len(res):], res)

	return address, nil
}

func BytesToAddress(b []byte) (Address, error) {
	var address Address
	if len(b) > 32 {
		return address, fmt.Errorf("value must be no more than 32 bytes")
	}

	copy(address[32-len(b):], b)
	return address, nil
}
