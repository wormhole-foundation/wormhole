// This tool can be used to confirm that the CoinkGecko price query still works after the token list is updated.
// Usage: go run check_query.go

package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/accounting"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/devnet"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/certusone/wormhole/node/pkg/wormconn"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCrypto "github.com/ethereum/go-ethereum/crypto"

	sdktx "github.com/cosmos/cosmos-sdk/types/tx"

	"golang.org/x/crypto/openpgp/armor" //nolint
	"google.golang.org/protobuf/proto"

	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	wormchainURL := string("localhost:9090")
	wormchainKeyPath := string("./dev.wormchain.key")
	contract := "wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh"
	guardianKeyPath := string("./dev.guardian.key")

	wormchainKey, err := devnet.LoadWormchainPrivKey(wormchainKeyPath)
	if err != nil {
		logger.Fatal("failed to load devnet wormchain private key", zap.Error(err))
	}

	wormchainConn, err := wormconn.NewConn(ctx, wormchainURL, wormchainKey)
	if err != nil {
		logger.Fatal("failed to connect to wormchain", zap.Error(err))
	}

	logger.Info("Connected to wormchain",
		zap.String("wormchainURL", wormchainURL),
		zap.String("wormchainKeyPath", wormchainKeyPath),
		zap.String("publicKey", wormchainConn.PublicKey()),
	)

	logger.Info("Loading guardian key", zap.String("guardianKeyPath", guardianKeyPath))
	gk, err := loadGuardianKey(guardianKeyPath)
	if err != nil {
		logger.Fatal("failed to load guardian key", zap.Error(err))
	}

	sequence := uint64(time.Now().Unix())

	if !testSubmit(ctx, logger, gk, wormchainConn, contract, "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16", sequence, 0, "Submit should succeed") {
		return
	}

	if !testSubmit(ctx, logger, gk, wormchainConn, contract, "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16", sequence, 1, "Duplicate transfer should succeed but not publish") {
		return
	}

	sequence += 1
	if !testSubmit(ctx, logger, gk, wormchainConn, contract, "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c17", sequence, -1, "Bad emitter address should fail") {
		return
	}
}

func testSubmit(
	ctx context.Context,
	logger *zap.Logger,
	gk *ecdsa.PrivateKey,
	wormchainConn *wormconn.ClientConn,
	contract string,
	emitterAddressStr string,
	sequence uint64,
	expectedResult int,
	tag string,
) bool {
	EmitterAddress, _ := vaa.StringToAddress(emitterAddressStr)
	TxHash, _ := vaa.StringToHash("82ea2536c5d1671830cb49120f94479e34b54596a8dd369fbc2666667a765f4b")
	Payload, _ := hex.DecodeString("010000000000000000000000000000000000000000000000000de0b6b3a76400000000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a0002000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d0c200000000000000000000000000000000000000000000000000000000000000000")
	gsIndex := uint32(0)

	msg := common.MessagePublication{
		TxHash:           TxHash,
		Timestamp:        time.Now(),
		Nonce:            uint32(0),
		Sequence:         sequence,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   EmitterAddress,
		ConsistencyLevel: uint8(15),
		Payload:          Payload,
	}

	txResp, err := accounting.SubmitObservationToContract(ctx, gk, gsIndex, wormchainConn, contract, &msg)
	if err != nil {
		logger.Error("acct: failed to broadcast Observation request", zap.Error(err))
		return false
	}

	// out, err := wormchainConn.BroadcastTxResponseToString(txResp)
	// if err != nil {
	// 	logger.Error("acct: failed to parse broadcast response", zap.Error(err))
	// 	return false
	// }

	result := CheckSubmitObservationResult(txResp)
	if result != expectedResult {
		out, err := wormchainConn.BroadcastTxResponseToString(txResp)
		if err != nil {
			logger.Error("acct: failed to parse broadcast response", zap.Error(err))
			return false
		}

		logger.Info("test failed", zap.String("test", tag), zap.Uint64("seqNo", sequence), zap.Int("result", result), zap.String("response", out))
		return false
	}

	logger.Info("test succeeded", zap.String("test", tag))
	return true
}

// checkResult() returns zero if the observation was submitted and the transfer should be queued up, a positive value
// if the transfer can be published immediately, and a negative value if an error occurred.
func CheckSubmitObservationResult(txResp *sdktx.BroadcastTxResponse) int {
	if strings.Contains(txResp.TxResponse.RawLog, "execute wasm contract failed") {
		if strings.Contains(txResp.TxResponse.RawLog, "already committed") {
			return 1

		}

		return -1
	}

	if strings.Contains(txResp.TxResponse.RawLog, "failed to execute message") {
		return -1
	}

	return 0
}

/*
Already Committed error:
2022-12-17T00:10:04.584Z        INFO    accounting/send_obs.go:94       Sent observation request to wormchain   {"resp": "{\"tx_response\":{\"height\":\"1280\",\"txhash\":\"5417A62D3830C6C128298404A1D7734082F3B52717F7824CC8D4DCB5A3E9EDF8\",\"codespace\":\"wasm\",\"code\":5,\"data\":\"\",\"raw_log\":\"failed to execute message; message index: 0: failed to handle `Observation`: transfer for key \\\"00002/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0000000000000000\\\" already committed: execute wasm contract failed\",\"logs\":[],\"info\":\"\",\"gas_wanted\":\"2000000\",\"gas_used\":\"117488\",\"tx\":null,\"timestamp\":\"\",\"events\":[{\"type\":\"tx\",\"attributes\":[{\"key\":\"ZmVl\",\"value\":null,\"index\":true},{\"key\":\"ZmVlX3BheWVy\",\"value\":\"d29ybWhvbGUxY3l5enB4cGx4ZHprZWVhN2t3c3lkYWRnODczNTdxbmEzemczdHE=\",\"index\":true}]},{\"type\":\"tx\",\"attributes\":[{\"key\":\"YWNjX3NlcQ==\",\"value\":\"d29ybWhvbGUxY3l5enB4cGx4ZHprZWVhN2t3c3lkYWRnODczNTdxbmEzemczdHEvMjA=\",\"index\":true}]},{\"type\":\"tx\",\"attributes\":[{\"key\":\"c2lnbmF0dXJl\",\"value\":\"cFRTVnVFYXIwYjBqSFJ2QXpPc2ZSanVHWVh4aURkT1JDenF4RmdqTXlFODdmbkJ1YytSNjl1NTVrYlVJZ2pWclhNcEgwV2hlVDN0N3B3Z1ZXUDlQK0E9PQ==\",\"index\":true}]}]}}"}

Success:
2022-12-17T00:09:19.103Z        INFO    accounting/send_obs.go:94       Sent observation request to wormchain   {"resp": "{\"tx_response\":{\"height\":\"1239\",\"txhash\":\"B39041E2CC3896CD76605C8CB2EA6037B6A007AEC424A6BC532FBE6ED017DF9D\",\"codespace\":\"\",\"code\":0,\"data\":\"0A260A242F636F736D7761736D2E7761736D2E76312E4D736745786563757465436F6E7472616374\",\"raw_log\":\"[{\\\"events\\\":[{\\\"type\\\":\\\"execute\\\",\\\"attributes\\\":[{\\\"key\\\":\\\"_contract_address\\\",\\\"value\\\":\\\"wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh\\\"}]},{\\\"type\\\":\\\"message\\\",\\\"attributes\\\":[{\\\"key\\\":\\\"action\\\",\\\"value\\\":\\\"/cosmwasm.wasm.v1.MsgExecuteContract\\\"},{\\\"key\\\":\\\"module\\\",\\\"value\\\":\\\"wasm\\\"},{\\\"key\\\":\\\"sender\\\",\\\"value\\\":\\\"wormhole1cyyzpxplxdzkeea7kwsydadg87357qna3zg3tq\\\"}]},{\\\"type\\\":\\\"wasm\\\",\\\"attributes\\\":[{\\\"key\\\":\\\"_contract_address\\\",\\\"value\\\":\\\"wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh\\\"},{\\\"key\\\":\\\"action\\\",\\\"value\\\":\\\"submit_observations\\\"},{\\\"key\\\":\\\"owner\\\",\\\"value\\\":\\\"wormhole1cyyzpxplxdzkeea7kwsydadg87357qna3zg3tq\\\"}]},{\\\"type\\\":\\\"wasm-Transfer\\\",\\\"attributes\\\":[{\\\"key\\\":\\\"_contract_address\\\",\\\"value\\\":\\\"wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh\\\"},{\\\"key\\\":\\\"emitter_chain\\\",\\\"value\\\":\\\"2\\\"},{\\\"key\\\":\\\"emitter_address\\\",\\\"value\\\":\\\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\\\"},{\\\"key\\\":\\\"sequence\\\",\\\"value\\\":\\\"0\\\"},{\\\"key\\\":\\\"nonce\\\",\\\"value\\\":\\\"0\\\"},{\\\"key\\\":\\\"tx_hash\\\",\\\"value\\\":\\\"82ea2536c5d1671830cb49120f94479e34b54596a8dd369fbc2666667a765f4b\\\"},{\\\"key\\\":\\\"payload\\\",\\\"value\\\":\\\"AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA3gtrOnZAAAAAAAAAAAAAAAAAAALYvmvwuqdOCpBwFmecrpGQ6A3QoAAgAAAAAAAAAAAAAAAMEIIJg/M0Vs576zoEb1qD+jTwJ9DCAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==\\\"}]}]}]\",\"logs\":[{\"msg_index\":0,\"log\":\"\",\"events\":[{\"type\":\"execute\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh\"}]},{\"type\":\"message\",\"attributes\":[{\"key\":\"action\",\"value\":\"/cosmwasm.wasm.v1.MsgExecuteContract\"},{\"key\":\"module\",\"value\":\"wasm\"},{\"key\":\"sender\",\"value\":\"wormhole1cyyzpxplxdzkeea7kwsydadg87357qna3zg3tq\"}]},{\"type\":\"wasm\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh\"},{\"key\":\"action\",\"value\":\"submit_observations\"},{\"key\":\"owner\",\"value\":\"wormhole1cyyzpxplxdzkeea7kwsydadg87357qna3zg3tq\"}]},{\"type\":\"wasm-Transfer\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh\"},{\"key\":\"emitter_chain\",\"value\":\"2\"},{\"key\":\"emitter_address\",\"value\":\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\"},{\"key\":\"sequence\",\"value\":\"0\"},{\"key\":\"nonce\",\"value\":\"0\"},{\"key\":\"tx_hash\",\"value\":\"82ea2536c5d1671830cb49120f94479e34b54596a8dd369fbc2666667a765f4b\"},{\"key\":\"payload\",\"value\":\"AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA3gtrOnZAAAAAAAAAAAAAAAAAAALYvmvwuqdOCpBwFmecrpGQ6A3QoAAgAAAAAAAAAAAAAAAMEIIJg/M0Vs576zoEb1qD+jTwJ9DCAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==\"}]}]}],\"info\":\"\",\"gas_wanted\":\"2000000\",\"gas_used\":\"210944\",\"tx\":null,\"timestamp\":\"\",\"events\":[{\"type\":\"tx\",\"attributes\":[{\"key\":\"ZmVl\",\"value\":null,\"index\":true},{\"key\":\"ZmVlX3BheWVy\",\"value\":\"d29ybWhvbGUxY3l5enB4cGx4ZHprZWVhN2t3c3lkYWRnODczNTdxbmEzemczdHE=\",\"index\":true}]},{\"type\":\"tx\",\"attributes\":[{\"key\":\"YWNjX3NlcQ==\",\"value\":\"d29ybWhvbGUxY3l5enB4cGx4ZHprZWVhN2t3c3lkYWRnODczNTdxbmEzemczdHEvMTk=\",\"index\":true}]},{\"type\":\"tx\",\"attributes\":[{\"key\":\"c2lnbmF0dXJl\",\"value\":\"anNQSEhGZURnaE8vSjhzVXJKQmtuQko2aGhJb2lzdkRPWVVYY1lFZ1FYRnRyOVpXQnRBbzV0T3hhUktrNVhXaVEvanF6UXA1TUJyMFFZMjJwMjlLT2c9PQ==\",\"index\":true}]},{\"type\":\"message\",\"attributes\":[{\"key\":\"YWN0aW9u\",\"value\":\"L2Nvc213YXNtLndhc20udjEuTXNnRXhlY3V0ZUNvbnRyYWN0\",\"index\":true}]},{\"type\":\"message\",\"attributes\":[{\"key\":\"bW9kdWxl\",\"value\":\"d2FzbQ==\",\"index\":true},{\"key\":\"c2VuZGVy\",\"value\":\"d29ybWhvbGUxY3l5enB4cGx4ZHprZWVhN2t3c3lkYWRnODczNTdxbmEzemczdHE=\",\"index\":true}]},{\"type\":\"execute\",\"attributes\":[{\"key\":\"X2NvbnRyYWN0X2FkZHJlc3M=\",\"value\":\"d29ybWhvbGUxNDY2bmYzenV4cHlhOHE5ZW14dWtkN3ZmdGFmNmg0cHNyMGEwN3NybDV6dzc0emg4NHlqcTRseWptaA==\",\"index\":true}]},{\"type\":\"wasm\",\"attributes\":[{\"key\":\"X2NvbnRyYWN0X2FkZHJlc3M=\",\"value\":\"d29ybWhvbGUxNDY2bmYzenV4cHlhOHE5ZW14dWtkN3ZmdGFmNmg0cHNyMGEwN3NybDV6dzc0emg4NHlqcTRseWptaA==\",\"index\":true},{\"key\":\"YWN0aW9u\",\"value\":\"c3VibWl0X29ic2VydmF0aW9ucw==\",\"index\":true},{\"key\":\"b3duZXI=\",\"value\":\"d29ybWhvbGUxY3l5enB4cGx4ZHprZWVhN2t3c3lkYWRnODczNTdxbmEzemczdHE=\",\"index\":true}]},{\"type\":\"wasm-Transfer\",\"attributes\":[{\"key\":\"X2NvbnRyYWN0X2FkZHJlc3M=\",\"value\":\"d29ybWhvbGUxNDY2bmYzenV4cHlhOHE5ZW14dWtkN3ZmdGFmNmg0cHNyMGEwN3NybDV6dzc0emg4NHlqcTRseWptaA==\",\"index\":true},{\"key\":\"ZW1pdHRlcl9jaGFpbg==\",\"value\":\"Mg==\",\"index\":true},{\"key\":\"ZW1pdHRlcl9hZGRyZXNz\",\"value\":\"MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDI5MGZiMTY3MjA4YWY0NTViYjEzNzc4MDE2M2I3YjdhOWExMGMxNg==\",\"index\":true},{\"key\":\"c2VxdWVuY2U=\",\"value\":\"MA==\",\"index\":true},{\"key\":\"bm9uY2U=\",\"value\":\"MA==\",\"index\":true},{\"key\":\"dHhfaGFzaA==\",\"value\":\"ODJlYTI1MzZjNWQxNjcxODMwY2I0OTEyMGY5NDQ3OWUzNGI1NDU5NmE4ZGQzNjlmYmMyNjY2NjY3YTc2NWY0Yg==\",\"index\":true},{\"key\":\"cGF5bG9hZA==\",\"value\":\"QVFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQTNndHJPblpBQUFBQUFBQUFBQUFBQUFBQUFBTFl2bXZ3dXFkT0NwQndGbWVjcnBHUTZBM1FvQUFnQUFBQUFBQUFBQUFBQUFBTUVJSUpnL00wVnM1NzZ6b0ViMXFEK2pUd0o5RENBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQT09\",\"index\":true}]}]}}"}

*/

const (
	GuardianKeyArmoredBlock = "WORMHOLE GUARDIAN PRIVATE KEY"
)

// loadGuardianKey loads a serialized guardian key from disk.
func loadGuardianKey(filename string) (*ecdsa.PrivateKey, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	p, err := armor.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read armored file: %w", err)
	}

	if p.Type != GuardianKeyArmoredBlock {
		return nil, fmt.Errorf("invalid block type: %s", p.Type)
	}

	b, err := io.ReadAll(p.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var m nodev1.GuardianKey
	err = proto.Unmarshal(b, &m)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize protobuf: %w", err)
	}

	gk, err := ethCrypto.ToECDSA(m.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize raw key data: %w", err)
	}

	return gk, nil
}

/*
DEBUG: obs: {
  key: {
    emitter_chain: 2,
    emitter_address: 'AAAAAAAAAAAAAAAAApD7FnIIr0VbsTd4AWO3t6mhDBY=',
    sequence: 0
  },
  nonce: 0,
  payload: 'AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA3gtrOnZAAAAAAAAAAAAAAAAAAALYvmvwuqdOCpBwFmecrpGQ6A3QoAAgAAAAAAAAAAAAAAAMEIIJg/M0Vs576zoEb1qD+jTwJ9DCAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==',
  tx_hash: '82ea2536c5d1671830cb49120f94479e34b54596a8dd369fbc2666667a765f4b'
}
*/
