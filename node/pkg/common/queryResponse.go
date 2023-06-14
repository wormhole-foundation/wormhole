package common

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
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
	ChainID    vaa.ChainID
	Status     QueryStatus
	Results    []EthCallQueryResponse
}

// CreatePerChainQueryResponseInternal creates a PerChainQueryResponseInternal and returns a pointer to it.
func CreatePerChainQueryResponseInternal(reqId string, reqIdx int, chainID vaa.ChainID, status QueryStatus, results []EthCallQueryResponse) *PerChainQueryResponseInternal {
	return &PerChainQueryResponseInternal{
		RequestID:  reqId,
		RequestIdx: reqIdx,
		ChainID:    chainID,
		Status:     status,
		Results:    results,
	}
}

var queryResponsePrefix = []byte("query_response_0000000000000000000|")

type QueryResponsePublication struct {
	Request           *gossipv1.SignedQueryRequest
	PerChainResponses []PerChainQueryResponse
}

type PerChainQueryResponse struct {
	ChainID   vaa.ChainID
	Responses []EthCallQueryResponse
}

type EthCallQueryResponse struct {
	Number *big.Int
	Hash   common.Hash
	Time   time.Time
	Result []byte
	// NOTE: If you modify this struct, please update the Equal() method for QueryResponsePublication.
}

const (
	QUERY_REQUEST_TYPE_ETH_CALL = uint8(1)
)

func (resp *QueryResponsePublication) RequestID() string {
	if resp == nil || resp.Request == nil {
		return "nil"
	}
	return hex.EncodeToString(resp.Request.Signature)
}

// MarshalQueryResponsePublication serializes the binary representation of a query response
func MarshalQueryResponsePublication(msg *QueryResponsePublication) ([]byte, error) {
	// TODO: copy request write checks to query module request handling
	// TODO: only receive the unmarshalled query request (see note in query.go)
	var queryRequest QueryRequest
	err := queryRequest.Unmarshal(msg.Request.QueryRequest)
	if err != nil {
		return nil, fmt.Errorf("received invalid message from query module")
	}

	// Validate things before we start marshalling.
	if err := queryRequest.Validate(); err != nil {
		return nil, fmt.Errorf("queryRequest is invalid: %w", err)
	}

	for idx := range msg.PerChainResponses {
		if err := ValidatePerChainResponse(&msg.PerChainResponses[idx]); err != nil {
			return nil, fmt.Errorf("invalid per chain response: %w", err)
		}
	}

	buf := new(bytes.Buffer)

	// Source
	// TODO: support writing off-chain and on-chain requests
	// Here, unset represents an off-chain request
	vaa.MustWrite(buf, binary.BigEndian, vaa.ChainIDUnset)
	buf.Write(msg.Request.Signature[:])

	// Request
	qrBuf, err := queryRequest.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query request")
	}
	buf.Write(qrBuf)

	// Per chain responses
	vaa.MustWrite(buf, binary.BigEndian, uint8(len(msg.PerChainResponses)))
	for idx := range msg.PerChainResponses {
		pcrBuf, err := MarshalPerChainResponse(&msg.PerChainResponses[idx])
		if err != nil {
			return nil, fmt.Errorf("failed to marshal per chain response: %w", err)
		}
		buf.Write(pcrBuf)
	}

	return buf.Bytes(), nil
}

// MarshalPerChainResponse marshalls a per chain query response.
func MarshalPerChainResponse(pcr *PerChainQueryResponse) ([]byte, error) {
	buf := new(bytes.Buffer)
	vaa.MustWrite(buf, binary.BigEndian, pcr.ChainID)
	vaa.MustWrite(buf, binary.BigEndian, uint8(len(pcr.Responses)))
	for _, resp := range pcr.Responses {
		vaa.MustWrite(buf, binary.BigEndian, resp.Number.Uint64())
		buf.Write(resp.Hash[:])
		vaa.MustWrite(buf, binary.BigEndian, resp.Time.UnixMicro())
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(resp.Result)))
		buf.Write(resp.Result)
	}
	return buf.Bytes(), nil
}

// ValidatePerChainResponse performs basic validation on a per chain query response.
func ValidatePerChainResponse(pcr *PerChainQueryResponse) error {
	if pcr.ChainID > math.MaxUint16 {
		return fmt.Errorf("invalid chain ID")
	}

	for _, resp := range pcr.Responses {
		if len(resp.Hash) != 32 {
			return fmt.Errorf("invalid length for block hash")
		}
		if len(resp.Result) > math.MaxUint32 {
			return fmt.Errorf("response data too long")
		}
	}

	return nil
}

// Unmarshal deserializes the binary representation of a query response
func UnmarshalQueryResponsePublication(data []byte) (*QueryResponsePublication, error) {
	// if len(data) < minMsgLength {
	// 	return nil, fmt.Errorf("message is too short")
	// }

	msg := &QueryResponsePublication{}

	reader := bytes.NewReader(data[:])

	// Request
	requestChain := vaa.ChainID(0)
	if err := binary.Read(reader, binary.BigEndian, &requestChain); err != nil {
		return nil, fmt.Errorf("failed to read request chain: %w", err)
	}
	if requestChain != vaa.ChainIDUnset {
		// TODO: support reading off-chain and on-chain requests
		return nil, fmt.Errorf("unsupported request chain: %d", requestChain)
	}

	signedQueryRequest := &gossipv1.SignedQueryRequest{}
	signature := [65]byte{}
	if n, err := reader.Read(signature[:]); err != nil || n != 65 {
		return nil, fmt.Errorf("failed to read signature [%d]: %w", n, err)
	}
	signedQueryRequest.Signature = signature[:]

	queryRequest := QueryRequest{}
	err := queryRequest.UnmarshalFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal query request: %w", err)
	}

	queryRequestBytes, err := queryRequest.Marshal()
	if err != nil {
		return nil, err
	}
	signedQueryRequest.QueryRequest = queryRequestBytes

	msg.Request = signedQueryRequest

	// Responses
	numPerChainResponses := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numPerChainResponses); err != nil {
		return nil, fmt.Errorf("failed to read number of per chain responses: %w", err)
	}

	for count := 0; count < int(numPerChainResponses); count++ {
		pcr, err := UnmarshalQueryPerChainResponseFromReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal per chain response: %w", err)
		}
		msg.PerChainResponses = append(msg.PerChainResponses, *pcr)
	}

	return msg, nil
}

func UnmarshalQueryPerChainResponseFromReader(reader *bytes.Reader) (*PerChainQueryResponse, error) {
	pcr := PerChainQueryResponse{}

	if err := binary.Read(reader, binary.BigEndian, &pcr.ChainID); err != nil {
		return nil, fmt.Errorf("failed to read chain ID: %w", err)
	}

	numResponses := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numResponses); err != nil {
		return nil, fmt.Errorf("failed to read number of responses: %w", err)
	}

	for count := 0; count < int(numResponses); count++ {
		queryResponse := EthCallQueryResponse{}

		responseNumber := uint64(0)
		if err := binary.Read(reader, binary.BigEndian, &responseNumber); err != nil {
			return nil, fmt.Errorf("failed to read response number: %w", err)
		}
		responseNumberBig := big.NewInt(0).SetUint64(responseNumber)
		queryResponse.Number = responseNumberBig

		responseHash := common.Hash{}
		if n, err := reader.Read(responseHash[:]); err != nil || n != 32 {
			return nil, fmt.Errorf("failed to read response hash [%d]: %w", n, err)
		}
		queryResponse.Hash = responseHash

		unixMicros := int64(0)
		if err := binary.Read(reader, binary.BigEndian, &unixMicros); err != nil {
			return nil, fmt.Errorf("failed to read response timestamp: %w", err)
		}
		queryResponse.Time = time.UnixMicro(unixMicros)

		responseResultLen := uint32(0)
		if err := binary.Read(reader, binary.BigEndian, &responseResultLen); err != nil {
			return nil, fmt.Errorf("failed to read response len: %w", err)
		}
		responseResult := make([]byte, responseResultLen)
		if n, err := reader.Read(responseResult[:]); err != nil || n != int(responseResultLen) {
			return nil, fmt.Errorf("failed to read result [%d]: %w", n, err)
		}
		queryResponse.Result = responseResult[:]

		pcr.Responses = append(pcr.Responses, queryResponse)
	}

	return &pcr, nil
}

// Similar to sdk/vaa/structs.go,
// In order to save space in the solana signature verification instruction, we hash twice so we only need to pass in
// the first hash (32 bytes) vs the full body data.
// TODO: confirm if this works / is worthwhile.
func (msg *QueryResponsePublication) SigningDigest() (common.Hash, error) {
	msgBytes, err := MarshalQueryResponsePublication(msg)
	if err != nil {
		return common.Hash{}, err
	}
	return GetQueryResponseDigestFromBytes(msgBytes), nil
}

// GetQueryResponseDigestFromBytes computes the digest bytes for a query response byte array.
func GetQueryResponseDigestFromBytes(b []byte) common.Hash {
	return crypto.Keccak256Hash(append(queryResponsePrefix, crypto.Keccak256Hash(b).Bytes()...))
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
		if !left.PerChainResponses[idx].Equal(&right.PerChainResponses[idx]) {
			return false
		}
	}
	return true
}

// Equal checks for equality on two per chain query responses.
func (left *PerChainQueryResponse) Equal(right *PerChainQueryResponse) bool {
	if left.ChainID != right.ChainID {
		return false
	}
	if len(left.Responses) != len(right.Responses) {
		return false
	}
	for idx := range left.Responses {
		if left.Responses[idx].Number.Cmp(right.Responses[idx].Number) != 0 {
			return false
		}
		if !bytes.Equal(left.Responses[idx].Hash.Bytes(), right.Responses[idx].Hash.Bytes()) {
			return false
		}
		if left.Responses[idx].Time != right.Responses[idx].Time {
			return false
		}
		if !bytes.Equal(left.Responses[idx].Result, right.Responses[idx].Result) {
			return false
		}
	}
	return true
}
