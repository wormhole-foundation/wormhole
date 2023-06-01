package common

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strings"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
)

const SignedQueryRequestChannelSize = 50

// QueryRequest is an internal representation of a query request.
type QueryRequest struct {
	SignedRequest *gossipv1.SignedQueryRequest
	Request       *gossipv1.QueryRequest
	RequestID     string
	ChainID       vaa.ChainID
}

// CreateQueryRequest creates a QueryRequest object from the signed query request.
func CreateQueryRequest(signedRequest *gossipv1.SignedQueryRequest, request *gossipv1.QueryRequest) *QueryRequest {
	return &QueryRequest{
		SignedRequest: signedRequest,
		Request:       request,
		RequestID:     hex.EncodeToString(signedRequest.Signature),
		ChainID:       vaa.ChainID(request.ChainId),
	}
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

// Marshal serializes the binary representation of a query response
func MarshalQueryRequest(queryRequest *gossipv1.QueryRequest) ([]byte, error) {
	buf := new(bytes.Buffer)

	switch req := queryRequest.Message.(type) {
	case *gossipv1.QueryRequest_EthCallQueryRequest:
		vaa.MustWrite(buf, binary.BigEndian, QUERY_REQUEST_TYPE_ETH_CALL)
		vaa.MustWrite(buf, binary.BigEndian, uint16(queryRequest.ChainId))
		vaa.MustWrite(buf, binary.BigEndian, queryRequest.Nonce) // uint32
		vaa.MustWrite(buf, binary.BigEndian, uint8(len(req.EthCallQueryRequest.CallData)))
		for _, callData := range req.EthCallQueryRequest.CallData {
			buf.Write(callData.To)
			vaa.MustWrite(buf, binary.BigEndian, uint32(len(callData.Data)))
			buf.Write(callData.Data)
		}
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(req.EthCallQueryRequest.Block)))
		// TODO: should this be an enum or the literal string?
		buf.Write([]byte(req.EthCallQueryRequest.Block))
	default:
		return nil, fmt.Errorf("invalid request type")
	}
	return buf.Bytes(), nil
}

// Unmarshal deserializes the binary representation of a query response
func UnmarshalQueryRequest(data []byte) (*gossipv1.QueryRequest, error) {
	reader := bytes.NewReader(data[:])
	return UnmarshalQueryRequestFromReader(reader)
}

func UnmarshalQueryRequestFromReader(reader *bytes.Reader) (*gossipv1.QueryRequest, error) {
	queryRequest := &gossipv1.QueryRequest{}

	requestType := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &requestType); err != nil {
		return nil, fmt.Errorf("failed to read request chain: %w", err)
	}
	if requestType != QUERY_REQUEST_TYPE_ETH_CALL {
		// TODO: support reading different types of request/response pairs
		return nil, fmt.Errorf("unsupported request type: %d", requestType)
	}

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

	numCallData := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numCallData); err != nil {
		return nil, fmt.Errorf("failed to read number of call data entries: %w", err)
	}

	for count := 0; count < int(numCallData); count++ {
		queryEthCallTo := [20]byte{}
		if n, err := reader.Read(queryEthCallTo[:]); err != nil || n != 20 {
			return nil, fmt.Errorf("failed to read call To [%d]: %w", n, err)
		}

		queryEthCallDataLen := uint32(0)
		if err := binary.Read(reader, binary.BigEndian, &queryEthCallDataLen); err != nil {
			return nil, fmt.Errorf("failed to read call Data len: %w", err)
		}
		queryEthCallData := make([]byte, queryEthCallDataLen)
		if n, err := reader.Read(queryEthCallData[:]); err != nil || n != int(queryEthCallDataLen) {
			return nil, fmt.Errorf("failed to read call To [%d]: %w", n, err)
		}

		callData := &gossipv1.EthCallQueryRequest_EthCallData{
			To:   queryEthCallTo[:],
			Data: queryEthCallData[:],
		}

		ethCallQueryRequest.CallData = append(ethCallQueryRequest.CallData, callData)
	}

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

	return queryRequest, nil
}

// ValidateQueryRequest does basic validation on a received query request.
func ValidateQueryRequest(queryRequest *gossipv1.QueryRequest) error {
	if queryRequest.ChainId > math.MaxUint16 {
		return fmt.Errorf("invalid chain id: %d is out of bounds", queryRequest.ChainId)
	}
	switch req := queryRequest.Message.(type) {
	case *gossipv1.QueryRequest_EthCallQueryRequest:
		if len(req.EthCallQueryRequest.Block) > math.MaxUint32 {
			return fmt.Errorf("request block too long")
		}
		if !strings.HasPrefix(req.EthCallQueryRequest.Block, "0x") {
			return fmt.Errorf("request block must be a hex number or hash starting with 0x")
		}
		for _, callData := range req.EthCallQueryRequest.CallData {
			if len(callData.To) != 20 {
				return fmt.Errorf("invalid length for To contract")
			}
			if len(callData.Data) > math.MaxUint32 {
				return fmt.Errorf("request data too long")
			}
		}
	default:
		return fmt.Errorf("received invalid message from query module")
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

func QueryRequestEqual(left *gossipv1.QueryRequest, right *gossipv1.QueryRequest) bool {
	if left.ChainId != right.ChainId {
		return false
	}
	if left.Nonce != right.Nonce {
		return false
	}

	switch reqLeft := left.Message.(type) {
	case *gossipv1.QueryRequest_EthCallQueryRequest:
		switch reqRight := right.Message.(type) {
		case *gossipv1.QueryRequest_EthCallQueryRequest:
			if reqLeft.EthCallQueryRequest.Block != reqRight.EthCallQueryRequest.Block {
				return false
			}
			if len(reqLeft.EthCallQueryRequest.CallData) != len(reqRight.EthCallQueryRequest.CallData) {
				return false
			}
			for idx := range reqLeft.EthCallQueryRequest.CallData {
				if !bytes.Equal(reqLeft.EthCallQueryRequest.CallData[idx].To, reqRight.EthCallQueryRequest.CallData[idx].To) {
					return false
				}
				if !bytes.Equal(reqLeft.EthCallQueryRequest.CallData[idx].Data, reqRight.EthCallQueryRequest.CallData[idx].Data) {
					return false
				}
			}
		default:
			return false
		}
	default:
		return false
	}

	return true
}
