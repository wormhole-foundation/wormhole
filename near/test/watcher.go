// http --verbose post http://localhost:3030  jsonrpc=2.0 id=dontcare method=block params:='{"block_id": 5004}'
// http --verbose post http://localhost:3030  jsonrpc=2.0 id=dontcare method=chunk params:='{"chunk_id": "AgVjJBCy5LBq9UBJuT1ZEhPJaY8DzUrpxbXGHHkqQkCb"}'
// http --verbose post http://localhost:3030  jsonrpc=2.0 id=dontcare method=EXPERIMENTAL_tx_status params:='["HZsEBFyo5fiRhApFx4SNm7Ao8anfdEiUyS3cVUf8riG1", "7144wormhole.test.near"]'

package main

import (
	"bytes"
	"fmt"
	"github.com/tidwall/gjson"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func getBlock(block uint64) ([]byte, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"block_id": %d}}`, block)
	resp, err := http.Post("http://localhost:3030", "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func getFinalBlock() ([]byte, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"finality": "final"}}`)
	resp, err := http.Post("http://localhost:3030", "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func getChunk(chunk string) ([]byte, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "chunk", "params": {"chunk_id": "%s"}}`, chunk)

	resp, err := http.Post("http://localhost:3030", "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func inspectBody(block uint64, body gjson.Result) error {
	fmt.Printf("block %d\n", block)

	result := body.Get("result.chunks.#.chunk_hash")
	for _, name := range result.Array() {
		chunk, err := getChunk(name.String())
		if (err != nil) {
			return err
		}
		receipts := gjson.ParseBytes(chunk).Get("result.receipts")
		for _, r := range receipts.Array() {
			p := r.Get("predecessor_id").String()
			if strings.HasSuffix(p, "wormhole.test.near") {
				a := r.Get("receipt.Action.actions.#.FunctionCall")
				for _, c := range a.Array() {
					if c.Get("method_name").String() == "message_published" {
						args := c.Get("args").String()
						rawDecodedText, err := base64.StdEncoding.DecodeString(args)
						if err != nil {
							return err
						}
						fmt.Printf("Decoded text: %s\n", rawDecodedText)
					}
				}
			}
		}
	}
	return nil
}

func main() {
	finalBody, _ := getFinalBlock()
	block := gjson.ParseBytes(finalBody).Get("result.chunks.0.height_created").Uint()

	for {
		finalBody, err := getFinalBlock()
		if err != nil {
			fmt.Printf(err.Error());
		} else {
			parsedFinalBody := gjson.ParseBytes(finalBody)
			lastBlock := parsedFinalBody.Get("result.chunks.0.height_created").Uint()

			for ; block <= lastBlock; block = block + 1 {
				if block == lastBlock {
					inspectBody(block, parsedFinalBody)
				} else {
					b, err := getBlock(block)
					if err != nil {
						fmt.Printf(err.Error());
						break
					} else {
						inspectBody(block, gjson.ParseBytes(b))
					}
				}
			}
		}
		time.Sleep(time.Second)
	}
}
