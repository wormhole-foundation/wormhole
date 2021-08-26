package terra

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/terra-project/terra.go/client"
	"github.com/terra-project/terra.go/key"
	"github.com/terra-project/terra.go/msg"
	"github.com/terra-project/terra.go/tx"

	"github.com/certusone/wormhole/node/pkg/vaa"
)

type submitVAAMsg struct {
	Params submitVAAParams `json:"submit_v_a_a"`
}

type submitVAAParams struct {
	VAA []byte `json:"vaa"`
}

// SubmitVAA prepares transaction with signed VAA and sends it to the Terra blockchain
func SubmitVAA(ctx context.Context, urlLCD string, chainID string, contractAddress string, feePayer string, signed *vaa.VAA) (*client.TxResponse, error) {

	// Serialize VAA
	vaaBytes, err := signed.Marshal()
	if err != nil {
		return nil, err
	}

	// Derive Raw Private Key
	privKey, err := key.DerivePrivKey(feePayer, key.CreateHDPath(0, 0))
	if err != nil {
		return nil, err
	}

	// Generate StdPrivKey
	tmKey, err := key.StdPrivKeyGen(privKey)
	if err != nil {
		return nil, err
	}

	// Generate Address from Public Key
	addr := msg.AccAddress(tmKey.PubKey().Address())

	// Create LCDClient
	LCDClient := client.NewLCDClient(
		urlLCD,
		chainID,
		msg.NewDecCoinFromDec("uusd", msg.NewDecFromIntWithPrec(msg.NewInt(15), 2)), // 0.15uusd
		msg.NewDecFromIntWithPrec(msg.NewInt(15), 1), tmKey, time.Second*15,
	)

	contract, err := msg.AccAddressFromBech32(contractAddress)
	if err != nil {
		return nil, err
	}

	// Create tx
	contractCall, err := json.Marshal(submitVAAMsg{
		Params: submitVAAParams{
			VAA: vaaBytes,
		}})

	if err != nil {
		return nil, err
	}

	executeContract := msg.NewExecuteContract(addr, contract, contractCall, msg.NewCoins())

	transaction, err := LCDClient.CreateAndSignTx(ctx, client.CreateTxOptions{
		Msgs: []msg.Msg{
			executeContract,
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
	return LCDClient.Broadcast(ctx, transaction)
}

// ReadKey reads file and returns its content as a string
func ReadKey(path string) (string, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
