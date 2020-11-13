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

// BroadcastReq broadcast request body
type BroadcastReq struct {
	Tx   tx.StdTxData `json:"tx"`
	Mode string       `json:"mode"`
}

// TxResponse response
type TxResponse struct {
	Height msg.Int `json:"height"`
	TxHash string  `json:"txhash"`
	Code   uint32  `json:"code,omitempty"`
	RawLog string  `json:"raw_log,omitempty"`
}

// Broadcast - no-lint
func (LCDClient LCDClient) Broadcast(stdTx tx.StdTx) (TxResponse, error) {
	broadcastReq := BroadcastReq{
		Tx:   stdTx.Value,
		Mode: "sync",
	}

	reqBytes, err := json.Marshal(broadcastReq)
	if err != nil {
		return TxResponse{}, sdkerrors.Wrap(err, "failed to marshal")
	}

	resp, err := http.Post(LCDClient.URL+"/txs", "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return TxResponse{}, sdkerrors.Wrap(err, "failed to broadcast")
	}

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return TxResponse{}, sdkerrors.Wrap(err, "failed to read response")
	}

	if resp.StatusCode != 200 {
		return TxResponse{}, fmt.Errorf("non 200 respose code %d, error: %s", resp.StatusCode, string(out))
	}

	var txResponse TxResponse
	err = json.Unmarshal(out, &txResponse)
	if err != nil {
		return TxResponse{}, sdkerrors.Wrap(err, "failed to unmarshal response")
	}

	if txResponse.Code != 0 {
		return txResponse, fmt.Errorf("Tx failed code %d, error: %s", txResponse.Code, txResponse.RawLog)
	}

	return txResponse, nil
}
