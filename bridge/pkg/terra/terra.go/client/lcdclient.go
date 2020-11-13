package client

import (
	"github.com/certusone/wormhole/bridge/pkg/terra/terra.go/key"
	"github.com/certusone/wormhole/bridge/pkg/terra/terra.go/msg"
	"github.com/certusone/wormhole/bridge/pkg/terra/terra.go/tx"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// LCDClient outer interface for building & signing & broadcasting tx
type LCDClient struct {
	URL           string
	ChainID       string
	GasPrice      msg.DecCoin
	GasAdjustment msg.Dec

	TmKey key.StdPrivKey
}

// NewLCDClient create new LCDClient
func NewLCDClient(URL, chainID string, gasPrice msg.DecCoin, gasAdjustment msg.Dec, tmKey key.StdPrivKey) LCDClient {
	return LCDClient{
		URL:           URL,
		ChainID:       chainID,
		GasPrice:      gasPrice,
		GasAdjustment: gasAdjustment,
		TmKey:         tmKey,
	}
}

// CreateTxOptions tx creation options
type CreateTxOptions struct {
	Msgs []msg.Msg
	Memo string

	// Optional parameters
	AccountNumber msg.Int
	Sequence      msg.Int
	Fee           tx.StdFee
}

// CreateAndSignTx build and sign tx
func (lcdClient LCDClient) CreateAndSignTx(options CreateTxOptions) (tx.StdTx, error) {
	stdTx := tx.NewStdTx(options.Msgs, options.Memo, options.Fee)
	if options.Fee.IsEmpty() {
		fee, err := lcdClient.EstimateFee(stdTx)
		if err != nil {
			return tx.StdTx{}, sdkerrors.Wrap(err, "failed to estimate fee")
		}

		stdTx.Value.Fee.Amount = fee.Fees
		stdTx.Value.Fee.Gas = fee.Gas
	}

	if (msg.Int{}) == options.AccountNumber ||
		(msg.Int{}) == options.Sequence ||
		options.AccountNumber.IsZero() {
		account, err := lcdClient.LoadAccount(msg.AccAddress(lcdClient.TmKey.PubKey().Address()))
		if err != nil {
			return tx.StdTx{}, sdkerrors.Wrap(err, "failed to load account")
		}

		options.AccountNumber = account.AccountNumber
		options.Sequence = account.Sequence
	}

	signature, err := stdTx.Sign(lcdClient.TmKey, lcdClient.ChainID, options.AccountNumber, options.Sequence)
	if err != nil {
		return tx.StdTx{}, sdkerrors.Wrap(err, "failed to sign tx")
	}

	stdTx.AppendSignatures(signature)
	return stdTx, nil
}
