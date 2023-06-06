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

const SignedQueryRequestChannelSize = 50

// PerChainQueryInternal is an internal representation of a query request that is passed to the watcher.
type PerChainQueryInternal struct {
	RequestID  string
	RequestIdx int
	ChainID    vaa.ChainID
	Request    *gossipv1.PerChainQueryRequest
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

// MarshalQueryRequest serializes the binary representation of a query request
func MarshalQueryRequest(queryRequest *gossipv1.QueryRequest) ([]byte, error) {
	buf := new(bytes.Buffer)

	vaa.MustWrite(buf, binary.BigEndian, queryRequest.Nonce) // uint32

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(queryRequest.PerChainQueries)))
	for _, perChainQuery := range queryRequest.PerChainQueries {
		pcqBuf, err := MarshalPerChainQueryRequest(perChainQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal per chain query")
		}
		buf.Write(pcqBuf)
	}

	return buf.Bytes(), nil
}

// MarshalQueryRequest serializes the binary representation of a per chain query request
func MarshalPerChainQueryRequest(perChainQuery *gossipv1.PerChainQueryRequest) ([]byte, error) {
	buf := new(bytes.Buffer)
	switch req := perChainQuery.Message.(type) {
	case *gossipv1.PerChainQueryRequest_EthCallQueryRequest:
		vaa.MustWrite(buf, binary.BigEndian, QUERY_REQUEST_TYPE_ETH_CALL)
		vaa.MustWrite(buf, binary.BigEndian, uint16(perChainQuery.ChainId))
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

// UnmarshalQueryRequest deserializes the binary representation of a query request from a byte array
func UnmarshalQueryRequest(data []byte) (*gossipv1.QueryRequest, error) {
	reader := bytes.NewReader(data[:])
	return UnmarshalQueryRequestFromReader(reader)
}

// UnmarshalQueryRequestFromReader deserializes the binary representation of a query request from an existing reader
func UnmarshalQueryRequestFromReader(reader *bytes.Reader) (*gossipv1.QueryRequest, error) {
	queryRequest := &gossipv1.QueryRequest{}

	queryNonce := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &queryNonce); err != nil {
		return nil, fmt.Errorf("failed to read request nonce: %w", err)
	}
	queryRequest.Nonce = queryNonce

	numPerChainQueries := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numPerChainQueries); err != nil {
		return nil, fmt.Errorf("failed to read number of per chain queries: %w", err)
	}

	for count := 0; count < int(numPerChainQueries); count++ {
		perChainQuery, err := UnmarshalPerChainQueryRequestFromReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal per chain query: %w", err)
		}
		queryRequest.PerChainQueries = append(queryRequest.PerChainQueries, perChainQuery)
	}

	return queryRequest, nil
}

// UnmarshalPerChainQueryRequest deserializes the binary representation of a per chain query request from a byte array
func UnmarshalPerChainQueryRequest(data []byte) (*gossipv1.PerChainQueryRequest, error) {
	reader := bytes.NewReader(data[:])
	return UnmarshalPerChainQueryRequestFromReader(reader)
}

// UnmarshalPerChainQueryRequestFromReader deserializes the binary representation of a per chain query request from an existing reader
func UnmarshalPerChainQueryRequestFromReader(reader *bytes.Reader) (*gossipv1.PerChainQueryRequest, error) {
	perChainQuery := &gossipv1.PerChainQueryRequest{}

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
	perChainQuery.ChainId = uint32(queryChain)

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

	perChainQuery.Message = &gossipv1.PerChainQueryRequest_EthCallQueryRequest{
		EthCallQueryRequest: ethCallQueryRequest,
	}

	return perChainQuery, nil
}

// ValidateQueryRequest does basic validation on a received query request.
func ValidateQueryRequest(queryRequest *gossipv1.QueryRequest) error {
	if len(queryRequest.PerChainQueries) == 0 {
		return fmt.Errorf("request does not contain any queries")
	}
	for _, perChainQuery := range queryRequest.PerChainQueries {
		if perChainQuery.ChainId > math.MaxUint16 {
			return fmt.Errorf("invalid chain id: %d is out of bounds", perChainQuery.ChainId)
		}
		switch req := perChainQuery.Message.(type) {
		case *gossipv1.PerChainQueryRequest_EthCallQueryRequest:
			if len(req.EthCallQueryRequest.Block) > math.MaxUint32 {
				return fmt.Errorf("request block too long")
			}
			if !strings.HasPrefix(req.EthCallQueryRequest.Block, "0x") {
				return fmt.Errorf("request block must be a hex number or hash starting with 0x")
			}
			if len(req.EthCallQueryRequest.CallData) == 0 {
				return fmt.Errorf("per chain query does not contain any requests")
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
	if left.Nonce != right.Nonce {
		return false
	}
	if len(left.PerChainQueries) != len(right.PerChainQueries) {
		return false
	}

	for idx := range left.PerChainQueries {
		if left.PerChainQueries[idx].ChainId != right.PerChainQueries[idx].ChainId {
			return false
		}

		switch reqLeft := left.PerChainQueries[idx].Message.(type) {
		case *gossipv1.PerChainQueryRequest_EthCallQueryRequest:
			switch reqRight := right.PerChainQueries[idx].Message.(type) {
			case *gossipv1.PerChainQueryRequest_EthCallQueryRequest:
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
	}

	return true
}
