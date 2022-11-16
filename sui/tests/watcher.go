package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"golang.org/x/net/websocket"
	"log"
	"os"
	"time"

	eth_common "github.com/ethereum/go-ethereum/common"
)

type SuiResult struct {
	Timestamp int64  `json:"timestamp"`
	TxDigest  string `json:"txDigest"`
	Event     struct {
		MoveEvent struct {
			PackageID         string `json:"packageId"`
			TransactionModule string `json:"transactionModule"`
			Sender            string `json:"sender"`
			Type              string `json:"type"`
			Fields            *struct {
				ConsistencyLevel uint8  `json:"consistency_level"`
				Nonce            uint64 `json:"nonce"`
				Payload          string `json:"payload"`
				Sender           uint64 `json:"sender"`
				Sequence         uint64 `json:"sequence"`
			} `json:"fields"`
			Bcs string `json:"bcs"`
		} `json:"moveEvent"`
	} `json:"event"`
}

type SuiEventMsg struct {
	Jsonrpc string  `json:"jsonrpc"`
	Method  *string `json:"method"`
	ID      *int    `json:"id"`
	result  *int    `json:"result"`
	Params  *struct {
		Subscription int64      `json:"subscription"`
		Result       *SuiResult `json:"result"`
	} `json:"params"`
}

type SuiTxnQuery struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Data       []SuiResult `json:"data"`
		NextCursor interface{} `json:"nextCursor"`
	} `json:"result"`
	ID int `json:"id"`
}

func inspectBody(body gjson.Result) error {
	txDigest := body.Get("txDigest")
	timestamp := body.Get("timestamp")
	packageId := body.Get("event.moveEvent.packageId") // defense in depth: check this
	account := body.Get("event.moveEvent.sender")      // defense in depth: check this
	consistency_level := body.Get("event.moveEvent.fields.consistency_level")
	nonce := body.Get("event.moveEvent.fields.nonce")
	payload := body.Get("event.moveEvent.fields.payload")
	sender := body.Get("event.moveEvent.fields.sender")
	sequence := body.Get("event.moveEvent.fields.sequence")

	if !txDigest.Exists() || !timestamp.Exists() || !packageId.Exists() || !account.Exists() || !consistency_level.Exists() || !nonce.Exists() || !payload.Exists() || !sender.Exists() || !sequence.Exists() {
		return errors.New("block parse error")
	}

	id, err := base64.StdEncoding.DecodeString(txDigest.String())
	if err != nil {
		fmt.Printf("txDigest decode error:  %s\n", txDigest.String())
		return err
	}

	var txHash = eth_common.BytesToHash(id) // 32 bytes = d3b136a6a182a40554b2fafbc8d12a7a22737c10c81e33b33d1dcb74c532708b
	fmt.Printf("\ntxHash: %s\n", txHash)

	pl, err := base64.StdEncoding.DecodeString(payload.String())
	if err != nil {
		fmt.Printf("payload decode error\n")
		return err
	}
	fmt.Printf("\npl: %s\n", pl)

	return nil
}

func main() {
	origin := "http://localhost/"
	url := "ws://localhost:9001"
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	s := fmt.Sprintf(`{"jsonrpc":"2.0", "id": 1, "method": "sui_subscribeEvent", "params": [{"Package": "%s"}]}`, os.Getenv("WORM_PACKAGE"))
	fmt.Printf("Sending: %s.\n", s)

	if _, err := ws.Write([]byte(s)); err != nil {
		log.Fatal(err)
	}
	for {
		var msg = make([]byte, 4096)
		var n int
		ws.SetReadDeadline(time.Now().Local().Add(1_000_000_000))
		if n, err = ws.Read(msg); err != nil {
			fmt.Printf("err")
		} else {
			fmt.Printf("\nReceived: %s.\n", msg[:n])
			parsedMsg := gjson.ParseBytes(msg[:n])

			var res SuiEventMsg
			err = json.Unmarshal(msg[:n], &res)
			if err != nil {
				fmt.Printf("SuiEventMsg: %s", err.Error())
			}

			if res.Method != nil {
				fmt.Printf("%s\n", *res.Method)
			} else {
				fmt.Printf("Method nil\n")
			}

			if res.ID != nil {
				fmt.Printf("%d\n", *res.ID)
			} else {
				fmt.Printf("ID nil\n")
			}

			result := parsedMsg.Get("params.result")
			if !result.Exists() {
				// Other messages come through on the channel.. we can ignore them safely
				continue
			}
			fmt.Printf("inspect body called\n")

			err := inspectBody(result)
			if err != nil {
				fmt.Printf("inspectBody: %s", err.Error())
			}
		}
	}
}
