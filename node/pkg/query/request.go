package query

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
)

// MSG_VERSION is the current version of the CCQ message protocol.
const MSG_VERSION uint8 = 1

// QueryRequest defines a cross chain query request to be submitted to the guardians.
// It is the payload of the SignedQueryRequest gossip message.
type QueryRequest struct {
	Nonce           uint32
	PerChainQueries []*PerChainQueryRequest
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

//
// Implementation of QueryRequest.
//

// Marshal serializes the binary representation of a query request.
// This method calls Validate() and relies on it to range checks lengths, etc.
func (queryRequest *QueryRequest) Marshal() ([]byte, error) {
	if err := queryRequest.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)

	vaa.MustWrite(buf, binary.BigEndian, MSG_VERSION)        // version
	vaa.MustWrite(buf, binary.BigEndian, queryRequest.Nonce) // uint32

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(queryRequest.PerChainQueries)))
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

	if version != MSG_VERSION {
		return fmt.Errorf("unsupported message version: %d", version)
	}

	if err := binary.Read(reader, binary.BigEndian, &queryRequest.Nonce); err != nil {
		return fmt.Errorf("failed to read request nonce: %w", err)
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

	return nil
}

// Validate does basic validation on a received query request.
func (queryRequest *QueryRequest) Validate() error {
	// Nothing to validate on the Nonce.
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

// Equal verifies that two query requests are equal.
func (left *QueryRequest) Equal(right *QueryRequest) bool {
	if left.Nonce != right.Nonce {
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
	vaa.MustWrite(buf, binary.BigEndian, uint32(len(queryBuf)))

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

	// Skip the query length.
	var queryLength uint32
	if err := binary.Read(reader, binary.BigEndian, &queryLength); err != nil {
		return fmt.Errorf("failed to read query length: %w", err)
	}

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
	default:
		return fmt.Errorf("unsupported query type: %d", queryType)
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

	switch leftEcq := left.Query.(type) {
	case *EthCallQueryRequest:
		switch rightEcd := right.Query.(type) {
		case *EthCallQueryRequest:
			return leftEcq.Equal(rightEcd)
		default:
			panic("unsupported query type on right, must be eth_call")
		}
	case *EthCallByTimestampQueryRequest:
		switch rightEcd := right.Query.(type) {
		case *EthCallByTimestampQueryRequest:
			return leftEcq.Equal(rightEcd)
		default:
			panic("unsupported query type on right, must be eth_call_by_timestamp")
		}
	case *EthCallWithFinalityQueryRequest:
		switch rightEcd := right.Query.(type) {
		case *EthCallWithFinalityQueryRequest:
			return leftEcq.Equal(rightEcd)
		default:
			panic("unsupported query type on right, must be eth_call_with_finality")
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
	vaa.MustWrite(buf, binary.BigEndian, uint32(len(ecd.BlockId)))
	buf.Write([]byte(ecd.BlockId))

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(ecd.CallData)))
	for _, callData := range ecd.CallData {
		buf.Write(callData.To)
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(callData.Data)))
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
		if callData.Data == nil || len(callData.Data) <= 0 {
			return fmt.Errorf("no call data data")
		}
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

	vaa.MustWrite(buf, binary.BigEndian, uint32(len(ecd.TargetBlockIdHint)))
	buf.Write([]byte(ecd.TargetBlockIdHint))

	vaa.MustWrite(buf, binary.BigEndian, uint32(len(ecd.FollowingBlockIdHint)))
	buf.Write([]byte(ecd.FollowingBlockIdHint))

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(ecd.CallData)))
	for _, callData := range ecd.CallData {
		buf.Write(callData.To)
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(callData.Data)))
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

	targetBlockIdHint := make([]byte, blockIdHintLen)
	if n, err := reader.Read(targetBlockIdHint[:]); err != nil || n != int(blockIdHintLen) {
		return fmt.Errorf("failed to read target block id hint [%d]: %w", n, err)
	}
	ecd.TargetBlockIdHint = string(targetBlockIdHint[:])

	blockIdHintLen = uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &blockIdHintLen); err != nil {
		return fmt.Errorf("failed to read following block id hint len: %w", err)
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
		if callData.Data == nil || len(callData.Data) <= 0 {
			return fmt.Errorf("no call data data")
		}
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
	vaa.MustWrite(buf, binary.BigEndian, uint32(len(ecd.BlockId)))
	buf.Write([]byte(ecd.BlockId))

	vaa.MustWrite(buf, binary.BigEndian, uint32(len(ecd.Finality)))
	buf.Write([]byte(ecd.Finality))

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(ecd.CallData)))
	for _, callData := range ecd.CallData {
		buf.Write(callData.To)
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(callData.Data)))
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

	blockId := make([]byte, blockIdLen)
	if n, err := reader.Read(blockId[:]); err != nil || n != int(blockIdLen) {
		return fmt.Errorf("failed to read target block id [%d]: %w", n, err)
	}
	ecd.BlockId = string(blockId[:])

	finalityLen := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &finalityLen); err != nil {
		return fmt.Errorf("failed to read finality len: %w", err)
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
		if callData.Data == nil || len(callData.Data) <= 0 {
			return fmt.Errorf("no call data data")
		}
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

func ValidatePerChainQueryRequestType(qt ChainSpecificQueryType) error {
	if qt != EthCallQueryRequestType && qt != EthCallByTimestampQueryRequestType && qt != EthCallWithFinalityQueryRequestType {
		return fmt.Errorf("invalid query request type: %d", qt)
	}
	return nil
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
