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
		// Payload of the VAA. This describes the action to be performed
		Payload vaaBody
	}

	// ChainID of a Wormhole chain
	ChainID uint8
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
		Signature [65]byte // TODO: hex marshaller
	}

	// AssetMeta describes an asset within the Wormhole protocol
	AssetMeta struct {
		// Chain is the ID of the chain the original version of the asset exists on
		Chain ChainID
		// Address is the address of the token contract/mint/equivalent.
		Address Address
		// Decimals is the number of decimals the token has
		Decimals uint8
	}

	vaaBody interface {
		getActionID() Action
		serialize() ([]byte, error)
	}

	BodyTransfer struct {
		// Nonce is a user given unique nonce for this transfer
		Nonce uint32
		// SourceChain is the id of the chain the transfer was initiated from
		SourceChain ChainID
		// TargetChain is the id of the chain the transfer is directed to
		TargetChain ChainID
		// TargetAddress is the address of the sender on SourceChain
		SourceAddress Address
		// TargetAddress is the address of the recipient on TargetChain
		TargetAddress Address
		// Asset is the asset to be transferred
		Asset *AssetMeta
		// Amount is the amount of tokens to be transferred
		Amount *big.Int
	}

	BodyGuardianSetUpdate struct {
		// Key is the new guardian set key
		Keys []common.Address
		// NewIndex is the index of the new guardian set
		NewIndex uint32
	}

	BodyContractUpgrade struct {
		// ChainID is the chain on which the contract should be upgraded
		ChainID uint8
		// NewContract is the address of the account containing the new contract.
		NewContract Address
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

func (c ChainID) String() string {
	switch c {
	case ChainIDUnset:
		return "unset"
	case ChainIDSolana:
		return "solana"
	case ChainIDEthereum:
		return "ethereum"
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
	default:
		return ChainIDUnset, fmt.Errorf("unknown chain ID: %s", s)
	}
}

const (
	ActionGuardianSetUpdate Action = 0x01
	ActionContractUpgrade   Action = 0x02
	ActionTransfer          Action = 0x10

	ChainIDUnset ChainID = 0
	// ChainIDSolana is the ChainID of Solana
	ChainIDSolana = 1
	// ChainIDEthereum is the ChainID of Ethereum
	ChainIDEthereum = 2

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

	var (
		action uint8
	)
	if err := binary.Read(reader, binary.BigEndian, &action); err != nil {
		return nil, fmt.Errorf("failed to read action: %w", err)
	}

	currentPos := len(data) - reader.Len()

	payloadReader := bytes.NewReader(data[currentPos:])
	var err error
	switch Action(action) {
	case ActionGuardianSetUpdate:
		v.Payload, err = parseBodyGuardianSetUpdate(payloadReader)
	case ActionTransfer:
		v.Payload, err = parseBodyTransfer(payloadReader)
	case ActionContractUpgrade:
		v.Payload, err = parseBodyContractUpgrade(payloadReader)
	default:
		return nil, fmt.Errorf("unknown action: %d", action)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

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

	hash := crypto.Keccak256Hash(body)
	return hash, nil
}

// VerifySignature verifies the signature of the VAA given the signer addresses.
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

func (v *VAA) serializeBody() ([]byte, error) {
	buf := new(bytes.Buffer)
	MustWrite(buf, binary.BigEndian, uint32(v.Timestamp.Unix()))
	MustWrite(buf, binary.BigEndian, v.Payload.getActionID())

	payloadData, err := v.Payload.serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize payload: %w", err)
	}

	buf.Write(payloadData)

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

func parseBodyTransfer(r io.Reader) (*BodyTransfer, error) {
	b := &BodyTransfer{}

	if err := binary.Read(r, binary.BigEndian, &b.Nonce); err != nil {
		return nil, fmt.Errorf("failed to read nonce: %w", err)
	}

	if err := binary.Read(r, binary.BigEndian, &b.SourceChain); err != nil {
		return nil, fmt.Errorf("failed to read source chain: %w", err)
	}

	if err := binary.Read(r, binary.BigEndian, &b.TargetChain); err != nil {
		return nil, fmt.Errorf("failed to read target chain: %w", err)
	}

	if n, err := r.Read(b.SourceAddress[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read source address: %w", err)
	}

	if n, err := r.Read(b.TargetAddress[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read target address: %w", err)
	}

	b.Asset = &AssetMeta{}
	if err := binary.Read(r, binary.BigEndian, &b.Asset.Chain); err != nil {
		return nil, fmt.Errorf("failed to read asset chain: %w", err)
	}
	if n, err := r.Read(b.Asset.Address[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read asset address: %w", err)
	}
	if err := binary.Read(r, binary.BigEndian, &b.Asset.Decimals); err != nil {
		return nil, fmt.Errorf("failed to read asset decimals: %w", err)
	}

	var amountBytes [32]byte
	if n, err := r.Read(amountBytes[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read amount: %w", err)
	}
	b.Amount = new(big.Int).SetBytes(amountBytes[:])

	return b, nil
}

func (v *BodyTransfer) getActionID() Action {
	return ActionTransfer
}

func (v *BodyTransfer) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	MustWrite(buf, binary.BigEndian, v.Nonce)
	MustWrite(buf, binary.BigEndian, v.SourceChain)
	MustWrite(buf, binary.BigEndian, v.TargetChain)
	buf.Write(v.SourceAddress[:])
	buf.Write(v.TargetAddress[:])

	if v.Asset == nil {
		return nil, fmt.Errorf("asset is empty")
	}
	MustWrite(buf, binary.BigEndian, v.Asset.Chain)
	buf.Write(v.Asset.Address[:])
	MustWrite(buf, binary.BigEndian, v.Asset.Decimals)

	if v.Amount == nil {
		return nil, fmt.Errorf("amount is empty")
	}
	buf.Write(common.LeftPadBytes(v.Amount.Bytes(), 32))

	return buf.Bytes(), nil
}

func parseBodyGuardianSetUpdate(r io.Reader) (*BodyGuardianSetUpdate, error) {
	b := &BodyGuardianSetUpdate{}

	if err := binary.Read(r, binary.BigEndian, &b.NewIndex); err != nil {
		return nil, fmt.Errorf("failed to read new index: %w", err)
	}

	keyLen := uint8(0)
	if err := binary.Read(r, binary.BigEndian, &keyLen); err != nil {
		return nil, fmt.Errorf("failed to read guardianset key len: %w", err)
	}
	for i := 0; i < int(keyLen); i++ {
		key := common.Address{}
		if n, err := r.Read(key[:]); err != nil || n != 20 {
			return nil, fmt.Errorf("failed to read guardianset key [%d]: %w", i, err)
		}
		b.Keys = append(b.Keys, key)
	}

	return b, nil
}

func (v *BodyGuardianSetUpdate) getActionID() Action {
	return ActionGuardianSetUpdate
}

func (v *BodyGuardianSetUpdate) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	MustWrite(buf, binary.BigEndian, v.NewIndex)
	MustWrite(buf, binary.BigEndian, uint8(len(v.Keys)))
	for _, key := range v.Keys {
		buf.Write(key.Bytes())
	}

	return buf.Bytes(), nil
}

func parseBodyContractUpgrade(r io.Reader) (*BodyContractUpgrade, error) {
	b := &BodyContractUpgrade{}

	if err := binary.Read(r, binary.BigEndian, &b.ChainID); err != nil {
		return nil, fmt.Errorf("failed to read chain id: %w", err)
	}

	if n, err := r.Read(b.NewContract[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read new contract address: %w", err)
	}

	return b, nil
}

func (v *BodyContractUpgrade) getActionID() Action {
	return ActionContractUpgrade
}

func (v *BodyContractUpgrade) serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	MustWrite(buf, binary.BigEndian, v.ChainID)
	buf.Write(v.NewContract[:])

	return buf.Bytes(), nil
}

// MustWrite calls binary.Write and panics on errors
func MustWrite(w io.Writer, order binary.ByteOrder, data interface{}) {
	if err := binary.Write(w, order, data); err != nil {
		panic(fmt.Errorf("failed to write binary data: %v", data).Error())
	}
}

// StringToAddress converts a hex-encoded adress into a vaa.Address
func StringToAddress(value string) (Address, error) {
	var address Address
	res, err := hex.DecodeString(value)
	if err != nil {
		return address, err
	}
	copy(address[:], res)
	return address, nil
}
