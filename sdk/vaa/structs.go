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
	"math"
	"math/big"
	"strconv"
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

	// number is a constraint for generic functions that can safely convert integer types to a ChainID (uint16).
	number interface {
		~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
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
	case ChainIDAlgorand:
		return "algorand"
	case ChainIDAurora:
		return "aurora"
	case ChainIDFantom:
		return "fantom"
	case ChainIDKarura:
		return "karura"
	case ChainIDAcala:
		return "acala"
	case ChainIDKlaytn:
		return "klaytn"
	case ChainIDCelo:
		return "celo"
	case ChainIDNear:
		return "near"
	case ChainIDMoonbeam:
		return "moonbeam"
	case ChainIDTerra2:
		return "terra2"
	case ChainIDInjective:
		return "injective"
	case ChainIDOsmosis:
		return "osmosis"
	case ChainIDSui:
		return "sui"
	case ChainIDAptos:
		return "aptos"
	case ChainIDAztec:
		return "aztec"
	case ChainIDArbitrum:
		return "arbitrum"
	case ChainIDOptimism:
		return "optimism"
	case ChainIDGnosis:
		return "gnosis"
	case ChainIDPythNet:
		return "pythnet"
	case ChainIDXpla:
		return "xpla"
	case ChainIDBtc:
		return "btc"
	case ChainIDBase:
		return "base"
	case ChainIDFileCoin:
		return "filecoin"
	case ChainIDSei:
		return "sei"
	case ChainIDRootstock:
		return "rootstock"
	case ChainIDScroll:
		return "scroll"
	case ChainIDMantle:
		return "mantle"
	case ChainIDBlast:
		return "blast"
	case ChainIDXLayer:
		return "xlayer"
	case ChainIDLinea:
		return "linea"
	case ChainIDBerachain:
		return "berachain"
	case ChainIDSeiEVM:
		return "seievm"
	case ChainIDEclipse:
		return "eclipse"
	case ChainIDBOB:
		return "bob"
	case ChainIDSnaxchain:
		return "snaxchain"
	case ChainIDUnichain:
		return "unichain"
	case ChainIDWorldchain:
		return "worldchain"
	case ChainIDInk:
		return "ink"
	case ChainIDHyperEVM:
		return "hyperevm"
	case ChainIDMonad:
		return "monad"
	case ChainIDMovement:
		return "movement"
	case ChainIDMezo:
		return "mezo"
	case ChainIDFogo:
		return "fogo"
	case ChainIDWormchain:
		return "wormchain"
	case ChainIDCosmoshub:
		return "cosmoshub"
	case ChainIDEvmos:
		return "evmos"
	case ChainIDKujira:
		return "kujira"
	case ChainIDNeutron:
		return "neutron"
	case ChainIDCelestia:
		return "celestia"
	case ChainIDStargaze:
		return "stargaze"
	case ChainIDSeda:
		return "seda"
	case ChainIDDymension:
		return "dymension"
	case ChainIDProvenance:
		return "provenance"
	case ChainIDNoble:
		return "noble"
	case ChainIDSepolia:
		return "sepolia"
	case ChainIDArbitrumSepolia:
		return "arbitrum_sepolia"
	case ChainIDBaseSepolia:
		return "base_sepolia"
	case ChainIDOptimismSepolia:
		return "optimism_sepolia"
	case ChainIDHolesky:
		return "holesky"
	case ChainIDPolygonSepolia:
		return "polygon_sepolia"
	default:
		return fmt.Sprintf("unknown chain ID: %d", c)
	}
}

// ChainIDFromNumber converts an unsigned integer into a ChainID. This function only determines whether the input is valid
// with respect to its type; it does not check whether the ChainID is actually registered or used anywhere.
// This function can be used to validate ChainID values that are deserialized from protobuf messages. (As protobuf
// does not support the uint16 type, ChainIDs are usually encoded as uint32.)
// https://protobuf.dev/reference/protobuf/proto3-spec/#fields
// Returns an error if the argument would overflow uint16.
func ChainIDFromNumber[N number](n N) (ChainID, error) {
	if n < 0 {
		return ChainIDUnset, fmt.Errorf("chainID cannot be negative but got %d", n)
	}
	switch any(n).(type) {
	case int8, uint8, int16, uint16:
		// Because these values have been checked to be non-negative, we can return early with a simple conversion.
		return ChainID(n), nil

	}
	// Use intermediate uint64 to safely handle conversion and allow comparison with MaxUint16.
	// This is safe to do because the negative case is already handled.
	val := uint64(n)
	if val > uint64(math.MaxUint16) {
		return ChainIDUnset, fmt.Errorf("chainID must be less than or equal to %d but got %d", math.MaxUint16, n)
	}
	return ChainID(n), nil

}

// KnownChainIDFromNumber converts an unsigned integer into a known ChainID. It is a wrapper function for ChainIDFromNumber
// that also checks whether the ChainID corresponds to a real, configured chain.
func KnownChainIDFromNumber[N number](n N) (ChainID, error) {
	id, err := ChainIDFromNumber(n)
	if err != nil {
		return ChainIDUnset, err
	}

	// NOTE: slice.Contains is not used here because some SDK integrators (e.g. wormchain, maybe others) use old versions of Go.
	for _, known := range GetAllNetworkIDs() {
		if id == known {
			return id, nil
		}
	}

	return ChainIDUnset, fmt.Errorf("no known ChainID for input %d", n)

}

// StringToKnownChainID converts from a string representation of a chain into a ChainID that is registered in the SDK.
// The argument can be either a numeric string representation of a number or a known chain name such as "solana".
// Inputs of unknown ChainIDs, including 0, will result in an error.
func StringToKnownChainID(s string) (ChainID, error) {

	// Try to convert from chain name first, and return early if it's found.
	id, err := ChainIDFromString(s)
	if err == nil {
		return id, nil
	}

	// Ensure that the string can be parsed into a uint16 in order to avoid overflow issues when converting
	// to ChainID (which is a uint16).
	u16, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return ChainIDUnset, err
	}

	return KnownChainIDFromNumber(u16)
}

// ChainIDFromString converts from a chain's full name (e.g. "solana") to its corresponding ChainID.
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
	case "algorand":
		return ChainIDAlgorand, nil
	case "aurora":
		return ChainIDAurora, nil
	case "fantom":
		return ChainIDFantom, nil
	case "karura":
		return ChainIDKarura, nil
	case "acala":
		return ChainIDAcala, nil
	case "klaytn":
		return ChainIDKlaytn, nil
	case "celo":
		return ChainIDCelo, nil
	case "near":
		return ChainIDNear, nil
	case "moonbeam":
		return ChainIDMoonbeam, nil
	case "terra2":
		return ChainIDTerra2, nil
	case "injective":
		return ChainIDInjective, nil
	case "osmosis":
		return ChainIDOsmosis, nil
	case "sui":
		return ChainIDSui, nil
	case "aptos":
		return ChainIDAptos, nil
	case "aztec":
		return ChainIDAztec, nil
	case "arbitrum":
		return ChainIDArbitrum, nil
	case "optimism":
		return ChainIDOptimism, nil
	case "gnosis":
		return ChainIDGnosis, nil
	case "pythnet":
		return ChainIDPythNet, nil
	case "xpla":
		return ChainIDXpla, nil
	case "btc":
		return ChainIDBtc, nil
	case "base":
		return ChainIDBase, nil
	case "filecoin":
		return ChainIDFileCoin, nil
	case "sei":
		return ChainIDSei, nil
	case "rootstock":
		return ChainIDRootstock, nil
	case "scroll":
		return ChainIDScroll, nil
	case "mantle":
		return ChainIDMantle, nil
	case "blast":
		return ChainIDBlast, nil
	case "xlayer":
		return ChainIDXLayer, nil
	case "linea":
		return ChainIDLinea, nil
	case "berachain":
		return ChainIDBerachain, nil
	case "seievm":
		return ChainIDSeiEVM, nil
	case "eclipse":
		return ChainIDEclipse, nil
	case "bob":
		return ChainIDBOB, nil
	case "snaxchain":
		return ChainIDSnaxchain, nil
	case "unichain":
		return ChainIDUnichain, nil
	case "worldchain":
		return ChainIDWorldchain, nil
	case "ink":
		return ChainIDInk, nil
	case "hyperevm":
		return ChainIDHyperEVM, nil
	case "monad":
		return ChainIDMonad, nil
	case "movement":
		return ChainIDMovement, nil
	case "mezo":
		return ChainIDMezo, nil
	case "fogo":
		return ChainIDFogo, nil
	case "wormchain":
		return ChainIDWormchain, nil
	case "cosmoshub":
		return ChainIDCosmoshub, nil
	case "evmos":
		return ChainIDEvmos, nil
	case "kujira":
		return ChainIDKujira, nil
	case "neutron":
		return ChainIDNeutron, nil
	case "celestia":
		return ChainIDCelestia, nil
	case "stargaze":
		return ChainIDStargaze, nil
	case "seda":
		return ChainIDSeda, nil
	case "dymension":
		return ChainIDDymension, nil
	case "provenance":
		return ChainIDProvenance, nil
	case "noble":
		return ChainIDNoble, nil
	case "sepolia":
		return ChainIDSepolia, nil
	case "arbitrum_sepolia":
		return ChainIDArbitrumSepolia, nil
	case "base_sepolia":
		return ChainIDBaseSepolia, nil
	case "optimism_sepolia":
		return ChainIDOptimismSepolia, nil
	case "holesky":
		return ChainIDHolesky, nil
	case "polygon_sepolia":
		return ChainIDPolygonSepolia, nil
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
		ChainIDTerra2,
		ChainIDInjective,
		ChainIDOsmosis,
		ChainIDSui,
		ChainIDAptos,
		ChainIDAztec,
		ChainIDArbitrum,
		ChainIDOptimism,
		ChainIDGnosis,
		ChainIDPythNet,
		ChainIDXpla,
		ChainIDBtc,
		ChainIDBase,
		ChainIDFileCoin,
		ChainIDSei,
		ChainIDRootstock,
		ChainIDScroll,
		ChainIDMantle,
		ChainIDBlast,
		ChainIDXLayer,
		ChainIDLinea,
		ChainIDBerachain,
		ChainIDSeiEVM,
		ChainIDEclipse,
		ChainIDBOB,
		ChainIDSnaxchain,
		ChainIDUnichain,
		ChainIDWorldchain,
		ChainIDInk,
		ChainIDHyperEVM,
		ChainIDMonad,
		ChainIDMovement,
		ChainIDMezo,
		ChainIDFogo,
		ChainIDWormchain,
		ChainIDCosmoshub,
		ChainIDEvmos,
		ChainIDKujira,
		ChainIDNeutron,
		ChainIDCelestia,
		ChainIDStargaze,
		ChainIDSeda,
		ChainIDDymension,
		ChainIDProvenance,
		ChainIDNoble,
		ChainIDSepolia,
		ChainIDArbitrumSepolia,
		ChainIDBaseSepolia,
		ChainIDOptimismSepolia,
		ChainIDHolesky,
		ChainIDPolygonSepolia,
	}
}

// NOTE: Please keep these in numerical order.
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
	// OBSOLETE: ChainIDNeon ChainID = 17
	// ChainIDTerra2 is the ChainID of Terra 2
	ChainIDTerra2 ChainID = 18
	// ChainIDInjective is the ChainID of Injective
	ChainIDInjective ChainID = 19
	// ChainIDOsmosis is the ChainID of Osmosis
	ChainIDOsmosis ChainID = 20
	// ChainIDSui is the ChainID of Sui
	ChainIDSui ChainID = 21
	// ChainIDAptos is the ChainID of Aptos
	ChainIDAptos ChainID = 22
	// ChainIDArbitrum is the ChainID of Arbitrum
	ChainIDArbitrum ChainID = 23
	// ChainIDOptimism is the ChainID of Optimism
	ChainIDOptimism ChainID = 24
	// ChainIDGnosis is the ChainID of Gnosis
	ChainIDGnosis ChainID = 25
	// ChainIDPythNet is the ChainID of PythNet
	ChainIDPythNet ChainID = 26
	// NOTE: 27 belongs to a chain that was never deployed.
	// ChainIDXpla is the ChainID of Xpla
	ChainIDXpla ChainID = 28
	//ChainIDBtc is the ChainID of Bitcoin
	ChainIDBtc ChainID = 29
	// ChainIDBase is the ChainID of Base
	ChainIDBase ChainID = 30
	// ChainIDFileCoin is the ChainID of FileCoin
	ChainIDFileCoin ChainID = 31
	// ChainIDSei is the ChainID of Sei
	ChainIDSei ChainID = 32
	// ChainIDRootstock is the ChainID of Rootstock
	ChainIDRootstock ChainID = 33
	// ChainIDScroll is the ChainID of Scroll
	ChainIDScroll ChainID = 34
	// ChainIDMantle is the ChainID of Mantle
	ChainIDMantle ChainID = 35
	// ChainIDBlast is the ChainID of Blast
	ChainIDBlast ChainID = 36
	// ChainIDXLayer is the ChainID of XLayer
	ChainIDXLayer ChainID = 37
	// ChainIDLinea is the ChainID of Linea
	ChainIDLinea ChainID = 38
	// ChainIDBerachain is the ChainID of Berachain
	ChainIDBerachain ChainID = 39
	// ChainIDSeiEVM is the ChainID of SeiEVM
	ChainIDSeiEVM ChainID = 40
	// ChainIDEclipse is the ChainID of Eclipse
	ChainIDEclipse ChainID = 41
	// ChainIDBOB is the ChainID of BOB
	ChainIDBOB ChainID = 42
	// ChainIDSnaxchain is the ChainID of Snaxchain
	ChainIDSnaxchain ChainID = 43
	// ChainIDUnichain is the ChainID of Unichain
	ChainIDUnichain ChainID = 44
	// ChainIDWorldchain is the ChainID of Worldchain
	ChainIDWorldchain ChainID = 45
	// ChainIDInk is the ChainID of Ink
	ChainIDInk ChainID = 46
	// ChainIDHyperEVM is the ChainID of HyperEVM
	ChainIDHyperEVM ChainID = 47
	// ChainIDMonad is the ChainID of Monad
	ChainIDMonad ChainID = 48
	// ChainIDMovement is the ChainID of Movement
	ChainIDMovement ChainID = 49
	// ChainIDMezo is the ChainID of Mezo
	ChainIDMezo ChainID = 50
	// ChainIDFogo is the ChainID of Fogo
	ChainIDFogo ChainID = 51
	// ChainIDAztec is the ChainID of Aztec
	ChainIDAztec ChainID = 52
	//ChainIDWormchain is the ChainID of Wormchain
	// Wormchain is in it's own range.
	ChainIDWormchain ChainID = 3104

	// The IBC chains start at 4000.
	// ChainIDCosmoshub is the ChainID of Cosmoshub
	ChainIDCosmoshub ChainID = 4000
	// ChainIDEvmos is the ChainID of Evmos
	ChainIDEvmos ChainID = 4001
	// ChainIDKujira is the ChainID of Kujira
	ChainIDKujira ChainID = 4002
	// ChainIDNeutron is the ChainID of Neutron
	ChainIDNeutron ChainID = 4003
	// ChainIDCelestia is the ChainID of Celestia
	ChainIDCelestia ChainID = 4004
	// ChainIDStargaze is the ChainID of Stargaze
	ChainIDStargaze ChainID = 4005
	// ChainIDSeda is the ChainID of Seda
	ChainIDSeda ChainID = 4006
	// ChainIDDymension is the ChainID of Dymension
	ChainIDDymension ChainID = 4007
	// ChainIDProvenance is the ChainID of Provenance
	ChainIDProvenance ChainID = 4008
	// ChainIDNoble is the ChainID of Noble
	ChainIDNoble ChainID = 4009
	// ChainIDSepolia is the ChainID of Sepolia

	// The Testnet only chains start at 10000.
	ChainIDSepolia ChainID = 10002
	// ChainIDArbitrumSepolia is the ChainID of Arbitrum on Sepolia
	ChainIDArbitrumSepolia ChainID = 10003
	// ChainIDBaseSepolia is the ChainID of Base on Sepolia
	ChainIDBaseSepolia ChainID = 10004
	// ChainIDOptimismSepolia is the ChainID of Optimism on Sepolia
	ChainIDOptimismSepolia ChainID = 10005
	// ChainIDHolesky is the ChainID of Holesky
	ChainIDHolesky ChainID = 10006
	// ChainIDPolygonSepolia is the ChainID of Polygon on Sepolia
	ChainIDPolygonSepolia ChainID = 10007
	// OBSOLETE: ChainIDMonadDevnet ChainID = 10008

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

	SupportedVAAVersion = 0x01
)

// UnmarshalBody deserializes the binary representation of a VAA's "BODY" properties
// The BODY fields are common among multiple types of VAA - v1, v2, etc
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

// signingBody returns the binary representation of the data that is relevant for signing and verifying the VAA
func (v *VAA) signingBody() []byte {
	return v.serializeBody()
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
	MustWrite(buf, binary.BigEndian, uint8(len(v.Signatures))) // #nosec G115 -- There will never be 256 guardians
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

// MessageID returns a human-readable emitter_chain/emitter_address/sequence tuple.
func (v *VAA) MessageID() string {
	return fmt.Sprintf("%d/%s/%d", v.EmitterChain, v.EmitterAddress, v.Sequence)
}

// UniqueID normalizes the ID of the VAA (any type) for the Attestation interface
// UniqueID returns the MessageID that uniquely identifies the Attestation
func (v *VAA) UniqueID() string {
	return v.MessageID()
}

// HexDigest returns the hex-encoded digest.
func (v *VAA) HexDigest() string {
	return hex.EncodeToString(v.SigningDigest().Bytes())
}

/*
SECURITY: Do not change this code! Changing it could result in two different hashes for
the same observation. But xDapps rely on the hash of an observation for replay protection.
*/
func (v *VAA) serializeBody() []byte {
	buf := new(bytes.Buffer)
	MustWrite(buf, binary.BigEndian, uint32(v.Timestamp.Unix())) // #nosec G115 -- This conversion is safe until year 2106
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
