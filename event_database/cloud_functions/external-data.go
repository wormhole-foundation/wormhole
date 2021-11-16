package p

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"net/http"
)

const cgBaseUrl = "https://api.coingecko.com/api/v3/"

type CoinGeckoCoin struct {
	Id     string `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}
type CoinGeckoCoins []CoinGeckoCoin

type CoinGeckoMarket [2]float64

type CoinGeckoMarketRes struct {
	Prices []CoinGeckoMarket `json:"prices"`
}

func fetchCoinGeckoCoins() map[string][]CoinGeckoCoin {
	url := fmt.Sprintf("%vcoins/list", cgBaseUrl)
	req, reqErr := http.NewRequest("GET", url, nil)
	if reqErr != nil {
		log.Fatalf("failed coins request, err: %v", reqErr)
	}

	res, resErr := http.DefaultClient.Do(req)
	if resErr != nil {
		log.Fatalf("failed get coins response, err: %v", resErr)
	}

	defer res.Body.Close()
	body, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		log.Fatalf("failed decoding coins body, err: %v", bodyErr)
	}

	var parsed []CoinGeckoCoin

	parseErr := json.Unmarshal(body, &parsed)
	if parseErr != nil {
		log.Printf("failed parsing body. err %v\n", parseErr)
	}
	var geckoCoins = map[string][]CoinGeckoCoin{}
	for _, coin := range parsed {
		geckoCoins[coin.Symbol] = append(geckoCoins[coin.Symbol], coin)
	}
	return geckoCoins

}

func fetchCoinGeckoPrice(coinId string, timestamp time.Time) (float64, error) {
	start, end := rangeFromTime(timestamp, 4)
	url := fmt.Sprintf("%vcoins/%v/market_chart/range?vs_currency=usd&from=%v&to=%v", cgBaseUrl, coinId, start.Unix(), end.Unix())
	req, reqErr := http.NewRequest("GET", url, nil)
	if reqErr != nil {
		log.Fatalf("failed coins request, err: %v\n", reqErr)
	}

	res, resErr := http.DefaultClient.Do(req)
	if resErr != nil {
		log.Fatalf("failed get coins response, err: %v\n", resErr)
	}

	defer res.Body.Close()
	body, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		log.Fatalf("failed decoding coins body, err: %v\n", bodyErr)
	}

	var parsed CoinGeckoMarketRes

	parseErr := json.Unmarshal(body, &parsed)
	if parseErr != nil {
		log.Printf("failed parsing body. err %v\n", parseErr)
	}
	if len(parsed.Prices) >= 1 {
		numPrices := len(parsed.Prices)
		middle := numPrices / 2
		// take the price in the middle of the range, as that should be
		// closest to the timestamp.
		price := parsed.Prices[middle][1]
		fmt.Printf("found a price for %v! %v\n", coinId, price)
		return price, nil
	}
	fmt.Println("no price found in coinGecko for", coinId)
	return 0, fmt.Errorf("no price found for %v", coinId)
}

const solanaTokenListURL = "https://raw.githubusercontent.com/solana-labs/token-list/main/src/tokens/solana.tokenlist.json"

type SolanaToken struct {
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Decimals int    `json:"decimals"`
}
type SolanaTokenListRes struct {
	Tokens []SolanaToken `json:"tokens"`
}

func fetchSolanaTokenList() map[string]SolanaToken {

	req, reqErr := http.NewRequest("GET", solanaTokenListURL, nil)
	if reqErr != nil {
		log.Fatalf("failed solana token list request, err: %v", reqErr)
	}

	res, resErr := http.DefaultClient.Do(req)
	if resErr != nil {
		log.Fatalf("failed get solana token list response, err: %v", resErr)
	}

	defer res.Body.Close()
	body, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		log.Fatalf("failed decoding solana token list body, err: %v", bodyErr)
	}

	var parsed SolanaTokenListRes

	parseErr := json.Unmarshal(body, &parsed)
	if parseErr != nil {
		log.Printf("failed parsing body. err %v\n", parseErr)
	}
	var solTokens = map[string]SolanaToken{}
	for _, token := range parsed.Tokens {
		if _, ok := solTokens[token.Address]; !ok {
			solTokens[token.Address] = token
		}
	}
	return solTokens
}
