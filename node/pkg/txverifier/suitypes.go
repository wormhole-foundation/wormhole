package txverifier

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"regexp"

	"github.com/certusone/wormhole/node/pkg/suiclient"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// The following structs mirror the on-chain Move types emitted/stored by the Sui core
// and token bridges. Their field order and types must match the Move definitions exactly
// so that BCS decoding of the gRPC-provided bytes succeeds. The layouts were validated
// against live mainnet objects.
//
// BCS notes:
//   - A Move `Balance<T>`/`Supply<T>` wraps a single `u64` and serializes as a bare u64.
//   - `wormhole::bytes32::Bytes32 { data: vector<u8> }` is a length-prefixed vector, NOT a
//     fixed array, so it decodes into a Go []byte (the length is always 32).
//   - A Move `UID`/`ID`/`address` serializes as a fixed 32-byte value (no length prefix).
//   - `token_registry::Key<C>` is an empty Move struct, which Sui serializes as a single
//     `dummy_field: bool` (1 byte).

// wormhole::bytes32::Bytes32 { data: vector<u8> }
type suiBytes32 struct {
	Data []byte
}

// wormhole::external_address::ExternalAddress { value: Bytes32 }
type suiExternalAddress struct {
	Value suiBytes32
}

//	token_bridge::native_asset::NativeAsset<C> {
//		custody: Balance<C>,            // serialized as u64
//		token_address: ExternalAddress,
//		decimals: u8,
//	}
type suiNativeAsset struct {
	Custody      uint64
	TokenAddress suiExternalAddress
	Decimals     uint8
}

//	token_bridge::wrapped_asset::ForeignInfo<C> {
//		token_chain: u16,
//		token_address: ExternalAddress,
//		native_decimals: u8,
//		symbol: String,                // serialized as vector<u8>
//	}
type suiForeignInfo struct {
	TokenChain     vaa.ChainID
	TokenAddress   suiExternalAddress
	NativeDecimals uint8
	Symbol         []byte
}

// 0x2::coin::TreasuryCap<C> { id: UID, total_supply: Supply<C> }.
// UID serializes as a 32-byte address and Supply<C> as a u64.
type suiTreasuryCap struct {
	ID          [32]byte
	TotalSupply uint64
}

//	token_bridge::wrapped_asset::WrappedAsset<C> {
//		info: ForeignInfo<C>,
//		treasury_cap: TreasuryCap<C>,
//		decimals: u8,
//		upgrade_cap: UpgradeCap,        // trailing; not needed and left undecoded
//	}
type suiWrappedAsset struct {
	Info        suiForeignInfo
	TreasuryCap suiTreasuryCap
	Decimals    uint8
}

//	0x2::dynamic_field::Field<Name, Value> {
//		id: UID,        // 32-byte address
//		name: Name,     // token_registry::Key<C>, an empty struct -> 1-byte dummy bool
//		value: Value,
//	}
type suiNativeAssetField struct {
	ID        [32]byte
	DummyName bool
	Value     suiNativeAsset
}

type suiWrappedAssetField struct {
	ID        [32]byte
	DummyName bool
	Value     suiWrappedAsset
}

// WormholeMessage mirrors the on-chain `publish_message::WormholeMessage` event emitted by
// the Sui core bridge. Field order and types must match the Move struct exactly so that BCS
// decoding of the gRPC event contents succeeds.
type WormholeMessage struct {
	Sender           [32]byte
	Sequence         uint64
	Nonce            uint32
	Payload          []byte
	ConsistencyLevel uint8
	Timestamp        uint64
}

// suiAssetInfo is the normalized token-registry asset data extracted from an object's BCS
// contents. The balance is the custodied amount for native assets, or the wrapped token's
// total supply for wrapped assets.
type suiAssetInfo struct {
	isWrapped    bool
	decimals     uint8
	tokenAddress string // hex-encoded, without a 0x prefix
	tokenChain   vaa.ChainID
	balance      *big.Int
}

// suiAssetType is the fully-qualified `<module>::<Type>` name of a token-registry asset value
// type. Only the two constants below are valid values.
type suiAssetType string

const (
	suiNativeAssetType  suiAssetType = "native_asset::NativeAsset"
	suiWrappedAssetType suiAssetType = "wrapped_asset::WrappedAsset"
)

// suiDynamicFieldTypeRegex matches a token-registry dynamic field object type of the form
//
//	0x...2::dynamic_field::Field<<pkg>::token_registry::Key<<coin>>,<pkg>::<asset>::<Asset><<coin>>>
//
// The framework address may be abbreviated (0x2) or fully padded (0x000...0002), and the
// comma separating the two type arguments may or may not be followed by whitespace; the gRPC
// API uses the padded address and no space, whereas the legacy JSON-RPC API used the short
// form and a trailing space.
var suiDynamicFieldTypeRegex = regexp.MustCompile(`^0x0*2::dynamic_field::Field<([^:]+)::token_registry::Key<([^>]+)>,\s*([^:]+)::([^<]+)<([^>]+)>>$`)

// Errors identifying which parseSuiAssetType check rejected an object type.
var (
	errNotTokenRegistryField = errors.New("object type is not a token-registry dynamic field")
	errNotAssetType          = errors.New("asset value type is not a native or wrapped token-bridge asset")
	errWrongPackageId        = errors.New("package ID does not match the token bridge package ID")
	errMismatchedCoinTypes   = errors.New("key coin type does not match the asset value coin type")
)

// parseSuiAssetType validates the type information of a token-registry dynamic field object and
// returns the validated asset value type. The following checks are performed:
//   - the type matches the token-registry dynamic field shape
//   - the asset type is a wrapped or native token-bridge asset
//   - both package IDs match the expected token bridge package ID
//   - the coin type referenced by the field key matches the coin type of the asset value
//
// On validation failure it returns the sentinel error for the check that failed.
func parseSuiAssetType(objectType string, expectedPackageId string) (suiAssetType, error) {
	matches := suiDynamicFieldTypeRegex.FindStringSubmatch(objectType)

	if len(matches) != 6 {
		return "", errNotTokenRegistryField
	}

	scanPackage1 := matches[1]
	scanCoinType1 := matches[2]
	scanPackage2 := matches[3]
	scanAssetType := suiAssetType(matches[4])
	scanCoinType2 := matches[5]

	if scanAssetType != suiWrappedAssetType && scanAssetType != suiNativeAssetType {
		return "", errNotAssetType
	}

	if scanPackage1 != expectedPackageId || scanPackage2 != expectedPackageId {
		return "", errWrongPackageId
	}

	if scanCoinType1 != scanCoinType2 {
		return "", errMismatchedCoinTypes
	}

	return scanAssetType, nil
}

// decodeSuiAssetObject decodes a token-registry dynamic field object's BCS contents into
// normalized asset info. `assetType` selects the decode layout.
func decodeSuiAssetObject(assetType suiAssetType, contents []byte) (*suiAssetInfo, error) {
	switch assetType {
	case suiWrappedAssetType:
		field, err := suiclient.DecodeBcs[suiWrappedAssetField](contents)
		if err != nil {
			return nil, fmt.Errorf("failed to BCS-decode WrappedAsset: %w", err)
		}

		chain, err := vaa.KnownChainIDFromNumber(field.Value.Info.TokenChain)
		if err != nil {
			return nil, fmt.Errorf("failed to convert token chain %d to a known chain id: %w", field.Value.Info.TokenChain, err)
		}

		return &suiAssetInfo{
			isWrapped:    true,
			decimals:     field.Value.Decimals,
			tokenAddress: hex.EncodeToString(field.Value.Info.TokenAddress.Value.Data),
			tokenChain:   chain,
			balance:      new(big.Int).SetUint64(field.Value.TreasuryCap.TotalSupply),
		}, nil

	case suiNativeAssetType:
		field, err := suiclient.DecodeBcs[suiNativeAssetField](contents)
		if err != nil {
			return nil, fmt.Errorf("failed to BCS-decode NativeAsset: %w", err)
		}

		return &suiAssetInfo{
			isWrapped:    false,
			decimals:     field.Value.Decimals,
			tokenAddress: hex.EncodeToString(field.Value.TokenAddress.Value.Data),
			tokenChain:   vaa.ChainIDSui,
			balance:      new(big.Int).SetUint64(field.Value.Custody),
		}, nil

	default:
		return nil, fmt.Errorf("object type is neither a native nor wrapped token-bridge asset: %s", assetType)
	}
}
