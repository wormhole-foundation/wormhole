package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
)

// QueryRequest defines a cross chain query request to be submitted to the guardians.
// It is the payload of the SignedQueryRequest gossip message.
type QueryRequest struct {
	Nonce           uint32
	PerChainQueries []*PerChainQueryRequest
}

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
	// BlockId identifies the block to be queried. It mus be a hex string starting with 0x. It may be a block number or a block hash.
	BlockId string

	// CallData is an array of specific queries to be performed on the specified block, in a single RPC call.
	CallData []*EthCallData
}

// EthCallData specifies the parameters to a single EVM eth_call request.
type EthCallData struct {
	// To specifies the contract address to be queried.
	To []byte

	// Data is the ABI encoded parameters to the query.
	Data []byte
}

const SignedQueryRequestChannelSize = 50
const EvmContractAddressLength = 20

// PerChainQueryInternal is an internal representation of a query request that is passed to the watcher.
type PerChainQueryInternal struct {
	RequestID  string
	RequestIdx int
	Request    *PerChainQueryRequest
}

// QueryRequestDigest returns the query signing prefix based on the environment.
func QueryRequestDigest(env Environment, b []byte) ethCommon.Hash {
	// TODO: should this use a different standard of signing messages, like https://eips.ethereum.org/EIPS/eip-712
	var queryRequestPrefix []byte
	if env == MainNet {
		queryRequestPrefix = []byte("mainnet_query_request_000000000000|")
	} else if env == TestNet {
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
		return ErrChanFull
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
	if len(queryRequest.PerChainQueries) == 0 {
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
		return fmt.Errorf("failed to read request chain: %w", err)
	}
	queryType := ChainSpecificQueryType(qt)

	if err := ValidatePerChainQueryRequestType(queryType); err != nil {
		return err
	}

	switch queryType {
	case EthCallQueryRequestType:
		q := EthCallQueryRequest{}
		if err := q.UnmarshalFromReader(reader); err != nil {
			return fmt.Errorf("failed to read request chain: %w", err)
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
			panic("unsupported query type on right") // We checked this above!
		}
	default:
		panic("unsupported query type on left") // We checked this above!
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

// UnmarshalEthCallQueryRequest deserializes an EVM eth_call query from a byte array
func (ecd *EthCallQueryRequest) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return ecd.UnmarshalFromReader(reader)
}

// UnmarshalFromReader  deserializes an EVM eth_call query from a byte array
func (ecd *EthCallQueryRequest) UnmarshalFromReader(reader *bytes.Reader) error {
	blockIdLen := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &blockIdLen); err != nil {
		return fmt.Errorf("failed to read call Data len: %w", err)
	}

	blockId := make([]byte, blockIdLen)
	if n, err := reader.Read(blockId[:]); err != nil || n != int(blockIdLen) {
		return fmt.Errorf("failed to read call To [%d]: %w", n, err)
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
			return fmt.Errorf("failed to read call To [%d]: %w", n, err)
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
	if len(ecd.CallData) == 0 {
		return fmt.Errorf("does not contain any call data")
	}
	if len(ecd.CallData) > math.MaxUint8 {
		return fmt.Errorf("too many call data entries")
	}
	for _, callData := range ecd.CallData {
		if callData.To == nil || len(callData.To) == 0 {
			return fmt.Errorf("no call data to")
		}
		if len(callData.To) != EvmContractAddressLength {
			return fmt.Errorf("invalid length for To contract")
		}
		if callData.Data == nil || len(callData.Data) == 0 {
			return fmt.Errorf("no call data data")
		}
		if len(callData.Data) > math.MaxUint32 {
			return fmt.Errorf("request data too long")
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

func ValidatePerChainQueryRequestType(qt ChainSpecificQueryType) error {
	if qt != EthCallQueryRequestType {
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
