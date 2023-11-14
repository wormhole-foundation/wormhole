// Usage:
// 		go run guardian_vaa_stats.go
// 		go run guardian_vaa_stats.go 2023-06-23 05:10:00 +0000 UTC
// This tool looks at the last 10 VAAs on each chain and tells you how many signatures each guardian has among those 10 VAAs.
// Wormscan is used as the datasource.
// If a timestamp is provided as argument, only VAAs after this timestamp are considered.
// For example, an output of
// 	solana:	map[0:7 1:10 2:10 3:5 4:10 6:10 9:7 10:10 11:5 13:9 14:10 15:8 16:10 17:10 18:10]
// means that on Solana, the Guardian with index 0 had its signature included in 7 out of 10 VAAs.
// Because Guardians usually do not collect more than 13 signatures on one VAA,
// it is expected that not every Guardian has a signature in every VAA and low participation rate is not necessarily indicative of a problem.

package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func getValidatorIndexForChain(chainId vaa.ChainID, onlyafter time.Time) (map[uint8]int, error) {
	url := fmt.Sprintf("https://api.wormholescan.io/api/v1/vaas/%d?page=1&pageSize=10&sortOrder=DESC", chainId)
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil) //nolint

	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "*/*")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	guardianParticipation := make(map[uint8]int, 19)

	vaaValues := gjson.GetBytes(body, "data.#.vaa")

	for _, vaaValue := range vaaValues.Array() {
		vaaBytes, err := base64.StdEncoding.DecodeString(vaaValue.String())
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		v, err := vaa.Unmarshal(vaaBytes)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}

		if v.Timestamp.Before(onlyafter) {
			continue
		}

		for _, sig := range v.Signatures {
			guardianParticipation[sig.Index] += 1
		}
	}

	return guardianParticipation, nil
}

func main() {
	var onlyafter time.Time = time.Unix(0, 0)
	if len(os.Args) > 1 {
		var err error
		onlyafter, err = time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", strings.Join(os.Args[1:], " "))
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Printf("Only looking for VAAs after %s\n\n", onlyafter)

	cids := vaa.GetAllNetworkIDs()
	for _, cid := range cids {
		gp, err := getValidatorIndexForChain(cid, onlyafter)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("%s\t%v\n", cid, gp)
	}
}
