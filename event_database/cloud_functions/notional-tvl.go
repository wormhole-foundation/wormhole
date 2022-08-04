// Package p contains an HTTP Cloud Function.
package p

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"cloud.google.com/go/bigtable"
)

type tvlResult struct {
	Last24HoursChange map[string]map[string]LockedAsset
	AllTime           map[string]map[string]LockedAsset
}

// an in-memory cache of previously calculated results
var warmTvlCache = map[string]map[string]map[string]LockedAsset{}
var muWarmTvlCache sync.RWMutex
var warmTvlFilePath = "tvl-cache.json"

var notionalTvlResultPath = "notional-tvl.json"

type LockedAsset struct {
	Symbol        string
	Name          string
	Address       string
	CoinGeckoId   string
	Amount        float64
	Notional      float64
	TokenPrice    float64
	TokenDecimals int
}

// finds the daily amount of each symbol transferred to each chain, from the specified start to the present.
func tvlInInterval(tbl *bigtable.Table, ctx context.Context, start time.Time) map[string]map[string]map[string]LockedAsset {
	if len(warmTvlCache) == 0 && loadCache {
		loadJsonToInterface(ctx, warmTvlFilePath, &muWarmTvlCache, &warmTvlCache)
	}

	results := map[string]map[string]map[string]LockedAsset{}

	now := time.Now().UTC()
	numPrevDays := int(now.Sub(start).Hours() / 24)

	var intervalsWG sync.WaitGroup
	// there will be a query for each previous day, plus today
	intervalsWG.Add(numPrevDays + 1)

	cacheNeedsUpdate := false

	for daysAgo := 0; daysAgo <= numPrevDays; daysAgo++ {
		go func(tbl *bigtable.Table, ctx context.Context, daysAgo int) {
			// start is the SOD, end is EOD
			// "0 daysAgo start" is 00:00:00 AM of the current day
			// "0 daysAgo end" is 23:59:59 of the current day (the future)

			// calculate the start and end times for the query
			hoursAgo := (24 * daysAgo)
			daysAgoDuration := -time.Duration(hoursAgo) * time.Hour
			n := now.Add(daysAgoDuration)
			year := n.Year()
			month := n.Month()
			day := n.Day()
			loc := n.Location()

			start := time.Date(year, month, day, 0, 0, 0, 0, loc)
			end := time.Date(year, month, day, 23, 59, 59, maxNano, loc)

			dateStr := start.Format("2006-01-02")

			muWarmTvlCache.Lock()
			// initialize the map for this date in the result set
			results[dateStr] = map[string]map[string]LockedAsset{}
			// check to see if there is cache data for this date/query
			if len(warmTvlCache) >= 1 {
				// have a cache, check if has the date

				if dateCache, ok := warmTvlCache[dateStr]; ok && len(dateCache) > 1 && useCache(dateStr) {
					// have a cache for this date
					if daysAgo >= 1 {
						// only use the cache for yesterday and older
						results[dateStr] = dateCache
						muWarmTvlCache.Unlock()
						intervalsWG.Done()
						return
					}
				}
			} else {
				// no cache for this query, initialize the map
				warmTvlCache = map[string]map[string]map[string]LockedAsset{}
			}
			muWarmTvlCache.Unlock()

			defer intervalsWG.Done()

			queryResult := fetchTransferRowsInInterval(tbl, ctx, "", start, end)

			// iterate through the rows and increment the count
			for _, row := range queryResult {
				if row.CoinGeckoCoinId == "" {
					log.Printf("skipping row without CoinGeckoCoinId. symbol: %v, amount %v", row.TokenSymbol, row.TokenAmount)
					continue
				}
				if row.TokenAddress == "" {
					// if the token address is missing, skip
					continue
				}

				if _, ok := results[dateStr][row.OriginChain]; !ok {
					results[dateStr][row.OriginChain] = map[string]LockedAsset{}
				}

				if _, ok := results[dateStr][row.OriginChain][row.TokenAddress]; !ok {
					results[dateStr][row.OriginChain][row.TokenAddress] = LockedAsset{
						Symbol:        row.TokenSymbol,
						Name:          row.TokenName,
						Address:       row.TokenAddress,
						CoinGeckoId:   row.CoinGeckoCoinId,
						TokenPrice:    row.TokenPrice,
						TokenDecimals: row.TokenDecimals,
						Amount:        0,
						Notional:      0,
					}
				}

				var amountChange float64
				amountChange = 0
				if row.OriginChain == row.LeavingChain {
					// this is a native asset leaving its chain:
					// add this to tokens of originChain
					amountChange = row.TokenAmount
				}
				if row.OriginChain == row.DestinationChain {
					// this is a native asset going back to its chain:
					// subtract this from tokens of originChain
					amountChange = row.TokenAmount * -1
				}

				if prevForChain, ok := results[dateStr][row.OriginChain][row.TokenAddress]; ok {
					prevForChain.Amount = prevForChain.Amount + amountChange
					results[dateStr][row.OriginChain][row.TokenAddress] = prevForChain
				}
			}
			if daysAgo >= 1 {
				// set the result in the cache
				muWarmTvlCache.Lock()
				if cacheData, ok := warmTvlCache[dateStr]; !ok || len(cacheData) <= 1 || !useCache(dateStr) {
					// cache does not have this date, persist it for other instances.
					warmTvlCache[dateStr] = results[dateStr]
					cacheNeedsUpdate = true
				}
				muWarmTvlCache.Unlock()
			}
		}(tbl, ctx, daysAgo)
	}

	intervalsWG.Wait()

	if cacheNeedsUpdate {
		persistInterfaceToJson(ctx, warmTvlFilePath, &muWarmTvlCache, warmTvlCache)
	}

	// create a set of all the keys from all dates/chains, to ensure the result objects all have the same chain keys
	seenChainSet := map[string]bool{}
	for _, chains := range results {
		for leaving := range chains {
			if _, ok := seenChainSet[leaving]; !ok {
				seenChainSet[leaving] = true
			}
		}
	}

	var muResult sync.RWMutex
	// ensure each chain object has all the same symbol keys:
	for date, chains := range results {
		// loop through seen chains
		for chain := range seenChainSet {
			// check that date has all the chains
			if _, ok := chains[chain]; !ok {
				muResult.Lock()
				results[date][chain] = map[string]LockedAsset{}
				muResult.Unlock()
			}
		}
	}

	return results
}

// adds dailyTotals to return a map with chainIds for keys, each value is a map of address/amount locked.
func tvlSinceDate(tbl *bigtable.Table, ctx context.Context, dailyTotals map[string]map[string]map[string]LockedAsset) map[string]map[string]LockedAsset {
	result := map[string]map[string]LockedAsset{}

	// loop through the query results to combine cache + fresh data
	for _, chains := range dailyTotals {
		for chain, tokens := range chains {
			// ensure the chain exists in the result map
			if _, ok := result[chain]; !ok {
				result[chain] = map[string]LockedAsset{}
			}
			for address, lockedAsset := range tokens {
				amount := lockedAsset.Amount
				if asset, ok := result[chain][address]; ok {
					// add the amount of this symbol transferred this day to the
					// amount already in the result (amount of this symbol previously transferred)
					asset.Amount = asset.Amount + amount
					result[chain][address] = asset
				} else {
					// have not seen this asset in previous days
					result[chain][address] = LockedAsset{
						Symbol:        lockedAsset.Symbol,
						Name:          lockedAsset.Name,
						Address:       lockedAsset.Address,
						CoinGeckoId:   lockedAsset.CoinGeckoId,
						Amount:        lockedAsset.Amount,
						TokenPrice:    lockedAsset.TokenPrice,
						TokenDecimals: lockedAsset.TokenDecimals,
					}
				}
			}
		}
	}

	return result
}

// returns the count of the rows in the query response
func tvlForInterval(tbl *bigtable.Table, ctx context.Context, start, end time.Time) map[string]map[string]LockedAsset {
	// query for all rows in time range, return result count
	queryResults := fetchTransferRowsInInterval(tbl, ctx, "", start, end)

	result := map[string]map[string]LockedAsset{}

	// iterate through the rows and increment the count for each index
	for _, row := range queryResults {
		if _, ok := result[row.OriginChain]; !ok {
			result[row.OriginChain] = map[string]LockedAsset{}
		}
		if row.TokenAddress == "" {
			// if the token address is missing, skip
			continue
		}
		if _, ok := result[row.OriginChain][row.TokenAddress]; !ok {
			result[row.OriginChain][row.TokenAddress] = LockedAsset{
				Symbol:      row.TokenSymbol,
				Name:        row.TokenName,
				Address:     row.TokenAddress,
				CoinGeckoId: row.CoinGeckoCoinId,
				Amount:      0,
				Notional:    0,
			}
		}

		var amountChange float64
		amountChange = 0
		// track notional changes for the previous 24 hour delta
		var notionalChange float64
		notionalChange = 0
		if row.OriginChain == row.LeavingChain {
			// this is a native asset leaving its chain:
			// add this to tvl of originChain
			amountChange = row.TokenAmount
			notionalChange = row.Notional
		}
		if row.OriginChain == row.DestinationChain {
			// this is a native asset going back to its chain:
			// subtract this from tvl of originChain
			amountChange = row.TokenAmount * -1
			notionalChange = row.Notional * -1
		}

		if prevForChain, ok := result[row.OriginChain][row.TokenAddress]; ok {
			prevForChain.Amount = prevForChain.Amount + amountChange
			prevForChain.Notional = prevForChain.Notional + notionalChange
			result[row.OriginChain][row.TokenAddress] = prevForChain
		}

		if prevAllChains, ok := result["*"][row.TokenAddress]; ok {
			prevAllChains.Amount = prevAllChains.Amount + amountChange
			prevAllChains.Notional = prevAllChains.Notional + notionalChange
			result["*"][row.TokenAddress] = prevAllChains
		}
	}
	return result
}

// calculates the value locked
func ComputeTVL(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for the preflight request
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// Set CORS headers for the main request.
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx := context.Background()

	now := time.Now().UTC()
	todaysDateStr := now.Format("2006-01-02")

	getNotionalAmounts := func(ctx context.Context, tokensLocked map[string]map[string]LockedAsset) map[string]map[string]LockedAsset {
		// create a map of all the coinIds
		seenCoinIds := map[string]bool{}
		for _, tokens := range tokensLocked {
			for _, lockedAsset := range tokens {
				coinId := lockedAsset.CoinGeckoId
				if coinId != "*" {
					seenCoinIds[coinId] = true
				}
			}
		}
		coinIdSet := []string{}
		for coinId := range seenCoinIds {
			coinIdSet = append(coinIdSet, coinId)
		}

		tokenPrices := fetchTokenPrices(ctx, coinIdSet)

		notionalLocked := map[string]map[string]LockedAsset{}

		// initialize the struct that will hold the total for all chains, all assets
		notionalLocked["*"] = map[string]LockedAsset{}
		notionalLocked["*"]["*"] = LockedAsset{
			Symbol:   "*",
			Name:     "all",
			Notional: 0,
		}
		for chain, tokens := range tokensLocked {
			notionalLocked[chain] = map[string]LockedAsset{}
			notionalLocked[chain]["*"] = LockedAsset{
				Symbol:  "all",
				Address: "*",
			}
			for address, lockedAsset := range tokens {
				if !isTokenActive(chain, address, todaysDateStr) {
					continue
				}

				coinId := lockedAsset.CoinGeckoId
				amount := lockedAsset.Amount
				if address != "*" {
					currentPrice := tokenPrices[coinId]
					notionalVal := amount * currentPrice
					if notionalVal <= 0 {
						continue
					}

					notionalLocked[chain][address] = LockedAsset{
						Symbol:        lockedAsset.Symbol,
						Name:          lockedAsset.Name,
						Address:       lockedAsset.Address,
						CoinGeckoId:   lockedAsset.CoinGeckoId,
						Amount:        lockedAsset.Amount,
						Notional:      roundToTwoDecimalPlaces(notionalVal),
						TokenPrice:    currentPrice,
						TokenDecimals: lockedAsset.TokenDecimals,
					}

					if asset, ok := notionalLocked[chain]["*"]; ok {
						asset.Notional = asset.Notional + notionalVal
						notionalLocked[chain]["*"] = asset
					}

				}
			}

			// add the chain total to the overall total
			if all, ok := notionalLocked["*"]["*"]; ok {
				all.Notional += notionalLocked[chain]["*"].Notional
				notionalLocked["*"]["*"] = all
			}

			// round the the amount for chain/*
			if asset, ok := notionalLocked[chain]["*"]; ok {
				asset.Notional = roundToTwoDecimalPlaces(asset.Notional)
				notionalLocked[chain]["*"] = asset
			}
		}
		return notionalLocked
	}

	var wg sync.WaitGroup

	// delta of last 24 hours
	last24HourDelta := map[string]map[string]LockedAsset{}
	wg.Add(1)
	go func() {
		last24HourInterval := -time.Duration(24) * time.Hour
		start := now.Add(last24HourInterval)
		defer wg.Done()
		transfers := tvlForInterval(tbl, ctx, start, now)
		last24HourDelta = getNotionalAmounts(ctx, transfers)
	}()

	// total since release
	allTimeLocked := map[string]map[string]LockedAsset{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		dailyTotalsAllTime := tvlInInterval(tbl, ctx, releaseDay)
		transfers := tvlSinceDate(tbl, ctx, dailyTotalsAllTime)
		allTimeLocked = getNotionalAmounts(ctx, transfers)
	}()

	wg.Wait()

	result := &tvlResult{
		Last24HoursChange: last24HourDelta,
		AllTime:           allTimeLocked,
	}

	persistInterfaceToJson(ctx, notionalTvlResultPath, &muWarmTvlCache, result)

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Println(err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func TVL(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for the preflight request
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// Set CORS headers for the main request.
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var last24Hours string

	// allow GET requests with querystring params, or POST requests with json body.
	switch r.Method {
	case http.MethodGet:
		queryParams := r.URL.Query()
		last24Hours = queryParams.Get("last24Hours")

	case http.MethodPost:
		// declare request body properties
		var d struct {
			Last24Hours string `json:"last24Hours"`
		}

		// deserialize request body
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			switch err {
			case io.EOF:
				// do nothing, empty body is ok
			default:
				log.Printf("json.NewDecoder: %v", err)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		}

		last24Hours = d.Last24Hours

	default:
		http.Error(w, "405 - Method Not Allowed", http.StatusMethodNotAllowed)
		log.Println("Method Not Allowed")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var cachedResult tvlResult
	loadJsonToInterface(ctx, notionalTvlResultPath, &muWarmTvlCache, &cachedResult)

	// delta of last 24 hours
	var last24HourDelta = map[string]map[string]LockedAsset{}
	if last24Hours != "" {
		last24HourDelta = cachedResult.Last24HoursChange
	}

	result := &tvlResult{
		Last24HoursChange: last24HourDelta,
		AllTime:           cachedResult.AllTime,
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Println(err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}
