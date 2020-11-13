package terra

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"github.com/terra-project/terra.go/client"
	"github.com/terra-project/terra.go/key"
	"github.com/terra-project/terra.go/msg"
	"github.com/terra-project/terra.go/tx"
)

type JSONArraySlice []uint8

func (u JSONArraySlice) MarshalJSON() ([]uint8, error) {
	var result string
	if u == nil {
		result = "null"
	} else {
		result = strings.Join(strings.Fields(fmt.Sprintf("%d", u)), ",")
	}
	return []byte(result), nil
}

type SubmitVAAMsg struct {
	Params SubmitVAAParams `json:"submit_v_a_a"`
}

type SubmitVAAParams struct {
	VAA JSONArraySlice `json:"vaa"`
}

// SubmitVAA prepares transaction with signed VAA and sends it to the Terra blockchain
func SubmitVAA(urlLCD string, chainID string, contractAddress string, feePayer string, signed *vaa.VAA) (client.TxResponse, error) {

	// Serialize VAA
	vaa, err := signed.Marshal()
	if err != nil {
		return client.TxResponse{}, err
	}

	// Derive Raw Private Key
	privKey, err := key.DerivePrivKey(feePayer, key.CreateHDPath(0, 0))
	if err != nil {
		return client.TxResponse{}, err
	}

	// Generate StdPrivKey
	tmKey, err := key.StdPrivKeyGen(privKey)
	if err != nil {
		return client.TxResponse{}, err
	}

	// Generate Address from Public Key
	addr := msg.AccAddress(tmKey.PubKey().Address())
	if err != nil {
		return client.TxResponse{}, err
	}

	// Create LCDClient
	LCDClient := client.NewLCDClient(
		urlLCD,
		chainID,
		msg.NewDecCoinFromDec("uusd", msg.NewDecFromIntWithPrec(msg.NewInt(15), 2)), // 0.15uusd
		msg.NewDecFromIntWithPrec(msg.NewInt(15), 1), tmKey,
	)

	contract, err := msg.AccAddressFromBech32(contractAddress)
	if err != nil {
		return client.TxResponse{}, err
	}

	// Create tx
	contractCall, err := json.Marshal(SubmitVAAMsg{
		Params: SubmitVAAParams{
			VAA: vaa,
		}})

	if err != nil {
		return client.TxResponse{}, err
	}

	executeContract := msg.NewExecuteContract(addr, contract, contractCall, msg.NewCoins())

	tx, err := LCDClient.CreateAndSignTx(client.CreateTxOptions{
		Msgs: []msg.Msg{
			executeContract,
		},
		Fee: tx.StdFee{
			Gas:    msg.NewInt(0),
			Amount: msg.NewCoins(),
		},
	})
	if err != nil {
		return client.TxResponse{}, err
	}

	// Broadcast
	return LCDClient.Broadcast(tx)
}
