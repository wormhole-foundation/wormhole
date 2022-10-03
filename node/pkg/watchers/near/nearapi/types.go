package nearapi

import (
	"errors"

	"github.com/tidwall/gjson"
)

const (
	BlockHashLen = 44
	chunkHashLen = 44
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
		//BlockHeightIncluded uint64
	}

	Chunk struct {
		Hash  string
		bytes []byte
		json  gjson.Result
	}

	Transaction struct {
		Hash       string
		ReceiverId string
	}
)

func newChunkFromBytes(bytes []byte) (Chunk, error) {
	if !gjson.ValidBytes(bytes) {
		return Chunk{}, errors.New("invalid json")
	}

	return Chunk{
		bytes: bytes,
		json:  gjson.ParseBytes(bytes),
	}, nil
}

func newBlockFromBytes(bytes []byte) (Block, error) {
	if !gjson.ValidBytes(bytes) {
		return Block{}, errors.New("invalid json")
	}

	json := gjson.ParseBytes(bytes)

	ts_nanosec := jsonGetUint(json, "result.header.timestamp")
	ts := uint64(ts_nanosec) / 1_000_000_000

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
	return uint64(ts_nanosec) / 1000000000
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
		if chunkHash.String() == "" || len(chunkHash.String()) != chunkHashLen {
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
		receiver_id := r.Get("receiver_id")
		if !hash.Exists() || !receiver_id.Exists() {
			continue
		}
		result = append(result, Transaction{hash.String(), receiver_id.String()})
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
