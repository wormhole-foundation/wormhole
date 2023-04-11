package nearapi

import (
	"errors"

	"github.com/tidwall/gjson"
)

type (
	BlockHeader struct {
		Hash           string
		PrevBlockHash  string
		Height         uint64
		Timestamp      uint64
		LastFinalBlock string
	}

	Block struct {
		bytes  []byte
		json   gjson.Result
		Header BlockHeader
	}

	ChunkHeader struct {
		Hash string
	}

	Chunk struct {
		Hash  string
		bytes []byte
		json  gjson.Result
	}

	Transaction struct {
		Hash     string
		SignerId string
	}
)

func NewChunkFromBytes(bytes []byte) (Chunk, error) {
	if !gjson.ValidBytes(bytes) {
		return Chunk{}, errors.New("invalid json")
	}

	json := gjson.ParseBytes(bytes)

	hash := jsonGetString(json, "result.header.chunk_hash")

	if hash == "" {
		return Chunk{}, errors.New("invalid json")
	}

	return Chunk{
		Hash:  hash,
		bytes: bytes,
		json:  json,
	}, nil
}

func NewBlockFromBytes(bytes []byte) (Block, error) {
	if !gjson.ValidBytes(bytes) {
		return Block{}, errors.New("invalid json")
	}

	json := gjson.ParseBytes(bytes)

	ts_nanosec := jsonGetUint(json, "result.header.timestamp")
	ts := ts_nanosec / 1_000_000_000

	header := BlockHeader{
		jsonGetString(json, "result.header.hash"),
		jsonGetString(json, "result.header.prev_hash"),
		jsonGetUint(json, "result.header.height"),
		ts,
		jsonGetString(json, "result.header.last_final_block"),
	}

	return Block{
		bytes:  bytes,
		json:   json,
		Header: header,
	}, nil
}

func (b Block) Timestamp() uint64 {
	ts_nanosec := jsonGetUint(b.json, "result.header.timestamp")
	return ts_nanosec / 1000000000
}

func (b Block) ChunkHashes() []ChunkHeader {
	chunks := make([]ChunkHeader, 0, 10) //capacity 10

	// get the hashes of all chunks in the block
	hashes := b.json.Get("result.chunks.#.chunk_hash")

	if !hashes.Exists() {
		// if there are no hashes, there's nothing to do. Return early.
		return nil
	}

	for _, chunkHash := range hashes.Array() {
		if chunkHash.String() == "" || IsWellFormedHash(chunkHash.String()) != nil {
			continue
		}
		chunks = append(chunks, ChunkHeader{chunkHash.String()})
	}
	return chunks
}

func (c Chunk) Height() uint64 {
	return jsonGetUint(c.json, "result.header.height_included")
}

func (c Chunk) Transactions() []Transaction {
	result := make([]Transaction, 0, 10)

	txns := c.json.Get("result.transactions")
	if !txns.Exists() {
		return nil
	}

	for _, r := range txns.Array() {
		hash := r.Get("hash")
		signer_id := r.Get("signer_id")
		if !hash.Exists() || !signer_id.Exists() {
			continue
		}
		result = append(result, Transaction{hash.String(), signer_id.String()})
	}
	return result
}

func jsonGetString(json gjson.Result, property string) string {
	if !json.Exists() {
		return ""
	}
	v := json.Get(property)
	if !v.Exists() {
		return ""
	}
	return v.String()
}

func jsonGetUint(json gjson.Result, property string) uint64 {
	if !json.Exists() {
		return 0
	}
	v := json.Get(property)
	if !v.Exists() {
		return 0
	}
	return v.Uint()
}
