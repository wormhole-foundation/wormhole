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
	"google.golang.org/protobuf/proto"
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

type QueryResponse struct {
	RequestID     string
	ChainID       vaa.ChainID
	Status        QueryStatus
	SignedRequest *gossipv1.SignedQueryRequest
	Result        *EthCallQueryResponse
}

func CreateQueryResponse(req *QueryRequest, status QueryStatus, result *EthCallQueryResponse) *QueryResponse {
	return &QueryResponse{
		RequestID:     req.RequestID,
		ChainID:       vaa.ChainID(req.Request.ChainId),
		SignedRequest: req.SignedRequest,
		Status:        status,
		Result:        result,
	}
}

var queryResponsePrefix = []byte("query_response_0000000000000000000|")

type EthCallQueryResponse struct {
	Number *big.Int
	Hash   common.Hash
	Time   time.Time
	Result []byte
}

type QueryResponsePublication struct {
	Request  *gossipv1.SignedQueryRequest
	Response EthCallQueryResponse
}

func (resp *QueryResponsePublication) RequestID() string {
	if resp == nil || resp.Request == nil {
		return "nil"
	}
	return hex.EncodeToString(resp.Request.Signature)
}

// Marshal serializes the binary representation of a query response
func (msg *QueryResponsePublication) Marshal() ([]byte, error) {
	// TODO: copy request write checks to query module request handling
	// TODO: only receive the unmarshalled query request (see note in query.go)
	var queryRequest gossipv1.QueryRequest
	err := proto.Unmarshal(msg.Request.QueryRequest, &queryRequest)
	if err != nil {
		return nil, fmt.Errorf("received invalid message from query module")
	}

	if err := ValidateQueryRequest(&queryRequest); err != nil {
		return nil, fmt.Errorf("queryRequest is invalid: %w", err)
	}

	if len(msg.Response.Hash) != 32 {
		return nil, fmt.Errorf("invalid length for block hash")
	}
	if len(msg.Response.Result) > math.MaxUint32 {
		return nil, fmt.Errorf("response data too long")
	}

	buf := new(bytes.Buffer)

	// Source
	// TODO: support writing off-chain and on-chain requests
	// Here, unset represents an off-chain request
	vaa.MustWrite(buf, binary.BigEndian, vaa.ChainIDUnset)
	buf.Write(msg.Request.Signature[:])

	// Request
	// TODO: support writing different types of request/response pairs
	switch req := queryRequest.Message.(type) {
	case *gossipv1.QueryRequest_EthCallQueryRequest:
		vaa.MustWrite(buf, binary.BigEndian, uint8(1))
		vaa.MustWrite(buf, binary.BigEndian, uint16(queryRequest.ChainId))
		vaa.MustWrite(buf, binary.BigEndian, queryRequest.Nonce) // uint32
		buf.Write(req.EthCallQueryRequest.To)
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(req.EthCallQueryRequest.Data)))
		buf.Write(req.EthCallQueryRequest.Data)
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(req.EthCallQueryRequest.Block)))
		// TODO: should this be an enum or the literal string?
		buf.Write([]byte(req.EthCallQueryRequest.Block))

		// Response
		// TODO: probably some kind of request/response pair validation
		// TODO: is uint64 safe?
		vaa.MustWrite(buf, binary.BigEndian, msg.Response.Number.Uint64())
		buf.Write(msg.Response.Hash[:])
		vaa.MustWrite(buf, binary.BigEndian, uint32(msg.Response.Time.Unix()))
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(msg.Response.Result)))
		buf.Write(msg.Response.Result)
		return buf.Bytes(), nil
	default:
		return nil, fmt.Errorf("received invalid message from query module")
	}
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

	requestType := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &requestType); err != nil {
		return nil, fmt.Errorf("failed to read request chain: %w", err)
	}
	if requestType != 1 {
		// TODO: support reading different types of request/response pairs
		return nil, fmt.Errorf("unsupported request type: %d", requestType)
	}

	queryRequest := &gossipv1.QueryRequest{}
	queryChain := vaa.ChainID(0)
	if err := binary.Read(reader, binary.BigEndian, &queryChain); err != nil {
		return nil, fmt.Errorf("failed to read request chain: %w", err)
	}
	queryRequest.ChainId = uint32(queryChain)

	queryNonce := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &queryNonce); err != nil {
		return nil, fmt.Errorf("failed to read request nonce: %w", err)
	}
	queryRequest.Nonce = queryNonce

	ethCallQueryRequest := &gossipv1.EthCallQueryRequest{}

	queryEthCallTo := [20]byte{}
	if n, err := reader.Read(queryEthCallTo[:]); err != nil || n != 20 {
		return nil, fmt.Errorf("failed to read call To [%d]: %w", n, err)
	}
	ethCallQueryRequest.To = queryEthCallTo[:]

	queryEthCallDataLen := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &queryEthCallDataLen); err != nil {
		return nil, fmt.Errorf("failed to read call Data len: %w", err)
	}
	queryEthCallData := make([]byte, queryEthCallDataLen)
	if n, err := reader.Read(queryEthCallData[:]); err != nil || n != int(queryEthCallDataLen) {
		return nil, fmt.Errorf("failed to read call To [%d]: %w", n, err)
	}
	ethCallQueryRequest.Data = queryEthCallData[:]

	queryEthCallBlockLen := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &queryEthCallBlockLen); err != nil {
		return nil, fmt.Errorf("failed to read call Data len: %w", err)
	}
	queryEthCallBlockBytes := make([]byte, queryEthCallBlockLen)
	if n, err := reader.Read(queryEthCallBlockBytes[:]); err != nil || n != int(queryEthCallBlockLen) {
		return nil, fmt.Errorf("failed to read call To [%d]: %w", n, err)
	}
	ethCallQueryRequest.Block = string(queryEthCallBlockBytes[:])

	queryRequest.Message = &gossipv1.QueryRequest_EthCallQueryRequest{
		EthCallQueryRequest: ethCallQueryRequest,
	}
	queryRequestBytes, err := proto.Marshal(queryRequest)
	if err != nil {
		return nil, err
	}
	signedQueryRequest.QueryRequest = queryRequestBytes

	msg.Request = signedQueryRequest

	// Response
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

	unixSeconds := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &unixSeconds); err != nil {
		return nil, fmt.Errorf("failed to read response timestamp: %w", err)
	}
	queryResponse.Time = time.Unix(int64(unixSeconds), 0)

	responseResultLen := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &responseResultLen); err != nil {
		return nil, fmt.Errorf("failed to read response len: %w", err)
	}
	responseResult := make([]byte, responseResultLen)
	if n, err := reader.Read(responseResult[:]); err != nil || n != int(responseResultLen) {
		return nil, fmt.Errorf("failed to read result [%d]: %w", n, err)
	}
	queryResponse.Result = responseResult[:]

	msg.Response = queryResponse

	return msg, nil
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

func GetQueryResponseDigestFromBytes(b []byte) common.Hash {
	return crypto.Keccak256Hash(append(queryResponsePrefix, crypto.Keccak256Hash(b).Bytes()...))
}
