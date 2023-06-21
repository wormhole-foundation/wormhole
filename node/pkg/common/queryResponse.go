package common

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

	// Source
	// TODO: support writing off-chain and on-chain requests
	// Here, unset represents an off-chain request
	vaa.MustWrite(buf, binary.BigEndian, vaa.ChainIDUnset)

	buf.Write(msg.Request.Signature[:])
	buf.Write(msg.Request.QueryRequest)

	// Per chain responses
	vaa.MustWrite(buf, binary.BigEndian, uint8(len(msg.PerChainResponses)))
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

	queryRequest := QueryRequest{}
	err := queryRequest.UnmarshalFromReader(reader)
	if err != nil {
		return fmt.Errorf("failed to unmarshal query request: %w", err)
	}

	queryRequestBytes, err := queryRequest.Marshal()
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

	return nil
}

// Validate does basic validation on a received query request.
func (msg *QueryResponsePublication) Validate() error {
	// Unmarshal and validate the contained query request.
	var queryRequest QueryRequest
	err := queryRequest.Unmarshal(msg.Request.QueryRequest)
	if err != nil {
		return fmt.Errorf("failed to unmarshal query request")
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

func (resp *QueryResponsePublication) RequestID() string {
	if resp == nil || resp.Request == nil {
		return "nil"
	}
	return hex.EncodeToString(resp.Request.Signature)
}

// Similar to sdk/vaa/structs.go,
// In order to save space in the solana signature verification instruction, we hash twice so we only need to pass in
// the first hash (32 bytes) vs the full body data.
// TODO: confirm if this works / is worthwhile.
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

	switch queryType {
	case EthCallQueryRequestType:
		r := EthCallQueryResponse{}
		if err := r.UnmarshalFromReader(reader); err != nil {
			return fmt.Errorf("failed to unmarshal eth call response: %w", err)
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

	switch leftEcq := left.Response.(type) {
	case *EthCallQueryResponse:
		switch rightEcd := right.Response.(type) {
		case *EthCallQueryResponse:
			return leftEcq.Equal(rightEcd)
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

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(ecr.Results)))
	for idx := range ecr.Results {
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(ecr.Results[idx])))
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
