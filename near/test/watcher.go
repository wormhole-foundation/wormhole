// http --verbose post http://localhost:3030  jsonrpc=2.0 id=dontcare method=block params:='{"block_id": 5004}'
// http --verbose post http://localhost:3030  jsonrpc=2.0 id=dontcare method=chunk params:='{"chunk_id": "AgVjJBCy5LBq9UBJuT1ZEhPJaY8DzUrpxbXGHHkqQkCb"}'
// http --verbose post http://localhost:3030  jsonrpc=2.0 id=dontcare method=EXPERIMENTAL_tx_status params:='["HZsEBFyo5fiRhApFx4SNm7Ao8anfdEiUyS3cVUf8riG1", "7144wormhole.test.near"]'

package main

import (
	"bytes"
	"fmt"
	"github.com/tidwall/gjson"
	//	"encoding/base64"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	//	"encoding/json"
)

func getTxStatus(tx string, src string) ([]byte, error) {
	s := fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "EXPERIMENTAL_tx_status", "params": ["%s", "%s"]}`, tx, src)
	fmt.Printf("%s\n", s)

	resp, err := http.Post("http://localhost:3030", "application/json", bytes.NewBuffer([]byte(s)))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

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
	if !result.Exists() {
		return nil
	}

	for _, name := range result.Array() {
		chunk, err := getChunk(name.String())
		if err != nil {
			return err
		}

		txns := gjson.ParseBytes(chunk).Get("result.transactions")
		if !txns.Exists() {
			continue
		}
		for _, r := range txns.Array() {
			hash := r.Get("hash")
			receiver_id := r.Get("receiver_id")
			if !hash.Exists() || !receiver_id.Exists() {
				continue
			}

			t, _ := getTxStatus(hash.String(), receiver_id.String())
			fmt.Printf("outcome:  %s\n", t)

			outcomes := gjson.ParseBytes(t).Get("result.receipts_outcome")

			if !outcomes.Exists() {
				continue
			}

			for _, o := range outcomes.Array() {
				outcome := o.Get("outcome")
				if !outcome.Exists() {
					continue
				}

				executor_id := outcome.Get("executor_id")
				if !executor_id.Exists() {
					continue
				}
				if executor_id.String() == "wormhole.test.near" {
					l := outcome.Get("logs")
					if !l.Exists() {
						continue
					}
					for _, log := range l.Array() {
						event := log.String()
						if !strings.HasPrefix(event, "EVENT_JSON:") {
							continue
						}
//						event_json := gjson.ParseBytes(event[11:])
						fmt.Printf("log: %s\n", event[11:])
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
			fmt.Printf(err.Error())
		} else {
			parsedFinalBody := gjson.ParseBytes(finalBody)
			lastBlock := parsedFinalBody.Get("result.chunks.0.height_created").Uint()

			for ; block <= lastBlock; block = block + 1 {
				if block == lastBlock {
					inspectBody(block, parsedFinalBody)
				} else {
					b, err := getBlock(block)
					//					fmt.Printf("block: %s\n", b);
					if err != nil {
						fmt.Printf(err.Error())
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
