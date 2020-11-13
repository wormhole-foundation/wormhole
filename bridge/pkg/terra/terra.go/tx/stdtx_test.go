package tx

import (
	"encoding/json"
	"testing"

	"github.com/certusone/wormhole/bridge/pkg/terra/terra.go/key"
	"github.com/certusone/wormhole/bridge/pkg/terra/terra.go/msg"
	"github.com/stretchr/testify/assert"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Test_StdTx(t *testing.T) {
	addr, _ := msg.AccAddressFromBech32("terra1cevwjzwft3pjuf5nc32d9kyrvh5y7fp9havw7k")
	stdTx := NewStdTx([]msg.Msg{
		msg.NewExecuteContract(addr, addr, []byte("{\"withdraw\": {\"position_idx\": \"1\", \"collateral\": {\"info\": {\"native_token\": {\"denom\": \"uusd\"}}, \"amount\": \"1000\"}}}"), msg.Coins{}),
	}, "", StdFee{
		Amount: msg.Coins{},
		Gas:    msg.NewInt(1000000),
	})

	bz, err := json.Marshal(stdTx)
	assert.NoError(t, err)
	assert.Equal(t, sdk.MustSortJSON(bz), sdk.MustSortJSON([]byte(
		`{"type":"core/StdTx","value":{"msg":[{"type":"wasm/MsgExecuteContract","value":{"sender":"terra1cevwjzwft3pjuf5nc32d9kyrvh5y7fp9havw7k","contract":"terra1cevwjzwft3pjuf5nc32d9kyrvh5y7fp9havw7k","execute_msg":"eyJ3aXRoZHJhdyI6IHsicG9zaXRpb25faWR4IjogIjEiLCAiY29sbGF0ZXJhbCI6IHsiaW5mbyI6IHsibmF0aXZlX3Rva2VuIjogeyJkZW5vbSI6ICJ1dXNkIn19LCAiYW1vdW50IjogIjEwMDAifX19","coins":[]}}],"fee":{"amount":[],"gas":"1000000"},"signatures":[],"memo":""}}`,
	)))
}

func Test_Sign(t *testing.T) {
	mnemonic := "essence gallery exit illegal nasty luxury sport trouble measure benefit busy almost bulb fat shed today produce glide meadow require impact fruit omit weasel"
	privKey, err := key.DerivePrivKey(mnemonic, key.CreateHDPath(0, 0))
	assert.NoError(t, err)
	tmKey, err := key.StdPrivKeyGen(privKey)
	assert.NoError(t, err)

	addr := msg.AccAddress(tmKey.PubKey().Address())
	assert.Equal(t, addr.String(), "terra1cevwjzwft3pjuf5nc32d9kyrvh5y7fp9havw7k")

	stdTx := NewStdTx([]msg.Msg{
		msg.NewExecuteContract(addr, addr, []byte("{\"withdraw\": {\"position_idx\": \"1\", \"collateral\": {\"info\": {\"native_token\": {\"denom\": \"uusd\"}}, \"amount\": \"1000\"}}}"), msg.Coins{}),
	}, "", StdFee{
		Amount: msg.Coins{},
		Gas:    msg.NewInt(1000000),
	})

	signature, err := stdTx.Sign(tmKey, "testnet", msg.NewInt(359), msg.NewInt(4))
	bz, err := json.Marshal(signature)
	assert.NoError(t, err)

	assert.Equal(t, bz, []byte(`{"pub_key":{"type":"tendermint/PubKeySecp256k1","value":"AmADjpxwusAnJ7ahD7+trzovH32w+LaRGVZSZUOd3E3d"},"signature":"NVHiNY72K/bypFqmy5sbwQe7CIxAOy6fq+DT36nfy4MVku130vKX93J3AZQ15W6JGmYQtvgSB+z5RyQE/NmPWQ=="}`))
}
