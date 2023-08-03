package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"strconv"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const ACCOUNTANT_DUMP = "data/accountant.json"
const TOKEN_ALLOW_LIST = "data/token-allowlist.json"
const TOKEN_METADATA = "data/token_metadata.json"

type (
	// Layout of the config data for each token
	tokenConfigEntry struct {
		chain       uint16
		addr        string
		symbol      string
		coinGeckoId string
		decimals    int64
		price       float64
	}
)

func tokenList() []tokenConfigEntry {
	return append(manualTokenList(), generatedMainnetTokenList()...)
}

type Account struct {
	Key     AccountKey `json:"key"`
	Balance string     `json:"balance"`
}

type AccountKey struct {
	ChainId      int    `json:"chain_id"`
	TokenChain   int    `json:"token_chain"`
	TokenAddress string `json:"token_address"`
}

type TokenEntry struct {
	price          float64
	decimalDivisor *big.Int
	symbol         string
	coinGeckoId    string
}

// TokenAllowList maps chainId -> (nativeAddress -> coingeckoId)
type TokenAllowList map[string]map[string]string

type TokenMetadata []TokenMetadataEntry
type TokenMetadataEntry struct {
	TokenChain    string `json:"token_chain"`
	TokenAddress  string `json:"token_address"`
	NativeAddress string `json:"native_address"`
}

// NativeAddressLookupTable maps strings "chainId/WhTokenAddress" -> string nativeAddress
type NativeAddressLookupTable map[string]string

type TokenFilter struct {
	nativeAddressTbl NativeAddressLookupTable
	tal              TokenAllowList
}

func NewTokenFilter() *TokenFilter {
	// Read Token Allow List

	jsonFileTAL, err := os.Open(TOKEN_ALLOW_LIST)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	byteValueTAL, _ := ioutil.ReadAll(jsonFileTAL)
	var tal = TokenAllowList{}
	json.Unmarshal(byteValueTAL, &tal)

	// Read Token MetaData

	jsonFileTM, err := os.Open(TOKEN_METADATA)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	byteValueTM, _ := ioutil.ReadAll(jsonFileTM)
	var tm = TokenMetadata{}
	var nativeToken = NativeAddressLookupTable{}
	json.Unmarshal(byteValueTM, &tm)

	for _, tme := range tm {
		tokenStr := fmt.Sprintf("%s/%s", tme.TokenChain, tme.TokenAddress)
		//fmt.Printf("%s: %s\n", tokenStr, tme.NativeAddress)
		nativeToken[tokenStr] = tme.NativeAddress
	}

	return &TokenFilter{
		nativeAddressTbl: nativeToken,
		tal:              tal,
	}
}

func (f *TokenFilter) isAllowed(chainId int, whTokenAddress string) bool {
	// find nativeAddress
	tokenStr := fmt.Sprintf("%d/%s", chainId, whTokenAddress)
	nativeAddress, ok := f.nativeAddressTbl[tokenStr]
	if !ok {
		return false
	}

	fmt.Sprintf("Native address for tokenStr: %s\n", nativeAddress)

	ce, ok := f.tal[strconv.Itoa(chainId)]
	if !ok {
		return false
	}

	_, ok = ce[nativeAddress]

	if !ok {
		return false
	}
	return true
}

func main() {
	tokenList := tokenList()

	parsedTokenList := make(map[string]TokenEntry) // Maps ChainID/TokenAddress to floor_value

	for _, t := range tokenList {
		// wormhole supports a maximum of 8 decimals
		if t.decimals > 8 {
			t.decimals = 8
		}
		decimalsFloat := big.NewFloat(math.Pow(10.0, float64(t.decimals)))
		decimals, _ := decimalsFloat.Int(nil)

		parsedTokenList[fmt.Sprintf("%d/%s", t.chain, t.addr)] = TokenEntry{
			price:          t.price,
			decimalDivisor: decimals,
			symbol:         t.symbol,
			coinGeckoId:    t.coinGeckoId,
		}
	}

	tokenFilter := NewTokenFilter()

	// Read Accountant dump

	jsonFile, err := os.Open(ACCOUNTANT_DUMP)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	byteValue, _ := ioutil.ReadAll(jsonFile)
	var accounts []Account
	json.Unmarshal(byteValue, &accounts)

	var tvmByChain = make(map[int]int64)
	var tvlByChain = make(map[int]int64)

	for _, a := range accounts {
		if a.Balance == "0" {
			continue
		}

		// process entry

		if _, ok := tvmByChain[a.Key.ChainId]; !ok {
			tvmByChain[a.Key.ChainId] = 0
		}

		te, ok := parsedTokenList[fmt.Sprintf("%d/%s", a.Key.TokenChain, a.Key.TokenAddress)]
		if !ok {
			continue
		}

		// check if in token allow list (TAL)
		if !tokenFilter.isAllowed(a.Key.TokenChain, a.Key.TokenAddress) {
			continue
		}

		var balanceI big.Int
		_, ok = balanceI.SetString(a.Balance, 10)
		if !ok {
			panic(a.Balance)
		}

		balanceI.Div(&balanceI, te.decimalDivisor)

		var balance big.Float
		balance.SetInt(&balanceI)

		notional := balance.Mul(&balance, big.NewFloat(te.price))

		notionalI, _ := notional.Int(nil)

		if notionalI.Cmp(big.NewInt(math.MaxInt64)) == 1 {
			panic("integer overflow")
		}

		notionalInt := notionalI.Int64()

		if notionalInt > 100_000_000 {
			fmt.Printf("Token %s (%s) from chain %d has balance %s (%d) on chain %d and notional value %d\n", te.symbol, a.Key.TokenAddress, a.Key.TokenChain, a.Balance, notionalInt, a.Key.ChainId, notionalInt)
		}

		if a.Key.ChainId == a.Key.TokenChain {
			// TVL
			if tvlByChain[a.Key.ChainId] > math.MaxInt64-notionalInt {
				panic("integer overflow 2")
			}
			tvlByChain[a.Key.ChainId] += notionalInt
		} else {
			// TVM
			if tvmByChain[a.Key.ChainId] > math.MaxInt64-notionalInt {
				panic("integer overflow 3")
			}
			tvmByChain[a.Key.ChainId] += notionalInt
		}
	}

	tvl := int64(0)
	tvm := int64(0)
	for chainId := range tvmByChain {
		tvl += tvlByChain[chainId]
		tvm += tvmByChain[chainId]

		p := message.NewPrinter(language.English)
		fmt.Printf("%d, %s|%s\n", chainId, p.Sprintf("%d", tvmByChain[chainId]), p.Sprintf("%d", tvlByChain[chainId]))
	}
	fmt.Printf("\nTVM: %d\n", tvm)
	fmt.Printf("\nTVL: %d\n", tvl)
}
