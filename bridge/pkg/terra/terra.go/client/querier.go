package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/certusone/wormhole/bridge/pkg/terra/terra.go/msg"
	"github.com/certusone/wormhole/bridge/pkg/terra/terra.go/tx"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// EstimateFeeReq request
type EstimateFeeReq struct {
	Tx            tx.StdTxData `json:"tx"`
	GasAdjustment string       `json:"gas_adjustment"`
	GasPrices     msg.DecCoins `json:"gas_prices"`
}

// EstimateFeeResp response
type EstimateFeeResp struct {
	Fees msg.Coins `json:"fees"`
	Gas  msg.Int   `json:"gas"`
}

// EstimateFeeResWrapper - wrapper for estimate fee query
type EstimateFeeResWrapper struct {
	Height msg.Int         `json:"height"`
	Result EstimateFeeResp `json:"result"`
}

// EstimateFee simulates gas and fee for a transaction
func (lcdClient LCDClient) EstimateFee(stdTx tx.StdTx) (res EstimateFeeResp, err error) {
	broadcastReq := EstimateFeeReq{
		Tx:            stdTx.Value,
		GasAdjustment: lcdClient.GasAdjustment.String(),
		GasPrices:     msg.DecCoins{lcdClient.GasPrice},
	}

	reqBytes, err := json.Marshal(broadcastReq)
	if err != nil {
		return EstimateFeeResp{}, sdkerrors.Wrap(err, "failed to marshal")
	}

	resp, err := http.Post(lcdClient.URL+"/txs/estimate_fee", "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return EstimateFeeResp{}, sdkerrors.Wrap(err, "failed to estimate")
	}

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return EstimateFeeResp{}, sdkerrors.Wrap(err, "failed to read response")
	}

	if resp.StatusCode != 200 {
		return EstimateFeeResp{}, fmt.Errorf("non 200 respose code %d, error: %s", resp.StatusCode, string(out))
	}

	var response EstimateFeeResWrapper
	err = json.Unmarshal(out, &response)
	if err != nil {
		return EstimateFeeResp{}, sdkerrors.Wrap(err, "failed to unmarshal response")
	}

	return response.Result, nil
}

// QueryAccountResData response
type QueryAccountResData struct {
	Address       msg.AccAddress `json:"address"`
	Coins         msg.Coins      `json:"coins"`
	AccountNumber msg.Int        `json:"account_number"`
	Sequence      msg.Int        `json:"sequence"`
}

// QueryAccountRes response
type QueryAccountRes struct {
	Type  string              `json:"type"`
	Value QueryAccountResData `json:"value"`
}

// QueryAccountResWrapper - wrapper for estimate fee query
type QueryAccountResWrapper struct {
	Height msg.Int         `json:"height"`
	Result QueryAccountRes `json:"result"`
}

// LoadAccount simulates gas and fee for a transaction
func (lcdClient LCDClient) LoadAccount(address msg.AccAddress) (res QueryAccountResData, err error) {
	resp, err := http.Get(lcdClient.URL + fmt.Sprintf("/auth/accounts/%s", address))
	if err != nil {
		return QueryAccountResData{}, sdkerrors.Wrap(err, "failed to estimate")
	}

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return QueryAccountResData{}, sdkerrors.Wrap(err, "failed to read response")
	}

	if resp.StatusCode != 200 {
		return QueryAccountResData{}, fmt.Errorf("non 200 respose code %d, error: %s", resp.StatusCode, string(out))
	}

	var response QueryAccountResWrapper
	err = json.Unmarshal(out, &response)
	if err != nil {
		return QueryAccountResData{}, sdkerrors.Wrap(err, "failed to unmarshal response")
	}

	return response.Result.Value, nil
}
