package ccq

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

func TestParseConfigFileDoesntExist(t *testing.T) {
	_, err := parseConfigFile("missingFile.json", false)
	require.Error(t, err)
	assert.Equal(t, `failed to open permissions file "missingFile.json": open missingFile.json: no such file or directory`, err.Error())
}

func TestParseConfigBadJson(t *testing.T) {
	str := `
	{
  "permissions": [
    {
      "userName": "Test User",
      "apiKey": "my_secret_key",
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "B4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "0x06fdde03"
          }
        }
      ]
    },
    {
      "userName": "Test User",
      "apiKey": "my_secret_key_2",
      "allowUnsigned": true,
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "0x06fdde03"
          }
        }
      ]
    }`

	_, err := parseConfig([]byte(str), false)
	require.Error(t, err)
	assert.Equal(t, `failed to unmarshal json: unexpected end of JSON input`, err.Error())
}

func TestParseConfigDuplicateUser(t *testing.T) {
	str := `
	{
  "permissions": [
    {
      "userName": "Test User",
      "apiKey": "my_secret_key",
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "B4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "0x06fdde03"
          }
        }
      ]
    },
    {
      "userName": "Test User",
      "apiKey": "my_secret_key_2",
      "allowUnsigned": true,
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "0x06fdde03"
          }
        }
      ]
    }
  ]
}`

	_, err := parseConfig([]byte(str), false)
	require.Error(t, err)
	assert.Equal(t, `UserName "Test User" is a duplicate`, err.Error())
}

func TestParseConfigDuplicateApiKey(t *testing.T) {
	str := `
	{
  "permissions": [
    {
      "userName": "Test User 1",
      "apiKey": "my_secret_key",
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "B4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "0x06fdde03"
          }
        }
      ]
    },
    {
      "userName": "Test User 2",
      "apiKey": "my_secret_key",
      "allowUnsigned": true,
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "0x06fdde03"
          }
        }
      ]
    }
  ]
}`

	_, err := parseConfig([]byte(str), false)
	require.Error(t, err)
	assert.Equal(t, `API key "my_secret_key" is a duplicate`, err.Error())
}

func TestParseConfigUnsupportedCallType(t *testing.T) {
	str := `
	{
  "permissions": [
    {
      "userName": "Test User",
      "apiKey": "my_secret_key",
      "allowedCalls": [
        {
          "badCallType": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "B4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "0x06fdde03"
          }
        }
      ]
    }
  ]
}`

	_, err := parseConfig([]byte(str), false)
	require.Error(t, err)
	assert.Equal(t, `unsupported call type for user "Test User", must be "ethCall", "ethCallByTimestamp", "ethCallWithFinality", "solAccount" or "solPDA"`, err.Error())
}

func TestParseConfigInvalidContractAddress(t *testing.T) {
	str := `
	{
  "permissions": [
    {
      "userName": "Test User",
      "apiKey": "my_secret_key",
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "HelloWorld",
            "call": "0x06fdde"
          }
        }
      ]
    }
  ]
}`

	_, err := parseConfig([]byte(str), false)
	require.Error(t, err)
	assert.Equal(t, `invalid contract address "HelloWorld" for user "Test User"`, err.Error())
}

func TestParseConfigInvalidEthCall(t *testing.T) {
	str := `
	{
  "permissions": [
    {
      "userName": "Test User",
      "apiKey": "my_secret_key",
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "B4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "HelloWorld"
          }
        }
      ]
    }
  ]
}`

	_, err := parseConfig([]byte(str), false)
	require.Error(t, err)
	assert.Equal(t, `invalid eth call "HelloWorld" for user "Test User"`, err.Error())
}

func TestParseConfigInvalidEthCallLength(t *testing.T) {
	str := `
	{
  "permissions": [
    {
      "userName": "Test User",
      "apiKey": "my_secret_key",
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "B4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "0x06fd"
          }
        }
      ]
    }
  ]
}`

	_, err := parseConfig([]byte(str), false)
	require.Error(t, err)
	assert.Equal(t, `eth call "0x06fd" for user "Test User" has an invalid length, must be 4 bytes`, err.Error())
}

func TestParseConfigDuplicateAllowedCallForUser(t *testing.T) {
	str := `
	{
  "permissions": [
    {
      "userName": "Test User",
      "apiKey": "my_secret_key",
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "B4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "0x06fdde03"
          }
        },
        {
          "ethCall": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "B4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "0x06fdde03"
          }
        }			
      ]
    }
  ]
}`

	_, err := parseConfig([]byte(str), false)
	require.Error(t, err)
	assert.Equal(t, `"ethCall:2:000000000000000000000000b4fbf271143f4fbf7b91a5ded31805e42b2208d6:06fdde03" is a duplicate allowed call for user "Test User"`, err.Error())
}

func TestParseConfigSuccess(t *testing.T) {
	str := `
	{
  "permissions": [
    {
      "userName": "Test User",
      "apiKey": "My_secret_key",
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "B4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "0x06fdde03"
          }
        },
        {
          "ethCallByTimestamp": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "B4FBF271143F4FBf7B91A5ded31805e42b2208d7",
            "call": "0x06fdde03"
          }
        },
        {
          "ethCallWithFinality": {
            "note:": "Decimals of WETH on Devnet",
            "chain": 2,
            "contractAddress": "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E",
            "call": "0x313ce567"
          }
        },        
        {
          "solAccount": {
            "note:": "Example NFT on Devnet",
            "chain": 1,
            "account": "BVxyYhm498L79r4HMQ9sxZ5bi41DmJmeWZ7SCS7Cyvna"
          }
        },
        {
          "solPDA": {
            "note:": "Core Bridge on Devnet",
            "chain": 1,
            "programAddress": "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"
          }
        }
      ]
    }
  ]
}`

	perms, err := parseConfig([]byte(str), false)
	require.NoError(t, err)
	assert.Equal(t, 1, len(perms))

	perm, exists := perms["my_secret_key"]
	require.True(t, exists)

	assert.Equal(t, 5, len(perm.allowedCalls))

	_, exists = perm.allowedCalls["ethCall:2:000000000000000000000000b4fbf271143f4fbf7b91a5ded31805e42b2208d6:06fdde03"]
	assert.True(t, exists)

	_, exists = perm.allowedCalls["ethCallByTimestamp:2:000000000000000000000000b4fbf271143f4fbf7b91a5ded31805e42b2208d7:06fdde03"]
	assert.True(t, exists)

	_, exists = perm.allowedCalls["ethCallWithFinality:2:000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e:313ce567"]
	assert.True(t, exists)

	_, exists = perm.allowedCalls["solAccount:1:BVxyYhm498L79r4HMQ9sxZ5bi41DmJmeWZ7SCS7Cyvna"]
	assert.True(t, exists)

	_, exists = perm.allowedCalls["solPDA:1:Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"]
	assert.True(t, exists)
}

func TestParseConfigAllowAnythingWhenNotEnabled(t *testing.T) {
	str := `
	{
  "permissions": [
    {
      "userName": "Test User",
      "apiKey": "my_secret_key",
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "B4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "0x06fdde03"
          }
        }
      ]
    },
    {
      "userName": "Test User2",
      "apiKey": "my_secret_key_2",
      "allowUnsigned": true,
      "allowAnything": true
    }
  ]
}`

	_, err := parseConfig([]byte(str), false)
	require.Error(t, err)
	assert.Equal(t, `UserName "Test User2" has "allowAnything" specified when the feature is not enabled`, err.Error())
}

func TestParseConfigAllowAnythingWithAllowedCallsIsInvalid(t *testing.T) {
	str := `
	{
  "permissions": [
    {
      "userName": "Test User",
      "apiKey": "my_secret_key",
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "B4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "0x06fdde03"
          }
        }
      ]
    },
    {
      "userName": "Test User2",
      "apiKey": "my_secret_key_2",
      "allowUnsigned": true,
      "allowAnything": true,
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "0x06fdde03"
          }
        }
      ]
    }
  ]
}`

	_, err := parseConfig([]byte(str), true)
	require.Error(t, err)
	assert.Equal(t, `UserName "Test User2" has "allowedCalls" specified with "allowAnything", which is not allowed`, err.Error())
}

func TestParseConfigAllowAnythingSuccess(t *testing.T) {
	str := `
	{
  "permissions": [
    {
      "userName": "Test User",
      "apiKey": "my_secret_key",
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of WETH on Goerli",
            "chain": 2,
            "contractAddress": "B4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "0x06fdde03"
          }
        }
      ]
    },
    {
      "userName": "Test User2",
      "apiKey": "my_secret_key_2",
      "allowUnsigned": true,
      "allowAnything": true
    }
  ]
}`

	perms, err := parseConfig([]byte(str), true)
	require.NoError(t, err)
	assert.Equal(t, 2, len(perms))

	perm, ok := perms["my_secret_key"]
	require.True(t, ok)
	assert.False(t, perm.allowAnything)

	perm, ok = perms["my_secret_key_2"]
	require.True(t, ok)
	assert.True(t, perm.allowAnything)
}

func TestParseConfigContractWildcard(t *testing.T) {
	str := `
	{
  "permissions": [
    {
      "userName": "Test User",
      "apiKey": "my_secret_key",
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of anything on Goerli",
            "chain": 2,
            "contractAddress": "*",
            "call": "0x06fdde03"
          }
        },
        {
          "ethCallByTimestamp": {
            "note:": "Total supply of WETH on Goerli",
            "chain": 2,
            "contractAddress": "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6",
            "call": "0x18160ddd"
          }    
        }   
      ]
    }
  ]
}`

	perms, err := parseConfig([]byte(str), true)
	require.NoError(t, err)
	assert.Equal(t, 1, len(perms))

	permsForUser, ok := perms["my_secret_key"]
	require.True(t, ok)
	assert.Equal(t, 2, len(permsForUser.allowedCalls))

	logger := zap.NewNop()

	type testCase struct {
		label           string
		callType        string
		chainID         vaa.ChainID
		contractAddress string
		data            string
		errText         string // empty string means success
	}

	var testCases = []testCase{
		{
			label:           "Wild card, success",
			callType:        "ethCall",
			chainID:         vaa.ChainIDEthereum,
			contractAddress: "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6",
			data:            "0x06fdde03",
			errText:         "",
		},
		{
			label:           "Wild card, success, different address",
			callType:        "ethCall",
			chainID:         vaa.ChainIDEthereum,
			contractAddress: "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d7",
			data:            "0x06fdde03",
			errText:         "",
		},
		{
			label:           "Wild card, wrong call type",
			callType:        "ethCallByTimestamp",
			chainID:         vaa.ChainIDEthereum,
			contractAddress: "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6",
			data:            "0x06fdde03",
			errText:         "not authorized",
		},
		{
			label:           "Wild card, wrong chain",
			callType:        "ethCall",
			chainID:         vaa.ChainIDBase,
			contractAddress: "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6",
			data:            "0x06fdde03",
			errText:         "not authorized",
		},
		{
			label:           "Specific, success",
			callType:        "ethCallByTimestamp",
			chainID:         vaa.ChainIDEthereum,
			contractAddress: "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6",
			data:            "0x18160ddd",
			errText:         "",
		},
		{
			label:           "Specific, wrong call type",
			callType:        "ethCall",
			chainID:         vaa.ChainIDEthereum,
			contractAddress: "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6",
			data:            "0x18160ddd",
			errText:         "not authorized",
		},
		{
			label:           "Specific, wrong chain",
			callType:        "ethCallByTimestamp",
			chainID:         vaa.ChainIDBase,
			contractAddress: "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6",
			data:            "0x18160ddd",
			errText:         "not authorized",
		},
		{
			label:           "Specific, wrong address",
			callType:        "ethCallByTimestamp",
			chainID:         vaa.ChainIDEthereum,
			contractAddress: "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d7",
			data:            "0x18160ddd",
			errText:         "not authorized",
		},
		{
			label:           "Specific, wrong data",
			callType:        "ethCallByTimestamp",
			chainID:         vaa.ChainIDEthereum,
			contractAddress: "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6",
			data:            "0x18160dde",
			errText:         "not authorized",
		},
	}

	for _, tst := range testCases {
		t.Run(tst.label, func(t *testing.T) {
			status, err := validateCallData(logger, permsForUser, tst.callType, tst.chainID, createCallData(t, tst.contractAddress, tst.data))
			if tst.errText == "" {
				require.NoError(t, err)
				assert.Equal(t, 200, status)
			} else {
				require.ErrorContains(t, err, tst.errText)
			}
		})
	}
}

func createCallData(t *testing.T, toStr string, dataStr string) []*query.EthCallData {
	t.Helper()
	to, err := vaa.StringToAddress(strings.TrimPrefix(toStr, "0x"))
	require.NoError(t, err)

	data, err := hex.DecodeString(strings.TrimPrefix(dataStr, "0x"))
	require.NoError(t, err)

	return []*query.EthCallData{
		{
			To:   to.Bytes(),
			Data: data,
		},
	}
}
