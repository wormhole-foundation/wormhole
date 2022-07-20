// Package p contains an HTTP Cloud Function.
package p

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"sort"
	"strconv"
	"sync"
	"time"

	"cloud.google.com/go/bigtable"
)

type tvlCumulativeResult struct {
	DailyLocked map[string]map[string]map[string]LockedAsset
}

// an in-memory cache of previously calculated results
var warmTvlCumulativeCache = map[string]map[string]map[string]LockedAsset{}
var muWarmTvlCumulativeCache sync.RWMutex
var warmTvlCumulativeCacheFilePath = "tvl-cumulative-cache.json"

var notionalTvlCumulativeResultPath = "notional-tvl-cumulative.json"

var coinGeckoPriceCacheFilePath = "coingecko-price-cache.json"
var coinGeckoPriceCache = map[string]map[string]float64{}
var loadedCoinGeckoPriceCache bool

// days to be excluded from the TVL result
var skipDays = map[string]bool{
	// for example:
	// "2022-02-19": true,
}

func loadAndUpdateCoinGeckoPriceCache(ctx context.Context, coinIds []string, now time.Time) {
	// at cold-start, load the price cache into memory, and fetch any missing token price histories and add them to the cache
	if !loadedCoinGeckoPriceCache {
		// load the price cache
		if loadCache {
			loadJsonToInterface(ctx, coinGeckoPriceCacheFilePath, &muWarmTvlCumulativeCache, &coinGeckoPriceCache)
			loadedCoinGeckoPriceCache = true
		}

		// find tokens missing price history
		missing := []string{}
		for _, coinId := range coinIds {
			found := false
			for _, prices := range coinGeckoPriceCache {
				if _, ok := prices[coinId]; ok {
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, coinId)
			}
		}

		// fetch missing price histories and add them to the cache
		priceHistories := fetchTokenPriceHistories(ctx, missing, releaseDay, now)
		for date, prices := range priceHistories {
			for coinId, price := range prices {
				if _, ok := coinGeckoPriceCache[date]; !ok {
					coinGeckoPriceCache[date] = map[string]float64{}
				}
				coinGeckoPriceCache[date][coinId] = price
			}
		}
	}

	// fetch today's latest prices
	today := now.Format("2006-01-02")
	coinGeckoPriceCache[today] = fetchTokenPrices(ctx, coinIds)

	// write to the cache file
	persistInterfaceToJson(ctx, coinGeckoPriceCacheFilePath, &muWarmCumulativeAddressesCache, coinGeckoPriceCache)
}

// calculates a running total of notional value transferred, by symbol, since the start time specified.
func createTvlCumulativeOfInterval(tbl *bigtable.Table, ctx context.Context, start time.Time) map[string]map[string]map[string]LockedAsset {
	if len(warmTvlCumulativeCache) == 0 && loadCache {
		loadJsonToInterface(ctx, warmTvlCumulativeCacheFilePath, &muWarmTvlCumulativeCache, &warmTvlCumulativeCache)
	}

	now := time.Now().UTC()
	today := now.Format("2006-01-02")

	cacheNeedsUpdate := false
	muWarmTvlCumulativeCache.Lock()
	if len(warmTvlCumulativeCache) == 0 {
		warmTvlCumulativeCache = map[string]map[string]map[string]LockedAsset{}
	}
	muWarmTvlCumulativeCache.Unlock()

	results := map[string]map[string]map[string]LockedAsset{}

	// fetch the amounts of transfers by symbol, for each day since launch (releaseDay)
	dailyAmounts := tvlInInterval(tbl, ctx, releaseDay)

	// create a slice of dates, order oldest first
	dateKeys := make([]string, 0, len(dailyAmounts))
	for k := range dailyAmounts {
		dateKeys = append(dateKeys, k)
	}
	sort.Strings(dateKeys)

	// iterate through the dates in the result set, and accumulate the amounts
	// of each token transfer by symbol, based on the destination of the transfer.
	for i, date := range dateKeys {
		results[date] = map[string]map[string]LockedAsset{"*": {"*": LockedAsset{}}}
		muWarmTvlCumulativeCache.RLock()
		if dateCache, ok := warmTvlCumulativeCache[date]; ok && useCache(date) && dateCache != nil {
			// have a cached value for this day, use it.
			// iterate through cache and copy values to the result
			for chain, tokens := range dateCache {
				results[date][chain] = map[string]LockedAsset{}
				for token, lockedAsset := range tokens {
					results[date][chain][token] = LockedAsset{
						Symbol:      lockedAsset.Symbol,
						Name:        lockedAsset.Name,
						Address:     lockedAsset.Address,
						CoinGeckoId: lockedAsset.CoinGeckoId,
						TokenPrice:  lockedAsset.TokenPrice,
						Amount:      lockedAsset.Amount,
						Notional:    lockedAsset.Notional,
					}
				}
			}
			muWarmTvlCumulativeCache.RUnlock()
		} else {
			// no cached value for this day, must calculate it
			muWarmTvlCumulativeCache.RUnlock()
			if i == 0 {
				// special case for first day, no need to sum.
				for chain, tokens := range dailyAmounts[date] {
					results[date][chain] = map[string]LockedAsset{}
					for token, lockedAsset := range tokens {
						results[date][chain][token] = LockedAsset{
							Symbol:      lockedAsset.Symbol,
							Name:        lockedAsset.Name,
							Address:     lockedAsset.Address,
							CoinGeckoId: lockedAsset.CoinGeckoId,
							TokenPrice:  lockedAsset.TokenPrice,
							Amount:      lockedAsset.Amount,
							Notional:    lockedAsset.Notional,
						}
					}
				}
			} else {
				// find the string of the previous day
				prevDate := dateKeys[i-1]
				prevDayAmounts := results[prevDate]
				thisDayAmounts := dailyAmounts[date]
				// iterate through all the transfers and add the previous day's amount, if it exists
				for chain, thisDaySymbols := range thisDayAmounts {
					// create a union of the symbols from this day, and previous days
					symbolsUnion := map[string]string{}
					for symbol := range prevDayAmounts[chain] {
						symbolsUnion[symbol] = symbol
					}
					for symbol := range thisDaySymbols {
						symbolsUnion[symbol] = symbol
					}
					// initialize the chain/symbol map for this date
					if _, ok := results[date][chain]; !ok {
						results[date][chain] = map[string]LockedAsset{}
					}
					// iterate through the union of symbols, creating an amount for each one,
					// and adding it the the results.
					for symbol := range symbolsUnion {

						asset := LockedAsset{}

						prevDayAmount := float64(0)
						if lockedAsset, ok := results[prevDate][chain][symbol]; ok {
							prevDayAmount = lockedAsset.Amount
							asset = lockedAsset
						}

						thisDayAmount := float64(0)
						if lockedAsset, ok := thisDaySymbols[symbol]; ok {
							thisDayAmount = lockedAsset.Amount
							// use today's locked asset, rather than prevDay's, for freshest price.
							asset = lockedAsset
						}
						cumulativeAmount := prevDayAmount + thisDayAmount

						results[date][chain][symbol] = LockedAsset{
							Symbol:      asset.Symbol,
							Name:        asset.Name,
							Address:     asset.Address,
							CoinGeckoId: asset.CoinGeckoId,
							TokenPrice:  asset.TokenPrice,
							Amount:      cumulativeAmount,
						}
					}
				}
			}
			// don't cache today
			if date != today {
				// set the result in the cache
				muWarmTvlCumulativeCache.Lock()
				if _, ok := warmTvlCumulativeCache[date]; !ok || !useCache(date) {
					// cache does not have this date, persist it for other instances.
					warmTvlCumulativeCache[date] = map[string]map[string]LockedAsset{}
					for chain, tokens := range results[date] {
						warmTvlCumulativeCache[date][chain] = map[string]LockedAsset{}
						for token, asset := range tokens {
							warmTvlCumulativeCache[date][chain][token] = LockedAsset{
								Symbol:      asset.Symbol,
								Name:        asset.Name,
								Address:     asset.Address,
								CoinGeckoId: asset.CoinGeckoId,
								TokenPrice:  asset.TokenPrice,
								Amount:      asset.Amount,
							}
						}
					}
					cacheNeedsUpdate = true
				}
				muWarmTvlCumulativeCache.Unlock()

			}
		}
	}

	if cacheNeedsUpdate {
		persistInterfaceToJson(ctx, warmTvlCumulativeCacheFilePath, &muWarmTvlCumulativeCache, warmTvlCumulativeCache)
	}

	// take the most recent n days, rather than returning all days since launch
	selectDays := map[string]map[string]map[string]LockedAsset{}
	days := getDaysInRange(start, now)
	for _, day := range days {
		selectDays[day] = map[string]map[string]LockedAsset{}
		for chain, tokens := range results[day] {
			selectDays[day][chain] = map[string]LockedAsset{}
			for symbol, asset := range tokens {
				selectDays[day][chain][symbol] = LockedAsset{
					Symbol:      asset.Symbol,
					Name:        asset.Name,
					Address:     asset.Address,
					CoinGeckoId: asset.CoinGeckoId,
					TokenPrice:  asset.TokenPrice,
					Amount:      asset.Amount,
				}
			}
		}
	}
	return selectDays

}

// calculates the cumulative value transferred each day since launch.
func ComputeTvlCumulative(w http.ResponseWriter, r *http.Request) {
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

	// days since launch day
	queryDays := int(time.Now().UTC().Sub(releaseDay).Hours() / 24)

	ctx := context.Background()

	dailyTvl := map[string]map[string]map[string]LockedAsset{}

	hours := (24 * queryDays)
	periodInterval := -time.Duration(hours) * time.Hour
	now := time.Now().UTC()
	prev := now.Add(periodInterval)
	start := time.Date(prev.Year(), prev.Month(), prev.Day(), 0, 0, 0, 0, prev.Location())

	transfers := createTvlCumulativeOfInterval(tbl, ctx, start)

	coinIdSet := map[string]bool{}
	for _, chains := range transfers {
		for _, assets := range chains {
			for _, asset := range assets {
				if asset.CoinGeckoId != "*" {
					coinIdSet[asset.CoinGeckoId] = true
				}
			}
		}
	}
	coinIds := []string{}
	for coinId := range coinIdSet {
		coinIds = append(coinIds, coinId)
	}
	loadAndUpdateCoinGeckoPriceCache(ctx, coinIds, now)

	// calculate the notional tvl based on the price of the tokens each day
	for date, chains := range transfers {
		if _, ok := skipDays[date]; ok {
			log.Println("skipping ", date)
			continue
		}
		dailyTvl[date] = map[string]map[string]LockedAsset{}
		dailyTvl[date]["*"] = map[string]LockedAsset{}
		dailyTvl[date]["*"]["*"] = LockedAsset{
			Symbol:   "*",
			Notional: 0,
		}
		for chain, tokens := range chains {
			if chain == "*" {
				continue
			}
			dailyTvl[date][chain] = map[string]LockedAsset{}
			dailyTvl[date][chain]["*"] = LockedAsset{
				Symbol:   "*",
				Notional: 0,
			}

			for symbol, asset := range tokens {
				if symbol == "*" {
					continue
				}

				// asset.TokenPrice is the price that was fetched when this token was last transferred, possibly before this date
				// prefer to use the cached price for this date if it's available, because it might be newer
				tokenPrice := asset.TokenPrice
				if prices, ok := coinGeckoPriceCache[date]; ok {
					if price, ok := prices[asset.CoinGeckoId]; ok {
						// use the cached price
						tokenPrice = price
					}
				}
				notional := asset.Amount * tokenPrice
				if notional <= 0 {
					continue
				}

				asset.Notional = roundToTwoDecimalPlaces(notional)

				// Note: disable individual symbols to reduce response size for now
				//// create a new LockAsset in order to exclude TokenPrice and Amount
				//dailyTvl[date][chain][symbol] = LockedAsset{
				//	Symbol:      asset.Symbol,
				//	Address:     asset.Address,
				//	CoinGeckoId: asset.CoinGeckoId,
				//	Notional:    asset.Notional,
				//}

				// add this asset's notional to the date/chain/*
				if allAssets, ok := dailyTvl[date][chain]["*"]; ok {
					allAssets.Notional += notional
					dailyTvl[date][chain]["*"] = allAssets
				}
			} // end symbols iteration

			// add chain total to the daily total
			if allAssets, ok := dailyTvl[date]["*"]["*"]; ok {
				allAssets.Notional += dailyTvl[date][chain]["*"].Notional
				dailyTvl[date]["*"]["*"] = allAssets
			}

			// round the day's chain total
			if allAssets, ok := dailyTvl[date][chain]["*"]; ok {
				allAssets.Notional = roundToTwoDecimalPlaces(allAssets.Notional)
				dailyTvl[date][chain]["*"] = allAssets
			}
		} // end chains iteration

		// round the daily total
		if allAssets, ok := dailyTvl[date]["*"]["*"]; ok {
			allAssets.Notional = roundToTwoDecimalPlaces(allAssets.Notional)
			dailyTvl[date]["*"]["*"] = allAssets
		}
	}

	result := &tvlCumulativeResult{
		DailyLocked: dailyTvl,
	}

	persistInterfaceToJson(ctx, notionalTvlCumulativeResultPath, &muWarmTvlCumulativeCache, result)

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

func TvlCumulative(w http.ResponseWriter, r *http.Request) {
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

	var numDays string
	var totalsOnly string
	switch r.Method {
	case http.MethodGet:
		queryParams := r.URL.Query()
		numDays = queryParams.Get("numDays")
		totalsOnly = queryParams.Get("totalsOnly")
	}

	var queryDays int
	if numDays == "" {
		// days since launch day
		queryDays = int(time.Now().UTC().Sub(releaseDay).Hours() / 24)
	} else {
		var convErr error
		queryDays, convErr = strconv.Atoi(numDays)
		if convErr != nil {
			fmt.Fprint(w, "numDays must be an integer")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	hours := (24 * queryDays)
	periodInterval := -time.Duration(hours) * time.Hour
	now := time.Now().UTC()
	prev := now.Add(periodInterval)
	start := time.Date(prev.Year(), prev.Month(), prev.Day(), 0, 0, 0, 0, prev.Location())
	startStr := start.Format("2006-01-02")

	var cachedResult tvlCumulativeResult
	loadJsonToInterface(ctx, notionalTvlCumulativeResultPath, &muWarmTvlCumulativeCache, &cachedResult)

	dailyLocked := map[string]map[string]map[string]LockedAsset{}
	for date, chains := range cachedResult.DailyLocked {
		if date >= startStr {
			if totalsOnly == "" {
				dailyLocked[date] = chains
			} else {
				dailyLocked[date] = map[string]map[string]LockedAsset{}
				for chain, addresses := range chains {
					dailyLocked[date][chain] = map[string]LockedAsset{}
					dailyLocked[date][chain]["*"] = addresses["*"]
				}
			}
		}
	}

	result := &tvlCumulativeResult{
		DailyLocked: dailyLocked,
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
