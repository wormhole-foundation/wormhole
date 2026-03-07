package query

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"

	solana "github.com/gagliardetto/solana-go"
)

const (
	// MSG_VERSION_V1 is the QueryRequest version prior to permissionless queries
	// and is unused in this version
	MSG_VERSION_V1 uint8 = 1
	// MSG_VERSION_V2 is the current version of the CCQ message protocol.
	// v2 adds timestamp (required) and optional staker address fields.
	MSG_VERSION_V2 uint8 = 2

	// Maximum sizes for untrusted length fields to prevent DoS via memory exhaustion.
	// These limits are enforced during unmarshaling before allocating memory.
	MAX_BLOCK_ID_LEN  = 256   // Max length for block ID strings (hex hash + prefix)
	MAX_FINALITY_LEN  = 128   // Max length for finality strings
	MAX_CALL_DATA_LEN = 65536 // Max length for call data (64KB per call)
	MAX_SEED_LEN      = 256   // Max length for Solana PDA seeds
)

// QueryRequest defines a cross chain query request to be submitted to the guardians.
// It is the payload of the SignedQueryRequest gossip message.
type QueryRequest struct {
	version         uint8 // Message version (set during unmarshal; Marshal always writes v2)
	Nonce           uint32
	Timestamp       uint64             // Unix seconds (REQUIRED for v2)
	StakerAddress   *ethCommon.Address // Optional staker address for delegated queries (nil = self-staking, v2 only)
	PerChainQueries []*PerChainQueryRequest
}

// Version returns the message version. Set during unmarshal; Marshal always writes v2.
func (qr *QueryRequest) Version() uint8 {
	return qr.version
}

// PerChainQueryRequest represents a query request for a single chain.
type PerChainQueryRequest struct {
	// ChainId indicates which chain this query is destine for.
	ChainId vaa.ChainID

	// Query is the chain specific query data.
	Query ChainSpecificQuery
}

// ChainSpecificQuery is the interface that must be implemented by a chain specific query.
type ChainSpecificQuery interface {
	Type() ChainSpecificQueryType
	Marshal() ([]byte, error)
	Unmarshal(data []byte) error
	UnmarshalFromReader(reader *bytes.Reader) error
	Validate() error
}

// ChainSpecificQueryType is used to interpret the data in a per chain query request.
type ChainSpecificQueryType uint8

// EthCallQueryRequestType is the type of an EVM eth_call query request.
const EthCallQueryRequestType ChainSpecificQueryType = 1

// EthCallQueryRequest implements ChainSpecificQuery for an EVM eth_call query request.
type EthCallQueryRequest struct {
	// BlockId identifies the block to be queried. It must be a hex string starting with 0x. It may be a block number or a block hash.
	BlockId string

	// CallData is an array of specific queries to be performed on the specified block, in a single RPC call.
	CallData []*EthCallData
}

func (ecr *EthCallQueryRequest) CallDataList() []*EthCallData {
	return ecr.CallData
}

// EthCallByTimestampQueryRequestType is the type of an EVM eth_call_by_timestamp query request.
const EthCallByTimestampQueryRequestType ChainSpecificQueryType = 2

// EthCallByTimestampQueryRequest implements ChainSpecificQuery for an EVM eth_call_by_timestamp query request.
type EthCallByTimestampQueryRequest struct {
	// TargetTimeInUs specifies the desired timestamp in microseconds.
	TargetTimestamp uint64

	// TargetBlockIdHint is optional. If specified, it identifies the block prior to the desired timestamp. It must be a hex string starting with 0x. It may be a block number or a block hash.
	TargetBlockIdHint string

	// FollowingBlockIdHint is optional. If specified, it identifies the block immediately following the desired timestamp. It must be a hex string starting with 0x. It may be a block number or a block hash.
	FollowingBlockIdHint string

	// CallData is an array of specific queries to be performed on the specified block, in a single RPC call.
	CallData []*EthCallData
}

func (ecr *EthCallByTimestampQueryRequest) CallDataList() []*EthCallData {
	return ecr.CallData
}

// EthCallWithFinalityQueryRequestType is the type of an EVM eth_call_with_finality query request.
const EthCallWithFinalityQueryRequestType ChainSpecificQueryType = 3

// EthCallWithFinalityQueryRequest implements ChainSpecificQuery for an EVM eth_call_with_finality query request.
type EthCallWithFinalityQueryRequest struct {
	// BlockId identifies the block to be queried. It must be a hex string starting with 0x. It may be a block number or a block hash.
	BlockId string

	// Finality is required. It identifies the level of finality the block must reach before the query is performed. Valid values are "finalized" and "safe".
	Finality string

	// CallData is an array of specific queries to be performed on the specified block, in a single RPC call.
	CallData []*EthCallData
}

func (ecr *EthCallWithFinalityQueryRequest) CallDataList() []*EthCallData {
	return ecr.CallData
}

// EthCallData specifies the parameters to a single EVM eth_call request.
type EthCallData struct {
	// To specifies the contract address to be queried.
	To []byte

	// Data is the ABI encoded parameters to the query.
	Data []byte
}

const EvmContractAddressLength = 20

////////////////////////////////// Solana Queries ////////////////////////////////////////////////

// SolanaAccountQueryRequestType is the type of a Solana sol_account query request.
const SolanaAccountQueryRequestType ChainSpecificQueryType = 4

// SolanaAccountQueryRequest implements ChainSpecificQuery for a Solana sol_account query request.
type SolanaAccountQueryRequest struct {
	// Commitment identifies the commitment level to be used in the queried. Currently it may only "finalized".
	// Before we can support "confirmed", we need a way to read the account data and the block information atomically.
	// We would also need to deal with the fact that queries are only handled in the finalized watcher and it does not
	// have access to the latest confirmed slot needed for MinContextSlot retries.
	Commitment string

	// The minimum slot that the request can be evaluated at. Zero means unused.
	MinContextSlot uint64

	// The offset of the start of data to be returned. Unused if DataSliceLength is zero.
	DataSliceOffset uint64

	// The length of the data to be returned. Zero means all data is returned.
	DataSliceLength uint64

	// Accounts is an array of accounts to be queried.
	Accounts [][SolanaPublicKeyLength]byte
}

// Solana public keys are fixed length.
const SolanaPublicKeyLength = solana.PublicKeyLength

// According to the Solana spec, the longest comment string is nine characters. Allow a few more, just in case.
// https://pkg.go.dev/github.com/gagliardetto/solana-go/rpc#CommitmentType
const SolanaMaxCommitmentLength = 12

// According to the spec, the query only supports up to 100 accounts.
// https://github.com/solana-labs/solana/blob/9d132441fdc6282a8be4bff0bc77d6a2fefe8b59/rpc-client-api/src/request.rs#L204
const SolanaMaxAccountsPerQuery = 100

func (saq *SolanaAccountQueryRequest) AccountList() [][SolanaPublicKeyLength]byte {
	return saq.Accounts
}

// SolanaPdaQueryRequestType is the type of a Solana sol_pda query request.
const SolanaPdaQueryRequestType ChainSpecificQueryType = 5

// SolanaPdaQueryRequest implements ChainSpecificQuery for a Solana sol_pda query request.
type SolanaPdaQueryRequest struct {
	// Commitment identifies the commitment level to be used in the queried. Currently it may only "finalized".
	// Before we can support "confirmed", we need a way to read the account data and the block information atomically.
	// We would also need to deal with the fact that queries are only handled in the finalized watcher and it does not
	// have access to the latest confirmed slot needed for MinContextSlot retries.
	Commitment string

	// The minimum slot that the request can be evaluated at. Zero means unused.
	MinContextSlot uint64

	// The offset of the start of data to be returned. Unused if DataSliceLength is zero.
	DataSliceOffset uint64

	// The length of the data to be returned. Zero means all data is returned.
	DataSliceLength uint64

	// PDAs is an array of PDAs to be queried.
	PDAs []SolanaPDAEntry
}

// SolanaPDAEntry defines a single Solana Program derived address (PDA).
type SolanaPDAEntry struct {
	ProgramAddress [SolanaPublicKeyLength]byte
	Seeds          [][]byte
}

// According to the spec, there may be at most 16 seeds.
// https://github.com/gagliardetto/solana-go/blob/6fe3aea02e3660d620433444df033fc3fe6e64c1/keys.go#L559
const SolanaMaxSeeds = solana.MaxSeeds

// According to the spec, a seed may be at most 32 bytes.
// https://github.com/gagliardetto/solana-go/blob/6fe3aea02e3660d620433444df033fc3fe6e64c1/keys.go#L557
const SolanaMaxSeedLen = solana.MaxSeedLength

func (spda *SolanaPdaQueryRequest) PDAList() []SolanaPDAEntry {
	return spda.PDAs
}

// PerChainQueryInternal is an internal representation of a query request that is passed to the watcher.
type PerChainQueryInternal struct {
	RequestID  string
	RequestIdx int
	Request    *PerChainQueryRequest
}

func (pcqi *PerChainQueryInternal) ID() string {
	return fmt.Sprintf("%s:%d", pcqi.RequestID, pcqi.RequestIdx)
}

// QueryRequestDigest returns the query signing prefix based on the environment.
func QueryRequestDigest(env common.Environment, b []byte) ethCommon.Hash {
	var queryRequestPrefix []byte
	if env == common.MainNet {
		queryRequestPrefix = []byte("mainnet_query_request_000000000000|")
	} else if env == common.TestNet {
		queryRequestPrefix = []byte("testnet_query_request_000000000000|")
	} else {
		queryRequestPrefix = []byte("devnet_query_request_0000000000000|")
	}

	return ethCrypto.Keccak256Hash(append(queryRequestPrefix, b...))
}

// PostSignedQueryRequest posts a signed query request to the specified channel.
func PostSignedQueryRequest(signedQueryReqSendC chan<- *gossipv1.SignedQueryRequest, req *gossipv1.SignedQueryRequest) error {
	select {
	case signedQueryReqSendC <- req:
		return nil
	default:
		return common.ErrChanFull
	}
}

func SignedQueryRequestEqual(left *gossipv1.SignedQueryRequest, right *gossipv1.SignedQueryRequest) bool {
	if !bytes.Equal(left.QueryRequest, right.QueryRequest) {
		return false
	}
	if !bytes.Equal(left.Signature, right.Signature) {
		return false
	}
	return true
}

//
// Implementation of QueryRequest.
//

// Marshal serializes the binary representation of a query request.
// This method calls validateForVersion() and relies on it to range check lengths, etc.
func (queryRequest *QueryRequest) Marshal() ([]byte, error) {
	// Marshal always produces v2 format, so validate with v2 rules
	// regardless of what version the struct was unmarshaled as.
	if err := queryRequest.validateForVersion(MSG_VERSION_V2); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)

	vaa.MustWrite(buf, binary.BigEndian, MSG_VERSION_V2)         // version (v2)
	vaa.MustWrite(buf, binary.BigEndian, queryRequest.Nonce)     // uint32
	vaa.MustWrite(buf, binary.BigEndian, queryRequest.Timestamp) // uint64

	// Write staker address with length prefix
	if queryRequest.StakerAddress != nil {
		vaa.MustWrite(buf, binary.BigEndian, uint8(20)) // staker address length
		buf.Write(queryRequest.StakerAddress[:])        // 20 bytes
	} else {
		vaa.MustWrite(buf, binary.BigEndian, uint8(0)) // no staker address (self-staking)
	}

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(queryRequest.PerChainQueries))) // #nosec G115 -- `PerChainQueries` length checked in `Validate`
	for _, perChainQuery := range queryRequest.PerChainQueries {
		pcqBuf, err := perChainQuery.Marshal()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal per chain query: %w", err)
		}
		buf.Write(pcqBuf)
	}

	return buf.Bytes(), nil
}

// Unmarshal deserializes the binary representation of a query request from a byte array
func (queryRequest *QueryRequest) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return queryRequest.UnmarshalFromReader(reader)
}

// UnmarshalFromReader deserializes the binary representation of a query request from an existing reader
func (queryRequest *QueryRequest) UnmarshalFromReader(reader *bytes.Reader) error {
	var version uint8
	if err := binary.Read(reader, binary.BigEndian, &version); err != nil {
		return fmt.Errorf("failed to read message version: %w", err)
	}

	if version != MSG_VERSION_V1 && version != MSG_VERSION_V2 {
		return fmt.Errorf("unsupported message version: %d", version)
	}
	queryRequest.version = version

	if err := binary.Read(reader, binary.BigEndian, &queryRequest.Nonce); err != nil {
		return fmt.Errorf("failed to read request nonce: %w", err)
	}

	// v2 adds timestamp and optional staker address fields.
	// v1 requests lack these fields; Timestamp remains 0 and StakerAddress nil.
	// Call sites that require v2 (e.g. the HTTP handler) enforce this via Validate().
	if version >= MSG_VERSION_V2 {
		if err := binary.Read(reader, binary.BigEndian, &queryRequest.Timestamp); err != nil {
			return fmt.Errorf("failed to read request timestamp: %w", err)
		}

		// Read staker address with length prefix
		var stakerLen uint8
		if err := binary.Read(reader, binary.BigEndian, &stakerLen); err != nil {
			return fmt.Errorf("failed to read staker address length: %w", err)
		}

		if stakerLen == 20 {
			var addr ethCommon.Address
			if n, err := reader.Read(addr[:]); err != nil || n != 20 {
				return fmt.Errorf("failed to read staker address [%d]: %w", n, err)
			}
			queryRequest.StakerAddress = &addr
		} else if stakerLen == 0 {
			queryRequest.StakerAddress = nil
		} else {
			return fmt.Errorf("invalid staker address length: expected 0 or 20, got %d", stakerLen)
		}
	}

	numPerChainQueries := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numPerChainQueries); err != nil {
		return fmt.Errorf("failed to read number of per chain queries: %w", err)
	}

	for count := 0; count < int(numPerChainQueries); count++ {
		perChainQuery := PerChainQueryRequest{}
		err := perChainQuery.UnmarshalFromReader(reader)
		if err != nil {
			return fmt.Errorf("failed to Unmarshal per chain query: %w", err)
		}
		queryRequest.PerChainQueries = append(queryRequest.PerChainQueries, &perChainQuery)
	}

	if reader.Len() != 0 {
		return fmt.Errorf("excess bytes in unmarshal")
	}

	if err := queryRequest.Validate(); err != nil {
		return fmt.Errorf("unmarshaled request failed validation: %w", err)
	}

	return nil
}

// Validate does basic validation on a received query request.
// Version 0 (unset) is treated as v2 so that directly constructed
// QueryRequest structs are validated with v2 rules by default.
func (queryRequest *QueryRequest) Validate() error {
	v := queryRequest.version
	if v == 0 {
		v = MSG_VERSION_V2
	}
	return queryRequest.validateForVersion(v)
}

// validateForVersion runs validation rules appropriate for the given version.
// v2 enforces timestamp freshness and staker address constraints;
// v1 skips those since the fields don't exist in the v1 wire format.
func (queryRequest *QueryRequest) validateForVersion(version uint8) error {
	// Nothing to validate on the Nonce.

	if version != MSG_VERSION_V1 {
		if queryRequest.Timestamp == 0 {
			return fmt.Errorf("timestamp is required")
		}

		now := uint64(time.Now().Unix())                  // #nosec G115 -- time.Now() always returns positive Unix timestamps
		age := int64(now) - int64(queryRequest.Timestamp) // #nosec G115 -- Safe: both values represent reasonable Unix timestamps

		// Reject requests older than 15 minutes
		if age > 900 {
			return fmt.Errorf("request timestamp too old: age %d seconds exceeds maximum of 900 seconds", age)
		}

		// Allow small clock skew (1 minute) for future timestamps
		if age < -60 {
			return fmt.Errorf("request timestamp too far in future: %d seconds ahead of server time", -age)
		}

		// Validate staker address if provided
		if queryRequest.StakerAddress != nil {
			// Reject zero address - it's not a valid staker
			if *queryRequest.StakerAddress == (ethCommon.Address{}) {
				return fmt.Errorf("staker address cannot be the zero address")
			}
		}
	}

	if len(queryRequest.PerChainQueries) <= 0 {
		return fmt.Errorf("request does not contain any per chain queries")
	}
	if len(queryRequest.PerChainQueries) > math.MaxUint8 {
		return fmt.Errorf("too many per chain queries")
	}
	for idx, perChainQuery := range queryRequest.PerChainQueries {
		if err := perChainQuery.Validate(); err != nil {
			return fmt.Errorf("failed to validate per chain query %d: %w", idx, err)
		}
	}
	return nil
}

// Equal verifies semantic equality of two query requests.
// The version field is intentionally excluded — two requests with identical
// nonce, timestamp, staker, and queries are equal regardless of wire version.
func (left *QueryRequest) Equal(right *QueryRequest) bool {
	if left.Nonce != right.Nonce {
		return false
	}

	if left.Timestamp != right.Timestamp {
		return false
	}

	// Compare staker addresses
	if (left.StakerAddress == nil) != (right.StakerAddress == nil) {
		return false
	}
	if left.StakerAddress != nil && *left.StakerAddress != *right.StakerAddress {
		return false
	}

	if len(left.PerChainQueries) != len(right.PerChainQueries) {
		return false
	}

	for idx := range left.PerChainQueries {
		if !left.PerChainQueries[idx].Equal(right.PerChainQueries[idx]) {
			return false
		}
	}
	return true
}

//
// Implementation of PerChainQueryRequest.
//

// Marshal serializes the binary representation of a per chain query request.
// This method calls Validate() and relies on it to range checks lengths, etc.
func (perChainQuery *PerChainQueryRequest) Marshal() ([]byte, error) {
	if err := perChainQuery.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	vaa.MustWrite(buf, binary.BigEndian, perChainQuery.ChainId)
	vaa.MustWrite(buf, binary.BigEndian, perChainQuery.Query.Type())
	queryBuf, err := perChainQuery.Query.Marshal()
	if err != nil {
		return nil, err
	}

	// Write the length of the query to facilitate on-chain parsing.
	if len(queryBuf) > math.MaxUint32 {
		return nil, fmt.Errorf("query too long")
	}
	vaa.MustWrite(buf, binary.BigEndian, uint32(len(queryBuf))) // #nosec G115 -- This conversion is safe as it is checked above

	buf.Write(queryBuf)
	return buf.Bytes(), nil
}

// Unmarshal deserializes the binary representation of a per chain query request from a byte array
func (perChainQuery *PerChainQueryRequest) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return perChainQuery.UnmarshalFromReader(reader)
}

// UnmarshalFromReader deserializes the binary representation of a per chain query request from an existing reader
func (perChainQuery *PerChainQueryRequest) UnmarshalFromReader(reader *bytes.Reader) error {
	if err := binary.Read(reader, binary.BigEndian, &perChainQuery.ChainId); err != nil {
		return fmt.Errorf("failed to read request chain: %w", err)
	}

	qt := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &qt); err != nil {
		return fmt.Errorf("failed to read request type: %w", err)
	}
	queryType := ChainSpecificQueryType(qt)

	if err := ValidatePerChainQueryRequestType(queryType); err != nil {
		return err
	}

	// Read and validate the query length to ensure canonical encoding.
	var queryLength uint32
	if err := binary.Read(reader, binary.BigEndian, &queryLength); err != nil {
		return fmt.Errorf("failed to read query length: %w", err)
	}

	// Record the position before reading the query data
	startPos := int64(reader.Size()) - int64(reader.Len())

	switch queryType {
	case EthCallQueryRequestType:
		q := EthCallQueryRequest{}
		if err := q.UnmarshalFromReader(reader); err != nil {
			return fmt.Errorf("failed to unmarshal eth call request: %w", err)
		}
		perChainQuery.Query = &q
	case EthCallByTimestampQueryRequestType:
		q := EthCallByTimestampQueryRequest{}
		if err := q.UnmarshalFromReader(reader); err != nil {
			return fmt.Errorf("failed to unmarshal eth call by timestamp request: %w", err)
		}
		perChainQuery.Query = &q
	case EthCallWithFinalityQueryRequestType:
		q := EthCallWithFinalityQueryRequest{}
		if err := q.UnmarshalFromReader(reader); err != nil {
			return fmt.Errorf("failed to unmarshal eth call with finality request: %w", err)
		}
		perChainQuery.Query = &q
	case SolanaAccountQueryRequestType:
		q := SolanaAccountQueryRequest{}
		if err := q.UnmarshalFromReader(reader); err != nil {
			return fmt.Errorf("failed to unmarshal solana account query request: %w", err)
		}
		perChainQuery.Query = &q
	case SolanaPdaQueryRequestType:
		q := SolanaPdaQueryRequest{}
		if err := q.UnmarshalFromReader(reader); err != nil {
			return fmt.Errorf("failed to unmarshal solana PDA query request: %w", err)
		}
		perChainQuery.Query = &q
	default:
		return fmt.Errorf("unsupported query type: %d", queryType)
	}

	// Validate that the actual bytes consumed match the declared length.
	endPos := int64(reader.Size()) - int64(reader.Len())
	bytesConsumed := endPos - startPos
	if bytesConsumed < 0 || bytesConsumed > int64(queryLength) {
		return fmt.Errorf("query consumed invalid byte count: %d", bytesConsumed)
	}
	actualLength := uint32(bytesConsumed) // #nosec G115 -- Validated above to be within uint32 range
	if actualLength != queryLength {
		return fmt.Errorf("query length mismatch: declared %d bytes, actual %d bytes", queryLength, actualLength)
	}

	return nil
}

// Validate does basic validation on a per chain query request.
func (perChainQuery *PerChainQueryRequest) Validate() error {
	str := perChainQuery.ChainId.String()
	if _, err := vaa.ChainIDFromString(str); err != nil {
		return fmt.Errorf("invalid chainID: %d", uint16(perChainQuery.ChainId))
	}

	if perChainQuery.Query == nil {
		return fmt.Errorf("query is nil")
	}

	if err := ValidatePerChainQueryRequestType(perChainQuery.Query.Type()); err != nil {
		return err
	}

	if err := perChainQuery.Query.Validate(); err != nil {
		return fmt.Errorf("chain specific query is invalid: %w", err)
	}

	return nil
}

func ValidatePerChainQueryRequestType(qt ChainSpecificQueryType) error {
	if qt != EthCallQueryRequestType && qt != EthCallByTimestampQueryRequestType && qt != EthCallWithFinalityQueryRequestType &&
		qt != SolanaAccountQueryRequestType && qt != SolanaPdaQueryRequestType {
		return fmt.Errorf("invalid query request type: %d", qt)
	}
	return nil
}

// Equal verifies that two query requests are equal.
func (left *PerChainQueryRequest) Equal(right *PerChainQueryRequest) bool {
	if left.ChainId != right.ChainId {
		return false
	}

	if left.Query == nil && right.Query == nil {
		return true
	}

	if left.Query == nil || right.Query == nil {
		return false
	}

	if left.Query.Type() != right.Query.Type() {
		return false
	}

	switch leftQuery := left.Query.(type) {
	case *EthCallQueryRequest:
		switch rightQuery := right.Query.(type) {
		case *EthCallQueryRequest:
			return leftQuery.Equal(rightQuery)
		default:
			panic("unsupported query type on right, must be eth_call")
		}
	case *EthCallByTimestampQueryRequest:
		switch rightQuery := right.Query.(type) {
		case *EthCallByTimestampQueryRequest:
			return leftQuery.Equal(rightQuery)
		default:
			panic("unsupported query type on right, must be eth_call_by_timestamp")
		}
	case *EthCallWithFinalityQueryRequest:
		switch rightQuery := right.Query.(type) {
		case *EthCallWithFinalityQueryRequest:
			return leftQuery.Equal(rightQuery)
		default:
			panic("unsupported query type on right, must be eth_call_with_finality")
		}
	case *SolanaAccountQueryRequest:
		switch rightQuery := right.Query.(type) {
		case *SolanaAccountQueryRequest:
			return leftQuery.Equal(rightQuery)
		default:
			panic("unsupported query type on right, must be sol_account")
		}
	case *SolanaPdaQueryRequest:
		switch rightQuery := right.Query.(type) {
		case *SolanaPdaQueryRequest:
			return leftQuery.Equal(rightQuery)
		default:
			panic("unsupported query type on right, must be sol_pda")
		}
	default:
		panic("unsupported query type on left")
	}
}

//
// Implementation of EthCallQueryRequest, which implements the ChainSpecificQuery interface.
//

func (e *EthCallQueryRequest) Type() ChainSpecificQueryType {
	return EthCallQueryRequestType
}

// Marshal serializes the binary representation of an EVM eth_call request.
// This method calls Validate() and relies on it to range checks lengths, etc.
func (ecd *EthCallQueryRequest) Marshal() ([]byte, error) {
	if err := ecd.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	vaa.MustWrite(buf, binary.BigEndian, uint32(len(ecd.BlockId))) // #nosec G115 -- This is validated in `Validate`
	buf.Write([]byte(ecd.BlockId))

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(ecd.CallData))) // #nosec G115 -- This is validated in `Validate`
	for _, callData := range ecd.CallData {
		buf.Write(callData.To)
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(callData.Data))) // #nosec G115 -- This is validated in `Validate`
		buf.Write(callData.Data)
	}
	return buf.Bytes(), nil
}

// Unmarshal deserializes an EVM eth_call query from a byte array
func (ecd *EthCallQueryRequest) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return ecd.UnmarshalFromReader(reader)
}

// UnmarshalFromReader  deserializes an EVM eth_call query from a byte array
func (ecd *EthCallQueryRequest) UnmarshalFromReader(reader *bytes.Reader) error {
	blockIdLen := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &blockIdLen); err != nil {
		return fmt.Errorf("failed to read block id len: %w", err)
	}

	if blockIdLen > MAX_BLOCK_ID_LEN {
		return fmt.Errorf("block id length %d exceeds maximum %d", blockIdLen, MAX_BLOCK_ID_LEN)
	}

	blockId := make([]byte, blockIdLen)
	if n, err := reader.Read(blockId[:]); err != nil || n != int(blockIdLen) {
		return fmt.Errorf("failed to read block id [%d]: %w", n, err)
	}
	ecd.BlockId = string(blockId[:])

	numCallData := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numCallData); err != nil {
		return fmt.Errorf("failed to read number of call data entries: %w", err)
	}

	for count := 0; count < int(numCallData); count++ {
		to := [EvmContractAddressLength]byte{}
		if n, err := reader.Read(to[:]); err != nil || n != EvmContractAddressLength {
			return fmt.Errorf("failed to read call To [%d]: %w", n, err)
		}

		dataLen := uint32(0)
		if err := binary.Read(reader, binary.BigEndian, &dataLen); err != nil {
			return fmt.Errorf("failed to read call Data len: %w", err)
		}

		if dataLen > MAX_CALL_DATA_LEN {
			return fmt.Errorf("call data length %d exceeds maximum %d", dataLen, MAX_CALL_DATA_LEN)
		}

		data := make([]byte, dataLen)
		if n, err := reader.Read(data[:]); err != nil || n != int(dataLen) {
			return fmt.Errorf("failed to read call data [%d]: %w", n, err)
		}

		callData := &EthCallData{
			To:   to[:],
			Data: data[:],
		}

		ecd.CallData = append(ecd.CallData, callData)
	}

	return nil
}

// Validate does basic validation on an EVM eth_call query.
func (ecd *EthCallQueryRequest) Validate() error {
	if len(ecd.BlockId) > math.MaxUint32 {
		return fmt.Errorf("block id too long")
	}
	if !strings.HasPrefix(ecd.BlockId, "0x") {
		return fmt.Errorf("block id must be a hex number or hash starting with 0x")
	}
	if len(ecd.CallData) <= 0 {
		return fmt.Errorf("does not contain any call data")
	}
	if len(ecd.CallData) > math.MaxUint8 {
		return fmt.Errorf("too many call data entries")
	}
	for _, callData := range ecd.CallData {
		if callData.To == nil || len(callData.To) <= 0 {
			return fmt.Errorf("no call data to")
		}
		if len(callData.To) != EvmContractAddressLength {
			return fmt.Errorf("invalid length for To contract")
		}
		//nolint:dupword // callData.Data is fine in the context of EVM.
		if callData.Data == nil || len(callData.Data) <= 0 {
			return fmt.Errorf("no call data data")
		}
		//nolint:dupword // callData.Data is fine in the context of EVM.
		if len(callData.Data) > math.MaxUint32 {
			return fmt.Errorf("call data data too long")
		}
	}

	return nil
}

// Equal verifies that two EVM eth_call queries are equal.
func (left *EthCallQueryRequest) Equal(right *EthCallQueryRequest) bool {
	if left.BlockId != right.BlockId {
		return false
	}
	if len(left.CallData) != len(right.CallData) {
		return false
	}
	for idx := range left.CallData {
		if !bytes.Equal(left.CallData[idx].To, right.CallData[idx].To) {
			return false
		}
		if !bytes.Equal(left.CallData[idx].Data, right.CallData[idx].Data) {
			return false
		}
	}

	return true
}

//
// Implementation of EthCallByTimestampQueryRequest, which implements the ChainSpecificQuery interface.
//

func (e *EthCallByTimestampQueryRequest) Type() ChainSpecificQueryType {
	return EthCallByTimestampQueryRequestType
}

// Marshal serializes the binary representation of an EVM eth_call_by_timestamp request.
// This method calls Validate() and relies on it to range checks lengths, etc.
func (ecd *EthCallByTimestampQueryRequest) Marshal() ([]byte, error) {
	if err := ecd.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	vaa.MustWrite(buf, binary.BigEndian, ecd.TargetTimestamp)

	vaa.MustWrite(buf, binary.BigEndian, uint32(len(ecd.TargetBlockIdHint))) // #nosec G115 -- This is validated in `Validate`
	buf.Write([]byte(ecd.TargetBlockIdHint))

	vaa.MustWrite(buf, binary.BigEndian, uint32(len(ecd.FollowingBlockIdHint))) // #nosec G115 -- This is validated in `Validate`
	buf.Write([]byte(ecd.FollowingBlockIdHint))

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(ecd.CallData))) // #nosec G115 -- This is validated in `Validate`
	for _, callData := range ecd.CallData {
		buf.Write(callData.To)
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(callData.Data))) // #nosec G115 -- This is validated in `Validate`
		buf.Write(callData.Data)
	}
	return buf.Bytes(), nil
}

// Unmarshal deserializes an EVM eth_call_by_timestamp query from a byte array
func (ecd *EthCallByTimestampQueryRequest) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return ecd.UnmarshalFromReader(reader)
}

// UnmarshalFromReader  deserializes an EVM eth_call_by_timestamp query from a byte array
func (ecd *EthCallByTimestampQueryRequest) UnmarshalFromReader(reader *bytes.Reader) error {
	if err := binary.Read(reader, binary.BigEndian, &ecd.TargetTimestamp); err != nil {
		return fmt.Errorf("failed to read timestamp: %w", err)
	}

	blockIdHintLen := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &blockIdHintLen); err != nil {
		return fmt.Errorf("failed to read target block id hint len: %w", err)
	}

	if blockIdHintLen > MAX_BLOCK_ID_LEN {
		return fmt.Errorf("target block id hint length %d exceeds maximum %d", blockIdHintLen, MAX_BLOCK_ID_LEN)
	}

	targetBlockIdHint := make([]byte, blockIdHintLen)
	if n, err := reader.Read(targetBlockIdHint[:]); err != nil || n != int(blockIdHintLen) {
		return fmt.Errorf("failed to read target block id hint [%d]: %w", n, err)
	}
	ecd.TargetBlockIdHint = string(targetBlockIdHint[:])

	blockIdHintLen = uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &blockIdHintLen); err != nil {
		return fmt.Errorf("failed to read following block id hint len: %w", err)
	}

	if blockIdHintLen > MAX_BLOCK_ID_LEN {
		return fmt.Errorf("following block id hint length %d exceeds maximum %d", blockIdHintLen, MAX_BLOCK_ID_LEN)
	}

	followingBlockIdHint := make([]byte, blockIdHintLen)
	if n, err := reader.Read(followingBlockIdHint[:]); err != nil || n != int(blockIdHintLen) {
		return fmt.Errorf("failed to read following block id hint [%d]: %w", n, err)
	}
	ecd.FollowingBlockIdHint = string(followingBlockIdHint[:])

	numCallData := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numCallData); err != nil {
		return fmt.Errorf("failed to read number of call data entries: %w", err)
	}

	for count := 0; count < int(numCallData); count++ {
		to := [EvmContractAddressLength]byte{}
		if n, err := reader.Read(to[:]); err != nil || n != EvmContractAddressLength {
			return fmt.Errorf("failed to read call To [%d]: %w", n, err)
		}

		dataLen := uint32(0)
		if err := binary.Read(reader, binary.BigEndian, &dataLen); err != nil {
			return fmt.Errorf("failed to read call Data len: %w", err)
		}

		if dataLen > MAX_CALL_DATA_LEN {
			return fmt.Errorf("call data length %d exceeds maximum %d", dataLen, MAX_CALL_DATA_LEN)
		}

		data := make([]byte, dataLen)
		if n, err := reader.Read(data[:]); err != nil || n != int(dataLen) {
			return fmt.Errorf("failed to read call data [%d]: %w", n, err)
		}

		callData := &EthCallData{
			To:   to[:],
			Data: data[:],
		}

		ecd.CallData = append(ecd.CallData, callData)
	}

	return nil
}

// Validate does basic validation on an EVM eth_call_by_timestamp query.
func (ecd *EthCallByTimestampQueryRequest) Validate() error {
	if ecd.TargetTimestamp == 0 {
		return fmt.Errorf("target timestamp may not be zero")
	}
	if len(ecd.TargetBlockIdHint) > math.MaxUint32 {
		return fmt.Errorf("target block id hint too long")
	}
	if (ecd.TargetBlockIdHint == "") != (ecd.FollowingBlockIdHint == "") {
		return fmt.Errorf("if either the target or following block id is unset, they both must be unset")
	}
	if ecd.TargetBlockIdHint != "" && !strings.HasPrefix(ecd.TargetBlockIdHint, "0x") {
		return fmt.Errorf("target block id must be a hex number or hash starting with 0x")
	}
	if len(ecd.FollowingBlockIdHint) > math.MaxUint32 {
		return fmt.Errorf("following block id hint too long")
	}
	if ecd.FollowingBlockIdHint != "" && !strings.HasPrefix(ecd.FollowingBlockIdHint, "0x") {
		return fmt.Errorf("following block id must be a hex number or hash starting with 0x")
	}
	if len(ecd.CallData) <= 0 {
		return fmt.Errorf("does not contain any call data")
	}
	if len(ecd.CallData) > math.MaxUint8 {
		return fmt.Errorf("too many call data entries")
	}
	for _, callData := range ecd.CallData {
		if callData.To == nil || len(callData.To) <= 0 {
			return fmt.Errorf("no call data to")
		}
		if len(callData.To) != EvmContractAddressLength {
			return fmt.Errorf("invalid length for To contract")
		}
		//nolint:dupword // callData.Data is fine in the context of EVM.
		if callData.Data == nil || len(callData.Data) <= 0 {
			return fmt.Errorf("no call data data")
		}
		//nolint:dupword // callData.Data is fine in the context of EVM.
		if len(callData.Data) > math.MaxUint32 {
			return fmt.Errorf("call data data too long")
		}
	}

	return nil
}

// Equal verifies that two EVM eth_call_by_timestamp queries are equal.
func (left *EthCallByTimestampQueryRequest) Equal(right *EthCallByTimestampQueryRequest) bool {
	if left.TargetTimestamp != right.TargetTimestamp {
		return false
	}
	if left.TargetBlockIdHint != right.TargetBlockIdHint {
		return false
	}
	if left.FollowingBlockIdHint != right.FollowingBlockIdHint {
		return false
	}
	if len(left.CallData) != len(right.CallData) {
		return false
	}
	for idx := range left.CallData {
		if !bytes.Equal(left.CallData[idx].To, right.CallData[idx].To) {
			return false
		}
		if !bytes.Equal(left.CallData[idx].Data, right.CallData[idx].Data) {
			return false
		}
	}

	return true
}

//
// Implementation of EthCallWithFinalityQueryRequest, which implements the ChainSpecificQuery interface.
//

func (e *EthCallWithFinalityQueryRequest) Type() ChainSpecificQueryType {
	return EthCallWithFinalityQueryRequestType
}

// Marshal serializes the binary representation of an EVM eth_call_with_finality request.
// This method calls Validate() and relies on it to range checks lengths, etc.
func (ecd *EthCallWithFinalityQueryRequest) Marshal() ([]byte, error) {
	if err := ecd.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	vaa.MustWrite(buf, binary.BigEndian, uint32(len(ecd.BlockId))) // #nosec G115 -- This is validated in `Validate`
	buf.Write([]byte(ecd.BlockId))

	vaa.MustWrite(buf, binary.BigEndian, uint32(len(ecd.Finality))) // #nosec G115 -- This is validated in `Validate`
	buf.Write([]byte(ecd.Finality))

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(ecd.CallData))) // #nosec G115 -- This is validated in `Validate`
	for _, callData := range ecd.CallData {
		buf.Write(callData.To)
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(callData.Data))) // #nosec G115 -- This is validated in `Validate`
		buf.Write(callData.Data)
	}
	return buf.Bytes(), nil
}

// Unmarshal deserializes an EVM eth_call_with_finality query from a byte array
func (ecd *EthCallWithFinalityQueryRequest) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return ecd.UnmarshalFromReader(reader)
}

// UnmarshalFromReader  deserializes an EVM eth_call_with_finality query from a byte array
func (ecd *EthCallWithFinalityQueryRequest) UnmarshalFromReader(reader *bytes.Reader) error {
	blockIdLen := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &blockIdLen); err != nil {
		return fmt.Errorf("failed to read target block id len: %w", err)
	}

	if blockIdLen > MAX_BLOCK_ID_LEN {
		return fmt.Errorf("block id length %d exceeds maximum %d", blockIdLen, MAX_BLOCK_ID_LEN)
	}

	blockId := make([]byte, blockIdLen)
	if n, err := reader.Read(blockId[:]); err != nil || n != int(blockIdLen) {
		return fmt.Errorf("failed to read target block id [%d]: %w", n, err)
	}
	ecd.BlockId = string(blockId[:])

	finalityLen := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &finalityLen); err != nil {
		return fmt.Errorf("failed to read finality len: %w", err)
	}

	if finalityLen > MAX_FINALITY_LEN {
		return fmt.Errorf("finality length %d exceeds maximum %d", finalityLen, MAX_FINALITY_LEN)
	}

	finality := make([]byte, finalityLen)
	if n, err := reader.Read(finality[:]); err != nil || n != int(finalityLen) {
		return fmt.Errorf("failed to read finality [%d]: %w", n, err)
	}
	ecd.Finality = string(finality[:])

	numCallData := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numCallData); err != nil {
		return fmt.Errorf("failed to read number of call data entries: %w", err)
	}

	for count := 0; count < int(numCallData); count++ {
		to := [EvmContractAddressLength]byte{}
		if n, err := reader.Read(to[:]); err != nil || n != EvmContractAddressLength {
			return fmt.Errorf("failed to read call To [%d]: %w", n, err)
		}

		dataLen := uint32(0)
		if err := binary.Read(reader, binary.BigEndian, &dataLen); err != nil {
			return fmt.Errorf("failed to read call Data len: %w", err)
		}

		if dataLen > MAX_CALL_DATA_LEN {
			return fmt.Errorf("call data length %d exceeds maximum %d", dataLen, MAX_CALL_DATA_LEN)
		}

		data := make([]byte, dataLen)
		if n, err := reader.Read(data[:]); err != nil || n != int(dataLen) {
			return fmt.Errorf("failed to read call data [%d]: %w", n, err)
		}

		callData := &EthCallData{
			To:   to[:],
			Data: data[:],
		}

		ecd.CallData = append(ecd.CallData, callData)
	}

	return nil
}

// Validate does basic validation on an EVM eth_call_with_finality query.
func (ecd *EthCallWithFinalityQueryRequest) Validate() error {
	if len(ecd.BlockId) > math.MaxUint32 {
		return fmt.Errorf("block id too long")
	}
	if ecd.BlockId == "" {
		return fmt.Errorf("block id is required")
	}
	if !strings.HasPrefix(ecd.BlockId, "0x") {
		return fmt.Errorf("block id must be a hex number or hash starting with 0x")
	}
	if len(ecd.Finality) > math.MaxUint32 {
		return fmt.Errorf("finality too long")
	}
	if ecd.Finality == "" {
		return fmt.Errorf("finality is required")
	}
	if ecd.Finality != "finalized" && ecd.Finality != "safe" {
		return fmt.Errorf(`finality must be "finalized" or "safe", is "%s"`, ecd.Finality)
	}
	if len(ecd.CallData) <= 0 {
		return fmt.Errorf("does not contain any call data")
	}
	if len(ecd.CallData) > math.MaxUint8 {
		return fmt.Errorf("too many call data entries")
	}
	for _, callData := range ecd.CallData {
		if callData.To == nil || len(callData.To) <= 0 {
			return fmt.Errorf("no call data to")
		}
		if len(callData.To) != EvmContractAddressLength {
			return fmt.Errorf("invalid length for To contract")
		}

		//nolint:dupword // callData.Data is fine in the context of EVM.
		if callData.Data == nil || len(callData.Data) <= 0 {
			return fmt.Errorf("no call data data")
		}
		//nolint:dupword // callData.Data is fine in the context of EVM.
		if len(callData.Data) > math.MaxUint32 {
			return fmt.Errorf("call data data too long")
		}
	}

	return nil
}

// Equal verifies that two EVM eth_call_with_finality queries are equal.
func (left *EthCallWithFinalityQueryRequest) Equal(right *EthCallWithFinalityQueryRequest) bool {
	if left.BlockId != right.BlockId {
		return false
	}
	if left.Finality != right.Finality {
		return false
	}
	if len(left.CallData) != len(right.CallData) {
		return false
	}
	for idx := range left.CallData {
		if !bytes.Equal(left.CallData[idx].To, right.CallData[idx].To) {
			return false
		}
		if !bytes.Equal(left.CallData[idx].Data, right.CallData[idx].Data) {
			return false
		}
	}

	return true
}

//
// Implementation of SolanaAccountQueryRequest, which implements the ChainSpecificQuery interface.
//

func (e *SolanaAccountQueryRequest) Type() ChainSpecificQueryType {
	return SolanaAccountQueryRequestType
}

// Marshal serializes the binary representation of a Solana sol_account request.
// This method calls Validate() and relies on it to range checks lengths, etc.
func (saq *SolanaAccountQueryRequest) Marshal() ([]byte, error) {
	if err := saq.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)

	vaa.MustWrite(buf, binary.BigEndian, uint32(len(saq.Commitment))) // #nosec G115 -- This is validated in `Validate`
	buf.Write([]byte(saq.Commitment))

	vaa.MustWrite(buf, binary.BigEndian, saq.MinContextSlot)
	vaa.MustWrite(buf, binary.BigEndian, saq.DataSliceOffset)
	vaa.MustWrite(buf, binary.BigEndian, saq.DataSliceLength)

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(saq.Accounts))) // #nosec G115 -- This is validated in `Validate`
	for _, acct := range saq.Accounts {
		buf.Write(acct[:])
	}
	return buf.Bytes(), nil
}

// Unmarshal deserializes a Solana sol_account query from a byte array
func (saq *SolanaAccountQueryRequest) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return saq.UnmarshalFromReader(reader)
}

// UnmarshalFromReader  deserializes a Solana sol_account query from a byte array
func (saq *SolanaAccountQueryRequest) UnmarshalFromReader(reader *bytes.Reader) error {
	length := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
		return fmt.Errorf("failed to read commitment len: %w", err)
	}

	if length > SolanaMaxCommitmentLength {
		return fmt.Errorf("commitment string is too long, may not be more than %d characters", SolanaMaxCommitmentLength)
	}

	commitment := make([]byte, length)
	if n, err := reader.Read(commitment[:]); err != nil || n != int(length) {
		return fmt.Errorf("failed to read commitment [%d]: %w", n, err)
	}
	saq.Commitment = string(commitment)

	if err := binary.Read(reader, binary.BigEndian, &saq.MinContextSlot); err != nil {
		return fmt.Errorf("failed to read min slot: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &saq.DataSliceOffset); err != nil {
		return fmt.Errorf("failed to read data slice offset: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &saq.DataSliceLength); err != nil {
		return fmt.Errorf("failed to read data slice length: %w", err)
	}

	numAccounts := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numAccounts); err != nil {
		return fmt.Errorf("failed to read number of account entries: %w", err)
	}

	for count := 0; count < int(numAccounts); count++ {
		account := [SolanaPublicKeyLength]byte{}
		if n, err := reader.Read(account[:]); err != nil || n != SolanaPublicKeyLength {
			return fmt.Errorf("failed to read account [%d]: %w", n, err)
		}
		saq.Accounts = append(saq.Accounts, account)
	}

	return nil
}

// Validate does basic validation on a Solana sol_account query.
func (saq *SolanaAccountQueryRequest) Validate() error {
	if len(saq.Commitment) > SolanaMaxCommitmentLength {
		return fmt.Errorf("commitment too long")
	}
	if saq.Commitment != "finalized" {
		return fmt.Errorf(`commitment must be "finalized"`)
	}

	if saq.DataSliceLength == 0 && saq.DataSliceOffset != 0 {
		return fmt.Errorf("data slice offset may not be set if data slice length is zero")
	}

	if len(saq.Accounts) <= 0 {
		return fmt.Errorf("does not contain any account entries")
	}
	if len(saq.Accounts) > SolanaMaxAccountsPerQuery {
		return fmt.Errorf("too many account entries, may not be more than %d", SolanaMaxAccountsPerQuery)
	}
	for _, acct := range saq.Accounts {
		// The account is fixed length, so don't need to check for nil.
		if len(acct) != SolanaPublicKeyLength {
			return fmt.Errorf("invalid account length")
		}
	}

	return nil
}

// Equal verifies that two Solana sol_account queries are equal.
func (left *SolanaAccountQueryRequest) Equal(right *SolanaAccountQueryRequest) bool {
	if left.Commitment != right.Commitment ||
		left.MinContextSlot != right.MinContextSlot ||
		left.DataSliceOffset != right.DataSliceOffset ||
		left.DataSliceLength != right.DataSliceLength {
		return false
	}

	if len(left.Accounts) != len(right.Accounts) {
		return false
	}
	for idx := range left.Accounts {
		if !bytes.Equal(left.Accounts[idx][:], right.Accounts[idx][:]) {
			return false
		}
	}

	return true
}

//
// Implementation of SolanaPdaQueryRequest, which implements the ChainSpecificQuery interface.
//

func (e *SolanaPdaQueryRequest) Type() ChainSpecificQueryType {
	return SolanaPdaQueryRequestType
}

// Marshal serializes the binary representation of a Solana sol_pda request.
// This method calls Validate() and relies on it to range checks lengths, etc.
func (spda *SolanaPdaQueryRequest) Marshal() ([]byte, error) {
	if err := spda.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)

	vaa.MustWrite(buf, binary.BigEndian, uint32(len(spda.Commitment))) // #nosec G115 -- This is validated in `Validate`
	buf.Write([]byte(spda.Commitment))

	vaa.MustWrite(buf, binary.BigEndian, spda.MinContextSlot)
	vaa.MustWrite(buf, binary.BigEndian, spda.DataSliceOffset)
	vaa.MustWrite(buf, binary.BigEndian, spda.DataSliceLength)

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(spda.PDAs))) // #nosec G115 -- This is validated in `Validate`
	for _, pda := range spda.PDAs {
		buf.Write(pda.ProgramAddress[:])
		vaa.MustWrite(buf, binary.BigEndian, uint8(len(pda.Seeds))) // #nosec G115 -- This is validated in `Validate`
		for _, seed := range pda.Seeds {
			vaa.MustWrite(buf, binary.BigEndian, uint32(len(seed))) // #nosec G115 -- This is validated in `Validate`
			buf.Write(seed)
		}
	}
	return buf.Bytes(), nil
}

// Unmarshal deserializes a Solana sol_pda query from a byte array
func (spda *SolanaPdaQueryRequest) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return spda.UnmarshalFromReader(reader)
}

// UnmarshalFromReader  deserializes a Solana sol_pda query from a byte array
func (spda *SolanaPdaQueryRequest) UnmarshalFromReader(reader *bytes.Reader) error {
	length := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
		return fmt.Errorf("failed to read commitment len: %w", err)
	}

	if length > SolanaMaxCommitmentLength {
		return fmt.Errorf("commitment string is too long, may not be more than %d characters", SolanaMaxCommitmentLength)
	}

	commitment := make([]byte, length)
	if n, err := reader.Read(commitment[:]); err != nil || n != int(length) {
		return fmt.Errorf("failed to read commitment [%d]: %w", n, err)
	}
	spda.Commitment = string(commitment)

	if err := binary.Read(reader, binary.BigEndian, &spda.MinContextSlot); err != nil {
		return fmt.Errorf("failed to read min slot: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &spda.DataSliceOffset); err != nil {
		return fmt.Errorf("failed to read data slice offset: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &spda.DataSliceLength); err != nil {
		return fmt.Errorf("failed to read data slice length: %w", err)
	}

	numPDAs := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numPDAs); err != nil {
		return fmt.Errorf("failed to read number of PDAs: %w", err)
	}

	for count := 0; count < int(numPDAs); count++ {
		programAddress := [SolanaPublicKeyLength]byte{}
		if n, err := reader.Read(programAddress[:]); err != nil || n != SolanaPublicKeyLength {
			return fmt.Errorf("failed to read program address [%d]: %w", n, err)
		}

		pda := SolanaPDAEntry{ProgramAddress: programAddress}
		numSeeds := uint8(0)
		if err := binary.Read(reader, binary.BigEndian, &numSeeds); err != nil {
			return fmt.Errorf("failed to read number of seeds: %w", err)
		}

		for count := 0; count < int(numSeeds); count++ {
			seedLen := uint32(0)
			if err := binary.Read(reader, binary.BigEndian, &seedLen); err != nil {
				return fmt.Errorf("failed to read call Data len: %w", err)
			}

			if seedLen > MAX_SEED_LEN {
				return fmt.Errorf("seed length %d exceeds maximum %d", seedLen, MAX_SEED_LEN)
			}

			seed := make([]byte, seedLen)
			if n, err := reader.Read(seed[:]); err != nil || n != int(seedLen) {
				return fmt.Errorf("failed to read seed [%d]: %w", n, err)
			}

			pda.Seeds = append(pda.Seeds, seed)
		}

		spda.PDAs = append(spda.PDAs, pda)
	}

	return nil
}

// Validate does basic validation on a Solana sol_pda query.
func (spda *SolanaPdaQueryRequest) Validate() error {
	if len(spda.Commitment) > SolanaMaxCommitmentLength {
		return fmt.Errorf("commitment too long")
	}
	if spda.Commitment != "finalized" {
		return fmt.Errorf(`commitment must be "finalized"`)
	}

	if spda.DataSliceLength == 0 && spda.DataSliceOffset != 0 {
		return fmt.Errorf("data slice offset may not be set if data slice length is zero")
	}

	if len(spda.PDAs) <= 0 {
		return fmt.Errorf("does not contain any PDAs entries")
	}
	if len(spda.PDAs) > SolanaMaxAccountsPerQuery {
		return fmt.Errorf("too many PDA entries, may not be more than %d", SolanaMaxAccountsPerQuery)
	}
	for _, pda := range spda.PDAs {
		// The program address is fixed length, so don't need to check for nil.
		if len(pda.ProgramAddress) != SolanaPublicKeyLength {
			return fmt.Errorf("invalid program address length")
		}

		if len(pda.Seeds) == 0 {
			return fmt.Errorf("PDA does not contain any seeds")
		}

		if len(pda.Seeds) > SolanaMaxSeeds {
			return fmt.Errorf("PDA contains too many seeds")
		}

		for _, seed := range pda.Seeds {
			if len(seed) == 0 {
				return fmt.Errorf("seed is null")
			}

			if len(seed) > SolanaMaxSeedLen {
				return fmt.Errorf("seed is too long")
			}
		}
	}

	return nil
}

// Equal verifies that two Solana sol_pda queries are equal.
func (left *SolanaPdaQueryRequest) Equal(right *SolanaPdaQueryRequest) bool {
	if left.Commitment != right.Commitment ||
		left.MinContextSlot != right.MinContextSlot ||
		left.DataSliceOffset != right.DataSliceOffset ||
		left.DataSliceLength != right.DataSliceLength {
		return false
	}

	if len(left.PDAs) != len(right.PDAs) {
		return false
	}
	for idx := range left.PDAs {
		if !bytes.Equal(left.PDAs[idx].ProgramAddress[:], right.PDAs[idx].ProgramAddress[:]) {
			return false
		}

		if len(left.PDAs[idx].Seeds) != len(right.PDAs[idx].Seeds) {
			return false
		}

		for idx2 := range left.PDAs[idx].Seeds {
			if !bytes.Equal(left.PDAs[idx].Seeds[idx2][:], right.PDAs[idx].Seeds[idx2][:]) {
				return false
			}
		}
	}

	return true
}
