package helpers

import (
	"fmt"
	"strings"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"

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
