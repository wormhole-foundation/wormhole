package tx

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/certusone/wormhole/bridge/pkg/terra/terra.go/key"
	"github.com/certusone/wormhole/bridge/pkg/terra/terra.go/msg"
)

// StdFee includes the amount of coins paid in fees and the maximum
// gas to be used by the transaction. The ratio yields an effective "gasprice",
// which must be above some miminum to be accepted into the mempool.
type StdFee struct {
	Amount msg.Coins `json:"amount" yaml:"amount"`
	Gas    msg.Int   `json:"gas" yaml:"gas"`
}

// IsEmpty return empty or not
func (stdFee StdFee) IsEmpty() bool {
	return ((msg.Int{}) == stdFee.Gas || stdFee.Gas.IsZero()) && stdFee.Amount.Empty()
}

// StdPubKey - tendermint style pubkey
type StdPubKey struct {
	Type  string `json:"type"`
	Value []byte `json:"value"`
}

// StdSignature represents a sig
type StdSignature struct {
	PubKey    StdPubKey `json:"pub_key"`
	Signature []byte    `json:"signature"`
}

// StdSignMsg is the body for a sign request
type StdSignMsg struct {
	AccountNumber msg.Int   `json:"account_number"`
	ChainID       string    `json:"chain_id"`
	Fee           StdFee    `json:"fee"`
	Msgs          []msg.Msg `json:"msgs"`
	Memo          string    `json:"memo"`
	Sequence      msg.Int   `json:"sequence"`
}

// StdTx - high level transaction
type StdTx struct {
	Type  string    `json:"type"`
	Value StdTxData `json:"value"`
}

// StdTxData - value part of MsgStoreCode
type StdTxData struct {
	Msgs       []msg.Msg      `json:"msg"`
	Fee        StdFee         `json:"fee"`
	Signatures []StdSignature `json:"signatures"`
	Memo       string         `json:"memo"`
}

// NewStdTx - create StdTx
func NewStdTx(msgs []msg.Msg, memo string, fee StdFee) StdTx {
	return StdTx{
		Type: "core/StdTx",
		Value: StdTxData{
			Msgs:       msgs,
			Fee:        fee,
			Memo:       memo,
			Signatures: []StdSignature{},
		},
	}
}

// Sign - generate signatures of the tx with given armored private key
// Only support Secp256k1 uses the Bitcoin secp256k1 ECDSA parameters.
func (stdTx StdTx) Sign(privKey key.StdPrivKey, chainID string, accountNumber, sequence msg.Int) (StdSignature, error) {
	bz, err := json.Marshal(StdSignMsg{
		AccountNumber: accountNumber,
		ChainID:       chainID,
		Fee:           stdTx.Value.Fee,
		Msgs:          stdTx.Value.Msgs,
		Memo:          stdTx.Value.Memo,
		Sequence:      sequence,
	})

	if err != nil {
		return StdSignature{}, sdkerrors.Wrap(err, "failed to marshal")
	}

	sigBytes, err := privKey.Sign(sdk.MustSortJSON(bz))
	if err != nil {
		return StdSignature{}, sdkerrors.Wrap(err, "failed to sign")
	}

	return StdSignature{
		PubKey: StdPubKey{
			Type:  "tendermint/PubKeySecp256k1",
			Value: privKey.PubKey().Bytes()[5:],
		},
		Signature: sigBytes,
	}, nil
}

// AppendSignatures append signature to StdTx
func (stdTx *StdTx) AppendSignatures(signature ...StdSignature) {
	stdTx.Value.Signatures = append(stdTx.Value.Signatures, signature...)
}
