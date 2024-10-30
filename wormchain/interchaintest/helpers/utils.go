package helpers

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"github.com/strangelove-ventures/interchaintest/v4/testreporter"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
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

// FindChannelByVersion queries all the channels of a given chain and returns the first with the given version. If no channel is found, it will fail the test.
func FindChannelByVersion(t *testing.T, ctx context.Context, eRep *testreporter.RelayerExecReporter, r ibc.Relayer, chainId string, version string) ibc.ChannelOutput {
	channels, err := r.GetChannels(ctx, eRep, chainId)
	require.NoError(t, err)

	channelIdx := slices.IndexFunc(channels, func(channel ibc.ChannelOutput) bool { return channel.Version == version })
	require.NotEqual(t, -1, channelIdx, "channel with version %s not found", version)

	return channels[channelIdx]
}
