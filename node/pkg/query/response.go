package query

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// QueryStatus is the status returned from the watcher to the query handler.
type QueryStatus int

const (
	// QuerySuccess means the query was successful and the response should be returned to the requester.
	QuerySuccess QueryStatus = 1

	// QueryRetryNeeded means the query failed, but a retry may be helpful.
	QueryRetryNeeded QueryStatus = 0

	// QueryFatalError means the query failed, and there is no point in retrying it.
	QueryFatalError QueryStatus = -1
)

// This is the query response returned from the watcher to the query handler.
type PerChainQueryResponseInternal struct {
	RequestID  string
	RequestIdx int
	ChainId    vaa.ChainID
	Status     QueryStatus
	Response   ChainSpecificResponse
}

// CreatePerChainQueryResponseInternal creates a PerChainQueryResponseInternal and returns a pointer to it.
func CreatePerChainQueryResponseInternal(reqId string, reqIdx int, chainId vaa.ChainID, status QueryStatus, response ChainSpecificResponse) *PerChainQueryResponseInternal {
	return &PerChainQueryResponseInternal{
		RequestID:  reqId,
		RequestIdx: reqIdx,
		ChainId:    chainId,
		Status:     status,
		Response:   response,
	}
}

var queryResponsePrefix = []byte("query_response_0000000000000000000|")

// QueryResponsePublication is the response to a QueryRequest.
type QueryResponsePublication struct {
	Request           *gossipv1.SignedQueryRequest
	PerChainResponses []*PerChainQueryResponse
}

// PerChainQueryResponse represents a query response for a single chain.
type PerChainQueryResponse struct {
	// ChainId indicates which chain this query was destine for.
	ChainId vaa.ChainID

	// Response is the chain specific query data.
	Response ChainSpecificResponse
}

// ChainSpecificResponse is the interface that must be implemented by a chain specific response.
type ChainSpecificResponse interface {
	Type() ChainSpecificQueryType
	Marshal() ([]byte, error)
	Unmarshal(data []byte) error
	UnmarshalFromReader(reader *bytes.Reader) error
	Validate() error
}

// EthCallQueryResponse implements ChainSpecificResponse for an EVM eth_call query response.
type EthCallQueryResponse struct {
	BlockNumber uint64
	Hash        common.Hash
	Time        time.Time

	// Results is the array of responses matching CallData in EthCallQueryRequest
	Results [][]byte
}

// EthCallByTimestampQueryResponse implements ChainSpecificResponse for an EVM eth_call_by_timestamp query response.
type EthCallByTimestampQueryResponse struct {
	TargetBlockNumber    uint64
	TargetBlockHash      common.Hash
	TargetBlockTime      time.Time
	FollowingBlockNumber uint64
	FollowingBlockHash   common.Hash
	FollowingBlockTime   time.Time

	// Results is the array of responses matching CallData in EthCallByTimestampQueryRequest
	Results [][]byte
}

// EthCallWithFinalityQueryResponse implements ChainSpecificResponse for an EVM eth_call_with_finality query response.
type EthCallWithFinalityQueryResponse struct {
	BlockNumber uint64
	Hash        common.Hash
	Time        time.Time

	// Results is the array of responses matching CallData in EthCallQueryRequest
	Results [][]byte
}

// SolanaAccountQueryResponse implements ChainSpecificResponse for a Solana sol_account query response.
type SolanaAccountQueryResponse struct {
	// SlotNumber is the slot number returned by the sol_account query
	SlotNumber uint64

	// BlockTime is the block time associated with the slot.
	BlockTime time.Time

	// BlockHash is the block hash associated with the slot.
	BlockHash [SolanaPublicKeyLength]byte

	Results []SolanaAccountResult
}

type SolanaAccountResult struct {
	// Lamports is the number of lamports assigned to the account.
	Lamports uint64

	// RentEpoch is the epoch at which this account will next owe rent.
	RentEpoch uint64

	// Executable is a boolean indicating if the account contains a program (and is strictly read-only).
	Executable bool

	// Owner is the public key of the owner of the account.
	Owner [SolanaPublicKeyLength]byte

	// Data is the data returned by the sol_account query.
	Data []byte
}

// SolanaPdaQueryResponse implements ChainSpecificResponse for a Solana sol_pda query response.
type SolanaPdaQueryResponse struct {
	// SlotNumber is the slot number returned by the sol_pda query
	SlotNumber uint64

	// BlockTime is the block time associated with the slot.
	BlockTime time.Time

	// BlockHash is the block hash associated with the slot.
	BlockHash [SolanaPublicKeyLength]byte

	Results []SolanaPdaResult
}

type SolanaPdaResult struct {
	// Account is the public key of the account derived from the PDA.
	Account [SolanaPublicKeyLength]byte

	// Bump is the bump value returned by the solana derivation function.
	Bump uint8

	// Lamports is the number of lamports assigned to the account.
	Lamports uint64

	// RentEpoch is the epoch at which this account will next owe rent.
	RentEpoch uint64

	// Executable is a boolean indicating if the account contains a program (and is strictly read-only).
	Executable bool

	// Owner is the public key of the owner of the account.
	Owner [SolanaPublicKeyLength]byte

	// Data is the data returned by the sol_pda query.
	Data []byte
}

//
// Implementation of QueryResponsePublication.
//

// Marshal serializes the binary representation of a query response.
// This method calls Validate() and relies on it to range checks lengths, etc.
func (msg *QueryResponsePublication) Marshal() ([]byte, error) {
	if err := msg.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)

	vaa.MustWrite(buf, binary.BigEndian, uint8(1)) // version

	// Source
	// TODO: support writing off-chain and on-chain requests
	// Here, unset represents an off-chain request
	vaa.MustWrite(buf, binary.BigEndian, vaa.ChainIDUnset)

	buf.Write(msg.Request.Signature[:])

	// Write the length of the request to facilitate on-chain parsing.
	if len(msg.Request.QueryRequest) > math.MaxUint32 {
		return nil, fmt.Errorf("request too long")
	}
	vaa.MustWrite(buf, binary.BigEndian, uint32(len(msg.Request.QueryRequest))) // #nosec G115 -- This is validated above

	buf.Write(msg.Request.QueryRequest)

	// Per chain responses
	vaa.MustWrite(buf, binary.BigEndian, uint8(len(msg.PerChainResponses))) // #nosec G115 -- This is validated in `Validate`
	for idx := range msg.PerChainResponses {
		pcrBuf, err := msg.PerChainResponses[idx].Marshal()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal per chain response: %w", err)
		}
		buf.Write(pcrBuf)
	}

	return buf.Bytes(), nil
}

// Unmarshal deserializes the binary representation of a query response
func (msg *QueryResponsePublication) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])

	var version uint8
	if err := binary.Read(reader, binary.BigEndian, &version); err != nil {
		return fmt.Errorf("failed to read message version: %w", err)
	}

	if version != 1 {
		return fmt.Errorf("unsupported message version: %d", version)
	}

	// Request
	requestChain := vaa.ChainID(0)
	if err := binary.Read(reader, binary.BigEndian, &requestChain); err != nil {
		return fmt.Errorf("failed to read request chain: %w", err)
	}
	if requestChain != vaa.ChainIDUnset {
		// TODO: support reading off-chain and on-chain requests
		return fmt.Errorf("unsupported request chain: %d", requestChain)
	}

	signedQueryRequest := &gossipv1.SignedQueryRequest{}
	signature := [65]byte{}
	if n, err := reader.Read(signature[:]); err != nil || n != 65 {
		return fmt.Errorf("failed to read signature [%d]: %w", n, err)
	}
	signedQueryRequest.Signature = signature[:]

	// Read the serialized request.
	queryRequestLen := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &queryRequestLen); err != nil {
		return fmt.Errorf("failed to read length of query request: %w", err)
	}

	queryRequestBytes := make([]byte, queryRequestLen)
	if n, err := reader.Read(queryRequestBytes[:]); err != nil || n != int(queryRequestLen) {
		return fmt.Errorf("failed to read query request [%d]: %w", n, err)
	}

	queryRequest := QueryRequest{}
	queryRequestReader := bytes.NewReader(queryRequestBytes[:])
	err := queryRequest.UnmarshalFromReader(queryRequestReader)
	if err != nil {
		return fmt.Errorf("failed to unmarshal query request: %w", err)
	}

	queryRequestBytes, err = queryRequest.Marshal()
	if err != nil {
		return err
	}
	signedQueryRequest.QueryRequest = queryRequestBytes
	msg.Request = signedQueryRequest

	// Responses
	numPerChainResponses := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numPerChainResponses); err != nil {
		return fmt.Errorf("failed to read number of per chain responses: %w", err)
	}

	for count := 0; count < int(numPerChainResponses); count++ {
		var pcr PerChainQueryResponse
		err := pcr.UnmarshalFromReader(reader)
		if err != nil {
			return fmt.Errorf("failed to unmarshal per chain response: %w", err)
		}
		msg.PerChainResponses = append(msg.PerChainResponses, &pcr)
	}

	if reader.Len() != 0 {
		return fmt.Errorf("excess bytes in unmarshal")
	}

	if err := msg.Validate(); err != nil {
		return fmt.Errorf("unmarshaled response failed validation: %w", err)
	}

	return nil
}

// Validate does basic validation on a received query request.
func (msg *QueryResponsePublication) Validate() error {
	// Unmarshal and validate the contained query request.
	var queryRequest QueryRequest
	err := queryRequest.Unmarshal(msg.Request.QueryRequest)
	if err != nil {
		return fmt.Errorf("failed to unmarshal query request: %w", err)
	}
	if err := queryRequest.Validate(); err != nil {
		return fmt.Errorf("query request is invalid: %w", err)
	}

	if len(msg.PerChainResponses) <= 0 {
		return fmt.Errorf("response does not contain any per chain responses")
	}
	if len(msg.PerChainResponses) > math.MaxUint8 {
		return fmt.Errorf("too many per chain responses")
	}
	if len(msg.PerChainResponses) != len(queryRequest.PerChainQueries) {
		return fmt.Errorf("number of responses does not match number of queries")
	}
	for idx, pcr := range msg.PerChainResponses {
		if err := pcr.Validate(); err != nil {
			return fmt.Errorf("failed to validate per chain query %d: %w", idx, err)
		}
		if pcr.Response.Type() != queryRequest.PerChainQueries[idx].Query.Type() {
			return fmt.Errorf("type of response %d does not match the query", idx)
		}
	}
	return nil
}

// Equal checks for equality on two query response publications.
func (left *QueryResponsePublication) Equal(right *QueryResponsePublication) bool {
	if !bytes.Equal(left.Request.QueryRequest, right.Request.QueryRequest) || !bytes.Equal(left.Request.Signature, right.Request.Signature) {
		return false
	}
	if len(left.PerChainResponses) != len(right.PerChainResponses) {
		return false
	}
	for idx := range left.PerChainResponses {
		if !left.PerChainResponses[idx].Equal(right.PerChainResponses[idx]) {
			return false
		}
	}
	return true
}

func (resp *QueryResponsePublication) Signature() string {
	if resp == nil || resp.Request == nil {
		return "nil"
	}
	return hex.EncodeToString(resp.Request.Signature)
}

// Similar to sdk/vaa/structs.go,
// In order to save space in the solana signature verification instruction, we hash twice so we only need to pass in
// the first hash (32 bytes) vs the full body data.
func (msg *QueryResponsePublication) SigningDigest() (common.Hash, error) {
	msgBytes, err := msg.Marshal()
	if err != nil {
		return common.Hash{}, err
	}
	return GetQueryResponseDigestFromBytes(msgBytes), nil
}

// GetQueryResponseDigestFromBytes computes the digest bytes for a query response byte array.
func GetQueryResponseDigestFromBytes(b []byte) common.Hash {
	return crypto.Keccak256Hash(append(queryResponsePrefix, crypto.Keccak256Hash(b).Bytes()...))
}

//
// Implementation of PerChainQueryResponse.
//

// Marshal marshalls a per chain query response.
func (perChainResponse *PerChainQueryResponse) Marshal() ([]byte, error) {
	if err := perChainResponse.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	vaa.MustWrite(buf, binary.BigEndian, perChainResponse.ChainId)
	vaa.MustWrite(buf, binary.BigEndian, perChainResponse.Response.Type())
	respBuf, err := perChainResponse.Response.Marshal()
	if err != nil {
		return nil, err
	}

	// Write the length of the response to facilitate on-chain parsing.
	if len(respBuf) > math.MaxUint32 {
		return nil, fmt.Errorf("response is too long")
	}
	vaa.MustWrite(buf, binary.BigEndian, uint32(len(respBuf))) // #nosec G115 -- This is validated above
	buf.Write(respBuf)
	return buf.Bytes(), nil
}

// Unmarshal deserializes the binary representation of a per chain query response from a byte array
func (perChainResponse *PerChainQueryResponse) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return perChainResponse.UnmarshalFromReader(reader)
}

// UnmarshalFromReader deserializes the binary representation of a per chain query response from an existing reader
func (perChainResponse *PerChainQueryResponse) UnmarshalFromReader(reader *bytes.Reader) error {
	if err := binary.Read(reader, binary.BigEndian, &perChainResponse.ChainId); err != nil {
		return fmt.Errorf("failed to read response chain: %w", err)
	}

	qt := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &qt); err != nil {
		return fmt.Errorf("failed to read response type: %w", err)
	}
	queryType := ChainSpecificQueryType(qt)

	if err := ValidatePerChainQueryRequestType(queryType); err != nil {
		return err
	}

	// Skip the response length.
	var respLength uint32
	if err := binary.Read(reader, binary.BigEndian, &respLength); err != nil {
		return fmt.Errorf("failed to read response length: %w", err)
	}

	switch queryType {
	case EthCallQueryRequestType:
		r := EthCallQueryResponse{}
		if err := r.UnmarshalFromReader(reader); err != nil {
			return fmt.Errorf("failed to unmarshal eth call response: %w", err)
		}
		perChainResponse.Response = &r
	case EthCallByTimestampQueryRequestType:
		r := EthCallByTimestampQueryResponse{}
		if err := r.UnmarshalFromReader(reader); err != nil {
			return fmt.Errorf("failed to unmarshal eth call by timestamp response: %w", err)
		}
		perChainResponse.Response = &r
	case EthCallWithFinalityQueryRequestType:
		r := EthCallWithFinalityQueryResponse{}
		if err := r.UnmarshalFromReader(reader); err != nil {
			return fmt.Errorf("failed to unmarshal eth call with finality response: %w", err)
		}
		perChainResponse.Response = &r
	case SolanaAccountQueryRequestType:
		r := SolanaAccountQueryResponse{}
		if err := r.UnmarshalFromReader(reader); err != nil {
			return fmt.Errorf("failed to unmarshal sol_account response: %w", err)
		}
		perChainResponse.Response = &r
	case SolanaPdaQueryRequestType:
		r := SolanaPdaQueryResponse{}
		if err := r.UnmarshalFromReader(reader); err != nil {
			return fmt.Errorf("failed to unmarshal sol_account response: %w", err)
		}
		perChainResponse.Response = &r
	default:
		return fmt.Errorf("unsupported query type: %d", queryType)
	}

	return nil
}

// ValidatePerChainResponse performs basic validation on a per chain query response.
func (perChainResponse *PerChainQueryResponse) Validate() error {
	str := perChainResponse.ChainId.String()
	if _, err := vaa.ChainIDFromString(str); err != nil {
		return fmt.Errorf("invalid chainID: %d", uint16(perChainResponse.ChainId))
	}

	if perChainResponse.Response == nil {
		return fmt.Errorf("response is nil")
	}

	if err := ValidatePerChainQueryRequestType(perChainResponse.Response.Type()); err != nil {
		return err
	}

	if err := perChainResponse.Response.Validate(); err != nil {
		return fmt.Errorf("chain specific response is invalid: %w", err)
	}

	return nil
}

// Equal checks for equality on two per chain query responses.
func (left *PerChainQueryResponse) Equal(right *PerChainQueryResponse) bool {
	if left.ChainId != right.ChainId {
		return false
	}

	if left.Response == nil && right.Response == nil {
		return true
	}

	if left.Response == nil || right.Response == nil {
		return false
	}

	if left.Response.Type() != right.Response.Type() {
		return false
	}

	switch leftResp := left.Response.(type) {
	case *EthCallQueryResponse:
		switch rightResp := right.Response.(type) {
		case *EthCallQueryResponse:
			return leftResp.Equal(rightResp)
		default:
			panic("unsupported query type on right") // We checked this above!
		}
	case *EthCallByTimestampQueryResponse:
		switch rightResp := right.Response.(type) {
		case *EthCallByTimestampQueryResponse:
			return leftResp.Equal(rightResp)
		default:
			panic("unsupported query type on right") // We checked this above!
		}
	case *EthCallWithFinalityQueryResponse:
		switch rightResp := right.Response.(type) {
		case *EthCallWithFinalityQueryResponse:
			return leftResp.Equal(rightResp)
		default:
			panic("unsupported query type on right") // We checked this above!
		}
	case *SolanaAccountQueryResponse:
		switch rightResp := right.Response.(type) {
		case *SolanaAccountQueryResponse:
			return leftResp.Equal(rightResp)
		default:
			panic("unsupported query type on right") // We checked this above!
		}
	case *SolanaPdaQueryResponse:
		switch rightResp := right.Response.(type) {
		case *SolanaPdaQueryResponse:
			return leftResp.Equal(rightResp)
		default:
			panic("unsupported query type on right") // We checked this above!
		}
	default:
		panic("unsupported query type on left") // We checked this above!
	}
}

//
// Implementation of EthCallQueryResponse, which implements the ChainSpecificResponse for an EVM eth_call query response.
//

func (e *EthCallQueryResponse) Type() ChainSpecificQueryType {
	return EthCallQueryRequestType
}

// Marshal serializes the binary representation of an EVM eth_call response.
// This method calls Validate() and relies on it to range checks lengths, etc.
func (ecr *EthCallQueryResponse) Marshal() ([]byte, error) {
	if err := ecr.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	vaa.MustWrite(buf, binary.BigEndian, ecr.BlockNumber)
	buf.Write(ecr.Hash[:])
	vaa.MustWrite(buf, binary.BigEndian, ecr.Time.UnixMicro())

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(ecr.Results))) // #nosec G115 -- This is validated in `Validate`
	for idx := range ecr.Results {
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(ecr.Results[idx]))) // #nosec G115 -- This is validated in `Validate`
		buf.Write(ecr.Results[idx])
	}

	return buf.Bytes(), nil
}

// Unmarshal deserializes an EVM eth_call response from a byte array
func (ecr *EthCallQueryResponse) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return ecr.UnmarshalFromReader(reader)
}

// UnmarshalFromReader  deserializes an EVM eth_call response from a byte array
func (ecr *EthCallQueryResponse) UnmarshalFromReader(reader *bytes.Reader) error {
	if err := binary.Read(reader, binary.BigEndian, &ecr.BlockNumber); err != nil {
		return fmt.Errorf("failed to read response number: %w", err)
	}

	responseHash := common.Hash{}
	if n, err := reader.Read(responseHash[:]); err != nil || n != 32 {
		return fmt.Errorf("failed to read response hash [%d]: %w", n, err)
	}
	ecr.Hash = responseHash

	unixMicros := int64(0)
	if err := binary.Read(reader, binary.BigEndian, &unixMicros); err != nil {
		return fmt.Errorf("failed to read response timestamp: %w", err)
	}
	ecr.Time = time.UnixMicro(unixMicros)

	numResults := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numResults); err != nil {
		return fmt.Errorf("failed to read number of results: %w", err)
	}

	for count := 0; count < int(numResults); count++ {
		resultLen := uint32(0)
		if err := binary.Read(reader, binary.BigEndian, &resultLen); err != nil {
			return fmt.Errorf("failed to read result len: %w", err)
		}
		result := make([]byte, resultLen)
		if n, err := reader.Read(result[:]); err != nil || n != int(resultLen) {
			return fmt.Errorf("failed to read result [%d]: %w", n, err)
		}

		ecr.Results = append(ecr.Results, result)
	}

	return nil
}

// Validate does basic validation on an EVM eth_call response.
func (ecr *EthCallQueryResponse) Validate() error {
	// Not checking for BlockNumber == 0, because maybe that could happen??

	if len(ecr.Hash) != 32 {
		return fmt.Errorf("invalid length for block hash")
	}

	if len(ecr.Results) <= 0 {
		return fmt.Errorf("does not contain any results")
	}
	if len(ecr.Results) > math.MaxUint8 {
		return fmt.Errorf("too many results")
	}
	for _, result := range ecr.Results {
		if len(result) > math.MaxUint32 {
			return fmt.Errorf("result too long")
		}
	}
	return nil
}

// Equal verifies that two EVM eth_call responses are equal.
func (left *EthCallQueryResponse) Equal(right *EthCallQueryResponse) bool {
	if left.BlockNumber != right.BlockNumber {
		return false
	}

	if !bytes.Equal(left.Hash.Bytes(), right.Hash.Bytes()) {
		return false
	}

	if left.Time != right.Time {
		return false
	}

	if len(left.Results) != len(right.Results) {
		return false
	}
	for idx := range left.Results {
		if !bytes.Equal(left.Results[idx], right.Results[idx]) {
			return false
		}
	}

	return true
}

//
// Implementation of EthCallByTimestampQueryResponse, which implements the ChainSpecificResponse for an EVM eth_call_by_timestamp query response.
//

func (e *EthCallByTimestampQueryResponse) Type() ChainSpecificQueryType {
	return EthCallByTimestampQueryRequestType
}

// Marshal serializes the binary representation of an EVM eth_call_by_timestamp response.
// This method calls Validate() and relies on it to range checks lengths, etc.
func (ecr *EthCallByTimestampQueryResponse) Marshal() ([]byte, error) {
	if err := ecr.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	vaa.MustWrite(buf, binary.BigEndian, ecr.TargetBlockNumber)
	buf.Write(ecr.TargetBlockHash[:])
	vaa.MustWrite(buf, binary.BigEndian, ecr.TargetBlockTime.UnixMicro())

	vaa.MustWrite(buf, binary.BigEndian, ecr.FollowingBlockNumber)
	buf.Write(ecr.FollowingBlockHash[:])
	vaa.MustWrite(buf, binary.BigEndian, ecr.FollowingBlockTime.UnixMicro())

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(ecr.Results))) // #nosec G115 -- This is validated in `Validate`
	for idx := range ecr.Results {
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(ecr.Results[idx]))) // #nosec G115 -- This is validated in `Validate`
		buf.Write(ecr.Results[idx])
	}

	return buf.Bytes(), nil
}

// Unmarshal deserializes an EVM eth_call_by_timestamp response from a byte array
func (ecr *EthCallByTimestampQueryResponse) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return ecr.UnmarshalFromReader(reader)
}

// UnmarshalFromReader  deserializes an EVM eth_call_by_timestamp response from a byte array
func (ecr *EthCallByTimestampQueryResponse) UnmarshalFromReader(reader *bytes.Reader) error {
	if err := binary.Read(reader, binary.BigEndian, &ecr.TargetBlockNumber); err != nil {
		return fmt.Errorf("failed to read response target block number: %w", err)
	}

	responseHash := common.Hash{}
	if n, err := reader.Read(responseHash[:]); err != nil || n != 32 {
		return fmt.Errorf("failed to read response target block hash [%d]: %w", n, err)
	}
	ecr.TargetBlockHash = responseHash

	unixMicros := int64(0)
	if err := binary.Read(reader, binary.BigEndian, &unixMicros); err != nil {
		return fmt.Errorf("failed to read response target block timestamp: %w", err)
	}
	ecr.TargetBlockTime = time.UnixMicro(unixMicros)

	if err := binary.Read(reader, binary.BigEndian, &ecr.FollowingBlockNumber); err != nil {
		return fmt.Errorf("failed to read response following block number: %w", err)
	}

	responseHash = common.Hash{}
	if n, err := reader.Read(responseHash[:]); err != nil || n != 32 {
		return fmt.Errorf("failed to read response following block hash [%d]: %w", n, err)
	}
	ecr.FollowingBlockHash = responseHash

	unixMicros = int64(0)
	if err := binary.Read(reader, binary.BigEndian, &unixMicros); err != nil {
		return fmt.Errorf("failed to read response following block timestamp: %w", err)
	}
	ecr.FollowingBlockTime = time.UnixMicro(unixMicros)

	numResults := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numResults); err != nil {
		return fmt.Errorf("failed to read number of results: %w", err)
	}

	for count := 0; count < int(numResults); count++ {
		resultLen := uint32(0)
		if err := binary.Read(reader, binary.BigEndian, &resultLen); err != nil {
			return fmt.Errorf("failed to read result len: %w", err)
		}
		result := make([]byte, resultLen)
		if n, err := reader.Read(result[:]); err != nil || n != int(resultLen) {
			return fmt.Errorf("failed to read result [%d]: %w", n, err)
		}

		ecr.Results = append(ecr.Results, result)
	}

	return nil
}

// Validate does basic validation on an EVM eth_call_by_timestamp response.
func (ecr *EthCallByTimestampQueryResponse) Validate() error {
	// Not checking for block numbers == 0, because maybe that could happen??

	if len(ecr.TargetBlockHash) != 32 {
		return fmt.Errorf("invalid length for target block hash")
	}

	if len(ecr.FollowingBlockHash) != 32 {
		return fmt.Errorf("invalid length for following block hash")
	}

	if len(ecr.Results) <= 0 {
		return fmt.Errorf("does not contain any results")
	}
	if len(ecr.Results) > math.MaxUint8 {
		return fmt.Errorf("too many results")
	}
	for _, result := range ecr.Results {
		if len(result) > math.MaxUint32 {
			return fmt.Errorf("result too long")
		}
	}
	return nil
}

// Equal verifies that two EVM eth_call_by_timestamp responses are equal.
func (left *EthCallByTimestampQueryResponse) Equal(right *EthCallByTimestampQueryResponse) bool {
	if left.TargetBlockNumber != right.TargetBlockNumber {
		return false
	}

	if !bytes.Equal(left.TargetBlockHash.Bytes(), right.TargetBlockHash.Bytes()) {
		return false
	}

	if left.TargetBlockTime != right.TargetBlockTime {
		return false
	}

	if left.FollowingBlockNumber != right.FollowingBlockNumber {
		return false
	}

	if !bytes.Equal(left.FollowingBlockHash.Bytes(), right.FollowingBlockHash.Bytes()) {
		return false
	}

	if left.FollowingBlockTime != right.FollowingBlockTime {
		return false
	}

	if len(left.Results) != len(right.Results) {
		return false
	}
	for idx := range left.Results {
		if !bytes.Equal(left.Results[idx], right.Results[idx]) {
			return false
		}
	}

	return true
}

//
// Implementation of EthCallWithFinalityQueryResponse, which implements the ChainSpecificResponse for an EVM eth_call_with_finality query response.
//

func (e *EthCallWithFinalityQueryResponse) Type() ChainSpecificQueryType {
	return EthCallWithFinalityQueryRequestType
}

// Marshal serializes the binary representation of an EVM eth_call_with_finality response.
// This method calls Validate() and relies on it to range checks lengths, etc.
func (ecr *EthCallWithFinalityQueryResponse) Marshal() ([]byte, error) {
	if err := ecr.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	vaa.MustWrite(buf, binary.BigEndian, ecr.BlockNumber)
	buf.Write(ecr.Hash[:])
	vaa.MustWrite(buf, binary.BigEndian, ecr.Time.UnixMicro())

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(ecr.Results))) // #nosec G115 -- This is validated in `Validate`
	for idx := range ecr.Results {
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(ecr.Results[idx]))) // #nosec G115 -- This is validated in `Validate`
		buf.Write(ecr.Results[idx])
	}

	return buf.Bytes(), nil
}

// Unmarshal deserializes an EVM eth_call_with_finality response from a byte array
func (ecr *EthCallWithFinalityQueryResponse) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return ecr.UnmarshalFromReader(reader)
}

// UnmarshalFromReader  deserializes an EVM eth_call_with_finality response from a byte array
func (ecr *EthCallWithFinalityQueryResponse) UnmarshalFromReader(reader *bytes.Reader) error {
	if err := binary.Read(reader, binary.BigEndian, &ecr.BlockNumber); err != nil {
		return fmt.Errorf("failed to read response number: %w", err)
	}

	responseHash := common.Hash{}
	if n, err := reader.Read(responseHash[:]); err != nil || n != 32 {
		return fmt.Errorf("failed to read response hash [%d]: %w", n, err)
	}
	ecr.Hash = responseHash

	unixMicros := int64(0)
	if err := binary.Read(reader, binary.BigEndian, &unixMicros); err != nil {
		return fmt.Errorf("failed to read response timestamp: %w", err)
	}
	ecr.Time = time.UnixMicro(unixMicros)

	numResults := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numResults); err != nil {
		return fmt.Errorf("failed to read number of results: %w", err)
	}

	for count := 0; count < int(numResults); count++ {
		resultLen := uint32(0)
		if err := binary.Read(reader, binary.BigEndian, &resultLen); err != nil {
			return fmt.Errorf("failed to read result len: %w", err)
		}
		result := make([]byte, resultLen)
		if n, err := reader.Read(result[:]); err != nil || n != int(resultLen) {
			return fmt.Errorf("failed to read result [%d]: %w", n, err)
		}

		ecr.Results = append(ecr.Results, result)
	}

	return nil
}

// Validate does basic validation on an EVM eth_call_with_finality response.
func (ecr *EthCallWithFinalityQueryResponse) Validate() error {
	// Not checking for BlockNumber == 0, because maybe that could happen??

	if len(ecr.Hash) != 32 {
		return fmt.Errorf("invalid length for block hash")
	}

	if len(ecr.Results) <= 0 {
		return fmt.Errorf("does not contain any results")
	}
	if len(ecr.Results) > math.MaxUint8 {
		return fmt.Errorf("too many results")
	}
	for _, result := range ecr.Results {
		if len(result) > math.MaxUint32 {
			return fmt.Errorf("result too long")
		}
	}
	return nil
}

// Equal verifies that two EVM eth_call_with_finality responses are equal.
func (left *EthCallWithFinalityQueryResponse) Equal(right *EthCallWithFinalityQueryResponse) bool {
	if left.BlockNumber != right.BlockNumber {
		return false
	}

	if !bytes.Equal(left.Hash.Bytes(), right.Hash.Bytes()) {
		return false
	}

	if left.Time != right.Time {
		return false
	}

	if len(left.Results) != len(right.Results) {
		return false
	}
	for idx := range left.Results {
		if !bytes.Equal(left.Results[idx], right.Results[idx]) {
			return false
		}
	}

	return true
}

//
// Implementation of SolanaAccountQueryResponse, which implements the ChainSpecificResponse for a Solana sol_account query response.
//

func (sar *SolanaAccountQueryResponse) Type() ChainSpecificQueryType {
	return SolanaAccountQueryRequestType
}

// Marshal serializes the binary representation of a Solana sol_account response.
// This method calls Validate() and relies on it to range check lengths, etc.
func (sar *SolanaAccountQueryResponse) Marshal() ([]byte, error) {
	if err := sar.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	vaa.MustWrite(buf, binary.BigEndian, sar.SlotNumber)
	vaa.MustWrite(buf, binary.BigEndian, sar.BlockTime.UnixMicro())
	buf.Write(sar.BlockHash[:])

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(sar.Results))) // #nosec G115 -- This is validated in `Validate`
	for _, res := range sar.Results {
		vaa.MustWrite(buf, binary.BigEndian, res.Lamports)
		vaa.MustWrite(buf, binary.BigEndian, res.RentEpoch)
		vaa.MustWrite(buf, binary.BigEndian, res.Executable)
		buf.Write(res.Owner[:])

		vaa.MustWrite(buf, binary.BigEndian, uint32(len(res.Data))) // #nosec G115 -- This is validated in `Validate`
		buf.Write(res.Data)
	}

	return buf.Bytes(), nil
}

// Unmarshal deserializes a Solana sol_account response from a byte array
func (sar *SolanaAccountQueryResponse) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return sar.UnmarshalFromReader(reader)
}

// UnmarshalFromReader  deserializes a Solana sol_account response from a byte array
func (sar *SolanaAccountQueryResponse) UnmarshalFromReader(reader *bytes.Reader) error {
	if err := binary.Read(reader, binary.BigEndian, &sar.SlotNumber); err != nil {
		return fmt.Errorf("failed to read slot number: %w", err)
	}

	blockTime := int64(0)
	if err := binary.Read(reader, binary.BigEndian, &blockTime); err != nil {
		return fmt.Errorf("failed to read block time: %w", err)
	}
	sar.BlockTime = time.UnixMicro(blockTime)
	if n, err := reader.Read(sar.BlockHash[:]); err != nil || n != SolanaPublicKeyLength {
		return fmt.Errorf("failed to read block hash [%d]: %w", n, err)
	}

	numResults := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numResults); err != nil {
		return fmt.Errorf("failed to read number of results: %w", err)
	}

	for count := 0; count < int(numResults); count++ {
		var result SolanaAccountResult

		if err := binary.Read(reader, binary.BigEndian, &result.Lamports); err != nil {
			return fmt.Errorf("failed to read lamports: %w", err)
		}

		if err := binary.Read(reader, binary.BigEndian, &result.RentEpoch); err != nil {
			return fmt.Errorf("failed to read rent epoch: %w", err)
		}

		if err := binary.Read(reader, binary.BigEndian, &result.Executable); err != nil {
			return fmt.Errorf("failed to read executable flag: %w", err)
		}

		if n, err := reader.Read(result.Owner[:]); err != nil || n != SolanaPublicKeyLength {
			return fmt.Errorf("failed to read owner [%d]: %w", n, err)
		}

		length := uint32(0)
		if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
			return fmt.Errorf("failed to read data len: %w", err)
		}
		result.Data = make([]byte, length)
		if n, err := reader.Read(result.Data[:]); err != nil || n != int(length) {
			return fmt.Errorf("failed to read data [%d]: %w", n, err)
		}

		sar.Results = append(sar.Results, result)
	}

	return nil
}

// Validate does basic validation on a Solana sol_account response.
func (sar *SolanaAccountQueryResponse) Validate() error {
	// Not checking for SlotNumber == 0, because maybe that could happen??
	// Not checking for BlockTime == 0, because maybe that could happen??

	// The block hash is fixed length, so don't need to check for nil.
	if len(sar.BlockHash) != SolanaPublicKeyLength {
		return fmt.Errorf("invalid block hash length")
	}

	if len(sar.Results) <= 0 {
		return fmt.Errorf("does not contain any results")
	}
	if len(sar.Results) > math.MaxUint8 {
		return fmt.Errorf("too many results")
	}
	for _, result := range sar.Results {
		// Owner is fixed length, so don't need to check for nil.
		if len(result.Owner) != SolanaPublicKeyLength {
			return fmt.Errorf("invalid owner length")
		}
		if len(result.Data) > math.MaxUint32 {
			return fmt.Errorf("data too long")
		}
	}

	return nil
}

// Equal verifies that two Solana sol_account responses are equal.
func (left *SolanaAccountQueryResponse) Equal(right *SolanaAccountQueryResponse) bool {
	if left.SlotNumber != right.SlotNumber ||
		left.BlockTime != right.BlockTime ||
		!bytes.Equal(left.BlockHash[:], right.BlockHash[:]) {
		return false
	}

	if len(left.Results) != len(right.Results) {
		return false
	}
	for idx := range left.Results {
		if left.Results[idx].Lamports != right.Results[idx].Lamports ||
			left.Results[idx].RentEpoch != right.Results[idx].RentEpoch ||
			left.Results[idx].Executable != right.Results[idx].Executable ||
			!bytes.Equal(left.Results[idx].Owner[:], right.Results[idx].Owner[:]) ||
			!bytes.Equal(left.Results[idx].Data, right.Results[idx].Data) {
			return false
		}
	}

	return true
}

//
// Implementation of SolanaPdaQueryResponse, which implements the ChainSpecificResponse for a Solana sol_pda query response.
//

func (sar *SolanaPdaQueryResponse) Type() ChainSpecificQueryType {
	return SolanaPdaQueryRequestType
}

// Marshal serializes the binary representation of a Solana sol_pda response.
// This method calls Validate() and relies on it to range check lengths, etc.
func (sar *SolanaPdaQueryResponse) Marshal() ([]byte, error) {
	if err := sar.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	vaa.MustWrite(buf, binary.BigEndian, sar.SlotNumber)
	vaa.MustWrite(buf, binary.BigEndian, sar.BlockTime.UnixMicro())
	buf.Write(sar.BlockHash[:])

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(sar.Results))) // #nosec G115 -- This is validated in `Validate`
	for _, res := range sar.Results {
		buf.Write(res.Account[:])
		vaa.MustWrite(buf, binary.BigEndian, res.Bump)
		vaa.MustWrite(buf, binary.BigEndian, res.Lamports)
		vaa.MustWrite(buf, binary.BigEndian, res.RentEpoch)
		vaa.MustWrite(buf, binary.BigEndian, res.Executable)
		buf.Write(res.Owner[:])

		vaa.MustWrite(buf, binary.BigEndian, uint32(len(res.Data))) // #nosec G115 -- This is validated in `Validate`
		buf.Write(res.Data)
	}

	return buf.Bytes(), nil
}

// Unmarshal deserializes a Solana sol_pda response from a byte array
func (sar *SolanaPdaQueryResponse) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return sar.UnmarshalFromReader(reader)
}

// UnmarshalFromReader  deserializes a Solana sol_pda response from a byte array
func (sar *SolanaPdaQueryResponse) UnmarshalFromReader(reader *bytes.Reader) error {
	if err := binary.Read(reader, binary.BigEndian, &sar.SlotNumber); err != nil {
		return fmt.Errorf("failed to read slot number: %w", err)
	}

	blockTime := int64(0)
	if err := binary.Read(reader, binary.BigEndian, &blockTime); err != nil {
		return fmt.Errorf("failed to read block time: %w", err)
	}
	sar.BlockTime = time.UnixMicro(blockTime)
	if n, err := reader.Read(sar.BlockHash[:]); err != nil || n != SolanaPublicKeyLength {
		return fmt.Errorf("failed to read block hash [%d]: %w", n, err)
	}

	numResults := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numResults); err != nil {
		return fmt.Errorf("failed to read number of results: %w", err)
	}

	for count := 0; count < int(numResults); count++ {
		var result SolanaPdaResult

		if n, err := reader.Read(result.Account[:]); err != nil || n != SolanaPublicKeyLength {
			return fmt.Errorf("failed to read account [%d]: %w", n, err)
		}

		if err := binary.Read(reader, binary.BigEndian, &result.Bump); err != nil {
			return fmt.Errorf("failed to read bump: %w", err)
		}

		if err := binary.Read(reader, binary.BigEndian, &result.Lamports); err != nil {
			return fmt.Errorf("failed to read lamports: %w", err)
		}

		if err := binary.Read(reader, binary.BigEndian, &result.RentEpoch); err != nil {
			return fmt.Errorf("failed to read rent epoch: %w", err)
		}

		if err := binary.Read(reader, binary.BigEndian, &result.Executable); err != nil {
			return fmt.Errorf("failed to read executable flag: %w", err)
		}

		if n, err := reader.Read(result.Owner[:]); err != nil || n != SolanaPublicKeyLength {
			return fmt.Errorf("failed to read owner [%d]: %w", n, err)
		}

		length := uint32(0)
		if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
			return fmt.Errorf("failed to read data len: %w", err)
		}
		result.Data = make([]byte, length)
		if n, err := reader.Read(result.Data[:]); err != nil || n != int(length) {
			return fmt.Errorf("failed to read data [%d]: %w", n, err)
		}

		sar.Results = append(sar.Results, result)
	}

	return nil
}

// Validate does basic validation on a Solana sol_pda response.
func (sar *SolanaPdaQueryResponse) Validate() error {
	// Not checking for SlotNumber == 0, because maybe that could happen??
	// Not checking for BlockTime == 0, because maybe that could happen??

	// The block hash is fixed length, so don't need to check for nil.
	if len(sar.BlockHash) != SolanaPublicKeyLength {
		return fmt.Errorf("invalid block hash length")
	}

	if len(sar.Results) <= 0 {
		return fmt.Errorf("does not contain any results")
	}
	if len(sar.Results) > math.MaxUint8 {
		return fmt.Errorf("too many results")
	}
	for _, result := range sar.Results {
		// Account is fixed length, so don't need to check for nil.
		if len(result.Account) != SolanaPublicKeyLength {
			return fmt.Errorf("invalid account length")
		}
		// Owner is fixed length, so don't need to check for nil.
		if len(result.Owner) != SolanaPublicKeyLength {
			return fmt.Errorf("invalid owner length")
		}
		if len(result.Data) > math.MaxUint32 {
			return fmt.Errorf("data too long")
		}
	}

	return nil
}

// Equal verifies that two Solana sol_pda responses are equal.
func (left *SolanaPdaQueryResponse) Equal(right *SolanaPdaQueryResponse) bool {
	if left.SlotNumber != right.SlotNumber ||
		left.BlockTime != right.BlockTime ||
		!bytes.Equal(left.BlockHash[:], right.BlockHash[:]) {
		return false
	}

	if len(left.Results) != len(right.Results) {
		return false
	}
	for idx := range left.Results {
		if !bytes.Equal(left.Results[idx].Account[:], right.Results[idx].Account[:]) ||
			left.Results[idx].Bump != right.Results[idx].Bump ||
			left.Results[idx].Lamports != right.Results[idx].Lamports ||
			left.Results[idx].RentEpoch != right.Results[idx].RentEpoch ||
			left.Results[idx].Executable != right.Results[idx].Executable ||
			!bytes.Equal(left.Results[idx].Owner[:], right.Results[idx].Owner[:]) ||
			!bytes.Equal(left.Results[idx].Data, right.Results[idx].Data) {
			return false
		}
	}

	return true
}
