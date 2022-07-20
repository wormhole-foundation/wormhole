// Package p contains an HTTP Cloud Function.
package p

import (
	// "bytes"
	"context"
	// "encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"cloud.google.com/go/bigtable"
)

type cumulativeResult struct {
	AllTime             map[string]map[string]float64
	AllTimeDurationDays int
	Daily               map[string]map[string]map[string]float64
}

// an in-memory cache of previously calculated results
var warmCumulativeCache = map[string]map[string]map[string]map[string]float64{}
var muWarmCumulativeCache sync.RWMutex
var warmCumulativeCacheFilePath = "notional-transferred-to-cumulative-cache.json"

var transferredToUpToYesterday = map[string]map[string]map[string]map[string]float64{}
var muTransferredToUpToYesterday sync.RWMutex
var transferredToUpToYesterdayFilePath = "notional-transferred-to-up-to-yesterday-cache.json"

// calculates the amount of each symbol transfered to each chain.
func transferredToSince(tbl *bigtable.Table, ctx context.Context, prefix string, start time.Time) map[string]map[string]float64 {
	if _, ok := transferredToUpToYesterday["*"]; !ok && loadCache {
		loadJsonToInterface(ctx, transferredToUpToYesterdayFilePath, &muTransferredToUpToYesterday, &transferredToUpToYesterday)
	}

	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	oneDayAgo := -time.Duration(24) * time.Hour
	yesterday := now.Add(oneDayAgo).Format("2006-01-02")

	result := map[string]map[string]float64{"*": {"*": 0}}

	// create the unique identifier for this query, for cache
	cachePrefix := createCachePrefix(prefix)
	muTransferredToUpToYesterday.Lock()
	if _, ok := transferredToUpToYesterday[cachePrefix]; !ok {
		transferredToUpToYesterday[cachePrefix] = map[string]map[string]map[string]float64{}
	}

	if cacheData, ok := transferredToUpToYesterday[cachePrefix][yesterday]; ok {
		// cache has data through midnight yesterday
		for chain, symbols := range cacheData {
			result[chain] = map[string]float64{}
			for symbol, amount := range symbols {
				result[chain][symbol] = amount
			}
		}
		// set the start to be the start of today
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}
	muTransferredToUpToYesterday.Unlock()

	dailyTotals := amountsTransferredToInInterval(tbl, ctx, prefix, start)

	// loop through the query results to combine cache + fresh data
	for _, chains := range dailyTotals {
		for chain, tokens := range chains {
			// ensure the chain exists in the result map
			if _, ok := result[chain]; !ok {
				result[chain] = map[string]float64{"*": 0}
			}
			for symbol, amount := range tokens {
				if _, ok := result[chain][symbol]; !ok {
					result[chain][symbol] = 0
				}
				// add the amount of this symbol transferred this day to the
				// amount already in the result (amount of this symbol prevoiusly transferred)
				result[chain][symbol] = result[chain][symbol] + amount
			}
		}
	}

	muTransferredToUpToYesterday.Lock()
	if _, ok := transferredToUpToYesterday[cachePrefix][yesterday]; !ok {
		transferredToUpToYesterday[cachePrefix][yesterday] = map[string]map[string]float64{}
		// no cache, populate it
		upToYesterday := map[string]map[string]float64{}
		for chain, tokens := range result {
			upToYesterday[chain] = map[string]float64{}
			for symbol, amount := range tokens {
				upToYesterday[chain][symbol] = amount
			}
		}
		for chain, tokens := range dailyTotals[today] {
			for symbol, amount := range tokens {
				// subtract the amounts from today, in order to create an "upToYesterday" amount
				upToYesterday[chain][symbol] = result[chain][symbol] - amount
			}
		}
		// loop again to assign values to the cache
		for chain, tokens := range upToYesterday {
			if _, ok := transferredToUpToYesterday[cachePrefix][yesterday][chain]; !ok {
				transferredToUpToYesterday[cachePrefix][yesterday][chain] = map[string]float64{}
			}
			for symbol, amount := range tokens {
				transferredToUpToYesterday[cachePrefix][yesterday][chain][symbol] = amount
			}
		}
		muTransferredToUpToYesterday.Unlock()
		// write the updated cache to disc
		persistInterfaceToJson(ctx, transferredToUpToYesterdayFilePath, &muTransferredToUpToYesterday, transferredToUpToYesterday)
	} else {
		muTransferredToUpToYesterday.Unlock()
	}

	return result
}

// returns a slice of dates (strings) for each day in the period. Dates formatted: "2021-12-30".
func getDaysInRange(start, end time.Time) []string {
	now := time.Now().UTC()
	numDays := int(end.Sub(start).Hours() / 24)
	days := []string{}
	for daysAgo := 0; daysAgo <= numDays; daysAgo++ {
		hoursAgo := (24 * daysAgo)
		daysAgoDuration := -time.Duration(hoursAgo) * time.Hour
		n := now.Add(daysAgoDuration)
		year := n.Year()
		month := n.Month()
		day := n.Day()
		loc := n.Location()

		start := time.Date(year, month, day, 0, 0, 0, 0, loc)
		dateStr := start.Format("2006-01-02")
		days = append(days, dateStr)
	}
	return days
}

// calcuates a running total of notional value transferred, by symbol, since the start time specified.
func createCumulativeAmountsOfInterval(tbl *bigtable.Table, ctx context.Context, prefix string, start time.Time) map[string]map[string]map[string]float64 {
	if _, ok := warmCumulativeCache["*"]; !ok && loadCache {
		loadJsonToInterface(ctx, warmCumulativeCacheFilePath, &muWarmCumulativeCache, &warmCumulativeCache)
	}

	now := time.Now().UTC()
	today := now.Format("2006-01-02")

	cachePrefix := createCachePrefix(prefix)
	cacheNeedsUpdate := false
	muWarmCumulativeCache.Lock()
	if _, ok := warmCumulativeCache[cachePrefix]; !ok {
		warmCumulativeCache[cachePrefix] = map[string]map[string]map[string]float64{}
	}
	muWarmCumulativeCache.Unlock()

	results := map[string]map[string]map[string]float64{}

	// fetch the amounts of transfers by symbol, for each day since launch (releaseDay)
	dailyAmounts := amountsTransferredToInInterval(tbl, ctx, prefix, releaseDay)

	// create a slice of dates, order oldest first
	dateKeys := make([]string, 0, len(dailyAmounts))
	for k := range dailyAmounts {
		dateKeys = append(dateKeys, k)
	}
	sort.Strings(dateKeys)

	// iterate through the dates in the result set, and accumulate the amounts
	// of each token transfer by symbol, based on the destination of the transfer.
	for i, date := range dateKeys {
		results[date] = map[string]map[string]float64{"*": {"*": 0}}
		muWarmCumulativeCache.RLock()
		if dateCache, ok := warmCumulativeCache[cachePrefix][date]; ok && dateCache != nil && useCache(date) {
			// have a cached value for this day, use it.
			// iterate through cache and copy values to the result
			for chain, tokens := range dateCache {
				results[date][chain] = map[string]float64{}
				for token, amount := range tokens {
					results[date][chain][token] = amount
				}
			}
			muWarmCumulativeCache.RUnlock()
		} else {
			// no cached value for this day, must calculate it
			muWarmCumulativeCache.RUnlock()
			if i == 0 {
				// special case for first day, no need to sum.
				for chain, tokens := range dailyAmounts[date] {
					results[date][chain] = map[string]float64{}
					for token, amount := range tokens {
						results[date][chain][token] = amount
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
						results[date][chain] = map[string]float64{"*": 0}
					}
					// iterate through the union of symbols, creating an amount for each one,
					// and adding it the the results.
					for symbol := range symbolsUnion {

						thisDayAmount := float64(0)
						if amt, ok := thisDaySymbols[symbol]; ok {
							thisDayAmount = amt
						}
						prevDayAmount := float64(0)
						if amt, ok := results[prevDate][chain][symbol]; ok {
							prevDayAmount = amt
						}
						cumulativeAmount := prevDayAmount + thisDayAmount

						results[date][chain][symbol] = cumulativeAmount
					}
				}
			}
			// dont cache today
			if date != today {
				// set the result in the cache
				muWarmCumulativeCache.Lock()
				if _, ok := warmCumulativeCache[cachePrefix][date]; !ok || !useCache(date) {
					// cache does not have this date, persist it for other instances.
					warmCumulativeCache[cachePrefix][date] = map[string]map[string]float64{}
					for chain, tokens := range results[date] {
						warmCumulativeCache[cachePrefix][date][chain] = map[string]float64{}
						for token, amount := range tokens {
							warmCumulativeCache[cachePrefix][date][chain][token] = amount
						}
					}
					cacheNeedsUpdate = true
				}
				muWarmCumulativeCache.Unlock()

			}
		}
	}

	if cacheNeedsUpdate {
		persistInterfaceToJson(ctx, warmCumulativeCacheFilePath, &muWarmCumulativeCache, warmCumulativeCache)
	}

	// take the most recent n days, rather than returning all days since launch
	selectDays := map[string]map[string]map[string]float64{}
	days := getDaysInRange(start, now)
	for _, day := range days {
		selectDays[day] = map[string]map[string]float64{}
		for chain, tokens := range results[day] {
			selectDays[day][chain] = map[string]float64{}
			for symbol, amount := range tokens {
				selectDays[day][chain][symbol] = amount
			}
		}
	}
	return selectDays

}

// calculates the cumulative value transferred each day since launch.
func NotionalTransferredToCumulative(w http.ResponseWriter, r *http.Request) {
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

	var numDays, forChain, forAddress, daily, allTime string

	// allow GET requests with querystring params, or POST requests with json body.
	switch r.Method {
	case http.MethodGet:
		queryParams := r.URL.Query()
		numDays = queryParams.Get("numDays")
		forChain = queryParams.Get("forChain")
		forAddress = queryParams.Get("forAddress")
		daily = queryParams.Get("daily")
		allTime = queryParams.Get("allTime")

	case http.MethodPost:
		// declare request body properties
		var d struct {
			NumDays    string `json:"numDays"`
			ForChain   string `json:"forChain"`
			ForAddress string `json:"forAddress"`
			Daily      string `json:"daily"`
			AllTime    string `json:"allTime"`
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

		numDays = d.NumDays
		forChain = d.ForChain
		forAddress = d.ForAddress
		daily = d.Daily
		allTime = d.AllTime

	default:
		http.Error(w, "405 - Method Not Allowed", http.StatusMethodNotAllowed)
		log.Println("Method Not Allowed")
		return
	}

	if daily == "" && allTime == "" {
		// none of the options were set, so set one
		allTime = "true"
	}

	var queryDays int
	if numDays == "" {
		queryDays = 30
	} else {
		var convErr error
		queryDays, convErr = strconv.Atoi(numDays)
		if convErr != nil {
			fmt.Fprint(w, "numDays must be an integer")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}

	// create the rowkey prefix for querying
	prefix := ""
	if forChain != "" {
		prefix = forChain
		// if the request is forChain, always groupBy chain
		if forAddress != "" {
			// if the request is forAddress, always groupBy address
			prefix = forChain + ":" + forAddress
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// total since launch
	periodTransfers := map[string]map[string]float64{}
	allTimeDays := int(time.Now().UTC().Sub(releaseDay).Hours() / 24)
	if allTime != "" {
		wg.Add(1)
		go func(prefix string) {
			defer wg.Done()
			transfers := transferredToSince(tbl, context.Background(), prefix, releaseDay)
			for chain, tokens := range transfers {
				periodTransfers[chain] = map[string]float64{}
				for symbol, amount := range tokens {
					periodTransfers[chain][symbol] = roundToTwoDecimalPlaces(amount)
				}
			}
		}(prefix)
	}

	// daily transfers by chain
	dailyTransfers := map[string]map[string]map[string]float64{}
	if daily != "" {
		wg.Add(1)
		go func(prefix string, queryDays int) {
			hours := (24 * queryDays)
			periodInterval := -time.Duration(hours) * time.Hour
			now := time.Now().UTC()
			prev := now.Add(periodInterval)
			start := time.Date(prev.Year(), prev.Month(), prev.Day(), 0, 0, 0, 0, prev.Location())
			defer wg.Done()
			transfers := createCumulativeAmountsOfInterval(tbl, ctx, prefix, start)
			for date, chains := range transfers {
				dailyTransfers[date] = map[string]map[string]float64{}
				for chain, tokens := range chains {
					dailyTransfers[date][chain] = map[string]float64{}
					for symbol, amount := range tokens {
						dailyTransfers[date][chain][symbol] = roundToTwoDecimalPlaces(amount)
					}
				}
			}

		}(prefix, queryDays)
	}

	wg.Wait()

	result := &cumulativeResult{
		AllTime:             periodTransfers,
		AllTimeDurationDays: allTimeDays,
		Daily:               dailyTransfers,
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
