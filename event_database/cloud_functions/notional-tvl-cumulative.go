// Package p contains an HTTP Cloud Function.
package p

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"sort"
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

var tvlUpToYesterday = map[string]map[string]map[string]map[string]float64{}
var muTvlUpToYesterday sync.RWMutex
var tvlUpToYesterdayFilePath = "tvl-up-to-yesterday-cache.json"

// token addresses blacklisted from TVL calculation
var tokensToSkip = map[string]bool{
	"0x04132bf45511d03a58afd4f1d36a29d229ccc574": true,
	"0xa79bd679ce21a2418be9e6f88b2186c9986bbe7d": true,
	"0x931c3987040c90b6db09981c7c91ba155d3fa31f": true,
	"0x8fb1a59ca2d57b51e5971a85277efe72c4492983": true,
	"0xd52d9ba6fcbadb1fe1e3aca52cbb72c4d9bbb4ec": true,
	"0x1353c55fd2beebd976d7acc4a7083b0618d94689": true,
	"0xf0fbdb8a402ec0fc626db974b8d019c902deb486": true,
	"0x1fd4a95f4335cf36cac85730289579c104544328": true,
	"0x358aa13c52544eccef6b0add0f801012adad5ee3": true,
	"0xbe32b7acd03bcc62f25ebabd169a35e69ef17601": true,
	"0x7ffb3d637014488b63fb9858e279385685afc1e2": true,
}

// days to be excluded from the TVL result
var skipDays = map[string]bool{
	// for example:
	// "2022-02-19": true,
}

// calcuates a running total of notional value transferred, by symbol, since the start time specified.
func createTvlCumulativeOfInterval(tbl *bigtable.Table, ctx context.Context, start time.Time) map[string]map[string]map[string]LockedAsset {
	if len(warmTvlCumulativeCache) == 0 {
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
					// initalize the chain/symbol map for this date
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
			// dont cache today
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

	// get the number of days to query - days since launch day
	queryDays := int(time.Now().UTC().Sub(releaseDay).Hours() / 24)

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	dailyTvl := map[string]map[string]map[string]LockedAsset{}

	hours := (24 * queryDays)
	periodInterval := -time.Duration(hours) * time.Hour
	now := time.Now().UTC()
	prev := now.Add(periodInterval)
	start := time.Date(prev.Year(), prev.Month(), prev.Day(), 0, 0, 0, 0, prev.Location())

	transfers := createTvlCumulativeOfInterval(tbl, ctx, start)

	// calculate the notional tvl based on the price of the tokens each day
	for date, chains := range transfers {
		if _, ok := skipDays[date]; ok {
			log.Println("skipping ", date)
			continue
		}
		dailyTvl[date] = map[string]map[string]LockedAsset{}
		dailyTvl[date]["*"] = map[string]LockedAsset{}
		dailyTvl[date]["*"]["*"] = LockedAsset{
			Symbol:      "*",
			Address:     "",
			CoinGeckoId: "",
			TokenPrice:  0,
			Amount:      0,
			Notional:    0,
		}
		for chain, tokens := range chains {
			if chain == "*" {
				continue
			}
			dailyTvl[date][chain] = map[string]LockedAsset{}
			dailyTvl[date][chain]["*"] = LockedAsset{
				Symbol:      "*",
				Address:     "",
				CoinGeckoId: "",
				TokenPrice:  0,
				Amount:      0,
				Notional:    0,
			}

			for symbol, asset := range tokens {
				if symbol == "*" {
					continue
				}
				if _, ok := tokensToSkip[symbol]; ok {
					log.Printf("going to skip %v, on chain %v, date %v", asset.Symbol, chain, date)
					continue
				}

				notional := asset.Amount * asset.TokenPrice
				if notional <= 0 {
					log.Printf("skipping token with no/negative value. notional: %v, chain: %v, symbol %v, address %v", notional, chain, asset.Symbol, asset.Address)
					continue
				}

				asset.Notional = roundToTwoDecimalPlaces(notional)
				dailyTvl[date][chain][symbol] = asset

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
