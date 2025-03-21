package solana

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func TestShimMatchPrefix(t *testing.T) {
	type test struct {
		input  string
		result bool
		label  string
	}
	tests := []test{
		{input: "d63264d12622074cdeadbeef", result: true, label: "Success"},
		{input: "d6", result: false, label: "Too_short"},
		{input: "", result: false, label: "Empty"},
		{input: shimPostMessageDiscriminatorStr, result: true, label: "Exact_match"},
		{input: "d73264d12622074cdeadbeef", result: false, label: "No_match"},
	}

	shimPostMessage, err := hex.DecodeString(shimPostMessageDiscriminatorStr)
	require.NoError(t, err)

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			buf, err := hex.DecodeString(tc.input)
			require.NoError(t, err)

			assert.Equal(t, tc.result, shimMatchPrefix(shimPostMessage, buf))
		})
	}
}

func Test_shimParsePostMessage(t *testing.T) {
	shimPostMessage, err := hex.DecodeString(shimPostMessageDiscriminatorStr)
	require.NoError(t, err)

	data, err := hex.DecodeString("d63264d12622074c2a000000010b00000068656c6c6f20776f726c64")
	require.NoError(t, err)

	postMsgData, err := shimParsePostMessage(shimPostMessage, data)
	require.NoError(t, err)
	require.NotNil(t, postMsgData)

	assert.Equal(t, uint32(42), postMsgData.Nonce)
	assert.Equal(t, consistencyLevelFinalized, postMsgData.ConsistencyLevel)
	assert.Equal(t, 11, len(postMsgData.Payload))
	assert.True(t, bytes.Equal([]byte("hello world"), postMsgData.Payload))
}

func Test_shimVerifyCoreMessage(t *testing.T) {
	data, err := hex.DecodeString("082a0000000000000001")
	require.NoError(t, err)

	coreMsgDataAsExpected, err := shimVerifyCoreMessage(data)
	require.NoError(t, err)
	assert.True(t, coreMsgDataAsExpected)
}

func Test_shimParseMessageEvent(t *testing.T) {
	shimMessageEvent, err := hex.DecodeString("e445a52e51cb9a1d441b8f004d4c8970")
	require.NoError(t, err)

	expectedEmitter, err := hex.DecodeString("041c657e845d65d009d59ceeb1dda172bd6bc9e7ee5a19e56573197cf7fdffde")
	require.NoError(t, err)

	data, err := hex.DecodeString("e445a52e51cb9a1d441b8f004d4c8970041c657e845d65d009d59ceeb1dda172bd6bc9e7ee5a19e56573197cf7fdffde00000000000000007c5b8167")
	require.NoError(t, err)

	msgEventData, err := shimParseMessageEvent(shimMessageEvent, data)
	require.NoError(t, err)
	require.NotNil(t, msgEventData)

	assert.True(t, bytes.Equal(expectedEmitter, msgEventData.EmitterAddress[:]))
	assert.Equal(t, uint64(0), msgEventData.Sequence)
	assert.Equal(t, uint32(1736530812), msgEventData.Timestamp)
}

func TestShimAlreadyProcessed(t *testing.T) {
	alreadyProcessed := ShimAlreadyProcessed{}
	assert.False(t, alreadyProcessed.exists(5, 7))
	alreadyProcessed.add(5, 7)
	assert.True(t, alreadyProcessed.exists(5, 7))
	assert.False(t, alreadyProcessed.exists(5, 8))
}

// WARNING: This only populates a few fields needed by the shim code!
func shimNewWatcherForTest(t *testing.T, msgC chan<- *common.MessagePublication) *SolanaWatcher {
	t.Helper()

	rawContract := "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
	contractAddress, err := solana.PublicKeyFromBase58(rawContract)
	require.NoError(t, err)

	shimContractStr := "EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX"
	shimContractAddr, err := solana.PublicKeyFromBase58(shimContractStr)
	require.NoError(t, err)

	s := &SolanaWatcher{
		contract:         contractAddress,
		rawContract:      rawContract,
		shimContractStr:  shimContractStr,
		shimContractAddr: shimContractAddr,
		chainID:          vaa.ChainIDSolana,
		commitment:       rpc.CommitmentFinalized,
		msgC:             msgC,
	}

	s.shimSetup()
	return s
}

func TestVerifyShimSetup(t *testing.T) {
	msgC := make(chan *common.MessagePublication, 10)
	s := shimNewWatcherForTest(t, msgC)
	assert.True(t, s.shimEnabled)
	assert.Equal(t, shimPostMessageDiscriminatorStr, hex.EncodeToString(s.shimPostMessageDiscriminator))
	assert.Equal(t, shimMessageEventDiscriminatorStr, hex.EncodeToString(s.shimMessageEventDiscriminator))
}

func TestShimDirect(t *testing.T) {
	eventJson := `
	{
		"blockTime": 1736530812,
		"meta": {
			"computeUnitsConsumed": 84252,
			"err": null,
			"fee": 5000,
			"innerInstructions": [
				{
					"index": 1,
					"instructions": [
						{
							"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9],
							"data": "TbyPDfUoyRxsr",
							"programIdIndex": 10,
							"stackHeight": 2
						},
						{
							"accounts": [0, 4],
							"data": "3Bxs4NLhqXb3ofom",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [4],
							"data": "9krTD1mFP1husSVM",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [4],
							"data": "SYXsBvR59WTsF4KEVN8LCQ1X9MekXCGPPNo3Af36taxCQBED",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [0, 3],
							"data": "3Bxs4bm7oSCPMeKR",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [3],
							"data": "9krTDGKFuDw9nLmM",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [3],
							"data": "SYXsBvR59WTsF4KEVN8LCQ1X9MekXCGPPNo3Af36taxCQBED",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [7],
							"data": "hTEY7jEqBPdDRkTWweeDPgyCUykRXEQVCUwrYmn4HZo84DdQrTJT2nBMiJFB3jXUVxHVd9mGq7BX9htuAN",
							"programIdIndex": 6,
							"stackHeight": 2
						}
					]
				}
			],
			"loadedAddresses": {
				"readonly": [],
				"writable": []
			},
			"logMessages": [
				"Program 11111111111111111111111111111111 invoke [1]",
				"Program 11111111111111111111111111111111 success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [1]",
				"Program log: Instruction: PostMessage",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [2]",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program log: Sequence: 0",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 60384 of 380989 compute units",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [2]",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 2000 of 318068 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 84102 of 399850 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success"
			],
			"postBalances": [
				499999999997496260, 1057920, 2350640170, 1552080, 946560, 1, 1141440, 0,
				1169280, 1009200, 1141440
			],
			"postTokenBalances": [],
			"preBalances": [
				500000000000000000, 1057920, 2350640070, 0, 0, 1, 1141440, 0, 1169280,
				1009200, 1141440
			],
			"preTokenBalances": [],
			"rewards": [],
			"status": {
				"Ok": null
			}
		},
		"slot": 3,
		"transaction": {
			"message": {
				"header": {
					"numReadonlySignedAccounts": 0,
					"numReadonlyUnsignedAccounts": 6,
					"numRequiredSignatures": 1
				},
				"accountKeys": [
					"H3kCPjpQDT4hgwWHr9E9pC99rZT2yHAwiwSwku6Bne9",
					"2yVjuQwpsvdsrywzsJJVs9Ueh4zayyo5DYJbBNc3DDpn",
					"9bFNrXNb2WTx8fMHXCheaZqkLZ3YCCaiqTftHxeintHy",
					"9vohBn118ZEctRmuTRvoUZg1B1HGfSH8C5QX6twtUFrJ",
					"HeccUHmoyMi5S6nuTcyUBh4w4me3FP541a52ErYJRT8a",
					"11111111111111111111111111111111",
					"EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX",
					"HQS31aApX3DDkuXgSpV9XyDUNtFgQ31pUn5BNWHG2PSp",
					"SysvarC1ock11111111111111111111111111111111",
					"SysvarRent111111111111111111111111111111111",
					"worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
				],
				"recentBlockhash": "CMqPGm4icRdNuHsWJUK4Kgu4Cbe2nDQkYNqugQkKPa4Y",
				"instructions": [
					{
						"accounts": [0, 2],
						"data": "3Bxs4HanWsHUZCbH",
						"programIdIndex": 5,
						"stackHeight": null
					},
					{
						"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9, 10, 7, 6],
						"data": "3Cn8VBJReY7Bku3RduhBfYpk7tiw1R6pKcTWv9R",
						"programIdIndex": 6,
						"stackHeight": null
					}
				],
				"indexToProgramIds": {}
			},
			"signatures": [
				"3NACxoZLehbdKGjTWZKTTXJPuovyqAih1AD1BrkYj8nzDAtjiQUEaNmhkoU1jcFfoPTAjrvnaLFgTafNWr3fBrdB"
			]
		},
		"version": "legacy"
	}
	`

	///////// A bunch of checks to verify we parsed the JSON correctly.
	var txRpc rpc.TransactionWithMeta
	err := json.Unmarshal([]byte(eventJson), &txRpc)
	require.NoError(t, err)

	tx, err := txRpc.GetParsedTransaction()
	require.NoError(t, err)

	require.Equal(t, 2, len(tx.Message.Instructions))
	require.Equal(t, 1, len(txRpc.Meta.InnerInstructions))

	///////// Now we start the real test.

	logger := zap.NewNop()
	msgC := make(chan *common.MessagePublication, 10)
	s := shimNewWatcherForTest(t, msgC)
	require.True(t, s.shimEnabled)

	var whProgramIndex uint16
	var shimProgramIndex uint16
	var shimFound bool
	for n, key := range tx.Message.AccountKeys {
		if key.Equals(s.contract) {
			whProgramIndex = uint16(n) // #nosec G115 -- The solana runtime can only support 64 accounts per transaction max
		}
		if key.Equals(s.shimContractAddr) {
			shimProgramIndex = uint16(n) // #nosec G115 -- The solana runtime can only support 64 accounts per transaction max
			shimFound = true
		}
	}

	require.Equal(t, uint16(10), whProgramIndex)
	require.True(t, shimFound)
	require.Equal(t, uint16(6), shimProgramIndex)

	alreadyProcessed := ShimAlreadyProcessed{}
	found, err := s.shimProcessTopLevelInstruction(logger, whProgramIndex, shimProgramIndex, tx, txRpc.Meta.InnerInstructions, 1, alreadyProcessed, false)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 1, len(s.msgC))

	msg := <-msgC
	require.NotNil(t, msg)

	expectedTxID, err := hex.DecodeString("7647cd98fd14c6e3cdfe35bc64bbc476abcdb5ab12e8d31e3151d132ed1e0eeb4595fda4779f69dbe00ff14aadad3fdcf537b88a22f48f3acb7b31f340670506")
	require.NoError(t, err)

	expectedEmitterAddress, err := vaa.StringToAddress("041c657e845d65d009d59ceeb1dda172bd6bc9e7ee5a19e56573197cf7fdffde")
	require.NoError(t, err)

	assert.Equal(t, expectedTxID, msg.TxID)
	assert.Equal(t, time.Unix(int64(1736530812), 0), msg.Timestamp)
	assert.Equal(t, uint32(42), msg.Nonce)
	assert.Equal(t, uint64(0), msg.Sequence)
	assert.Equal(t, vaa.ChainIDSolana, msg.EmitterChain)
	assert.Equal(t, expectedEmitterAddress, msg.EmitterAddress)
	assert.Equal(t, uint8(1), msg.ConsistencyLevel)
	assert.Equal(t, "hello world", string(msg.Payload))
	assert.False(t, msg.IsReobservation)
	assert.False(t, msg.Unreliable)
}

func TestShimFromIntegrator(t *testing.T) {
	eventJson := `
	{
		"blockTime": 1736542615,
		"meta": {
			"computeUnitsConsumed": 48958,
			"err": null,
			"fee": 5000,
			"innerInstructions": [
				{
					"index": 1,
					"instructions": [
						{
							"accounts": [1, 4, 11, 3, 0, 2, 9, 5, 10, 12, 8, 7],
							"data": "BeHixXyfSZ8dzFJzxTYRV18L6KSgTuqcTjaqeXgDVbXHC7mCjAgSyhz",
							"programIdIndex": 7,
							"stackHeight": 2
						},
						{
							"accounts": [1, 4, 11, 3, 0, 2, 9, 5, 10],
							"data": "T4xyMHqZi66JU",
							"programIdIndex": 12,
							"stackHeight": 3
						},
						{
							"accounts": [8],
							"data": "hTEY7jEqBPdDRkTWweeDPgzBpsiybJCHnVTVt8aCDem8p58yeQcQLJWk7hgGHrX79qZyKmCM89vCgPY7SE",
							"programIdIndex": 7,
							"stackHeight": 3
						}
					]
				}
			],
			"loadedAddresses": { "readonly": [], "writable": [] },
			"logMessages": [
				"Program 11111111111111111111111111111111 invoke [1]",
				"Program 11111111111111111111111111111111 success",
				"Program AEwubmehHNvkMXoH2C5MgDSemZgQ3HUSYpeaF3UrNZdQ invoke [1]",
				"Program log: Instruction: PostMessage",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [2]",
				"Program log: Instruction: PostMessage",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [3]",
				"Program log: Sequence: 1",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 18679 of 375180 compute units",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [3]",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 2000 of 353964 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 33649 of 385286 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success",
				"Program AEwubmehHNvkMXoH2C5MgDSemZgQ3HUSYpeaF3UrNZdQ consumed 48808 of 399850 compute units",
				"Program AEwubmehHNvkMXoH2C5MgDSemZgQ3HUSYpeaF3UrNZdQ success"
			],
			"postBalances": [
				499999999997491140, 1057920, 2350640270, 946560, 1552080, 1, 1141440,
				1141440, 0, 1169280, 1009200, 0, 1141440
			],
			"postTokenBalances": [],
			"preBalances": [
				499999999997496260, 1057920, 2350640170, 946560, 1552080, 1, 1141440,
				1141440, 0, 1169280, 1009200, 0, 1141440
			],
			"preTokenBalances": [],
			"rewards": [],
			"status": { "Ok": null }
		},
		"slot": 5,
		"transaction": {
			"message": {
				"header": {
					"numReadonlySignedAccounts": 0,
					"numReadonlyUnsignedAccounts": 8,
					"numRequiredSignatures": 1
				},
				"accountKeys": [
					"H3kCPjpQDT4hgwWHr9E9pC99rZT2yHAwiwSwku6Bne9",
					"2yVjuQwpsvdsrywzsJJVs9Ueh4zayyo5DYJbBNc3DDpn",
					"9bFNrXNb2WTx8fMHXCheaZqkLZ3YCCaiqTftHxeintHy",
					"G4zDzQLktwvU4rn6A4dSAy9eU76cJxppCaumZhjjhXjv",
					"GXUAWs1h6Nh1KLByvfeEyig9yn92LmKMjXDNxHGddyXR",
					"11111111111111111111111111111111",
					"AEwubmehHNvkMXoH2C5MgDSemZgQ3HUSYpeaF3UrNZdQ",
					"EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX",
					"HQS31aApX3DDkuXgSpV9XyDUNtFgQ31pUn5BNWHG2PSp",
					"SysvarC1ock11111111111111111111111111111111",
					"SysvarRent111111111111111111111111111111111",
					"UvCifi1D8qj5FSJQdWL3KENnmaZjm62XUMa7NReceer",
					"worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
				],
				"recentBlockhash": "EqNQXbHebHwD1Vs4BSStmUVh2y6GjMxF3NBsDXsYuvRh",
				"instructions": [
					{
						"accounts": [0, 2],
						"data": "3Bxs4HanWsHUZCbH",
						"programIdIndex": 5,
						"stackHeight": null
					},
					{
						"accounts": [0, 7, 1, 4, 11, 3, 2, 9, 5, 10, 12, 8],
						"data": "cpyiD6CEaBD",
						"programIdIndex": 6,
						"stackHeight": null
					}
				],
				"indexToProgramIds": {}
			},
			"signatures": [
				"G4jVHcH6F4Np1NRvYC6ridv5jGfPSVGgiEVZrjprpMdBFhJH7eVxUuxsvkDF2rkx4JseUftz3HnWoSomGt3czSY"
			]
		},
		"version": "legacy"
	}
	`

	///////// A bunch of checks to verify we parsed the JSON correctly.
	var txRpc rpc.TransactionWithMeta
	err := json.Unmarshal([]byte(eventJson), &txRpc)
	require.NoError(t, err)

	tx, err := txRpc.GetParsedTransaction()
	require.NoError(t, err)

	require.Equal(t, 2, len(tx.Message.Instructions))
	require.Equal(t, 1, len(txRpc.Meta.InnerInstructions))

	///////// Now we start the real test.

	logger := zap.NewNop()
	msgC := make(chan *common.MessagePublication, 10)
	s := shimNewWatcherForTest(t, msgC)
	require.True(t, s.shimEnabled)

	var whProgramIndex uint16
	var shimProgramIndex uint16
	var shimFound bool
	for n, key := range tx.Message.AccountKeys {
		if key.Equals(s.contract) {
			whProgramIndex = uint16(n) // #nosec G115 -- The solana runtime can only support 64 accounts per transaction max
		}
		if key.Equals(s.shimContractAddr) {
			shimProgramIndex = uint16(n) // #nosec G115 -- The solana runtime can only support 64 accounts per transaction max
			shimFound = true
		}
	}

	require.Equal(t, uint16(12), whProgramIndex)
	require.True(t, shimFound)
	require.Equal(t, uint16(7), shimProgramIndex)

	alreadyProcessed := ShimAlreadyProcessed{}
	found, err := s.shimProcessInnerInstruction(logger, whProgramIndex, shimProgramIndex, tx, txRpc.Meta.InnerInstructions[0].Instructions, 0, 0, alreadyProcessed, false)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 1, len(s.msgC))

	msg := <-msgC
	require.NotNil(t, msg)

	expectedTxID, err := hex.DecodeString("0cfdad68fdee85b49aea65e48c0d8def74f0968e7e1cf2c33305cfc33fec02a4742895c1d32f7c4093f75133104e70bd126fbbf8b71e5d8cb723a390cd976305")
	require.NoError(t, err)

	expectedEmitterAddress, err := vaa.StringToAddress("0726d66bf942e942332ddf34a2edb7b83c4cdfd25b15d4247e2e15057cdfc3cf")
	require.NoError(t, err)

	assert.Equal(t, expectedTxID, msg.TxID)
	assert.Equal(t, time.Unix(int64(1736542615), 0), msg.Timestamp)
	assert.Equal(t, uint32(0), msg.Nonce)
	assert.Equal(t, uint64(1), msg.Sequence)
	assert.Equal(t, vaa.ChainIDSolana, msg.EmitterChain)
	assert.Equal(t, expectedEmitterAddress, msg.EmitterAddress)
	assert.Equal(t, uint8(1), msg.ConsistencyLevel)
	assert.Equal(t, "your message goes here!", string(msg.Payload))
	assert.False(t, msg.IsReobservation)
	assert.False(t, msg.Unreliable)
}

func TestShimDirectWithMultipleShimTransactions(t *testing.T) {
	eventJson := `
	{
		"blockTime": 1736530812,
		"meta": {
			"computeUnitsConsumed": 84252,
			"err": null,
			"fee": 5000,
			"innerInstructions": [
				{
					"index": 1,
					"instructions": [
						{
							"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9],
							"data": "TbyPDfUoyRxsr",
							"programIdIndex": 10,
							"stackHeight": 2
						},
						{
							"accounts": [0, 4],
							"data": "3Bxs4NLhqXb3ofom",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [4],
							"data": "9krTD1mFP1husSVM",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [4],
							"data": "SYXsBvR59WTsF4KEVN8LCQ1X9MekXCGPPNo3Af36taxCQBED",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [0, 3],
							"data": "3Bxs4bm7oSCPMeKR",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [3],
							"data": "9krTDGKFuDw9nLmM",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [3],
							"data": "SYXsBvR59WTsF4KEVN8LCQ1X9MekXCGPPNo3Af36taxCQBED",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [7],
							"data": "hTEY7jEqBPdDRkTWweeDPgyCUykRXEQVCUwrYmn4HZo84DdQrTJT2nBMiJFB3jXUVxHVd9mGq7BX9htuAN",
							"programIdIndex": 6,
							"stackHeight": 2
						}
					]
				},
				{
					"index": 2,
					"instructions": [
						{
							"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9],
							"data": "TbyPDfUoyRxsr",
							"programIdIndex": 10,
							"stackHeight": 2
						},
						{
							"accounts": [7],
							"data": "hTEY7jEqBPdDRkTWweeDPgyCUykRXEQVCUwrYmn4HZo84DdQrTJT2nBMiJFB3jXUVxJb75nmkwJkw2Varz",
							"programIdIndex": 6,
							"stackHeight": 2
						}
					]
				}				
			],
			"loadedAddresses": {
				"readonly": [],
				"writable": []
			},
			"logMessages": [
				"Program 11111111111111111111111111111111 invoke [1]",
				"Program 11111111111111111111111111111111 success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [1]",
				"Program log: Instruction: PostMessage",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [2]",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program log: Sequence: 0",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 60384 of 380989 compute units",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [2]",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 2000 of 318068 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 84102 of 399850 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success"
			],
			"postBalances": [
				499999999997496260, 1057920, 2350640170, 1552080, 946560, 1, 1141440, 0,
				1169280, 1009200, 1141440
			],
			"postTokenBalances": [],
			"preBalances": [
				500000000000000000, 1057920, 2350640070, 0, 0, 1, 1141440, 0, 1169280,
				1009200, 1141440
			],
			"preTokenBalances": [],
			"rewards": [],
			"status": {
				"Ok": null
			}
		},
		"slot": 3,
		"transaction": {
			"message": {
				"header": {
					"numReadonlySignedAccounts": 0,
					"numReadonlyUnsignedAccounts": 6,
					"numRequiredSignatures": 1
				},
				"accountKeys": [
					"H3kCPjpQDT4hgwWHr9E9pC99rZT2yHAwiwSwku6Bne9",
					"2yVjuQwpsvdsrywzsJJVs9Ueh4zayyo5DYJbBNc3DDpn",
					"9bFNrXNb2WTx8fMHXCheaZqkLZ3YCCaiqTftHxeintHy",
					"9vohBn118ZEctRmuTRvoUZg1B1HGfSH8C5QX6twtUFrJ",
					"HeccUHmoyMi5S6nuTcyUBh4w4me3FP541a52ErYJRT8a",
					"11111111111111111111111111111111",
					"EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX",
					"HQS31aApX3DDkuXgSpV9XyDUNtFgQ31pUn5BNWHG2PSp",
					"SysvarC1ock11111111111111111111111111111111",
					"SysvarRent111111111111111111111111111111111",
					"worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
				],
				"recentBlockhash": "CMqPGm4icRdNuHsWJUK4Kgu4Cbe2nDQkYNqugQkKPa4Y",
				"instructions": [
					{
						"accounts": [0, 2],
						"data": "3Bxs4HanWsHUZCbH",
						"programIdIndex": 5,
						"stackHeight": null
					},
					{
						"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9, 10, 7, 6],
						"data": "3Cn8VBJReY7Bku3RduhBfYpk7tiw1R6pKcTWv9R",
						"programIdIndex": 6,
						"stackHeight": null
					},
					{
						"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9, 10, 7, 6],
						"data": "3Cn8VBJReY7BmhufSsqDhta36ruYh9KwEjTqqv3",
						"programIdIndex": 6,
						"stackHeight": null
					}					
				],
				"indexToProgramIds": {}
			},
			"signatures": [
				"3NACxoZLehbdKGjTWZKTTXJPuovyqAih1AD1BrkYj8nzDAtjiQUEaNmhkoU1jcFfoPTAjrvnaLFgTafNWr3fBrdB"
			]
		},
		"version": "legacy"
	}
	`

	///////// A bunch of checks to verify we parsed the JSON correctly.
	var txRpc rpc.TransactionWithMeta
	err := json.Unmarshal([]byte(eventJson), &txRpc)
	require.NoError(t, err)

	tx, err := txRpc.GetParsedTransaction()
	require.NoError(t, err)

	require.Equal(t, 3, len(tx.Message.Instructions))
	require.Equal(t, 2, len(txRpc.Meta.InnerInstructions))

	///////// Set up the watcher and do the one-time transaction processing.

	logger := zap.NewNop()
	msgC := make(chan *common.MessagePublication, 10)
	s := shimNewWatcherForTest(t, msgC)
	require.True(t, s.shimEnabled)

	var whProgramIndex uint16
	var shimProgramIndex uint16
	var shimFound bool
	for n, key := range tx.Message.AccountKeys {
		if key.Equals(s.contract) {
			whProgramIndex = uint16(n) // #nosec G115 -- The solana runtime can only support 64 accounts per transaction max
		}
		if key.Equals(s.shimContractAddr) {
			shimProgramIndex = uint16(n) // #nosec G115 -- The solana runtime can only support 64 accounts per transaction max
			shimFound = true
		}
	}

	require.Equal(t, uint16(10), whProgramIndex)
	require.True(t, shimFound)
	require.Equal(t, uint16(6), shimProgramIndex)

	expectedTxID, err := hex.DecodeString("7647cd98fd14c6e3cdfe35bc64bbc476abcdb5ab12e8d31e3151d132ed1e0eeb4595fda4779f69dbe00ff14aadad3fdcf537b88a22f48f3acb7b31f340670506")
	require.NoError(t, err)

	expectedEmitterAddress, err := vaa.StringToAddress("041c657e845d65d009d59ceeb1dda172bd6bc9e7ee5a19e56573197cf7fdffde")
	require.NoError(t, err)

	//////////// Process the first shim top level instruction.

	alreadyProcessed := ShimAlreadyProcessed{}
	found, err := s.shimProcessTopLevelInstruction(logger, whProgramIndex, shimProgramIndex, tx, txRpc.Meta.InnerInstructions, 1, alreadyProcessed, false)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 1, len(s.msgC))

	msg := <-msgC
	require.NotNil(t, msg)

	assert.Equal(t, expectedTxID, msg.TxID)
	assert.Equal(t, time.Unix(int64(1736530812), 0), msg.Timestamp)
	assert.Equal(t, uint32(42), msg.Nonce)
	assert.Equal(t, uint64(0), msg.Sequence)
	assert.Equal(t, vaa.ChainIDSolana, msg.EmitterChain)
	assert.Equal(t, expectedEmitterAddress, msg.EmitterAddress)
	assert.Equal(t, uint8(1), msg.ConsistencyLevel)
	assert.Equal(t, "hello world", string(msg.Payload))
	assert.False(t, msg.IsReobservation)
	assert.False(t, msg.Unreliable)

	//////////// Process the second shim top level instruction.

	found, err = s.shimProcessTopLevelInstruction(logger, whProgramIndex, shimProgramIndex, tx, txRpc.Meta.InnerInstructions, 2, alreadyProcessed, false)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 1, len(s.msgC))

	msg = <-msgC
	require.NotNil(t, msg)

	assert.Equal(t, expectedTxID, msg.TxID)
	assert.Equal(t, time.Unix(int64(1736530813), 0), msg.Timestamp)
	assert.Equal(t, uint32(43), msg.Nonce)
	assert.Equal(t, uint64(1), msg.Sequence)
	assert.Equal(t, vaa.ChainIDSolana, msg.EmitterChain)
	assert.Equal(t, expectedEmitterAddress, msg.EmitterAddress)
	assert.Equal(t, uint8(1), msg.ConsistencyLevel)
	assert.Equal(t, "hello world", string(msg.Payload))
	assert.False(t, msg.IsReobservation)
	assert.False(t, msg.Unreliable)
}

func TestShimFromIntegratorWithMultipleShimTransactions(t *testing.T) {
	eventJson := `
	{
		"blockTime": 1736542615,
		"meta": {
			"computeUnitsConsumed": 48958,
			"err": null,
			"fee": 5000,
			"innerInstructions": [
				{
					"index": 1,
					"instructions": [
						{
							"accounts": [1, 4, 11, 3, 0, 2, 9, 5, 10, 12, 8, 7],
							"data": "BeHixXyfSZ8dzFJzxTYRV18L6KSgTuqcTjaqeXgDVbXHC7mCjAgSyhz",
							"programIdIndex": 7,
							"stackHeight": 2
						},
						{
							"accounts": [1, 4, 11, 3, 0, 2, 9, 5, 10],
							"data": "T4xyMHqZi66JU",
							"programIdIndex": 12,
							"stackHeight": 3
						},
						{
							"accounts": [8],
							"data": "hTEY7jEqBPdDRkTWweeDPgzBpsiybJCHnVTVt8aCDem8p58yeQcQLJWk7hgGHrX79qZyKmCM89vCgPY7SE",
							"programIdIndex": 7,
							"stackHeight": 3
						},
						{
							"accounts": [1, 4, 11, 3, 0, 2, 9, 5, 10, 12, 8, 7],
							"data": "BeHixXyfSZ8gpCS9kw5xbo7V9NN3f6bDP3Bi4G3sPsbod54LvCUimUU",
							"programIdIndex": 7,
							"stackHeight": 2
						},
						{
							"accounts": [1, 4, 11, 3, 0, 2, 9, 5, 10],
							"data": "T4xyMHqZi66JU",
							"programIdIndex": 12,
							"stackHeight": 3
						},
						{
							"accounts": [8],
							"data": "hTEY7jEqBPdDRkTWweeDPgzBpsiybJCHnVTVt8aCDem8p58yeQcQLJWk7hgGHrX79qb4ohDr3z3STi8o8r",
							"programIdIndex": 7,
							"stackHeight": 3
						}
					]
				}
			],
			"loadedAddresses": { "readonly": [], "writable": [] },
			"logMessages": [
				"Program 11111111111111111111111111111111 invoke [1]",
				"Program 11111111111111111111111111111111 success",
				"Program AEwubmehHNvkMXoH2C5MgDSemZgQ3HUSYpeaF3UrNZdQ invoke [1]",
				"Program log: Instruction: PostMessage",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [2]",
				"Program log: Instruction: PostMessage",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [3]",
				"Program log: Sequence: 1",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 18679 of 375180 compute units",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [3]",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 2000 of 353964 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 33649 of 385286 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success",
				"Program AEwubmehHNvkMXoH2C5MgDSemZgQ3HUSYpeaF3UrNZdQ consumed 48808 of 399850 compute units",
				"Program AEwubmehHNvkMXoH2C5MgDSemZgQ3HUSYpeaF3UrNZdQ success"
			],
			"postBalances": [
				499999999997491140, 1057920, 2350640270, 946560, 1552080, 1, 1141440,
				1141440, 0, 1169280, 1009200, 0, 1141440
			],
			"postTokenBalances": [],
			"preBalances": [
				499999999997496260, 1057920, 2350640170, 946560, 1552080, 1, 1141440,
				1141440, 0, 1169280, 1009200, 0, 1141440
			],
			"preTokenBalances": [],
			"rewards": [],
			"status": { "Ok": null }
		},
		"slot": 5,
		"transaction": {
			"message": {
				"header": {
					"numReadonlySignedAccounts": 0,
					"numReadonlyUnsignedAccounts": 8,
					"numRequiredSignatures": 1
				},
				"accountKeys": [
					"H3kCPjpQDT4hgwWHr9E9pC99rZT2yHAwiwSwku6Bne9",
					"2yVjuQwpsvdsrywzsJJVs9Ueh4zayyo5DYJbBNc3DDpn",
					"9bFNrXNb2WTx8fMHXCheaZqkLZ3YCCaiqTftHxeintHy",
					"G4zDzQLktwvU4rn6A4dSAy9eU76cJxppCaumZhjjhXjv",
					"GXUAWs1h6Nh1KLByvfeEyig9yn92LmKMjXDNxHGddyXR",
					"11111111111111111111111111111111",
					"AEwubmehHNvkMXoH2C5MgDSemZgQ3HUSYpeaF3UrNZdQ",
					"EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX",
					"HQS31aApX3DDkuXgSpV9XyDUNtFgQ31pUn5BNWHG2PSp",
					"SysvarC1ock11111111111111111111111111111111",
					"SysvarRent111111111111111111111111111111111",
					"UvCifi1D8qj5FSJQdWL3KENnmaZjm62XUMa7NReceer",
					"worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
				],
				"recentBlockhash": "EqNQXbHebHwD1Vs4BSStmUVh2y6GjMxF3NBsDXsYuvRh",
				"instructions": [
					{
						"accounts": [0, 2],
						"data": "3Bxs4HanWsHUZCbH",
						"programIdIndex": 5,
						"stackHeight": null
					},
					{
						"accounts": [0, 7, 1, 4, 11, 3, 2, 9, 5, 10, 12, 8],
						"data": "cpyiD6CEaBD",
						"programIdIndex": 6,
						"stackHeight": null
					}
				],
				"indexToProgramIds": {}
			},
			"signatures": [
				"G4jVHcH6F4Np1NRvYC6ridv5jGfPSVGgiEVZrjprpMdBFhJH7eVxUuxsvkDF2rkx4JseUftz3HnWoSomGt3czSY"
			]
		},
		"version": "legacy"
	}
	`

	///////// A bunch of checks to verify we parsed the JSON correctly.
	var txRpc rpc.TransactionWithMeta
	err := json.Unmarshal([]byte(eventJson), &txRpc)
	require.NoError(t, err)

	tx, err := txRpc.GetParsedTransaction()
	require.NoError(t, err)

	require.Equal(t, 2, len(tx.Message.Instructions))
	require.Equal(t, 1, len(txRpc.Meta.InnerInstructions))

	///////// Set up the watcher and do the one-time transaction processing.

	logger := zap.NewNop()
	msgC := make(chan *common.MessagePublication, 10)
	s := shimNewWatcherForTest(t, msgC)
	require.True(t, s.shimEnabled)

	var whProgramIndex uint16
	var shimProgramIndex uint16
	var shimFound bool
	for n, key := range tx.Message.AccountKeys {
		if key.Equals(s.contract) {
			whProgramIndex = uint16(n) // #nosec G115 -- The solana runtime can only support 64 accounts per transaction max
		}
		if key.Equals(s.shimContractAddr) {
			shimProgramIndex = uint16(n) // #nosec G115 -- The solana runtime can only support 64 accounts per transaction max
			shimFound = true
		}
	}

	require.Equal(t, uint16(12), whProgramIndex)
	require.True(t, shimFound)
	require.Equal(t, uint16(7), shimProgramIndex)

	expectedTxID, err := hex.DecodeString("0cfdad68fdee85b49aea65e48c0d8def74f0968e7e1cf2c33305cfc33fec02a4742895c1d32f7c4093f75133104e70bd126fbbf8b71e5d8cb723a390cd976305")
	require.NoError(t, err)

	expectedEmitterAddress, err := vaa.StringToAddress("0726d66bf942e942332ddf34a2edb7b83c4cdfd25b15d4247e2e15057cdfc3cf")
	require.NoError(t, err)

	//////////// Process the first shim inner instruction.

	alreadyProcessed := ShimAlreadyProcessed{}
	found, err := s.shimProcessInnerInstruction(logger, whProgramIndex, shimProgramIndex, tx, txRpc.Meta.InnerInstructions[0].Instructions, 0, 0, alreadyProcessed, false)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 1, len(s.msgC))

	msg := <-msgC
	require.NotNil(t, msg)

	assert.Equal(t, expectedTxID, msg.TxID)
	assert.Equal(t, time.Unix(int64(1736542615), 0), msg.Timestamp)
	assert.Equal(t, uint32(0), msg.Nonce)
	assert.Equal(t, uint64(1), msg.Sequence)
	assert.Equal(t, vaa.ChainIDSolana, msg.EmitterChain)
	assert.Equal(t, expectedEmitterAddress, msg.EmitterAddress)
	assert.Equal(t, uint8(1), msg.ConsistencyLevel)
	assert.Equal(t, "your message goes here!", string(msg.Payload))
	assert.False(t, msg.IsReobservation)
	assert.False(t, msg.Unreliable)

	//////////// Process the second shim inner instruction.

	found, err = s.shimProcessInnerInstruction(logger, whProgramIndex, shimProgramIndex, tx, txRpc.Meta.InnerInstructions[0].Instructions, 0, 3, alreadyProcessed, false)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 1, len(s.msgC))

	msg = <-msgC
	require.NotNil(t, msg)

	assert.Equal(t, expectedTxID, msg.TxID)
	assert.Equal(t, time.Unix(int64(1736542616), 0), msg.Timestamp)
	assert.Equal(t, uint32(42), msg.Nonce)
	assert.Equal(t, uint64(2), msg.Sequence)
	assert.Equal(t, vaa.ChainIDSolana, msg.EmitterChain)
	assert.Equal(t, expectedEmitterAddress, msg.EmitterAddress)
	assert.Equal(t, uint8(1), msg.ConsistencyLevel)
	assert.Equal(t, "your message goes here!", string(msg.Payload))
	assert.False(t, msg.IsReobservation)
	assert.False(t, msg.Unreliable)
}

func TestShimDirectWithExtraWhEventBeforeShimEventShouldFail(t *testing.T) {
	eventJson := `
	{
		"blockTime": 1736530812,
		"meta": {
			"computeUnitsConsumed": 84252,
			"err": null,
			"fee": 5000,
			"innerInstructions": [
				{
					"index": 1,
					"instructions": [
						{
							"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9],
							"data": "TbyPDfUoyRxsr",
							"programIdIndex": 10,
							"stackHeight": 2
						},
						{
							"accounts": [0, 4],
							"data": "3Bxs4NLhqXb3ofom",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [4],
							"data": "9krTD1mFP1husSVM",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [4],
							"data": "SYXsBvR59WTsF4KEVN8LCQ1X9MekXCGPPNo3Af36taxCQBED",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [0, 3],
							"data": "3Bxs4bm7oSCPMeKR",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [3],
							"data": "9krTDGKFuDw9nLmM",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [3],
							"data": "SYXsBvR59WTsF4KEVN8LCQ1X9MekXCGPPNo3Af36taxCQBED",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9],
							"data": "TbyPDfUoyRxsr",
							"programIdIndex": 10,
							"stackHeight": 2
						},
						{
							"accounts": [7],
							"data": "hTEY7jEqBPdDRkTWweeDPgyCUykRXEQVCUwrYmn4HZo84DdQrTJT2nBMiJFB3jXUVxHVd9mGq7BX9htuAN",
							"programIdIndex": 6,
							"stackHeight": 2
						}
					]
				}
			],
			"loadedAddresses": {
				"readonly": [],
				"writable": []
			},
			"logMessages": [
				"Program 11111111111111111111111111111111 invoke [1]",
				"Program 11111111111111111111111111111111 success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [1]",
				"Program log: Instruction: PostMessage",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [2]",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program log: Sequence: 0",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 60384 of 380989 compute units",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [2]",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 2000 of 318068 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 84102 of 399850 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success"
			],
			"postBalances": [
				499999999997496260, 1057920, 2350640170, 1552080, 946560, 1, 1141440, 0,
				1169280, 1009200, 1141440
			],
			"postTokenBalances": [],
			"preBalances": [
				500000000000000000, 1057920, 2350640070, 0, 0, 1, 1141440, 0, 1169280,
				1009200, 1141440
			],
			"preTokenBalances": [],
			"rewards": [],
			"status": {
				"Ok": null
			}
		},
		"slot": 3,
		"transaction": {
			"message": {
				"header": {
					"numReadonlySignedAccounts": 0,
					"numReadonlyUnsignedAccounts": 6,
					"numRequiredSignatures": 1
				},
				"accountKeys": [
					"H3kCPjpQDT4hgwWHr9E9pC99rZT2yHAwiwSwku6Bne9",
					"2yVjuQwpsvdsrywzsJJVs9Ueh4zayyo5DYJbBNc3DDpn",
					"9bFNrXNb2WTx8fMHXCheaZqkLZ3YCCaiqTftHxeintHy",
					"9vohBn118ZEctRmuTRvoUZg1B1HGfSH8C5QX6twtUFrJ",
					"HeccUHmoyMi5S6nuTcyUBh4w4me3FP541a52ErYJRT8a",
					"11111111111111111111111111111111",
					"EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX",
					"HQS31aApX3DDkuXgSpV9XyDUNtFgQ31pUn5BNWHG2PSp",
					"SysvarC1ock11111111111111111111111111111111",
					"SysvarRent111111111111111111111111111111111",
					"worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
				],
				"recentBlockhash": "CMqPGm4icRdNuHsWJUK4Kgu4Cbe2nDQkYNqugQkKPa4Y",
				"instructions": [
					{
						"accounts": [0, 2],
						"data": "3Bxs4HanWsHUZCbH",
						"programIdIndex": 5,
						"stackHeight": null
					},
					{
						"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9, 10, 7, 6],
						"data": "3Cn8VBJReY7Bku3RduhBfYpk7tiw1R6pKcTWv9R",
						"programIdIndex": 6,
						"stackHeight": null
					}
				],
				"indexToProgramIds": {}
			},
			"signatures": [
				"3NACxoZLehbdKGjTWZKTTXJPuovyqAih1AD1BrkYj8nzDAtjiQUEaNmhkoU1jcFfoPTAjrvnaLFgTafNWr3fBrdB"
			]
		},
		"version": "legacy"
	}
	`

	///////// A bunch of checks to verify we parsed the JSON correctly.
	var txRpc rpc.TransactionWithMeta
	err := json.Unmarshal([]byte(eventJson), &txRpc)
	require.NoError(t, err)

	tx, err := txRpc.GetParsedTransaction()
	require.NoError(t, err)

	require.Equal(t, 2, len(tx.Message.Instructions))
	require.Equal(t, 1, len(txRpc.Meta.InnerInstructions))

	///////// Now we start the real test.

	logger := zap.NewNop()
	msgC := make(chan *common.MessagePublication, 10)
	s := shimNewWatcherForTest(t, msgC)
	require.True(t, s.shimEnabled)

	var whProgramIndex uint16
	var shimProgramIndex uint16
	var shimFound bool
	for n, key := range tx.Message.AccountKeys {
		if key.Equals(s.contract) {
			whProgramIndex = uint16(n) // #nosec G115 -- The solana runtime max transaction size is 1232 bytes. So we'd never be able to have this many accounts.
		}
		if key.Equals(s.shimContractAddr) {
			shimProgramIndex = uint16(n) // #nosec G115 -- The solana runtime max transaction size is 1232 bytes. So we'd never be able to have this many accounts.
			shimFound = true
		}
	}

	require.Equal(t, uint16(10), whProgramIndex)
	require.True(t, shimFound)
	require.Equal(t, uint16(6), shimProgramIndex)

	alreadyProcessed := ShimAlreadyProcessed{}
	found, err := s.shimProcessTopLevelInstruction(logger, whProgramIndex, shimProgramIndex, tx, txRpc.Meta.InnerInstructions, 1, alreadyProcessed, false)
	require.ErrorContains(t, err, "detected multiple inner core instructions when there should not be")
	require.False(t, found)
	require.Equal(t, 0, len(s.msgC))
	require.Equal(t, 1, len(alreadyProcessed)) // The first core event will have been added.
}

func TestShimDirectWithExtraShimEventsShouldFail(t *testing.T) {
	eventJson := `
	{
		"blockTime": 1736530812,
		"meta": {
			"computeUnitsConsumed": 84252,
			"err": null,
			"fee": 5000,
			"innerInstructions": [
				{
					"index": 1,
					"instructions": [
						{
							"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9],
							"data": "TbyPDfUoyRxsr",
							"programIdIndex": 10,
							"stackHeight": 2
						},
						{
							"accounts": [0, 4],
							"data": "3Bxs4NLhqXb3ofom",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [4],
							"data": "9krTD1mFP1husSVM",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [4],
							"data": "SYXsBvR59WTsF4KEVN8LCQ1X9MekXCGPPNo3Af36taxCQBED",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [0, 3],
							"data": "3Bxs4bm7oSCPMeKR",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [3],
							"data": "9krTDGKFuDw9nLmM",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [3],
							"data": "SYXsBvR59WTsF4KEVN8LCQ1X9MekXCGPPNo3Af36taxCQBED",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [7],
							"data": "hTEY7jEqBPdDRkTWweeDPgyCUykRXEQVCUwrYmn4HZo84DdQrTJT2nBMiJFB3jXUVxHVd9mGq7BX9htuAN",
							"programIdIndex": 6,
							"stackHeight": 2
						},
						{
							"accounts": [7],
							"data": "hTEY7jEqBPdDRkTWweeDPgyCUykRXEQVCUwrYmn4HZo84DdQrTJT2nBMiJFB3jXUVxHVd9mGq7BX9htuAN",
							"programIdIndex": 6,
							"stackHeight": 2
						}
					]
				}
			],
			"loadedAddresses": {
				"readonly": [],
				"writable": []
			},
			"logMessages": [
				"Program 11111111111111111111111111111111 invoke [1]",
				"Program 11111111111111111111111111111111 success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [1]",
				"Program log: Instruction: PostMessage",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [2]",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program log: Sequence: 0",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 60384 of 380989 compute units",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [2]",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 2000 of 318068 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 84102 of 399850 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success"
			],
			"postBalances": [
				499999999997496260, 1057920, 2350640170, 1552080, 946560, 1, 1141440, 0,
				1169280, 1009200, 1141440
			],
			"postTokenBalances": [],
			"preBalances": [
				500000000000000000, 1057920, 2350640070, 0, 0, 1, 1141440, 0, 1169280,
				1009200, 1141440
			],
			"preTokenBalances": [],
			"rewards": [],
			"status": {
				"Ok": null
			}
		},
		"slot": 3,
		"transaction": {
			"message": {
				"header": {
					"numReadonlySignedAccounts": 0,
					"numReadonlyUnsignedAccounts": 6,
					"numRequiredSignatures": 1
				},
				"accountKeys": [
					"H3kCPjpQDT4hgwWHr9E9pC99rZT2yHAwiwSwku6Bne9",
					"2yVjuQwpsvdsrywzsJJVs9Ueh4zayyo5DYJbBNc3DDpn",
					"9bFNrXNb2WTx8fMHXCheaZqkLZ3YCCaiqTftHxeintHy",
					"9vohBn118ZEctRmuTRvoUZg1B1HGfSH8C5QX6twtUFrJ",
					"HeccUHmoyMi5S6nuTcyUBh4w4me3FP541a52ErYJRT8a",
					"11111111111111111111111111111111",
					"EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX",
					"HQS31aApX3DDkuXgSpV9XyDUNtFgQ31pUn5BNWHG2PSp",
					"SysvarC1ock11111111111111111111111111111111",
					"SysvarRent111111111111111111111111111111111",
					"worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
				],
				"recentBlockhash": "CMqPGm4icRdNuHsWJUK4Kgu4Cbe2nDQkYNqugQkKPa4Y",
				"instructions": [
					{
						"accounts": [0, 2],
						"data": "3Bxs4HanWsHUZCbH",
						"programIdIndex": 5,
						"stackHeight": null
					},
					{
						"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9, 10, 7, 6],
						"data": "3Cn8VBJReY7Bku3RduhBfYpk7tiw1R6pKcTWv9R",
						"programIdIndex": 6,
						"stackHeight": null
					}
				],
				"indexToProgramIds": {}
			},
			"signatures": [
				"3NACxoZLehbdKGjTWZKTTXJPuovyqAih1AD1BrkYj8nzDAtjiQUEaNmhkoU1jcFfoPTAjrvnaLFgTafNWr3fBrdB"
			]
		},
		"version": "legacy"
	}
	`

	///////// A bunch of checks to verify we parsed the JSON correctly.
	var txRpc rpc.TransactionWithMeta
	err := json.Unmarshal([]byte(eventJson), &txRpc)
	require.NoError(t, err)

	tx, err := txRpc.GetParsedTransaction()
	require.NoError(t, err)

	require.Equal(t, 2, len(tx.Message.Instructions))
	require.Equal(t, 1, len(txRpc.Meta.InnerInstructions))

	///////// Now we start the real test.

	logger := zap.NewNop()
	msgC := make(chan *common.MessagePublication, 10)
	s := shimNewWatcherForTest(t, msgC)
	require.True(t, s.shimEnabled)

	var whProgramIndex uint16
	var shimProgramIndex uint16
	var shimFound bool
	for n, key := range tx.Message.AccountKeys {
		if key.Equals(s.contract) {
			whProgramIndex = uint16(n) // #nosec G115 -- The solana runtime max transaction size is 1232 bytes. So we'd never be able to have this many accounts.
		}
		if key.Equals(s.shimContractAddr) {
			shimProgramIndex = uint16(n) // #nosec G115 -- The solana runtime max transaction size is 1232 bytes. So we'd never be able to have this many accounts.
			shimFound = true
		}
	}

	require.Equal(t, uint16(10), whProgramIndex)
	require.True(t, shimFound)
	require.Equal(t, uint16(6), shimProgramIndex)

	alreadyProcessed := ShimAlreadyProcessed{}
	found, err := s.shimProcessTopLevelInstruction(logger, whProgramIndex, shimProgramIndex, tx, txRpc.Meta.InnerInstructions, 1, alreadyProcessed, false)
	require.ErrorContains(t, err, "detected multiple shim message event instructions when there should not be")
	require.False(t, found)
	require.Equal(t, 0, len(s.msgC))
	require.Equal(t, 2, len(alreadyProcessed)) // The first core and shim events will have been added.
}

func TestShimDirectWithExtraCoreEventShouldFail(t *testing.T) {
	eventJson := `
	{
		"blockTime": 1736530812,
		"meta": {
			"computeUnitsConsumed": 84252,
			"err": null,
			"fee": 5000,
			"innerInstructions": [
				{
					"index": 1,
					"instructions": [
						{
							"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9],
							"data": "TbyPDfUoyRxsr",
							"programIdIndex": 10,
							"stackHeight": 2
						},
						{
							"accounts": [0, 4],
							"data": "3Bxs4NLhqXb3ofom",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [4],
							"data": "9krTD1mFP1husSVM",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [4],
							"data": "SYXsBvR59WTsF4KEVN8LCQ1X9MekXCGPPNo3Af36taxCQBED",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [0, 3],
							"data": "3Bxs4bm7oSCPMeKR",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [3],
							"data": "9krTDGKFuDw9nLmM",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [3],
							"data": "SYXsBvR59WTsF4KEVN8LCQ1X9MekXCGPPNo3Af36taxCQBED",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [7],
							"data": "hTEY7jEqBPdDRkTWweeDPgyCUykRXEQVCUwrYmn4HZo84DdQrTJT2nBMiJFB3jXUVxHVd9mGq7BX9htuAN",
							"programIdIndex": 6,
							"stackHeight": 2
						},
						{
							"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9],
							"data": "TbyPDfUoyRxsr",
							"programIdIndex": 10,
							"stackHeight": 2
						}						
					]
				}
			],
			"loadedAddresses": {
				"readonly": [],
				"writable": []
			},
			"logMessages": [
				"Program 11111111111111111111111111111111 invoke [1]",
				"Program 11111111111111111111111111111111 success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [1]",
				"Program log: Instruction: PostMessage",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [2]",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program log: Sequence: 0",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 60384 of 380989 compute units",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [2]",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 2000 of 318068 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 84102 of 399850 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success"
			],
			"postBalances": [
				499999999997496260, 1057920, 2350640170, 1552080, 946560, 1, 1141440, 0,
				1169280, 1009200, 1141440
			],
			"postTokenBalances": [],
			"preBalances": [
				500000000000000000, 1057920, 2350640070, 0, 0, 1, 1141440, 0, 1169280,
				1009200, 1141440
			],
			"preTokenBalances": [],
			"rewards": [],
			"status": {
				"Ok": null
			}
		},
		"slot": 3,
		"transaction": {
			"message": {
				"header": {
					"numReadonlySignedAccounts": 0,
					"numReadonlyUnsignedAccounts": 6,
					"numRequiredSignatures": 1
				},
				"accountKeys": [
					"H3kCPjpQDT4hgwWHr9E9pC99rZT2yHAwiwSwku6Bne9",
					"2yVjuQwpsvdsrywzsJJVs9Ueh4zayyo5DYJbBNc3DDpn",
					"9bFNrXNb2WTx8fMHXCheaZqkLZ3YCCaiqTftHxeintHy",
					"9vohBn118ZEctRmuTRvoUZg1B1HGfSH8C5QX6twtUFrJ",
					"HeccUHmoyMi5S6nuTcyUBh4w4me3FP541a52ErYJRT8a",
					"11111111111111111111111111111111",
					"EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX",
					"HQS31aApX3DDkuXgSpV9XyDUNtFgQ31pUn5BNWHG2PSp",
					"SysvarC1ock11111111111111111111111111111111",
					"SysvarRent111111111111111111111111111111111",
					"worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
				],
				"recentBlockhash": "CMqPGm4icRdNuHsWJUK4Kgu4Cbe2nDQkYNqugQkKPa4Y",
				"instructions": [
					{
						"accounts": [0, 2],
						"data": "3Bxs4HanWsHUZCbH",
						"programIdIndex": 5,
						"stackHeight": null
					},
					{
						"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9, 10, 7, 6],
						"data": "3Cn8VBJReY7Bku3RduhBfYpk7tiw1R6pKcTWv9R",
						"programIdIndex": 6,
						"stackHeight": null
					}
				],
				"indexToProgramIds": {}
			},
			"signatures": [
				"3NACxoZLehbdKGjTWZKTTXJPuovyqAih1AD1BrkYj8nzDAtjiQUEaNmhkoU1jcFfoPTAjrvnaLFgTafNWr3fBrdB"
			]
		},
		"version": "legacy"
	}
	`

	///////// A bunch of checks to verify we parsed the JSON correctly.
	var txRpc rpc.TransactionWithMeta
	err := json.Unmarshal([]byte(eventJson), &txRpc)
	require.NoError(t, err)

	tx, err := txRpc.GetParsedTransaction()
	require.NoError(t, err)

	require.Equal(t, 2, len(tx.Message.Instructions))
	require.Equal(t, 1, len(txRpc.Meta.InnerInstructions))

	///////// Now we start the real test.

	logger := zap.NewNop()
	msgC := make(chan *common.MessagePublication, 10)
	s := shimNewWatcherForTest(t, msgC)
	require.True(t, s.shimEnabled)

	var whProgramIndex uint16
	var shimProgramIndex uint16
	var shimFound bool
	for n, key := range tx.Message.AccountKeys {
		if key.Equals(s.contract) {
			whProgramIndex = uint16(n) // #nosec G115 -- The solana runtime max transaction size is 1232 bytes. So we'd never be able to have this many accounts.
		}
		if key.Equals(s.shimContractAddr) {
			shimProgramIndex = uint16(n) // #nosec G115 -- The solana runtime max transaction size is 1232 bytes. So we'd never be able to have this many accounts.
			shimFound = true
		}
	}

	require.Equal(t, uint16(10), whProgramIndex)
	require.True(t, shimFound)
	require.Equal(t, uint16(6), shimProgramIndex)

	alreadyProcessed := ShimAlreadyProcessed{}
	found, err := s.shimProcessTopLevelInstruction(logger, whProgramIndex, shimProgramIndex, tx, txRpc.Meta.InnerInstructions, 1, alreadyProcessed, false)
	require.ErrorContains(t, err, "detected multiple inner core instructions when there should not be")
	require.False(t, found)
	require.Equal(t, 0, len(s.msgC))
	require.Equal(t, 2, len(alreadyProcessed)) // The first core and shim events will have been added.
}

func TestShimTopLevelEmptyInstructionsShouldFail(t *testing.T) {
	eventJson := `
	{
		"blockTime": 1736530812,
		"meta": {
			"computeUnitsConsumed": 84252,
			"err": null,
			"fee": 5000,
			"innerInstructions": [
				{
					"index": 1,
					"instructions": [
						{
							"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9],
							"data": "TbyPDfUoyRxsr",
							"programIdIndex": 10,
							"stackHeight": 2
						},
						{
							"accounts": [0, 4],
							"data": "3Bxs4NLhqXb3ofom",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [4],
							"data": "9krTD1mFP1husSVM",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [4],
							"data": "SYXsBvR59WTsF4KEVN8LCQ1X9MekXCGPPNo3Af36taxCQBED",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [0, 3],
							"data": "3Bxs4bm7oSCPMeKR",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [3],
							"data": "9krTDGKFuDw9nLmM",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [3],
							"data": "SYXsBvR59WTsF4KEVN8LCQ1X9MekXCGPPNo3Af36taxCQBED",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [7],
							"data": "hTEY7jEqBPdDRkTWweeDPgyCUykRXEQVCUwrYmn4HZo84DdQrTJT2nBMiJFB3jXUVxHVd9mGq7BX9htuAN",
							"programIdIndex": 6,
							"stackHeight": 2
						},
						{
							"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9],
							"data": "TbyPDfUoyRxsr",
							"programIdIndex": 10,
							"stackHeight": 2
						}						
					]
				}
			],
			"loadedAddresses": {
				"readonly": [],
				"writable": []
			},
			"logMessages": [
				"Program 11111111111111111111111111111111 invoke [1]",
				"Program 11111111111111111111111111111111 success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [1]",
				"Program log: Instruction: PostMessage",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [2]",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program log: Sequence: 0",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 60384 of 380989 compute units",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [2]",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 2000 of 318068 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 84102 of 399850 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success"
			],
			"postBalances": [
				499999999997496260, 1057920, 2350640170, 1552080, 946560, 1, 1141440, 0,
				1169280, 1009200, 1141440
			],
			"postTokenBalances": [],
			"preBalances": [
				500000000000000000, 1057920, 2350640070, 0, 0, 1, 1141440, 0, 1169280,
				1009200, 1141440
			],
			"preTokenBalances": [],
			"rewards": [],
			"status": {
				"Ok": null
			}
		},
		"slot": 3,
		"transaction": {
			"message": {
				"header": {
					"numReadonlySignedAccounts": 0,
					"numReadonlyUnsignedAccounts": 6,
					"numRequiredSignatures": 1
				},
				"accountKeys": [
					"H3kCPjpQDT4hgwWHr9E9pC99rZT2yHAwiwSwku6Bne9",
					"2yVjuQwpsvdsrywzsJJVs9Ueh4zayyo5DYJbBNc3DDpn",
					"9bFNrXNb2WTx8fMHXCheaZqkLZ3YCCaiqTftHxeintHy",
					"9vohBn118ZEctRmuTRvoUZg1B1HGfSH8C5QX6twtUFrJ",
					"HeccUHmoyMi5S6nuTcyUBh4w4me3FP541a52ErYJRT8a",
					"11111111111111111111111111111111",
					"EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX",
					"HQS31aApX3DDkuXgSpV9XyDUNtFgQ31pUn5BNWHG2PSp",
					"SysvarC1ock11111111111111111111111111111111",
					"SysvarRent111111111111111111111111111111111",
					"worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
				],
				"recentBlockhash": "CMqPGm4icRdNuHsWJUK4Kgu4Cbe2nDQkYNqugQkKPa4Y",
				"instructions": [
				],
				"indexToProgramIds": {}
			},
			"signatures": [
				"3NACxoZLehbdKGjTWZKTTXJPuovyqAih1AD1BrkYj8nzDAtjiQUEaNmhkoU1jcFfoPTAjrvnaLFgTafNWr3fBrdB"
			]
		},
		"version": "legacy"
	}
	`

	///////// A bunch of checks to verify we parsed the JSON correctly.
	var txRpc rpc.TransactionWithMeta
	err := json.Unmarshal([]byte(eventJson), &txRpc)
	require.NoError(t, err)

	tx, err := txRpc.GetParsedTransaction()
	require.NoError(t, err)

	require.Equal(t, 0, len(tx.Message.Instructions))
	require.Equal(t, 1, len(txRpc.Meta.InnerInstructions))

	///////// Now we start the real test.

	logger := zap.NewNop()
	msgC := make(chan *common.MessagePublication, 10)
	s := shimNewWatcherForTest(t, msgC)
	require.True(t, s.shimEnabled)

	var whProgramIndex uint16
	var shimProgramIndex uint16
	var shimFound bool
	for n, key := range tx.Message.AccountKeys {
		if key.Equals(s.contract) {
			whProgramIndex = uint16(n) // #nosec G115 -- The solana runtime max transaction size is 1232 bytes. So we'd never be able to have this many accounts.
		}
		if key.Equals(s.shimContractAddr) {
			shimProgramIndex = uint16(n) // #nosec G115 -- The solana runtime max transaction size is 1232 bytes. So we'd never be able to have this many accounts.
			shimFound = true
		}
	}

	require.Equal(t, uint16(10), whProgramIndex)
	require.True(t, shimFound)
	require.Equal(t, uint16(6), shimProgramIndex)

	alreadyProcessed := ShimAlreadyProcessed{}
	found, err := s.shimProcessTopLevelInstruction(logger, whProgramIndex, shimProgramIndex, tx, txRpc.Meta.InnerInstructions, 1, alreadyProcessed, false)
	require.ErrorContains(t, err, "topLevelIndex 1 is greater than the total number of instructions in the tx message, 0")
	require.False(t, found)
	require.Equal(t, 0, len(s.msgC))
	require.Equal(t, 0, len(alreadyProcessed))
}

func TestShimProcessInnerInstructions_OutOfBoundsStartIndexShouldFail(t *testing.T) {
	eventJson := `
	{
		"blockTime": 1736542615,
		"meta": {
			"computeUnitsConsumed": 48958,
			"err": null,
			"fee": 5000,
			"innerInstructions": [
				{
					"index": 1,
					"instructions": [
						{
							"accounts": [1, 4, 11, 3, 0, 2, 9, 5, 10, 12, 8, 7],
							"data": "BeHixXyfSZ8dzFJzxTYRV18L6KSgTuqcTjaqeXgDVbXHC7mCjAgSyhz",
							"programIdIndex": 7,
							"stackHeight": 2
						},
						{
							"accounts": [1, 4, 11, 3, 0, 2, 9, 5, 10],
							"data": "T4xyMHqZi66JU",
							"programIdIndex": 12,
							"stackHeight": 3
						},
						{
							"accounts": [8],
							"data": "hTEY7jEqBPdDRkTWweeDPgzBpsiybJCHnVTVt8aCDem8p58yeQcQLJWk7hgGHrX79qZyKmCM89vCgPY7SE",
							"programIdIndex": 7,
							"stackHeight": 3
						}
					]
				}
			],
			"loadedAddresses": { "readonly": [], "writable": [] },
			"logMessages": [
				"Program 11111111111111111111111111111111 invoke [1]",
				"Program 11111111111111111111111111111111 success",
				"Program AEwubmehHNvkMXoH2C5MgDSemZgQ3HUSYpeaF3UrNZdQ invoke [1]",
				"Program log: Instruction: PostMessage",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [2]",
				"Program log: Instruction: PostMessage",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [3]",
				"Program log: Sequence: 1",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 18679 of 375180 compute units",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [3]",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 2000 of 353964 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 33649 of 385286 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success",
				"Program AEwubmehHNvkMXoH2C5MgDSemZgQ3HUSYpeaF3UrNZdQ consumed 48808 of 399850 compute units",
				"Program AEwubmehHNvkMXoH2C5MgDSemZgQ3HUSYpeaF3UrNZdQ success"
			],
			"postBalances": [
				499999999997491140, 1057920, 2350640270, 946560, 1552080, 1, 1141440,
				1141440, 0, 1169280, 1009200, 0, 1141440
			],
			"postTokenBalances": [],
			"preBalances": [
				499999999997496260, 1057920, 2350640170, 946560, 1552080, 1, 1141440,
				1141440, 0, 1169280, 1009200, 0, 1141440
			],
			"preTokenBalances": [],
			"rewards": [],
			"status": { "Ok": null }
		},
		"slot": 5,
		"transaction": {
			"message": {
				"header": {
					"numReadonlySignedAccounts": 0,
					"numReadonlyUnsignedAccounts": 8,
					"numRequiredSignatures": 1
				},
				"accountKeys": [
					"H3kCPjpQDT4hgwWHr9E9pC99rZT2yHAwiwSwku6Bne9",
					"2yVjuQwpsvdsrywzsJJVs9Ueh4zayyo5DYJbBNc3DDpn",
					"9bFNrXNb2WTx8fMHXCheaZqkLZ3YCCaiqTftHxeintHy",
					"G4zDzQLktwvU4rn6A4dSAy9eU76cJxppCaumZhjjhXjv",
					"GXUAWs1h6Nh1KLByvfeEyig9yn92LmKMjXDNxHGddyXR",
					"11111111111111111111111111111111",
					"AEwubmehHNvkMXoH2C5MgDSemZgQ3HUSYpeaF3UrNZdQ",
					"EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX",
					"HQS31aApX3DDkuXgSpV9XyDUNtFgQ31pUn5BNWHG2PSp",
					"SysvarC1ock11111111111111111111111111111111",
					"SysvarRent111111111111111111111111111111111",
					"UvCifi1D8qj5FSJQdWL3KENnmaZjm62XUMa7NReceer",
					"worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
				],
				"recentBlockhash": "EqNQXbHebHwD1Vs4BSStmUVh2y6GjMxF3NBsDXsYuvRh",
				"instructions": [
					{
						"accounts": [0, 2],
						"data": "3Bxs4HanWsHUZCbH",
						"programIdIndex": 5,
						"stackHeight": null
					},
					{
						"accounts": [0, 7, 1, 4, 11, 3, 2, 9, 5, 10, 12, 8],
						"data": "cpyiD6CEaBD",
						"programIdIndex": 6,
						"stackHeight": null
					}
				],
				"indexToProgramIds": {}
			},
			"signatures": [
				"G4jVHcH6F4Np1NRvYC6ridv5jGfPSVGgiEVZrjprpMdBFhJH7eVxUuxsvkDF2rkx4JseUftz3HnWoSomGt3czSY"
			]
		},
		"version": "legacy"
	}
	`

	///////// A bunch of checks to verify we parsed the JSON correctly.
	var txRpc rpc.TransactionWithMeta
	err := json.Unmarshal([]byte(eventJson), &txRpc)
	require.NoError(t, err)

	tx, err := txRpc.GetParsedTransaction()
	require.NoError(t, err)

	require.Equal(t, 2, len(tx.Message.Instructions))
	require.Equal(t, 1, len(txRpc.Meta.InnerInstructions))

	///////// Now we start the real test.

	logger := zap.NewNop()
	msgC := make(chan *common.MessagePublication, 10)
	s := shimNewWatcherForTest(t, msgC)
	require.True(t, s.shimEnabled)

	var whProgramIndex uint16
	var shimProgramIndex uint16
	var shimFound bool
	for n, key := range tx.Message.AccountKeys {
		if key.Equals(s.contract) {
			whProgramIndex = uint16(n) // #nosec G115 -- The solana runtime max transaction size is 1232 bytes. So we'd never be able to have this many accounts.
		}
		if key.Equals(s.shimContractAddr) {
			shimProgramIndex = uint16(n) // #nosec G115 -- The solana runtime max transaction size is 1232 bytes. So we'd never be able to have this many accounts.
			shimFound = true
		}
	}

	require.Equal(t, uint16(12), whProgramIndex)
	require.True(t, shimFound)
	require.Equal(t, uint16(7), shimProgramIndex)

	alreadyProcessed := ShimAlreadyProcessed{}
	found, err := s.shimProcessInnerInstruction(logger, whProgramIndex, shimProgramIndex, tx, txRpc.Meta.InnerInstructions[0].Instructions, 0, len(txRpc.Meta.InnerInstructions[0].Instructions), alreadyProcessed, false)
	require.ErrorContains(t, err, "startIdx 3 is out of bounds of slice innerInstructions (length: 3)")
	require.False(t, found)
	require.Equal(t, 0, len(s.msgC))
	require.Equal(t, 0, len(alreadyProcessed))
}

func TestShimWhPostMessageInUnexpectedFormatShouldNotBeCountedAsShimMessage(t *testing.T) {
	// The WH instruction in the first slot is `012a0000000000000001` which is a reliable with no payload.
	// The WH instruction for a shim event should be `082a0000000000000001` which is unreliable with no payload.
	// So this instruction should not be counted as part of a shim event.
	eventJson := `
	{
		"meta": {
			"innerInstructions": [
				{
					"index": 1,
					"instructions": [
						{
							"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9],
							"data": "4o1AAQkKMUkzL",
							"programIdIndex": 10,
							"stackHeight": 2
						},
						{
							"accounts": [7],
							"data": "hTEY7jEqBPdDRkTWweeDPgyCUykRXEQVCUwrYmn4HZo84DdQrTJT2nBMiJFB3jXUVxHVd9mGq7BX9htuAN",
							"programIdIndex": 6,
							"stackHeight": 2
						},
						{
							"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9],
							"data": "TbyPDfUoyRxsr",
							"programIdIndex": 10,
							"stackHeight": 2
						}
					]
				}
			]
		},
		"transaction": {
			"message": {
				"accountKeys": [
					"H3kCPjpQDT4hgwWHr9E9pC99rZT2yHAwiwSwku6Bne9",
					"2yVjuQwpsvdsrywzsJJVs9Ueh4zayyo5DYJbBNc3DDpn",
					"9bFNrXNb2WTx8fMHXCheaZqkLZ3YCCaiqTftHxeintHy",
					"9vohBn118ZEctRmuTRvoUZg1B1HGfSH8C5QX6twtUFrJ",
					"HeccUHmoyMi5S6nuTcyUBh4w4me3FP541a52ErYJRT8a",
					"11111111111111111111111111111111",
					"EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX",
					"HQS31aApX3DDkuXgSpV9XyDUNtFgQ31pUn5BNWHG2PSp",
					"SysvarC1ock11111111111111111111111111111111",
					"SysvarRent111111111111111111111111111111111",
					"worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
				],
				"instructions": [
					{
						"accounts": [0, 2],
						"data": "3Bxs4HanWsHUZCbH",
						"programIdIndex": 5,
						"stackHeight": null
					},
					{
						"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9, 10, 7, 6],
						"data": "3Cn8VBJReY7Bku3RduhBfYpk7tiw1R6pKcTWv9R",
						"programIdIndex": 6,
						"stackHeight": null
					}
				]
			},
			"signatures": [
				"3NACxoZLehbdKGjTWZKTTXJPuovyqAih1AD1BrkYj8nzDAtjiQUEaNmhkoU1jcFfoPTAjrvnaLFgTafNWr3fBrdB"
			]
		}
	}
	`

	///////// A bunch of checks to verify we parsed the JSON correctly.
	var txRpc rpc.TransactionWithMeta
	err := json.Unmarshal([]byte(eventJson), &txRpc)
	require.NoError(t, err)

	tx, err := txRpc.GetParsedTransaction()
	require.NoError(t, err)

	require.Equal(t, 2, len(tx.Message.Instructions))
	require.Equal(t, 1, len(txRpc.Meta.InnerInstructions))

	///////// Set up the watcher and do the one-time transaction processing.

	logger := zap.NewNop()
	msgC := make(chan *common.MessagePublication, 10)
	s := shimNewWatcherForTest(t, msgC)
	require.True(t, s.shimEnabled)

	var whProgramIndex uint16
	var shimProgramIndex uint16
	var shimFound bool
	for n, key := range tx.Message.AccountKeys {
		if key.Equals(s.contract) {
			whProgramIndex = uint16(n) // #nosec G115 -- The solana runtime max transaction size is 1232 bytes. So we'd never be able to have this many accounts.
		}
		if key.Equals(s.shimContractAddr) {
			shimProgramIndex = uint16(n) // #nosec G115 -- The solana runtime max transaction size is 1232 bytes. So we'd never be able to have this many accounts.
			shimFound = true
		}
	}

	require.Equal(t, uint16(10), whProgramIndex)
	require.True(t, shimFound)
	require.Equal(t, uint16(6), shimProgramIndex)

	alreadyProcessed := ShimAlreadyProcessed{}
	found, err := s.shimProcessTopLevelInstruction(
		logger,
		whProgramIndex,
		shimProgramIndex,
		tx,
		txRpc.Meta.InnerInstructions,
		1,
		alreadyProcessed,
		false,
	)

	require.ErrorContains(t, err, "detected an inner shim message event instruction before the core event for shim instruction")
	require.False(t, found)
	require.Equal(t, 0, len(s.msgC))
	require.Equal(t, 0, len(alreadyProcessed))
}

func TestShimProcessRestWithNullEventShouldFail(t *testing.T) {
	eventJson := `
	{
		"blockTime": 1736530812,
		"meta": {
			"computeUnitsConsumed": 84252,
			"err": null,
			"fee": 5000,
			"innerInstructions": [
				{
					"index": 1,
					"instructions": [
						{
							"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9],
							"data": "TbyPDfUoyRxsr",
							"programIdIndex": 10,
							"stackHeight": 2
						},
						{
							"accounts": [0, 4],
							"data": "3Bxs4NLhqXb3ofom",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [4],
							"data": "9krTD1mFP1husSVM",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [4],
							"data": "SYXsBvR59WTsF4KEVN8LCQ1X9MekXCGPPNo3Af36taxCQBED",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [0, 3],
							"data": "3Bxs4bm7oSCPMeKR",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [3],
							"data": "9krTDGKFuDw9nLmM",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [3],
							"data": "SYXsBvR59WTsF4KEVN8LCQ1X9MekXCGPPNo3Af36taxCQBED",
							"programIdIndex": 5,
							"stackHeight": 3
						},
						{
							"accounts": [7],
							"data": "hTEY7jEqBPdDRkTWweeDPgyCUykRXEQVCUwrYmn4HZo84DdQrTJT2nBMiJFB3jXUVxHVd9mGq7BX9htuAN",
							"programIdIndex": 6,
							"stackHeight": 2
						}
					]
				}
			],
			"loadedAddresses": {
				"readonly": [],
				"writable": []
			},
			"logMessages": [
				"Program 11111111111111111111111111111111 invoke [1]",
				"Program 11111111111111111111111111111111 success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [1]",
				"Program log: Instruction: PostMessage",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [2]",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program log: Sequence: 0",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program 11111111111111111111111111111111 invoke [3]",
				"Program 11111111111111111111111111111111 success",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 60384 of 380989 compute units",
				"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX invoke [2]",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 2000 of 318068 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX consumed 84102 of 399850 compute units",
				"Program EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX success"
			],
			"postBalances": [
				499999999997496260, 1057920, 2350640170, 1552080, 946560, 1, 1141440, 0,
				1169280, 1009200, 1141440
			],
			"postTokenBalances": [],
			"preBalances": [
				500000000000000000, 1057920, 2350640070, 0, 0, 1, 1141440, 0, 1169280,
				1009200, 1141440
			],
			"preTokenBalances": [],
			"rewards": [],
			"status": {
				"Ok": null
			}
		},
		"slot": 3,
		"transaction": {
			"message": {
				"header": {
					"numReadonlySignedAccounts": 0,
					"numReadonlyUnsignedAccounts": 6,
					"numRequiredSignatures": 1
				},
				"accountKeys": [
					"H3kCPjpQDT4hgwWHr9E9pC99rZT2yHAwiwSwku6Bne9",
					"2yVjuQwpsvdsrywzsJJVs9Ueh4zayyo5DYJbBNc3DDpn",
					"9bFNrXNb2WTx8fMHXCheaZqkLZ3YCCaiqTftHxeintHy",
					"9vohBn118ZEctRmuTRvoUZg1B1HGfSH8C5QX6twtUFrJ",
					"HeccUHmoyMi5S6nuTcyUBh4w4me3FP541a52ErYJRT8a",
					"11111111111111111111111111111111",
					"EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX",
					"HQS31aApX3DDkuXgSpV9XyDUNtFgQ31pUn5BNWHG2PSp",
					"SysvarC1ock11111111111111111111111111111111",
					"SysvarRent111111111111111111111111111111111",
					"worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
				],
				"recentBlockhash": "CMqPGm4icRdNuHsWJUK4Kgu4Cbe2nDQkYNqugQkKPa4Y",
				"instructions": [
					{
						"accounts": [0, 2],
						"data": "3Bxs4HanWsHUZCbH",
						"programIdIndex": 5,
						"stackHeight": null
					},
					{
						"accounts": [1, 3, 0, 4, 0, 2, 8, 5, 9, 10, 7, 6],
						"data": "3Cn8VBJReY7Bku3RduhBfYpk7tiw1R6pKcTWv9R",
						"programIdIndex": 6,
						"stackHeight": null
					}
				],
				"indexToProgramIds": {}
			},
			"signatures": [
				"3NACxoZLehbdKGjTWZKTTXJPuovyqAih1AD1BrkYj8nzDAtjiQUEaNmhkoU1jcFfoPTAjrvnaLFgTafNWr3fBrdB"
			]
		},
		"version": "legacy"
	}
	`

	///////// A bunch of checks to verify we parsed the JSON correctly.
	var txRpc rpc.TransactionWithMeta
	err := json.Unmarshal([]byte(eventJson), &txRpc)
	require.NoError(t, err)

	tx, err := txRpc.GetParsedTransaction()
	require.NoError(t, err)

	require.Equal(t, 2, len(tx.Message.Instructions))
	require.Equal(t, 1, len(txRpc.Meta.InnerInstructions))

	///////// Now we start the real test.

	logger := zap.NewNop()
	msgC := make(chan *common.MessagePublication, 10)
	s := shimNewWatcherForTest(t, msgC)
	require.True(t, s.shimEnabled)

	var whProgramIndex uint16
	var shimProgramIndex uint16
	var shimFound bool
	for n, key := range tx.Message.AccountKeys {
		if key.Equals(s.contract) {
			whProgramIndex = uint16(n) // #nosec G115 -- The solana runtime max transaction size is 1232 bytes. So we'd never be able to have this many accounts.
		}
		if key.Equals(s.shimContractAddr) {
			shimProgramIndex = uint16(n) // #nosec G115 -- The solana runtime max transaction size is 1232 bytes. So we'd never be able to have this many accounts.
			shimFound = true
		}
	}

	require.Equal(t, uint16(10), whProgramIndex)
	require.True(t, shimFound)
	require.Equal(t, uint16(6), shimProgramIndex)

	alreadyProcessed := ShimAlreadyProcessed{}
	err = s.shimProcessRest(logger, whProgramIndex, shimProgramIndex, tx, txRpc.Meta.InnerInstructions[0].Instructions, 0, 10, nil, alreadyProcessed, false, true)
	require.ErrorContains(t, err, "postMessage is nil")
	require.Equal(t, 0, len(s.msgC))
	require.Equal(t, 0, len(alreadyProcessed))
}
