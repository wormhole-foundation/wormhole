package client

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/certusone/wormhole/bridge/pkg/terra/terra.go/key"
	"github.com/certusone/wormhole/bridge/pkg/terra/terra.go/msg"
)

func Test_Transaction(t *testing.T) {
	mnemonic := "essence gallery exit illegal nasty luxury sport trouble measure benefit busy almost bulb fat shed today produce glide meadow require impact fruit omit weasel"
	privKey, err := key.DerivePrivKey(mnemonic, key.CreateHDPath(0, 0))
	assert.NoError(t, err)
	tmKey, err := key.StdPrivKeyGen(privKey)
	assert.NoError(t, err)

	addr := msg.AccAddress(tmKey.PubKey().Address())
	assert.Equal(t, addr.String(), "terra1cevwjzwft3pjuf5nc32d9kyrvh5y7fp9havw7k")

	toAddr, err := msg.AccAddressFromBech32("terra1t849fxw7e8ney35mxemh4h3ayea4zf77dslwna")
	assert.NoError(t, err)

	LCDClient := NewLCDClient(
		"http://127.0.0.1:1317",
		"testnet",
		msg.NewDecCoinFromDec("uusd", msg.NewDecFromIntWithPrec(msg.NewInt(15), 2)), // 0.15uusd
		msg.NewDecFromIntWithPrec(msg.NewInt(15), 1), tmKey,
	)

	tx, err := LCDClient.CreateAndSignTx(CreateTxOptions{
		Msgs: []msg.Msg{
			msg.NewSend(addr, toAddr, msg.NewCoins(msg.NewInt64Coin("uusd", 100000000))), // 100UST
			msg.NewSwapSend(addr, toAddr, msg.NewInt64Coin("uusd", 1000000), "ukrw"),
		},
		Memo: "",
	})
	assert.NoError(t, err)

	res, err := LCDClient.Broadcast(tx)
	assert.NoError(t, err)
	assert.Equal(t, res.Code, uint32(0))
}
