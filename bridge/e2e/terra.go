package e2e

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/certusone/wormhole/bridge/pkg/devnet"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"github.com/tendermint/tendermint/libs/rand"
	"github.com/terra-project/terra.go/client"
	"github.com/terra-project/terra.go/key"
	"github.com/terra-project/terra.go/msg"
	"github.com/terra-project/terra.go/tx"
	"github.com/tidwall/gjson"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type lockAssetsMsg struct {
	Params lockAssetsParams `json:"lock_assets"`
}

type increaseAllowanceMsg struct {
	Params increaseAllowanceParams `json:"increase_allowance"`
}

type lockAssetsParams struct {
	Asset       string `json:"asset"`
	Amount      string `json:"amount"`
	Recipient   []byte `json:"recipient"`
	TargetChain uint8  `json:"target_chain"`
	Nonce       uint32 `json:"nonce"`
}

type increaseAllowanceParams struct {
	Spender string `json:"spender"`
	Amount  string `json:"amount"`
}

// TerraClient encapsulates Terra LCD client and fee payer signing address
type TerraClient struct {
	lcdClient client.LCDClient
	address   msg.AccAddress
}

func (tc TerraClient) lockAssets(t *testing.T, ctx context.Context, token string, amount *big.Int, recipient [32]byte, targetChain uint8, nonce uint32) (*client.TxResponse, error) {
	bridgeContract, err := msg.AccAddressFromBech32(devnet.TerraBridgeAddress)
	if err != nil {
		return nil, err
	}

	tokenContract, err := msg.AccAddressFromBech32(token)
	if err != nil {
		return nil, err
	}

	// Create tx
	increaseAllowanceCall, err := json.Marshal(increaseAllowanceMsg{
		Params: increaseAllowanceParams{
			Spender: devnet.TerraBridgeAddress,
			Amount:  amount.String(),
		}})

	if err != nil {
		return nil, err
	}

	lockAssetsCall, err := json.Marshal(lockAssetsMsg{
		Params: lockAssetsParams{
			Asset:       token,
			Amount:      amount.String(),
			Recipient:   recipient[:],
			TargetChain: targetChain,
			Nonce:       nonce,
		}})

	if err != nil {
		return nil, err
	}

	t.Logf("increaseAllowanceCall\n %s", increaseAllowanceCall)
	t.Logf("lockAssetsCall\n %s", lockAssetsCall)

	executeIncreaseAllowance := msg.NewExecuteContract(tc.address, tokenContract, increaseAllowanceCall, msg.NewCoins())
	executeLockAssets := msg.NewExecuteContract(tc.address, bridgeContract, lockAssetsCall, msg.NewCoins())

	transaction, err := tc.lcdClient.CreateAndSignTx(ctx, client.CreateTxOptions{
		Msgs: []msg.Msg{
			executeIncreaseAllowance,
			executeLockAssets,
		},
		Fee: tx.StdFee{
			Gas:    msg.NewInt(0),
			Amount: msg.NewCoins(),
		},
	})
	if err != nil {
		return nil, err
	}

	// Broadcast
	return tc.lcdClient.Broadcast(ctx, transaction)
}

// NewTerraClient creates new TerraClient instance to work
func NewTerraClient() (*TerraClient, error) {
	// Derive Raw Private Key
	privKey, err := key.DerivePrivKey(devnet.TerraFeePayerKey, key.CreateHDPath(0, 0))
	if err != nil {
		return nil, err
	}

	// Generate StdPrivKey
	tmKey, err := key.StdPrivKeyGen(privKey)
	if err != nil {
		return nil, err
	}

	// Generate Address from Public Key
	address := msg.AccAddress(tmKey.PubKey().Address())

	// Terra client
	lcdClient := client.NewLCDClient(
		devnet.TerraLCDURL,
		devnet.TerraChainID,
		msg.NewDecCoinFromDec("uusd", msg.NewDecFromIntWithPrec(msg.NewInt(15), 2)), // 0.15uusd
		msg.NewDecFromIntWithPrec(msg.NewInt(15), 1), tmKey, time.Second*15,
	)

	return &TerraClient{
		lcdClient: *lcdClient,
		address:   address,
	}, nil
}

func getTerraBalance(ctx context.Context, token string) (*big.Int, error) {
	json, err := terraQuery(ctx, token, fmt.Sprintf("{\"balance\":{\"address\":\"%s\"}}", devnet.TerraMainTestAddress))
	if err != nil {
		return nil, err
	}
	balance := gjson.Get(json, "result.balance").String()
	parsed, success := new(big.Int).SetString(balance, 10)

	if !success {
		return nil, fmt.Errorf("cannot parse balance: %s", balance)
	}

	return parsed, nil
}

func getAssetAddress(ctx context.Context, contract string, chain uint8, asset []byte) (string, error) {
	json, err := terraQuery(ctx, contract, fmt.Sprintf("{\"wrapped_registry\":{\"chain\":%d,\"address\":\"%s\"}}",
		chain,
		base64.StdEncoding.EncodeToString(asset)))
	if err != nil {
		return "", err
	}
	return gjson.Get(json, "result.address").String(), nil
}

func terraQuery(ctx context.Context, contract string, query string) (string, error) {

	requestURL := fmt.Sprintf("%s/wasm/contracts/%s/store?query_msg=%s", devnet.TerraLCDURL, contract, query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("http request error: %w", err)
	}

	client := &http.Client{
		Timeout: time.Second * 15,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http execution error: %w", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("http read error: %w", err)
	}

	return string(body), nil
}

// waitTerraAsset waits for asset contract to be deployed on terra
func waitTerraAsset(t *testing.T, ctx context.Context, contract string, chain uint8, asset []byte) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	assetAddress := ""

	err := wait.PollUntil(1*time.Second, func() (bool, error) {

		address, err := getAssetAddress(ctx, contract, chain, asset)
		if err != nil {
			t.Log(err)
			return true, nil
		}

		assetAddress = address
		return false, nil
	}, ctx.Done())

	if err != nil {
		t.Error(err)
	}
	return assetAddress, err
}

// waitTerraBalance waits for target account before to increase.
func waitTerraBalance(t *testing.T, ctx context.Context, token string, before *big.Int, target int64) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	err := wait.PollUntil(1*time.Second, func() (bool, error) {

		after, err := getTerraBalance(ctx, token)
		if err != nil {
			return false, err
		}

		d := new(big.Int).Sub(after, before)
		t.Logf("CW20 balance after: %d -> %d, delta %d", before, after, d)

		if after.Cmp(before) != 0 {
			if d.Cmp(new(big.Int).SetInt64(target)) != 0 {
				t.Errorf("expected CW20 delta of %v, got: %v", target, d)
			}
			return true, nil
		}
		return false, nil
	}, ctx.Done())

	if err != nil {
		t.Error(err)
	}
}

func waitTerraUnknownBalance(t *testing.T, ctx context.Context, contract string, chain uint8, asset []byte, before *big.Int, target int64) {

	token, err := waitTerraAsset(t, ctx, contract, chain, asset)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	err = wait.PollUntil(1*time.Second, func() (bool, error) {

		after, err := getTerraBalance(ctx, token)
		if err != nil {
			return false, err
		}

		d := new(big.Int).Sub(after, before)
		t.Logf("CW20 balance after: %d -> %d, delta %d", before, after, d)

		if after.Cmp(before) != 0 {
			if d.Cmp(new(big.Int).SetInt64(target)) != 0 {
				t.Errorf("expected CW20 delta of %v, got: %v", target, d)
			}
			return true, nil
		}
		return false, nil
	}, ctx.Done())

	if err != nil {
		t.Error(err)
	}
}

func testTerraLockup(t *testing.T, ctx context.Context, tc *TerraClient,
	c *kubernetes.Clientset, token string, destination string, amount int64, precisionLoss int) {

	// Store balance of source CW20 token
	beforeCw20, err := getTerraBalance(ctx, token)
	if err != nil {
		t.Log(err) // account may not yet exist, defaults to 0
	}
	t.Logf("CW20 balance: %v", beforeCw20)

	// Store balance of destination SPL token
	beforeSPL, err := getSPLBalance(ctx, c, destination)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("SPL balance: %d", beforeSPL)

	// Send lockup
	tx, err := tc.lockAssets(
		t, ctx,
		// asset address
		token,
		// token amount
		new(big.Int).SetInt64(amount),
		// recipient address on target chain
		devnet.MustBase58ToEthAddress(destination),
		// target chain
		vaa.ChainIDSolana,
		// random nonce
		rand.Uint32(),
	)
	if err != nil {
		t.Error(err)
	}

	t.Logf("sent lockup tx: %s", tx.TxHash)

	// Destination account increases by full amount.
	waitSPLBalance(t, ctx, c, destination, beforeSPL, int64(float64(amount)/math.Pow10(precisionLoss)))

	// Source account decreases by the full amount.
	waitTerraBalance(t, ctx, token, beforeCw20, -int64(amount))
}
