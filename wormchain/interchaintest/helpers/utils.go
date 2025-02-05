package helpers

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	abcitypes "github.com/tendermint/tendermint/abci/types"
)

func MustAccAddressFromBech32(address string, bech32Prefix string) sdk.AccAddress {
	if len(strings.TrimSpace(address)) == 0 {
		panic("empty address string is not allowed")
	}

	bz, err := sdk.GetFromBech32(address, bech32Prefix)
	if err != nil {
		panic(err)
	}

	err = sdk.VerifyAddressFormat(bz)
	if err != nil {
		panic(err)
	}

	return sdk.AccAddress(bz)
}

func FindEventAttribute(t *testing.T, chain *cosmos.CosmosChain, txHash string, eventType string, attributeKey string, attributeValue string) bool {
	tx, err := chain.GetTransaction(txHash)
	require.NoError(t, err)
	for _, event := range tx.Events {
		if event.Type == eventType {
			for _, attribute := range event.Attributes {
				if string(attribute.Key) == attributeKey && string(attribute.Value) == attributeValue {
					fmt.Println("Found: ", eventType, " ", attributeKey, " ", attributeValue)
					return true
				}
			}
		}
	}
	fmt.Println("Not found: ", eventType, " ", attributeKey, " ", attributeValue, "!")
	return false
}

// FindOpenChannelByVersion queries all the channels of a given chain and returns the first with the given version. If no channel is found, it will fail the test.
func FindOpenChannelByVersion(
	t *testing.T,
	ctx context.Context,
	eRep *testreporter.RelayerExecReporter,
	r ibc.Relayer,
	chain *cosmos.CosmosChain,
	version string) ibc.ChannelOutput {
	// iterate up to 20 times to allow for chain to catch up
	for i := 0; i < 20; i++ {

		channels, err := r.GetChannels(ctx, eRep, chain.Config().ChainID)
		require.NoError(t, err)

		channelIdx := slices.IndexFunc(channels, func(channel ibc.ChannelOutput) bool {
			return channel.State == "STATE_OPEN" && channel.Version == version
		})
		if channelIdx != -1 {
			return channels[channelIdx]
		}
		testutil.WaitForBlocks(ctx, 1, chain)
	}

	require.Failf(t, "channel with version %s not found", version)
	return ibc.ChannelOutput{}
}

func GetIBCTx(
	c *cosmos.CosmosChain,
	txHash string,
) (tx ibc.Tx, _ error) {
	txResp, err := c.GetTransaction(txHash)
	if err != nil {
		return tx, fmt.Errorf("failed to get transaction %s: %w", txHash, err)
	}
	tx.Height = uint64(txResp.Height)
	tx.TxHash = txHash
	// In cosmos, user is charged for entire gas requested, not the actual gas used.
	tx.GasSpent = txResp.GasWanted

	const evType = "send_packet"
	events := txResp.Events

	var (
		seq, _           = AttributeValue(events, evType, "packet_sequence")
		srcPort, _       = AttributeValue(events, evType, "packet_src_port")
		srcChan, _       = AttributeValue(events, evType, "packet_src_channel")
		dstPort, _       = AttributeValue(events, evType, "packet_dst_port")
		dstChan, _       = AttributeValue(events, evType, "packet_dst_channel")
		timeoutHeight, _ = AttributeValue(events, evType, "packet_timeout_height")
		timeoutTs, _     = AttributeValue(events, evType, "packet_timeout_timestamp")
		data, _          = AttributeValue(events, evType, "packet_data")
	)
	tx.Packet.SourcePort = srcPort
	tx.Packet.SourceChannel = srcChan
	tx.Packet.DestPort = dstPort
	tx.Packet.DestChannel = dstChan
	tx.Packet.TimeoutHeight = timeoutHeight
	tx.Packet.Data = []byte(data)

	seqNum, err := strconv.Atoi(seq)
	if err != nil {
		return tx, fmt.Errorf("invalid packet sequence from events %s: %w", seq, err)
	}
	tx.Packet.Sequence = uint64(seqNum)

	timeoutNano, err := strconv.ParseUint(timeoutTs, 10, 64)
	if err != nil {
		return tx, fmt.Errorf("invalid packet timestamp timeout %s: %w", timeoutTs, err)
	}
	tx.Packet.TimeoutTimestamp = ibc.Nanoseconds(timeoutNano)

	return tx, nil
}

func AttributeValue(events []abcitypes.Event, eventType, attrKey string) (string, bool) {
	for _, event := range events {
		if event.Type != eventType {
			continue
		}
		for _, attr := range event.Attributes {
			if string(attr.Key) == attrKey {
				return string(attr.Value), true
			}
		}
	}
	return "", false
}
