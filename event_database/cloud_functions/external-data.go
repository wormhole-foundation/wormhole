package p

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"net/http"

	"github.com/certusone/wormhole/node/pkg/vaa"
)

const cgBaseUrl = "https://api.coingecko.com/api/v3/"
const cgProBaseUrl = "https://pro-api.coingecko.com/api/v3/"

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
type CoinGeckoErrorRes struct {
	Error string `json:"error"`
}

func fetchCoinGeckoCoins() map[string][]CoinGeckoCoin {
	defer timeTrack(time.Now(), "fetchCoinGeckoCoins")
	baseUrl := cgBaseUrl
	cgApiKey := os.Getenv("COINGECKO_API_KEY")
	if cgApiKey != "" {
		baseUrl = cgProBaseUrl
	}
	url := fmt.Sprintf("%vcoins/list", baseUrl)
	req, reqErr := http.NewRequest("GET", url, nil)
	if reqErr != nil {
		log.Fatalf("failed coins request, err: %v", reqErr)
	}

	if cgApiKey != "" {
		req.Header.Set("X-Cg-Pro-Api-Key", cgApiKey)
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
		log.Printf("fetchCoinGeckoCoins failed parsing body. err %v\n", parseErr)
	}
	var geckoCoins = map[string][]CoinGeckoCoin{}
	for _, coin := range parsed {
		symbol := strings.ToLower(coin.Symbol)
		geckoCoins[symbol] = append(geckoCoins[symbol], coin)
	}
	return geckoCoins

}

func chainIdToCoinGeckoPlatform(chain vaa.ChainID) string {
	switch chain {
	case vaa.ChainIDSolana:
		return "solana"
	case vaa.ChainIDEthereum:
		return "ethereum"
	case vaa.ChainIDTerra:
		return "terra"
	case vaa.ChainIDBSC:
		return "binance-smart-chain"
	case vaa.ChainIDPolygon:
		return "polygon-pos"
	case vaa.ChainIDAvalanche:
		return "avalanche"
	case vaa.ChainIDOasis:
		return "oasis"
	case vaa.ChainIDAlgorand:
		return "algorand"
	case vaa.ChainIDAurora:
		return "aurora"
	case vaa.ChainIDFantom:
		return "fantom"
	case vaa.ChainIDKarura:
		return "karura"
	case vaa.ChainIDAcala:
		return "acala"
	case vaa.ChainIDKlaytn:
		return "klay-token"
	case vaa.ChainIDCelo:
		return "celo"
	case vaa.ChainIDTerra2:
		return "" // TODO
	case vaa.ChainIDEthereumRopsten:
		return "ethereum"
	}
	return ""
}

func fetchCoinGeckoCoinFromContract(chainId vaa.ChainID, address string) CoinGeckoCoin {
	baseUrl := cgBaseUrl
	cgApiKey := os.Getenv("COINGECKO_API_KEY")
	if cgApiKey != "" {
		baseUrl = cgProBaseUrl
	}
	platform := chainIdToCoinGeckoPlatform(chainId)
	url := fmt.Sprintf("%vcoins/%v/contract/%v", baseUrl, platform, address)
	req, reqErr := http.NewRequest("GET", url, nil)
	if reqErr != nil {
		log.Fatalf("failed contract request, err: %v\n", reqErr)
	}
	if cgApiKey != "" {
		req.Header.Set("X-Cg-Pro-Api-Key", cgApiKey)
	}

	res, resErr := http.DefaultClient.Do(req)
	if resErr != nil {
		log.Fatalf("failed get contract response, err: %v\n", resErr)
	}

	defer res.Body.Close()
	body, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		log.Fatalf("failed decoding contract body, err: %v\n", bodyErr)
	}

	var parsed CoinGeckoCoin

	parseErr := json.Unmarshal(body, &parsed)
	if parseErr != nil {
		log.Printf("fetchCoinGeckoCoinFromContract failed parsing body. err %v\n", parseErr)
		var errRes CoinGeckoErrorRes
		if err := json.Unmarshal(body, &errRes); err == nil {
			if errRes.Error == "Could not find coin with the given id" {
				log.Printf("Could not find CoinGecko coin by contract address, for chain %v, address, %v\n", chainId, address)
			} else {
				log.Println("Failed calling CoinGecko, got err", errRes.Error)
			}
		}
	}

	return parsed
}

func fetchCoinGeckoCoinId(chainId vaa.ChainID, address, symbol, name string) (coinId, foundSymbol, foundName string) {
	// try coingecko, return if good
	// if coingecko does not work, try chain-specific options

	// initialize strings that will be returned if we find a symbol/name
	// when looking up this token by contract address
	newSymbol := ""
	newName := ""

	if symbol == "" && chainId == vaa.ChainIDSolana {
		// try to lookup the symbol in solana token list, from the address
		if token, ok := solanaTokens[address]; ok {
			symbol = token.Symbol
			name = token.Name
			newSymbol = token.Symbol
			newName = token.Name
		}
	}
	if _, ok := coinGeckoCoins[strings.ToLower(symbol)]; ok {
		tokens := coinGeckoCoins[strings.ToLower(symbol)]
		if len(tokens) == 1 {
			// only one match found for this symbol
			return tokens[0].Id, newSymbol, newName
		}
		for _, token := range tokens {
			if token.Name == name {
				// found token by name match
				return token.Id, newSymbol, newName
			}
			if strings.Contains(strings.ToLower(strings.ReplaceAll(name, " ", "")), strings.ReplaceAll(token.Id, "-", "")) {
				// found token by id match
				log.Println("found token by symbol and name match", name)
				return token.Id, newSymbol, newName
			}
		}
		// more than one symbol with this name, let contract lookup try
	}
	coin := fetchCoinGeckoCoinFromContract(chainId, address)
	if coin.Id != "" {
		return coin.Id, newSymbol, newName
	}
	// could not find a CoinGecko coin
	return "", newSymbol, newName
}

func fetchCoinGeckoPrice(coinId string, timestamp time.Time) (float64, error) {
	hourAgo := time.Now().Add(-time.Duration(1) * time.Hour)
	withinLastHour := timestamp.After(hourAgo)
	start, end := rangeFromTime(timestamp, 12)

	baseUrl := cgBaseUrl
	cgApiKey := os.Getenv("COINGECKO_API_KEY")
	if cgApiKey != "" {
		baseUrl = cgProBaseUrl
	}
	url := fmt.Sprintf("%vcoins/%v/market_chart/range?vs_currency=usd&from=%v&to=%v", baseUrl, coinId, start.Unix(), end.Unix())
	req, reqErr := http.NewRequest("GET", url, nil)
	if reqErr != nil {
		log.Fatalf("failed coins request, err: %v\n", reqErr)
	}
	if cgApiKey != "" {
		req.Header.Set("X-Cg-Pro-Api-Key", cgApiKey)
	}

	res, resErr := http.DefaultClient.Do(req)
	if resErr != nil {
		log.Fatalf("failed get coins response, err: %v\n", resErr)
	}
	if res.StatusCode >= 400 {
		log.Fatal("failed to get CoinGecko prices. Status", res.Status)
	}

	defer res.Body.Close()
	body, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		log.Fatalf("failed decoding coins body, err: %v\n", bodyErr)
	}

	var parsed CoinGeckoMarketRes

	parseErr := json.Unmarshal(body, &parsed)
	if parseErr != nil {
		log.Printf("fetchCoinGeckoPrice failed parsing body. err %v\n", parseErr)
		var errRes CoinGeckoErrorRes
		if err := json.Unmarshal(body, &errRes); err == nil {
			log.Println("Failed calling CoinGecko, got err", errRes.Error)
		}
	}
	if len(parsed.Prices) >= 1 {
		var priceIndex int
		if withinLastHour {
			// use the last price in the list, latest price
			priceIndex = len(parsed.Prices) - 1
		} else {
			// use a price from the middle of the list, as that should be
			// closest to the timestamp.
			numPrices := len(parsed.Prices)
			priceIndex = numPrices / 2
		}
		price := parsed.Prices[priceIndex][1]
		log.Printf("found a price of $%f for %v!\n", price, coinId)
		return price, nil
	}
	log.Println("no price found in coinGecko for", coinId)
	return 0, fmt.Errorf("no price found for %v", coinId)
}

type Price struct {
	USD float64 `json:"usd"`
}
type CoinGeckoCoinPrices map[string]Price

// takes a list of CoinGeckoCoinIds, returns a map of { coinId: price }.
func fetchCoinGeckoPrices(coinIds []string) (map[string]float64, error) {
	baseUrl := cgBaseUrl
	cgApiKey := os.Getenv("COINGECKO_API_KEY")
	if cgApiKey != "" {
		baseUrl = cgProBaseUrl
	}
	url := fmt.Sprintf("%vsimple/price?ids=%v&vs_currencies=usd", baseUrl, strings.Join(coinIds, ","))
	req, reqErr := http.NewRequest("GET", url, nil)
	if reqErr != nil {
		log.Fatalf("failed coins request, err: %v\n", reqErr)
	}
	if cgApiKey != "" {
		req.Header.Set("X-Cg-Pro-Api-Key", cgApiKey)
	}

	res, resErr := http.DefaultClient.Do(req)
	if resErr != nil {
		log.Fatalf("failed get coins response, err: %v\n", resErr)
	}
	if res.StatusCode >= 400 {
		log.Fatal("failed to get CoinGecko prices. Status", res.Status)
	}

	defer res.Body.Close()
	body, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		log.Fatalf("failed decoding coins body, err: %v\n", bodyErr)
	}

	var parsed CoinGeckoCoinPrices

	parseErr := json.Unmarshal(body, &parsed)
	if parseErr != nil {
		log.Printf("fetchCoinGeckoPrice failed parsing body. err %v\n", parseErr)
		var errRes CoinGeckoErrorRes
		if err := json.Unmarshal(body, &errRes); err == nil {
			log.Println("Failed calling CoinGecko, got err", errRes.Error)
		}
	}
	priceMap := map[string]float64{}
	for coinId, price := range parsed {
		price := price.USD
		priceMap[coinId] = price

	}
	return priceMap, nil
}

// takes a list of CoinGeckoCoinIds, returns a map of { coinId: price }.
// makes batches of requests to CoinGecko.
func fetchTokenPrices(ctx context.Context, coinIds []string) map[string]float64 {
	allPrices := map[string]float64{}

	// Split the list into batches, otherwise the request could be too large
	batch := 100

	for i := 0; i < len(coinIds); i += batch {
		j := i + batch
		if j > len(coinIds) {
			j = len(coinIds)
		}

		prices, err := fetchCoinGeckoPrices(coinIds[i:j])
		if err != nil {
			log.Fatalf("failed to get price for coinIds. err %v", err)
		}
		for coinId, price := range prices {
			allPrices[coinId] = price
		}

		// CoinGecko rate limit is low (5/second), be very cautious about bursty requests
		time.Sleep(time.Millisecond * 200)
	}

	return allPrices
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
	defer timeTrack(time.Now(), "fetchSolanaTokenList")

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
		log.Printf("fetchSolanaTokenList failed parsing body. err %v\n", parseErr)
	}
	var solTokens = map[string]SolanaToken{}
	for _, token := range parsed.Tokens {
		if _, ok := solTokens[token.Address]; !ok {
			solTokens[token.Address] = token
		}
	}
	return solTokens
}

const solanaBeachPublicBaseURL = "https://prod-api.solana.surf/v1/"
const solanaBeachPrivateBaseURL = "https://api.solanabeach.io/v1/"

type SolanaBeachAccountOwner struct {
	Owner SolanaBeachAccountOwnerAddress `json:"owner"`
}
type SolanaBeachAccountOwnerAddress struct {
	Address string `json:"address"`
}
type SolanaBeachAccountResponse struct {
	Value struct {
		Extended struct {
			SolanaBeachAccountOwner
		} `json:"extended"`
	} `json:"value"`
}

func fetchSolanaAccountOwner(account string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	baseUrl := solanaBeachPublicBaseURL

	sbApiKey := os.Getenv("SOLANABEACH_API_KEY")
	if sbApiKey != "" {
		baseUrl = solanaBeachPrivateBaseURL
	}

	url := fmt.Sprintf("%vaccount/%v", baseUrl, account)
	req, reqErr := http.NewRequestWithContext(ctx, "GET", url, nil)
	if reqErr != nil {
		log.Printf("failed solanabeach request, err: %v", reqErr)
		return ""
	}

	if sbApiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", sbApiKey))
	}

	res, resErr := http.DefaultClient.Do(req)
	if resErr != nil {
		log.Printf("failed get solana beach account response, err: %v", resErr)
		return ""
	}

	defer res.Body.Close()
	body, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		log.Printf("failed decoding solana beach account body, err: %v", bodyErr)
		return ""
	}

	var parsed SolanaBeachAccountResponse

	parseErr := json.Unmarshal(body, &parsed)
	if parseErr != nil {
		log.Printf("fetchSolanaAccountOwner failed parsing body. err %v\n", parseErr)
		return ""
	}
	address := parsed.Value.Extended.Owner.Address
	if address == "" {
		log.Println("failed to find owner address for Solana account", account)
	}
	return address
}
