package txverifier

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

const SUI_CHAIN_ID = 21

// The SuiApi interface defines the functions that are required to interact with the Sui RPC.
type SuiApiInterface interface {
	QueryEvents(filter string, cursor string, limit int, descending bool) (SuiQueryEventsResponse, error)
	GetTransactionBlock(txDigest string) (SuiGetTransactionBlockResponse, error)
	TryMultiGetPastObjects(objectId string, version string, previousVersion string) (SuiTryMultiGetPastObjectsResponse, error)
}

// This struct defines the standard properties that get returned from the RPC.
// It includes the ErrorMessage and Error fields as well, with a standard implementation
// of a `GetError()` function. `suiApiRequest` requires `GetError()` for standard
// API error handling.
type SuiApiStandardResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	// error_msg is typically populated when a non-api-related error occurs (like ratelimiting)
	ErrorMessage *string `json:"error_msg"`
	// error is typically populated when an api-related error occurs
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (e SuiApiStandardResponse) GetError() error {
	if e.ErrorMessage != nil {
		return fmt.Errorf("error from Sui RPC: %s", *e.ErrorMessage)
	}

	if e.Error != nil {
		return fmt.Errorf("error from Sui RPC: %s", e.Error.Message)
	}

	return nil
}

// The response object for suix_queryEvents
type SuiQueryEventsResponse struct {
	SuiApiStandardResponse
	Result SuiQueryEventsResult `json:"result"`
}

type SuiQueryEventsResult struct {
	Data []SuiEvent `json:"data"`
}

type SuiEvent struct {
	ID struct {
		TxDigest *string `json:"txDigest"`
		EventSeq *string `json:"eventSeq"`
	} `json:"id"`
	PackageID         *string `json:"packageId"`
	TransactionModule *string `json:"transactionModule"`
	Sender            *string `json:"sender"`
	Type              *string `json:"type"`
	// Bcs               *string          `json:"bcs"`
	Timestamp *string          `json:"timestampMs"`
	Message   *WormholeMessage `json:"parsedJson"`
}

// The response object for sui_GetTransactionBlock
type SuiGetTransactionBlockResponse struct {
	SuiApiStandardResponse
	Result SuiGetTransactionBlockResult `json:"result"`
}

type SuiGetTransactionBlockResult struct {
	ObjectChanges []ObjectChange `json:"objectChanges"`
	Events        []SuiEvent     `json:"events"`
}

type ObjectChange struct {
	ObjectType      string `json:"objectType"`
	ObjectId        string `json:"objectId"`
	Version         string `json:"version"`
	PreviousVersion string `json:"previousVersion"`
}

// Validate the type information of the object change. The following checks are performed:
//   - pass the object through a regex that extracts the package ID, coin type, and asset type
//   - ensure that the asset type is wrapped or native
//   - ensure that the package IDs match the expected package ID
//   - ensure that the coin types match
func (o ObjectChange) ValidateTypeInformation(expectedPackageId string) (success bool) {
	// AI generated regex
	re := regexp.MustCompile(`^0x2::dynamic_field::Field<([^:]+)::token_registry::Key<([^>]+)>, ([^:]+)::([^<]+)<([^>]+)>>$`)
	matches := re.FindStringSubmatch(o.ObjectType)

	if len(matches) == 6 {
		scanPackage1 := matches[1]
		scanCoinType1 := matches[2]
		scanPackage2 := matches[3]
		scanAssetType := matches[4]
		scanCoinType2 := matches[5]

		// Ensure that the asset type is wrapped or native
		if scanAssetType != "wrapped_asset::WrappedAsset" && scanAssetType != "native_asset::NativeAsset" {
			return false
		}

		// Ensure that the package IDs match the expected package ID
		if scanPackage1 != expectedPackageId || scanPackage2 != expectedPackageId {
			return false
		}

		// Ensure that the coin types match
		if scanCoinType1 != scanCoinType2 {
			return false
		}

		return true
	}

	// No matches were found
	return false
}

// The response object for suix_tryMultiGetPastObjects
type SuiTryMultiGetPastObjectsResponse struct {
	SuiApiStandardResponse
	Result []SuiTryMultiGetPastObjectsResult `json:"result"`
}

// Gets the balance difference of the two result objects.
func (r SuiTryMultiGetPastObjectsResponse) GetBalanceDiff() (*big.Int, error) {

	if len(r.Result) != 2 {
		return big.NewInt(0), fmt.Errorf("incorrect number of results received")
	}

	// Determine if the asset is wrapped or native
	isWrapped, err := r.Result[0].IsWrapped()
	if err != nil {
		return big.NewInt(0), fmt.Errorf("error in checking if object is wrapped: %w", err)
	}

	// TODO: Should we check that the other asset is also wrapped?
	newBalance, err := r.Result[0].GetVersionBalance(isWrapped)
	if err != nil {
		return big.NewInt(0), fmt.Errorf("error in getting new balance: %w", err)
	}

	oldBalance, err := r.Result[1].GetVersionBalance(isWrapped)
	if err != nil {
		return big.NewInt(0), fmt.Errorf("error in getting old balance: %w", err)
	}

	difference := newBalance.Sub(newBalance, oldBalance)
	// If the asset is wrapped, it means that the balance was burned, so the difference should be negative.
	if isWrapped {
		difference = difference.Mul(difference, big.NewInt(-1))
	}

	return difference, nil
}

// Gets the decimals
func (r SuiTryMultiGetPastObjectsResponse) GetDecimals() (uint8, error) {
	decimals0, err0 := r.Result[0].GetDecimals()
	decimals1, err1 := r.Result[1].GetDecimals()

	if err0 != nil {
		return 0, fmt.Errorf("error in getting decimals: %w", err0)
	} else if err1 != nil {
		return 0, fmt.Errorf("error in getting decimals: %w", err1)
	} else if decimals0 != decimals1 {
		return 0, fmt.Errorf("decimals do not match")
	}

	return decimals0, nil
}

func (r SuiTryMultiGetPastObjectsResponse) GetTokenAddress() (string, error) {
	tokenAddress0, err0 := r.Result[0].GetTokenAddress()
	tokenAddress1, err1 := r.Result[1].GetTokenAddress()

	if err0 != nil {
		return "", fmt.Errorf("error in getting token address: %w", err0)
	} else if err1 != nil {
		return "", fmt.Errorf("error in getting token address: %w", err1)
	} else if tokenAddress0 != tokenAddress1 {
		return "", fmt.Errorf("token addresses do not match")
	}

	return tokenAddress0, nil
}

func (r SuiTryMultiGetPastObjectsResponse) GetTokenChain() (uint16, error) {
	chain0, err0 := r.Result[0].GetTokenChain()
	chain1, err1 := r.Result[1].GetTokenChain()

	if err0 != nil {
		return 0, fmt.Errorf("error in getting token chain: %w", err0)
	} else if err1 != nil {
		return 0, fmt.Errorf("error in getting token chain: %w", err1)
	} else if chain0 != chain1 {
		return 0, fmt.Errorf("token chain ids do not match")
	}

	return chain0, nil
}

func (r SuiTryMultiGetPastObjectsResponse) GetObjectId() (string, error) {
	objectId, err := r.Result[0].GetObjectId()
	if err != nil {
		return "", fmt.Errorf("could not get object id")
	}

	return objectId, nil
}

func (r SuiTryMultiGetPastObjectsResponse) GetVersion() (string, error) {
	version, err := r.Result[0].GetVersion()
	if err != nil {
		return "", fmt.Errorf("could not get object id")
	}

	return version, nil
}

func (r SuiTryMultiGetPastObjectsResponse) GetPreviousVersion() (string, error) {
	previousVersion, err := r.Result[1].GetVersion()
	if err != nil {
		return "", fmt.Errorf("could not get object id")
	}

	return previousVersion, nil
}

func (r SuiTryMultiGetPastObjectsResponse) GetObjectType() (string, error) {
	type0, err0 := r.Result[0].GetObjectType()
	type1, err1 := r.Result[1].GetObjectType()

	if err0 != nil {
		return "", fmt.Errorf("error in getting token chain: %w", err0)
	} else if err1 != nil {
		return "", fmt.Errorf("error in getting token chain: %w", err1)
	} else if type0 != type1 {
		return "", fmt.Errorf("token chain ids do not match")
	}

	return type0, nil
}

// The result object for suix_tryMultiGetPastObjects.
type SuiTryMultiGetPastObjectsResult struct {
	Status  string           `json:"status"`
	Details *json.RawMessage `json:"details"`
}

// Check if the result object is wrapped.
func (r SuiTryMultiGetPastObjectsResult) IsWrapped() (bool, error) {
	path := "content.type"
	objectType, err := extractFromJsonPath[string](*r.Details, path)

	if err != nil {
		return false, fmt.Errorf("error in extracting object type: %w", err)
	}

	return strings.Contains(objectType, "wrapped_asset::WrappedAsset"), nil
}

// Get the balance of the result object.
func (r SuiTryMultiGetPastObjectsResult) GetVersionBalance(isWrapped bool) (*big.Int, error) {

	var path string
	supplyInt := big.NewInt(0)

	// The path to use for a native asset
	pathNative := "content.fields.value.fields.custody"

	// The path to use for a wrapped asset
	pathWrapped := "content.fields.value.fields.treasury_cap.fields.total_supply.fields.value"

	if isWrapped {
		path = pathWrapped
	} else {
		path = pathNative
	}

	supply, err := extractFromJsonPath[string](*r.Details, path)

	if err != nil {
		return supplyInt, fmt.Errorf("error in extracting wormhole balance: %w", err)
	}

	supplyInt, success := supplyInt.SetString(supply, 10)

	if !success {
		return supplyInt, fmt.Errorf("error converting supply to int: %w", err)
	}

	return supplyInt, nil
}

// Get the result object's decimals.
func (r SuiTryMultiGetPastObjectsResult) GetDecimals() (uint8, error) {
	// token_bridge::wrapped_asset::decimals() and token_bridge::native_asset::decimals()
	// both store the decimals used for truncation in the NativeAsset or WrappedAsset's `decimals()` field
	path := "content.fields.value.fields.decimals"

	decimals, err := extractFromJsonPath[float64](*r.Details, path)

	if err != nil {
		return 0, fmt.Errorf("error in extracting decimals: %w", err)
	}

	return uint8(decimals), nil
}

// Get the result object's token address. This will be the address of the token
// on it's chain of origin.
func (r SuiTryMultiGetPastObjectsResult) GetTokenAddress() (tokenAddress string, err error) {
	var path string

	// The path to use for a native asset
	pathNative := "content.fields.value.fields.token_address.fields.value.fields.data"

	// The path to use for a wrapped asset
	pathWrapped := "content.fields.value.fields.info.fields.token_address.fields.value.fields.data"

	wrapped, err := r.IsWrapped()

	if err != nil {
		return "", fmt.Errorf("error in checking if object is wrapped: %w", err)
	}

	if wrapped {
		path = pathWrapped
	} else {
		path = pathNative
	}

	data, err := extractFromJsonPath[[]interface{}](*r.Details, path)

	if err != nil {
		return "", fmt.Errorf("error in extracting token address: %w", err)
	}

	// data is of type []interface{}, and each element is of type float64.
	// We need to covnert each element to a byte, and then convert the []byte to a hex string.
	addrBytes := make([]byte, len(data))

	for i, v := range data {
		if f, ok := v.(float64); ok {
			addrBytes[i] = byte(f)
		} else {
			return "", fmt.Errorf("error in converting token data to float type")
		}
	}

	return hex.EncodeToString(addrBytes), nil
}

// Get the token's chain ID. This will be the chain ID of the network the token
// originated from.
func (r SuiTryMultiGetPastObjectsResult) GetTokenChain() (uint16, error) {

	wrapped, err := r.IsWrapped()

	if err != nil {
		return 0, fmt.Errorf("error in checking if object is wrapped: %w", err)
	}

	if !wrapped {
		return SUI_CHAIN_ID, nil
	}

	path := "content.fields.value.fields.info.fields.token_chain"

	chain, err := extractFromJsonPath[float64](*r.Details, path)

	if err != nil {
		return 0, fmt.Errorf("error in extracting chain: %w", err)
	}

	return uint16(chain), nil
}

func (r SuiTryMultiGetPastObjectsResult) GetObjectId() (string, error) {
	path := "objectId"

	objectId, err := extractFromJsonPath[string](*r.Details, path)

	if err != nil {
		return "", fmt.Errorf("error in extracting objectId: %w", err)
	}

	return objectId, nil
}

func (r SuiTryMultiGetPastObjectsResult) GetVersion() (string, error) {
	path := "version"

	version, err := extractFromJsonPath[string](*r.Details, path)

	if err != nil {
		return "", fmt.Errorf("error in extracting version: %w", err)
	}

	return version, nil
}

func (r SuiTryMultiGetPastObjectsResult) GetObjectType() (string, error) {
	path := "type"

	version, err := extractFromJsonPath[string](*r.Details, path)

	if err != nil {
		return "", fmt.Errorf("error in extracting version: %w", err)
	}

	return version, nil
}

// Definition of the WormholeMessage event
type WormholeMessage struct {
	ConsistencyLevel *uint8  `json:"consistency_level"`
	Nonce            *uint64 `json:"nonce"`
	Payload          []byte  `json:"payload"`
	Sender           *string `json:"sender"`
	Sequence         *string `json:"sequence"`
	Timestamp        *string `json:"timestamp"`
}
