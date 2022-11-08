package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"golang.org/x/net/websocket"
	"log"
	"os"
	"time"

	eth_common "github.com/ethereum/go-ethereum/common"
)

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
			parsedMsg := gjson.ParseBytes(msg)

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
