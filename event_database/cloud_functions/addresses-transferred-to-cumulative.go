// Package p contains an HTTP Cloud Function.
package p

import (
	"context"
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

type cumulativeAddressesResult struct {
	AllTimeAmounts      map[string]map[string]float64
	AllTimeCounts       map[string]int
	AllTimeDurationDays int
	DailyAmounts        map[string]map[string]map[string]float64
	DailyCounts         map[string]map[string]int
}

// an in-memory cache of previously calculated results
var warmCumulativeAddressesCache = map[string]map[string]map[string]map[string]float64{}
var muWarmCumulativeAddressesCache sync.RWMutex
var warmCumulativeAddressesCacheFilePath = "addresses-transferred-to-cumulative-cache.json"

var addressesToUpToYesterday = map[string]map[string]map[string]map[string]float64{}
var muAddressesToUpToYesterday sync.RWMutex
var addressesToUpToYesterdayFilePath = "addresses-transferred-to-up-to-yesterday-cache.json"

// finds all the unique addresses that have received tokens since a particular moment.
func addressesTransferredToSince(tbl *bigtable.Table, ctx context.Context, prefix string, start time.Time) map[string]map[string]float64 {
	if _, ok := addressesToUpToYesterday["*"]; !ok && loadCache {
		loadJsonToInterface(ctx, addressesToUpToYesterdayFilePath, &muAddressesToUpToYesterday, &addressesToUpToYesterday)
	}

	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	oneDayAgo := -time.Duration(24) * time.Hour
	yesterday := now.Add(oneDayAgo).Format("2006-01-02")

	result := map[string]map[string]float64{}

	// create the unique identifier for this query, for cache
	cachePrefix := createCachePrefix(prefix)
	muAddressesToUpToYesterday.Lock()
	if _, ok := addressesToUpToYesterday[cachePrefix]; !ok {
		addressesToUpToYesterday[cachePrefix] = map[string]map[string]map[string]float64{}
	}

	if cacheData, ok := addressesToUpToYesterday[cachePrefix][yesterday]; ok {
		// cache has data through midnight yesterday
		for chain, addresses := range cacheData {
			result[chain] = map[string]float64{}
			for address, amount := range addresses {
				result[chain][address] = amount
			}
		}
		// set the start to be the start of today
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}
	muAddressesToUpToYesterday.Unlock()

	// fetch data for days not in the cache
	dailyAddresses := createAddressesOfInterval(tbl, ctx, prefix, start)

	// loop through the query results to combine cache + fresh data
	for _, chains := range dailyAddresses {
		for chain, addresses := range chains {
			// ensure the chain exists in the result map
			if _, ok := result[chain]; !ok {
				result[chain] = map[string]float64{}
			}
			for address, amount := range addresses {
				if _, ok := result[chain][address]; !ok {
					result[chain][address] = 0
				}
				// add the amount the address received this day to the
				// amount already in the result (amount the address has recieved so far)
				result[chain][address] = result[chain][address] + amount
			}
		}
	}

	muAddressesToUpToYesterday.Lock()
	if _, ok := addressesToUpToYesterday[cachePrefix][yesterday]; !ok {
		addressesToUpToYesterday[cachePrefix][yesterday] = map[string]map[string]float64{}
		// no cache, populate it
		upToYesterday := map[string]map[string]float64{}
		for chain, addresses := range result {
			upToYesterday[chain] = map[string]float64{}
			for address, amount := range addresses {
				upToYesterday[chain][address] = amount
			}
		}
		for chain, addresses := range dailyAddresses[today] {
			for address, amount := range addresses {
				// subtract the amounts from today, in order to create an "upToYesterday" amount
				upToYesterday[chain][address] = result[chain][address] - amount
			}
		}
		// loop again to assign values to the cache
		for chain, addresses := range upToYesterday {
			if _, ok := addressesToUpToYesterday[cachePrefix][yesterday][chain]; !ok {
				addressesToUpToYesterday[cachePrefix][yesterday][chain] = map[string]float64{}
			}
			for address, amount := range addresses {
				addressesToUpToYesterday[cachePrefix][yesterday][chain][address] = amount
			}
		}
		muAddressesToUpToYesterday.Unlock()
		// write cache to disc
		persistInterfaceToJson(ctx, addressesToUpToYesterdayFilePath, &muAddressesToUpToYesterday, addressesToUpToYesterday)
	} else {
		muAddressesToUpToYesterday.Unlock()
	}

	return result
}

// calcuates a map of recepient address to notional value received, by chain, since the start time specified.
func createCumulativeAddressesOfInterval(tbl *bigtable.Table, ctx context.Context, prefix string, start time.Time) map[string]map[string]map[string]float64 {
	if _, ok := warmCumulativeAddressesCache["*"]; !ok && loadCache {
		loadJsonToInterface(ctx, warmCumulativeAddressesCacheFilePath, &muWarmCumulativeAddressesCache, &warmCumulativeAddressesCache)
	}

	now := time.Now().UTC()
	today := now.Format("2006-01-02")

	cachePrefix := createCachePrefix(prefix)
	cacheNeedsUpdate := false
	muWarmCumulativeAddressesCache.Lock()
	if _, ok := warmCumulativeAddressesCache[cachePrefix]; !ok {
		warmCumulativeAddressesCache[cachePrefix] = map[string]map[string]map[string]float64{}
	}
	muWarmCumulativeAddressesCache.Unlock()

	results := map[string]map[string]map[string]float64{}

	dailyAddresses := createAddressesOfInterval(tbl, ctx, prefix, releaseDay)

	dateKeys := make([]string, 0, len(dailyAddresses))
	for k := range dailyAddresses {
		dateKeys = append(dateKeys, k)
	}
	sort.Strings(dateKeys)

	// iterate through the dates in the result set, and accumulate the amounts
	// of each token transfer by symbol, based on the destination of the transfer.
	for i, date := range dateKeys {
		results[date] = map[string]map[string]float64{"*": {"*": 0}}
		muWarmCumulativeAddressesCache.RLock()
		if dateCache, ok := warmCumulativeAddressesCache[cachePrefix][date]; ok && dateCache != nil && useCache(date) {
			// have a cached value for this day, use it.
			// iterate through cache and copy values to the result
			for chain, addresses := range dateCache {
				results[date][chain] = map[string]float64{}
				for address, amount := range addresses {
					results[date][chain][address] = amount
				}
			}
			muWarmCumulativeAddressesCache.RUnlock()
		} else {
			// no cached value for this day, must calculate it
			muWarmCumulativeAddressesCache.RUnlock()
			if i == 0 {
				// special case for first day, no need to sum.
				for chain, addresses := range dailyAddresses[date] {
					results[date][chain] = map[string]float64{}
					for address, amount := range addresses {
						results[date][chain][address] = amount
					}
				}
			} else {
				// find the string of the previous day
				prevDate := dateKeys[i-1]
				prevDayChains := results[prevDate]
				thisDayChains := dailyAddresses[date]
				for chain, thisDayAddresses := range thisDayChains {
					// create a union of the addresses from this day, and previous days
					addressUnion := map[string]string{}
					for address := range prevDayChains[chain] {
						addressUnion[address] = address
					}
					for address := range thisDayAddresses {
						addressUnion[address] = address
					}
					// initalize the chain/symbol map for this date
					if _, ok := results[date][chain]; !ok {
						results[date][chain] = map[string]float64{}
					}

					// iterate through the union of addresses, creating an amount for each one,
					// and adding it the the results.
					for address := range addressUnion {
						thisDayAmount := float64(0)
						if amt, ok := thisDayAddresses[address]; ok {
							thisDayAmount = amt
						}
						prevDayAmount := float64(0)
						if prevAmount, ok := results[prevDate][chain][address]; ok && prevAmount != 0 {
							prevDayAmount = prevAmount
						}
						cumulativeAmount := prevDayAmount + thisDayAmount
						results[date][chain][address] = cumulativeAmount
					}
				}
			}
			// dont cache today
			if date != today {
				// set the result in the cache
				muWarmCumulativeAddressesCache.Lock()
				if _, ok := warmCumulativeAddressesCache[cachePrefix][date]; !ok || !useCache(date) {
					// cache does not have this date, persist it for other instances.
					warmCumulativeAddressesCache[cachePrefix][date] = map[string]map[string]float64{}
					for chain, addresses := range results[date] {
						warmCumulativeAddressesCache[cachePrefix][date][chain] = map[string]float64{}
						for address, amount := range addresses {
							warmCumulativeAddressesCache[cachePrefix][date][chain][address] = amount
						}
					}
					cacheNeedsUpdate = true
				}
				muWarmCumulativeAddressesCache.Unlock()
			}
		}
	}

	if cacheNeedsUpdate {
		persistInterfaceToJson(ctx, warmCumulativeAddressesCacheFilePath, &muWarmCumulativeAddressesCache, warmCumulativeAddressesCache)
	}

	// take the most recent n days, rather than returning all days since launch
	selectDays := map[string]map[string]map[string]float64{}
	days := getDaysInRange(start, now)
	for _, day := range days {
		selectDays[day] = map[string]map[string]float64{}
		for chain, addresses := range results[day] {
			selectDays[day][chain] = map[string]float64{}
			for address, amount := range addresses {
				selectDays[day][chain][address] = amount
			}
		}
	}
	return selectDays

}

// finds unique addresses that tokens have been transferred to.
func AddressesTransferredToCumulative(w http.ResponseWriter, r *http.Request) {
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

	var numDays, forChain, forAddress, daily, allTime, counts, amounts string

	// allow GET requests with querystring params, or POST requests with json body.
	switch r.Method {
	case http.MethodGet:
		queryParams := r.URL.Query()
		numDays = queryParams.Get("numDays")
		forChain = queryParams.Get("forChain")
		forAddress = queryParams.Get("forAddress")
		daily = queryParams.Get("daily")
		allTime = queryParams.Get("allTime")
		counts = queryParams.Get("counts")
		amounts = queryParams.Get("amounts")

	case http.MethodPost:
		// declare request body properties
		var d struct {
			NumDays    string `json:"numDays"`
			ForChain   string `json:"forChain"`
			ForAddress string `json:"forAddress"`
			Daily      string `json:"daily"`
			AllTime    string `json:"allTime"`
			Counts     string `json:"counts"`
			Amounts    string `json:"amounts"`
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
		counts = d.Counts
		amounts = d.Amounts

	default:
		http.Error(w, "405 - Method Not Allowed", http.StatusMethodNotAllowed)
		log.Println("Method Not Allowed")
		return
	}

	if daily == "" && allTime == "" {
		// none of the options were set, so set one
		allTime = "true"
	}
	if counts == "" && amounts == "" {
		// neither of the options were set, so set one
		counts = "true"
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

	// total of the last numDays
	addressesDailyAmounts := map[string]map[string]float64{}
	addressesDailyCounts := map[string]int{}
	allTimeDays := int(time.Now().UTC().Sub(releaseDay).Hours() / 24)
	if allTime != "" {
		wg.Add(1)
		go func(prefix string) {

			defer wg.Done()
			periodAmounts := addressesTransferredToSince(tbl, ctx, prefix, releaseDay)
			if amounts != "" {
				for chain, addresses := range periodAmounts {
					addressesDailyAmounts[chain] = map[string]float64{}
					for address, amount := range addresses {
						addressesDailyAmounts[chain][address] = roundToTwoDecimalPlaces(amount)
					}
				}
			}
			if counts != "" {
				for chain, addresses := range periodAmounts {
					// need to sum all the chains to get the total count of addresses,
					// since addresses are not unique across chains.
					numAddresses := len(addresses)
					addressesDailyCounts[chain] = len(addresses)
					addressesDailyCounts["*"] = addressesDailyCounts["*"] + numAddresses
				}
			}
		}(prefix)
	}

	// daily totals
	dailyAmounts := map[string]map[string]map[string]float64{}
	dailyCounts := map[string]map[string]int{}
	if daily != "" {
		wg.Add(1)
		go func(prefix string, queryDays int) {
			hours := (24 * queryDays)
			periodInterval := -time.Duration(hours) * time.Hour
			now := time.Now().UTC()
			prev := now.Add(periodInterval)
			start := time.Date(prev.Year(), prev.Month(), prev.Day(), 0, 0, 0, 0, prev.Location())
			defer wg.Done()
			dailyTotals := createCumulativeAddressesOfInterval(tbl, ctx, prefix, start)
			if amounts != "" {
				for date, chains := range dailyTotals {
					dailyAmounts[date] = map[string]map[string]float64{}
					for chain, addresses := range chains {
						dailyAmounts[date][chain] = map[string]float64{}
						for address, amount := range addresses {
							dailyAmounts[date][chain][address] = roundToTwoDecimalPlaces(amount)
						}
					}
				}
			}
			if counts != "" {
				for date, chains := range dailyTotals {
					dailyCounts[date] = map[string]int{}
					for chain, addresses := range chains {
						// need to sum all the chains to get the total count of addresses,
						// since addresses are not unique across chains.
						numAddresses := len(addresses)
						dailyCounts[date][chain] = numAddresses
						dailyCounts[date]["*"] = dailyCounts[date]["*"] + numAddresses
					}
				}
			}
		}(prefix, queryDays)
	}

	wg.Wait()

	result := &cumulativeAddressesResult{
		AllTimeAmounts:      addressesDailyAmounts,
		AllTimeCounts:       addressesDailyCounts,
		AllTimeDurationDays: allTimeDays,
		DailyAmounts:        dailyAmounts,
		DailyCounts:         dailyCounts,
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Println(err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	w.Write(jsonBytes)
}
