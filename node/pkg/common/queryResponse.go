package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/ethereum/go-ethereum/common"
	eth_hexutil "github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"google.golang.org/protobuf/proto"
)

var queryResponsePrefix = []byte("query_response_0000000000000000000|")

type EthCallQueryResponse struct {
	Number *eth_hexutil.Big
	Hash   common.Hash
	Time   eth_hexutil.Uint64
	Result []byte
}

type QueryResponsePublication struct {
	Request  *gossipv1.SignedQueryRequest
	Response EthCallQueryResponse
}

func (msg *QueryResponsePublication) Marshal() ([]byte, error) {
	// TODO: copy request write checks to query module request handling
	// TODO: only receive the unmarshalled query request (see note in query.go)
	var queryRequest gossipv1.QueryRequest
	err := proto.Unmarshal(msg.Request.QueryRequest, &queryRequest)
	if err != nil {
		return nil, fmt.Errorf("received invalid message from query module")
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
		vaa.MustWrite(buf, binary.BigEndian, queryRequest.ChainId) // uint32
		vaa.MustWrite(buf, binary.BigEndian, queryRequest.Nonce)   // uint32
		if len(req.EthCallQueryRequest.To) != 20 {
			return nil, fmt.Errorf("invalid length for To contract")
		}
		buf.Write(req.EthCallQueryRequest.To)
		if len(req.EthCallQueryRequest.Data) > math.MaxUint32 {
			return nil, fmt.Errorf("request data too long")
		}
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(req.EthCallQueryRequest.Data)))
		buf.Write(req.EthCallQueryRequest.Data)
		if len(req.EthCallQueryRequest.Block) > math.MaxUint32 {
			return nil, fmt.Errorf("request block too long")
		}
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(req.EthCallQueryRequest.Block)))
		// TODO: should this be an enum or the literal string?
		buf.Write([]byte(req.EthCallQueryRequest.Block))

		// Response
		// TODO: probably some kind of request/response pair validation
		vaa.MustWrite(buf, binary.BigEndian, msg.Response.Number.ToInt().Uint64())
		if len(msg.Response.Hash) != 32 {
			return nil, fmt.Errorf("invalid length for block hash")
		}
		buf.Write(msg.Response.Hash[:])
		vaa.MustWrite(buf, binary.BigEndian, uint32(time.Unix(int64(msg.Response.Time), 0).Unix()))
		if len(msg.Response.Result) > math.MaxUint32 {
			return nil, fmt.Errorf("response data too long")
		}
		vaa.MustWrite(buf, binary.BigEndian, uint32(len(msg.Response.Result)))
		buf.Write(msg.Response.Result)
		return buf.Bytes(), nil
	default:
		return nil, fmt.Errorf("received invalid message from query module")
	}
}

// TODO
// Unmarshal deserializes the binary representation of a VAA
// func UnmarshalMessagePublication(data []byte) (*MessagePublication, error) {
// 	if len(data) < minMsgLength {
// 		return nil, fmt.Errorf("message is too short")
// 	}

// 	msg := &MessagePublication{}

// 	reader := bytes.NewReader(data[:])

// 	txHash := common.Hash{}
// 	if n, err := reader.Read(txHash[:]); err != nil || n != 32 {
// 		return nil, fmt.Errorf("failed to read TxHash [%d]: %w", n, err)
// 	}
// 	msg.TxHash = txHash

// 	unixSeconds := uint32(0)
// 	if err := binary.Read(reader, binary.BigEndian, &unixSeconds); err != nil {
// 		return nil, fmt.Errorf("failed to read timestamp: %w", err)
// 	}
// 	msg.Timestamp = time.Unix(int64(unixSeconds), 0)

// 	if err := binary.Read(reader, binary.BigEndian, &msg.Nonce); err != nil {
// 		return nil, fmt.Errorf("failed to read nonce: %w", err)
// 	}

// 	if err := binary.Read(reader, binary.BigEndian, &msg.Sequence); err != nil {
// 		return nil, fmt.Errorf("failed to read sequence: %w", err)
// 	}

// 	if err := binary.Read(reader, binary.BigEndian, &msg.ConsistencyLevel); err != nil {
// 		return nil, fmt.Errorf("failed to read consistency level: %w", err)
// 	}

// 	if err := binary.Read(reader, binary.BigEndian, &msg.EmitterChain); err != nil {
// 		return nil, fmt.Errorf("failed to read emitter chain: %w", err)
// 	}

// 	emitterAddress := vaa.Address{}
// 	if n, err := reader.Read(emitterAddress[:]); err != nil || n != 32 {
// 		return nil, fmt.Errorf("failed to read emitter address [%d]: %w", n, err)
// 	}
// 	msg.EmitterAddress = emitterAddress

// 	payload := make([]byte, reader.Len())
// 	n, err := reader.Read(payload)
// 	if err != nil || n == 0 {
// 		return nil, fmt.Errorf("failed to read payload [%d]: %w", n, err)
// 	}
// 	msg.Payload = payload[:n]

// 	return msg, nil
// }

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
