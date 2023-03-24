package vaa

import (
	"bytes"
	"crypto/ecdsa"
	"encoding"
	"encoding/binary"
	"encoding/hex"
	"errors"
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

	BatchVAA struct {
		// Version of the VAA schema
		Version uint8
		// GuardianSetIndex is the index of the guardian set that signed this VAA
		GuardianSetIndex uint32
		// SignatureData is the signature of the guardian set
		Signatures []*Signature

		// EmitterChain the VAAs were emitted on
		EmitterChain ChainID

		// The chain-native identifier of the transaction that created the batch VAA.
		TransactionID common.Hash

		// array of Observation VAA hashes
		Hashes []common.Hash

		// Observations in the batch
		Observations []*Observation
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

	Observation struct {
		// Index of the observation in a Batch array
		Index uint8
		// Signed Observation data
		Observation *VAA
	}

	TransferPayloadHdr struct {
		Type          uint8
		Amount        *big.Int
		OriginAddress Address
		OriginChain   ChainID
		TargetAddress Address
		TargetChain   ChainID
	}

	// Attestation interface contains the methods common to all VAA types
	Attestation interface {
		encoding.BinaryMarshaler
		encoding.BinaryUnmarshaler
		serializeBody()
		signingBody() []byte
		SigningMsg() common.Hash
		VerifySignatures(addrs []common.Address) bool
		UniqueID() string
		HexDigest() string
		AddSignature(key *ecdsa.PrivateKey, index uint8)
		GetEmitterChain() ChainID
	}
)

const (
	ConsistencyLevelPublishImmediately = uint8(200)
	ConsistencyLevelSafe               = uint8(201)
)

func (a Address) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, a)), nil
}

// Standard marshal stores the Address like this: "[0,0,0,0,0,0,0,0,0,0,0,0,2,144,251,22,114,8,175,69,91,177,55,120,1,99,183,183,169,161,12,22]"
// The above MarshalJSON stores it like this (66 bytes): ""0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16""
func (a *Address) UnmarshalJSON(data []byte) error {
	addr, err := StringToAddress(strings.Trim(string(data), `"`))
	if err != nil {
		return err
	}
	*a = addr
	return nil
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
	case ChainIDNear:
		return "near"
	case ChainIDAptos:
		return "aptos"
	case ChainIDSui:
		return "sui"
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
	case ChainIDArbitrum:
		return "arbitrum"
	case ChainIDOptimism:
		return "optimism"
	case ChainIDPythNet:
		return "pythnet"
	case ChainIDWormchain:
		return "wormchain"
	case ChainIDXpla:
		return "xpla"
	case ChainIDBtc:
		return "btc"
	case ChainIDBase:
		return "base"
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
	case "near":
		return ChainIDNear, nil
	case "sui":
		return ChainIDSui, nil
	case "aptos":
		return ChainIDAptos, nil
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
	case "arbitrum":
		return ChainIDArbitrum, nil
	case "optimism":
		return ChainIDOptimism, nil
	case "pythnet":
		return ChainIDPythNet, nil
	case "wormchain":
		return ChainIDWormchain, nil
	case "xpla":
		return ChainIDXpla, nil
	case "btc":
		return ChainIDBtc, nil
	case "base":
		return ChainIDBase, nil
	default:
		return ChainIDUnset, fmt.Errorf("unknown chain ID: %s", s)
	}
}

func GetAllNetworkIDs() []ChainID {
	return []ChainID{
		ChainIDSolana,
		ChainIDEthereum,
		ChainIDTerra,
		ChainIDBSC,
		ChainIDPolygon,
		ChainIDAvalanche,
		ChainIDOasis,
		ChainIDAlgorand,
		ChainIDAurora,
		ChainIDFantom,
		ChainIDKarura,
		ChainIDAcala,
		ChainIDKlaytn,
		ChainIDCelo,
		ChainIDNear,
		ChainIDMoonbeam,
		ChainIDNeon,
		ChainIDTerra2,
		ChainIDInjective,
		ChainIDSui,
		ChainIDAptos,
		ChainIDArbitrum,
		ChainIDOptimism,
		ChainIDPythNet,
		ChainIDXpla,
		ChainIDBtc,
		ChainIDBase,
		ChainIDWormchain,
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
	// ChainIDNear is the ChainID of Near
	ChainIDNear ChainID = 15
	// ChainIDMoonbeam is the ChainID of Moonbeam
	ChainIDMoonbeam ChainID = 16
	// ChainIDNeon is the ChainID of Neon
	ChainIDNeon ChainID = 17
	// ChainIDTerra2 is the ChainID of Terra 2
	ChainIDTerra2 ChainID = 18
	// ChainIDInjective is the ChainID of Injective
	ChainIDInjective ChainID = 19
	// ChainIDSui is the ChainID of Sui
	ChainIDSui ChainID = 21
	// ChainIDAptos is the ChainID of Aptos
	ChainIDAptos ChainID = 22
	// ChainIDArbitrum is the ChainID of Arbitrum
	ChainIDArbitrum ChainID = 23
	// ChainIDOptimism is the ChainID of Optimism
	ChainIDOptimism ChainID = 24
	// ChainIDPythNet is the ChainID of PythNet
	ChainIDPythNet ChainID = 26
	// ChainIDXpla is the ChainID of Xpla
	ChainIDXpla ChainID = 28
	//ChainIDBtc is the ChainID of Bitcoin
	ChainIDBtc ChainID = 29
	// ChainIDBase is the ChainID of Base
	ChainIDBase ChainID = 30
	//ChainIDWormchain is the ChainID of Wormchain
	ChainIDWormchain ChainID = 3104

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
	//  BATCH
	//  - Length of Observation Hashes (1 byte) <== minimum one
	//  - Observation Hash (32 bytes)
	//  - Length of Observations (1 byte) <== minimum one
	//  - Observation Index (1 byte)
	//  - Observation Length (1 byte)
	//  - Observation, aka BODY, aka Headless (51 bytes)
	// From Above:
	// HEADER: 1 + 4 + 1 + 0 = 6
	// BODY: 4 + 4 + 2 + 32 + 8  + 1 + 0 = 51
	// BATCH: 1 + 32 + 1 + 1 + 1 + 51 = 88
	//
	// More details here: https://docs.wormholenetwork.com/wormhole/vaas
	minHeadlessVAALength = 51 // HEADER
	minVAALength         = 57 // HEADER + BODY
	minBatchVAALength    = 94 // HEADER + BATCH

	SupportedVAAVersion = 0x01
	BatchVAAVersion     = 0x02
)

// UnmarshalBody deserializes the binary representation of a VAA's "BODY" properties
// The BODY fields are common among multiple types of VAA - v1, v2 (BatchVAA), etc
func UnmarshalBody(data []byte, reader *bytes.Reader, v *VAA) (*VAA, error) {
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

	// Make sure to only read the payload if the VAA has one; VAAs may have a 0 length payload
	if reader.Len() != 0 {
		payload := make([]byte, reader.Len())
		n, err := reader.Read(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to read payload [%d]: %w", n, err)
		}

		v.Payload = payload[:n]
	} else {
		v.Payload = []byte{}
	}

	return v, nil
}

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

	return UnmarshalBody(data, reader, v)
}

// UnmarshalBatch deserializes the binary representation of a BatchVAA
func UnmarshalBatch(data []byte) (*BatchVAA, error) {
	if len(data) < minBatchVAALength {
		return nil, fmt.Errorf("BatchVAA.Observation is too short")
	}
	v := &BatchVAA{}

	v.Version = data[0]
	if v.Version != BatchVAAVersion {
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

	v.Signatures = make([]*Signature, int(lenSignatures))
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
			Index:     uint8(index),
			Signature: signature,
		}
	}

	lenHashes, err := reader.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("failed to read hashes length [%w]", err)
	}
	numHashes := int(lenHashes)

	v.Hashes = make([]common.Hash, numHashes)
	for i := 0; i < int(lenHashes); i++ {
		hash := [32]byte{}
		if n, err := reader.Read(hash[:]); err != nil || n != 32 {
			return nil, fmt.Errorf("failed to read hash [%d]: %w", i, err)
		}
		v.Hashes[i] = common.BytesToHash(hash[:])
	}

	lenObservations, err := reader.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("failed to read observations length: %w", err)
	}
	numObservations := int(lenObservations)

	if numHashes != numObservations {
		// should never happen, check anyway
		return nil, fmt.Errorf(
			"failed unmarshaling BatchVAA, observations differs from hashes")
	}

	v.Observations = make([]*Observation, numObservations)
	for i := 0; i < int(lenObservations); i++ {
		index, err := reader.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("failed to read Observation index [%d]: %w", i, err)
		}
		obsvIndex := uint8(index)

		obsvLength := uint32(0)
		if err := binary.Read(reader, binary.BigEndian, &obsvLength); err != nil {
			return nil, fmt.Errorf("failed to read Observation length: %w", err)
		}
		numBytes := int(obsvLength)

		// ensure numBytes is within expected bounds before allocating arrays
		// cannot be negative
		if numBytes < 0 {
			return nil, fmt.Errorf(
				"failed to read Observation index: %v, byte length is negative", i)
		}
		// cannot be longer than what is left in the array
		if numBytes > reader.Len() {
			return nil, fmt.Errorf(
				"failed to read Observation index: %v, byte length is erroneous", i)
		}

		obs := make([]byte, numBytes)
		if n, err := reader.Read(obs[:]); err != nil || n == 0 {
			return nil, fmt.Errorf("failed to read Observation bytes [%d]: %w", n, err)
		}

		// ensure the observation meets the minimum length of headless VAAs
		if len(obs) < minHeadlessVAALength {
			return nil, fmt.Errorf(
				"BatchVAA.Observation is too short. Index: %v", obsvIndex)
		}

		// decode the observation, which is just the "BODY" fields of a v1 VAA
		headless, err := UnmarshalBody(data, bytes.NewReader(obs[:]), &VAA{})

		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal Observation VAA. %w", err)
		}

		// check for malformed data - verify that the hash of the observation matches what was supplied
		// the guardian has no interest in or use for observations after the batch has been signed, but still check
		obsHash := headless.SigningDigest()
		if obsHash != v.Hashes[obsvIndex] {
			return nil, fmt.Errorf(
				"BatchVAA Observation %v does not match supplied hash", obsvIndex)
		}

		v.Observations[i] = &Observation{
			Index:       obsvIndex,
			Observation: headless,
		}
	}

	return v, nil
}

// signingBody returns the binary representation of the data that is relevant for signing and verifying the VAA
func (v *VAA) signingBody() []byte {
	return v.serializeBody()
}

// signingBody returns the binary representation of the data that is relevant for signing and verifying the VAA
func (v *BatchVAA) signingBody() []byte {
	buf := new(bytes.Buffer)

	// add the VAA version
	MustWrite(buf, binary.BigEndian, v.Version)

	// create the hash array from the Observations of the BatchVAA
	hashes := v.ObsvHashArray()

	MustWrite(buf, binary.BigEndian, hashes)

	return buf.Bytes()
}

func doubleKeccak(bz []byte) common.Hash {
	// In order to save space in the solana signature verification instruction, we hash twice so we only need to pass in
	// the first hash (32 bytes) vs the full body data.
	return crypto.Keccak256Hash(crypto.Keccak256Hash(bz).Bytes())
}

// This is a temporary method to produce a vaa signing digest on raw bytes.
// It is error prone and we should use `v.SigningDigest()` instead.
// whenever possible.
// This will be removed in a subsequent release.
func DeprecatedSigningDigest(bz []byte) common.Hash {
	return doubleKeccak(bz)
}

// MessageSigningDigest returns the hash of the data prepended with it's signing prefix.
// This is intending to be used for signing messages of different types from VAA's.
// The message prefix helps protect from message collisions.
func MessageSigningDigest(prefix []byte, data []byte) (common.Hash, error) {
	if len(prefix) < 32 {
		// Prefixes must be at least 32 bytes
		// https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0009_guardian_key.md
		return common.Hash([32]byte{}), errors.New("prefix must be at least 32 bytes")
	}
	return crypto.Keccak256Hash(prefix[:], data), nil
}

// SigningDigest returns the hash of the vaa hash to be signed directly.
// This is used for signature generation and verification
func (v *VAA) SigningDigest() common.Hash {
	return doubleKeccak(v.signingBody())
}

// BatchSigningDigest returns the hash of the batch vaa hash to be signed directly.
// This is used for signature generation and verification
func (v *BatchVAA) SigningDigest() common.Hash {
	return doubleKeccak(v.signingBody())
}

// ObsvHashArray creates an array of hashes of Observation.
// hashes in the array have the index position of their Observation.Index.
func (v *BatchVAA) ObsvHashArray() []common.Hash {
	hashes := make([]common.Hash, len(v.Observations))
	for _, msg := range v.Observations {
		obsIndex := msg.Index
		hashes[obsIndex] = msg.Observation.SigningDigest()
	}

	return hashes
}

// Verify Signature checks that the provided address matches the address that created the signature for the provided digest
// Digest should be the output of SigningMsg(data).Bytes()
// Should not be public as other message types should be verified using a message prefix.
func verifySignature(vaa_digest []byte, signature *Signature, address common.Address) bool {
	// retrieve the address that signed the data
	pubKey, err := crypto.Ecrecover(vaa_digest, signature.Signature[:])
	if err != nil {
		return false
	}
	addr := common.BytesToAddress(crypto.Keccak256(pubKey[1:])[12:])

	// check that the recovered address equals the provided address
	return addr == address
}

// Digest should be the output of SigningMsg(data).Bytes()
// Should not be public as other message types should be verified using a message prefix.
func verifySignatures(vaa_digest []byte, signatures []*Signature, addresses []common.Address) bool {
	if len(addresses) < len(signatures) {
		return false
	}

	last_index := -1
	signing_addresses := []common.Address{}

	for _, sig := range signatures {
		if int(sig.Index) >= len(addresses) {
			return false
		}

		// Ensure increasing indexes
		if int(sig.Index) <= last_index {
			return false
		}
		last_index = int(sig.Index)

		// verify this signature
		addr := addresses[sig.Index]
		ok := verifySignature(vaa_digest, sig, addr)
		if !ok {
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

// Operating on bytes directly is error prone.  We should use `vaa.VerifyingSignatures()` whenever possible.
// This function will be removed in a subsequent release.
func DeprecatedVerifySignatures(vaaBody []byte, signatures []*Signature, addresses []common.Address) bool {
	vaaDigest := doubleKeccak(vaaBody)
	return verifySignatures(vaaDigest[:], signatures, addresses)
}

func VerifyMessageSignature(prefix []byte, messageBody []byte, signatures *Signature, addresses common.Address) bool {
	if len(prefix) < 32 {
		return false
	}
	msgDigest, err := MessageSigningDigest(prefix, messageBody)
	if err != nil {
		return false
	}
	return verifySignature(msgDigest[:], signatures, addresses)
}

// VerifySignatures verifies the signature of the VAA given the signer addresses.
// Returns true if the signatures were verified successfully.
func (v *VAA) VerifySignatures(addresses []common.Address) bool {
	return verifySignatures(v.SigningDigest().Bytes(), v.Signatures, addresses)
}

// VerifySignatures verifies the signature of the BatchVAA given the signer addresses.
// Returns true if the signatures were verified successfully.
func (v *BatchVAA) VerifySignatures(addresses []common.Address) bool {
	return verifySignatures(v.SigningDigest().Bytes(), v.Signatures, addresses)
}

// Marshal returns the binary representation of the BatchVAA
func (v *BatchVAA) Marshal() ([]byte, error) {
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

// Serializes the body of the BatchVAA.
func (v *BatchVAA) serializeBody() []byte {
	buf := new(bytes.Buffer)

	hashes := v.ObsvHashArray()

	MustWrite(buf, binary.BigEndian, uint8(len(hashes)))
	MustWrite(buf, binary.BigEndian, hashes)

	MustWrite(buf, binary.BigEndian, uint8(len(v.Observations)))
	for _, obsv := range v.Observations {

		MustWrite(buf, binary.BigEndian, uint8(obsv.Index))

		obsvBytes := obsv.Observation.serializeBody()

		lenBytes := len(obsvBytes)
		MustWrite(buf, binary.BigEndian, uint32(lenBytes))
		buf.Write(obsvBytes)
	}

	return buf.Bytes()
}

// Verify is a function on the VAA that takes a complete set of guardian keys as input and attempts certain checks with respect to this guardian.
// Verify will return nil if the VAA passes checks.  Otherwise, Verify will return an error containing the text of the first check to fail.
// NOTE:  Verify will not work correctly if a subset of the guardian set keys is passed in.  The complete guardian set must be passed in.
// Verify does the following checks:
// - If the guardian does not have or know its own guardian set keys, then the VAA cannot be verified.
// - Quorum is calculated on the guardian set passed in and checks if the VAA has enough signatures.
// - The signatures in the VAA is verified against the guardian set keys.
func (v *VAA) Verify(addresses []common.Address) error {
	if addresses == nil {
		return errors.New("no addresses were provided")
	}

	// Check if VAA doesn't have any signatures
	if len(v.Signatures) == 0 {
		return errors.New("VAA was not signed")
	}

	// Verify VAA has enough signatures for quorum
	quorum := CalculateQuorum(len(addresses))
	if len(v.Signatures) < quorum {
		return errors.New("VAA did not have a quorum")
	}

	// Verify VAA signatures to prevent a DoS attack on our local store.
	if !v.VerifySignatures(addresses) {
		return errors.New("VAA had bad signatures")
	}

	return nil
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

// implement encoding.BinaryMarshaler interface for the VAA struct
func (v VAA) MarshalBinary() ([]byte, error) {
	return v.Marshal()
}

// implement encoding.BinaryUnmarshaler interface for the VAA struct
func (v *VAA) UnmarshalBinary(data []byte) error {
	vaa, err := Unmarshal(data)
	if err != nil {
		return err
	}

	// derefernce the stuct created by Unmarshal, and assign it to the method's context
	*v = *vaa
	return nil
}

// implement encoding.BinaryMarshaler interface for BatchVAA struct
func (b BatchVAA) MarshalBinary() ([]byte, error) {
	return b.Marshal()
}

// implement encoding.BinaryUnmarshaler interface for BatchVAA struct
func (b *BatchVAA) UnmarshalBinary(data []byte) error {
	batch, err := UnmarshalBatch(data)
	if err != nil {
		return err
	}

	// derefernce the stuct created by Unmarshal, and assign it to the method's context
	*b = *batch
	return nil
}

// MessageID returns a human-readable emitter_chain/emitter_address/sequence tuple.
func (v *VAA) MessageID() string {
	return fmt.Sprintf("%d/%s/%d", v.EmitterChain, v.EmitterAddress, v.Sequence)
}

// BatchID returns a human-readable emitter_chain/transaction_hex
func (v *BatchVAA) BatchID() string {
	if len(v.Observations) == 0 {
		// cant have a batch without Observations, but check just be safe
		panic("Cannot create a BatchID from BatchVAA with no Observations.")
	}
	nonce := v.Observations[0].Observation.Nonce
	return fmt.Sprintf("%d/%s/%d", v.EmitterChain, hex.EncodeToString(v.TransactionID.Bytes()), nonce)
}

// UniqueID normalizes the ID of the VAA (any type) for the Attestation interface
// UniqueID returns the MessageID that uniquely identifies the Attestation
func (v *VAA) UniqueID() string {
	return v.MessageID()
}

// UniqueID returns the BatchID that uniquely identifies the Attestation
func (b *BatchVAA) UniqueID() string {
	return b.BatchID()
}

// GetTransactionID implements the processor.Batch interface for *BatchVAA.
func (v *BatchVAA) GetTransactionID() common.Hash {
	return v.TransactionID
}

// HexDigest returns the hex-encoded digest.
func (v *VAA) HexDigest() string {
	return hex.EncodeToString(v.SigningDigest().Bytes())
}

// HexDigest returns the hex-encoded digest.
func (b *BatchVAA) HexDigest() string {
	return hex.EncodeToString(b.SigningDigest().Bytes())
}

/*
SECURITY: Do not change this code! Changing it could result in two different hashes for
the same observation. But xDapps rely on the hash of an observation for replay protection.
*/
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
	sig, err := crypto.Sign(v.SigningDigest().Bytes(), key)
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

// creates signature of BatchVAA.Hashes and adds it to BatchVAA.Signatures.
func (v *BatchVAA) AddSignature(key *ecdsa.PrivateKey, index uint8) {

	sig, err := crypto.Sign(v.SigningDigest().Bytes(), key)
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

// GetEmitterChain implements the processor.Batch interface for *BatchVAA.
func (v *BatchVAA) GetEmitterChain() ChainID {
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

// StringToHash converts a hex-encoded string into a common.Hash
func StringToHash(value string) (common.Hash, error) {
	var tx common.Hash

	// Make sure we have enough to decode
	if len(value) < 2 {
		return tx, fmt.Errorf("value must be at least 1 byte")
	}

	// Trim any preceding "0x" to the address
	value = strings.TrimPrefix(value, "0x")

	res, err := hex.DecodeString(value)
	if err != nil {
		return tx, err
	}

	tx = common.BytesToHash(res)

	return tx, nil
}

func BytesToHash(b []byte) (common.Hash, error) {
	var hash common.Hash
	if len(b) > 32 {
		return hash, fmt.Errorf("value must be no more than 32 bytes")
	}

	hash = common.BytesToHash(b)
	return hash, nil
}
