package connectors

import (
	"context"
	"fmt"
	"math/big"
	"time"
)

func GetLatestBlock(ctx context.Context, conn Connector) (*NewBlock, error) {
	return GetBlockByFinality(ctx, conn, Latest)
}

func GetBlockByFinality(ctx context.Context, conn Connector, blockFinality FinalityLevel) (*NewBlock, error) {
	return GetBlock(ctx, conn, blockFinality.String(), blockFinality)
}

func GetBlockByNumberUint64(ctx context.Context, conn Connector, blockNum uint64, blockFinality FinalityLevel) (*NewBlock, error) {
	return GetBlock(ctx, conn, "0x"+fmt.Sprintf("%x", blockNum), blockFinality)
}

func GetBlock(ctx context.Context, conn Connector, str string, blockFinality FinalityLevel) (*NewBlock, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var m BlockMarshaller
	err := conn.RawCallContext(timeout, &m, "eth_getBlockByNumber", str, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get block for %s: %w", str, err)
	}
	if m.Number == nil {
		return nil, fmt.Errorf("failed to unmarshal block for %s: Number is nil", str)
	}
	n := big.Int(*m.Number)

	var l1bn *big.Int
	if m.L1BlockNumber != nil {
		bn := big.Int(*m.L1BlockNumber)
		l1bn = &bn
	}

	return &NewBlock{
		Number:        &n,
		Time:          uint64(m.Time),
		Hash:          m.Hash,
		L1BlockNumber: l1bn,
		Finality:      blockFinality,
	}, nil
}
