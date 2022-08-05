// Package p contains an HTTP Cloud Function.
package p

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"cloud.google.com/go/bigtable"
)

type transfersResult struct {
	Last24Hours        map[string]map[string]map[string]float64
	WithinPeriod       map[string]map[string]map[string]float64
	PeriodDurationDays int
	Daily              map[string]map[string]map[string]map[string]float64
}

// an in-memory cache of previously calculated results
var warmTransfersCache = map[string]map[string]map[string]map[string]map[string]float64{}
var muWarmTransfersCache sync.RWMutex
var warmTransfersCacheFilePath = "notional-transferred-cache.json"

// finds the daily amount of each symbol transferred from each chain, to each chain,
// from the specified start to the present.
func createTransfersOfInterval(tbl *bigtable.Table, ctx context.Context, prefix string, start time.Time) map[string]map[string]map[string]map[string]float64 {
	if _, ok := warmTransfersCache["*"]; !ok && loadCache {
		loadJsonToInterface(ctx, warmTransfersCacheFilePath, &muWarmTransfersCache, &warmTransfersCache)
	}

	results := map[string]map[string]map[string]map[string]float64{}

	now := time.Now().UTC()
	numPrevDays := int(now.Sub(start).Hours() / 24)

	var intervalsWG sync.WaitGroup
	// there will be a query for each previous day, plus today
	intervalsWG.Add(numPrevDays + 1)

	// create the unique identifier for this query, for cache
	cachePrefix := createCachePrefix(prefix)

	cacheNeedsUpdate := false

	for daysAgo := 0; daysAgo <= numPrevDays; daysAgo++ {
		go func(tbl *bigtable.Table, ctx context.Context, prefix string, daysAgo int) {
			// start is the SOD, end is EOD
			// "0 daysAgo start" is 00:00:00 AM of the current day
			// "0 daysAgo end" is 23:59:59 of the current day (the future)

			// calulate the start and end times for the query
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

			muWarmTransfersCache.Lock()
			// initialize the map for this date in the result set
			results[dateStr] = map[string]map[string]map[string]float64{"*": {"*": {"*": 0}}}
			// check to see if there is cache data for this date/query
			if dates, ok := warmTransfersCache[cachePrefix]; ok {
				// have a cache for this query

				if dateCache, ok := dates[dateStr]; ok && len(dateCache) > 1 && useCache(dateStr) {
					// have a cache for this date

					if daysAgo >= 1 {
						// only use the cache for yesterday and older
						results[dateStr] = dateCache
						muWarmTransfersCache.Unlock()
						intervalsWG.Done()
						return
					}
				}
			} else {
				// no cache for this query, initialize the map
				warmTransfersCache[cachePrefix] = map[string]map[string]map[string]map[string]float64{}
			}
			muWarmTransfersCache.Unlock()

			defer intervalsWG.Done()

			queryResult := fetchTransferRowsInInterval(tbl, ctx, prefix, start, end)

			// iterate through the rows and increment the amounts
			for _, row := range queryResult {
				if _, ok := results[dateStr][row.LeavingChain]; !ok {
					results[dateStr][row.LeavingChain] = map[string]map[string]float64{"*": {"*": 0}}
				}
				if _, ok := results[dateStr][row.LeavingChain][row.DestinationChain]; !ok {
					results[dateStr][row.LeavingChain][row.DestinationChain] = map[string]float64{"*": 0}
				}
				if _, ok := results[dateStr]["*"][row.DestinationChain]; !ok {
					results[dateStr]["*"][row.DestinationChain] = map[string]float64{"*": 0}
				}
				// add the transfer data to the result set every possible way:
				// by symbol, aggregated by: "leaving chain", "arriving at chain", "from any chain", "to any chain".

				// add to the total amount leaving this chain, going to any chain, for all symbols
				results[dateStr][row.LeavingChain]["*"]["*"] = results[dateStr][row.LeavingChain]["*"]["*"] + row.Notional
				// add to the total amount leaving this chain, going to the destination chain, for all symbols
				results[dateStr][row.LeavingChain][row.DestinationChain]["*"] = results[dateStr][row.LeavingChain][row.DestinationChain]["*"] + row.Notional
				// add to the total amount of this symbol leaving this chain, going to any chain
				results[dateStr][row.LeavingChain]["*"][row.TokenSymbol] = results[dateStr][row.LeavingChain]["*"][row.TokenSymbol] + row.Notional
				// add to the total amount of this symbol leaving this chain, going to the destination chain
				results[dateStr][row.LeavingChain][row.DestinationChain][row.TokenSymbol] = results[dateStr][row.LeavingChain][row.DestinationChain][row.TokenSymbol] + row.Notional

				// add to the total amount arriving at the destination chain, coming from anywhere, including all symbols
				results[dateStr]["*"][row.DestinationChain]["*"] = results[dateStr]["*"][row.DestinationChain]["*"] + row.Notional
				// add to the total amount of this symbol arriving at the destination chain
				results[dateStr]["*"][row.DestinationChain][row.TokenSymbol] = results[dateStr]["*"][row.DestinationChain][row.TokenSymbol] + row.Notional
				// add to the total amount of this symbol transferred, from any chain, to any chain
				results[dateStr]["*"]["*"][row.TokenSymbol] = results[dateStr]["*"]["*"][row.TokenSymbol] + row.Notional
				// and finally, total/total/total: amount of all symbols transferred from any chain to any other chain
				results[dateStr]["*"]["*"]["*"] = results[dateStr]["*"]["*"]["*"] + row.Notional
			}
			if daysAgo >= 1 {
				// set the result in the cache
				muWarmTransfersCache.Lock()
				if cacheData, ok := warmTransfersCache[cachePrefix][dateStr]; !ok || len(cacheData) == 1 || !useCache(dateStr) {
					// cache does not have this date, add the data, and mark the cache stale
					warmTransfersCache[cachePrefix][dateStr] = results[dateStr]
					cacheNeedsUpdate = true
				}
				muWarmTransfersCache.Unlock()
			}
		}(tbl, ctx, prefix, daysAgo)
	}

	intervalsWG.Wait()

	if cacheNeedsUpdate {
		persistInterfaceToJson(ctx, warmTransfersCacheFilePath, &muWarmTransfersCache, warmTransfersCache)
	}

	// having consistent keys in each object is helpful for clients, explorer GUI
	// create a set of all the keys from all dates/chains, to ensure the result objects all have the same chain keys
	seenChainSet := map[string]bool{}
	for _, chains := range results {
		for leaving, dests := range chains {
			seenChainSet[leaving] = true

			for dest := range dests {
				seenChainSet[dest] = true
			}
		}
	}

	var muResult sync.RWMutex
	// ensure each chain object has all the same symbol keys:
	for date, chains := range results {
		for chain := range seenChainSet {
			if _, ok := chains[chain]; !ok {
				muResult.Lock()
				results[date][chain] = map[string]map[string]float64{"*": {"*": 0}}
				muResult.Unlock()
			}
		}
		for leaving := range chains {
			for chain := range seenChainSet {
				// check that date has all the chains
				if _, ok := chains[chain]; !ok {
					muResult.Lock()
					results[date][leaving][chain] = map[string]float64{"*": 0}
					muResult.Unlock()
				}
			}
		}
	}

	return results
}

// calculates the amount of each symbol that has gone from each chain, to each other chain, since the specified day.
func transferredSinceDate(tbl *bigtable.Table, ctx context.Context, prefix string, start time.Time) map[string]map[string]map[string]float64 {
	result := map[string]map[string]map[string]float64{"*": {"*": {"*": 0}}}

	dailyTotals := createTransfersOfInterval(tbl, ctx, prefix, start)

	for _, leaving := range dailyTotals {
		for chain, dests := range leaving {
			// ensure the chain exists in the result map
			if _, ok := result[chain]; !ok {
				result[chain] = map[string]map[string]float64{"*": {"*": 0}}
			}
			for dest, tokens := range dests {
				if _, ok := result[chain][dest]; !ok {
					result[chain][dest] = map[string]float64{"*": 0}
				}
				for symbol, amount := range tokens {
					if _, ok := result[chain][dest][symbol]; !ok {
						result[chain][dest][symbol] = 0
					}
					// add the amount of this symbol transferred this day to the
					// amount already in the result (amount of this symbol prevoiusly transferred)
					result[chain][dest][symbol] = result[chain][dest][symbol] + amount
				}
			}
		}
	}

	// create a set of chainIDs, the union of source and destination chains,
	// to ensure the result objects all have the same keys.
	seenChainSet := map[string]bool{}
	for leaving, dests := range result {
		seenChainSet[leaving] = true
		for dest := range dests {
			seenChainSet[dest] = true
		}
	}
	// make sure the root of the map has all the chainIDs
	for chain := range seenChainSet {
		if _, ok := result[chain]; !ok {
			result[chain] = map[string]map[string]float64{"*": {"*": 0}}
		}
	}
	// make sure that each chain at the root (leaving) as a key (destination) for each chain
	for leaving, dests := range result {
		for chain := range seenChainSet {
			// check that date has all the chains
			if _, ok := dests[chain]; !ok {
				result[leaving][chain] = map[string]float64{"*": 0}
			}
		}
	}

	return result
}

// returns the count of the rows in the query response
func transfersForInterval(tbl *bigtable.Table, ctx context.Context, prefix string, start, end time.Time) map[string]map[string]map[string]float64 {
	// query for all rows in time range, return result count
	queryResults := fetchTransferRowsInInterval(tbl, ctx, prefix, start, end)

	result := map[string]map[string]map[string]float64{"*": {"*": {"*": 0}}}

	// iterate through the rows and increment the count for each index
	for _, row := range queryResults {
		if _, ok := result[row.LeavingChain]; !ok {
			result[row.LeavingChain] = map[string]map[string]float64{"*": {"*": 0}}
		}
		if _, ok := result[row.LeavingChain][row.DestinationChain]; !ok {
			result[row.LeavingChain][row.DestinationChain] = map[string]float64{"*": 0}
		}
		if _, ok := result["*"][row.DestinationChain]; !ok {
			result["*"][row.DestinationChain] = map[string]float64{"*": 0}
		}
		// add the transfer data to the result set every possible way:
		// by symbol, aggregated by: "leaving chain", "arriving at chain", "from any chain", "to any chain".

		// add to the total amount leaving this chain, going to any chain, for all symbols
		result[row.LeavingChain]["*"]["*"] = result[row.LeavingChain]["*"]["*"] + row.Notional
		// add to the total amount leaving this chain, going to the destination chain, for all symbols
		result[row.LeavingChain][row.DestinationChain]["*"] = result[row.LeavingChain][row.DestinationChain]["*"] + row.Notional
		// add to the total amount of this symbol leaving this chain, going to any chain
		result[row.LeavingChain]["*"][row.TokenSymbol] = result[row.LeavingChain]["*"][row.TokenSymbol] + row.Notional
		// add to the total amount of this symbol leaving this chain, going to the destination chain
		result[row.LeavingChain][row.DestinationChain][row.TokenSymbol] = result[row.LeavingChain][row.DestinationChain][row.TokenSymbol] + row.Notional

		// add to the total amount arriving at the destination chain, coming from anywhere, including all symbols
		result["*"][row.DestinationChain]["*"] = result["*"][row.DestinationChain]["*"] + row.Notional
		// add to the total amount of this symbol arriving at the destination chain
		result["*"][row.DestinationChain][row.TokenSymbol] = result["*"][row.DestinationChain][row.TokenSymbol] + row.Notional
		// add to the total amount of this symbol transferred, from any chain, to any chain
		result["*"]["*"][row.TokenSymbol] = result["*"]["*"][row.TokenSymbol] + row.Notional
		// and finally, total/total/total: amount of all symbols transferred from any chain to any other chain
		result["*"]["*"]["*"] = result["*"]["*"]["*"] + row.Notional
	}

	// create a set of chainIDs, the union of source and destination chains,
	// to ensure the result objects all have the same keys.
	seenChainSet := map[string]bool{}
	for leaving, dests := range result {
		seenChainSet[leaving] = true
		for dest := range dests {
			seenChainSet[dest] = true
		}
	}

	// make sure the root of the map has all the chainIDs
	for chain := range seenChainSet {
		if _, ok := result[chain]; !ok {
			result[chain] = map[string]map[string]float64{"*": {"*": 0}}
		}
	}
	// make sure that each chain at the root (leaving) as a key (destination) for each chain
	for leaving, dests := range result {
		for chain := range seenChainSet {
			// check that date has all the chains
			if _, ok := dests[chain]; !ok {
				result[leaving][chain] = map[string]float64{"*": 0}
			}
		}
	}
	return result
}

// finds the value that has been transferred from each chain to each other, by symbol.
func NotionalTransferred(w http.ResponseWriter, r *http.Request) {
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

	var numDays, forChain, forAddress, daily, last24Hours, forPeriod string

	// allow GET requests with querystring params, or POST requests with json body.
	switch r.Method {
	case http.MethodGet:
		queryParams := r.URL.Query()
		numDays = queryParams.Get("numDays")
		forChain = queryParams.Get("forChain")
		forAddress = queryParams.Get("forAddress")
		daily = queryParams.Get("daily")
		last24Hours = queryParams.Get("last24Hours")
		forPeriod = queryParams.Get("forPeriod")

	case http.MethodPost:
		// declare request body properties
		var d struct {
			NumDays     string `json:"numDays"`
			ForChain    string `json:"forChain"`
			ForAddress  string `json:"forAddress"`
			Daily       string `json:"daily"`
			Last24Hours string `json:"last24Hours"`
			ForPeriod   string `json:"forPeriod"`
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
		last24Hours = d.Last24Hours
		forPeriod = d.ForPeriod

	default:
		http.Error(w, "405 - Method Not Allowed", http.StatusMethodNotAllowed)
		log.Println("Method Not Allowed")
		return
	}

	if daily == "" && last24Hours == "" && forPeriod == "" {
		// none of the options were set, so set one
		last24Hours = "true"
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

	// total of last 24 hours
	last24HourCount := map[string]map[string]map[string]float64{}
	if last24Hours != "" {
		wg.Add(1)
		go func(prefix string) {
			last24HourInterval := -time.Duration(24) * time.Hour
			now := time.Now().UTC()
			start := now.Add(last24HourInterval)
			defer wg.Done()
			transfers := transfersForInterval(tbl, ctx, prefix, start, now)
			for chain, dests := range transfers {
				last24HourCount[chain] = map[string]map[string]float64{}
				for dest, tokens := range dests {
					last24HourCount[chain][dest] = map[string]float64{}
					for symbol, amount := range tokens {
						last24HourCount[chain][dest][symbol] = roundToTwoDecimalPlaces(amount)
					}
				}
			}

		}(prefix)
	}

	// transfers of the last numDays
	periodTransfers := map[string]map[string]map[string]float64{}
	if forPeriod != "" {
		wg.Add(1)
		go func(prefix string) {
			hours := (24 * queryDays)
			periodInterval := -time.Duration(hours) * time.Hour

			now := time.Now().UTC()
			prev := now.Add(periodInterval)
			start := time.Date(prev.Year(), prev.Month(), prev.Day(), 0, 0, 0, 0, prev.Location())

			defer wg.Done()
			transfers := transferredSinceDate(tbl, ctx, prefix, start)
			for chain, dests := range transfers {
				periodTransfers[chain] = map[string]map[string]float64{}
				for dest, tokens := range dests {
					periodTransfers[chain][dest] = map[string]float64{}
					for symbol, amount := range tokens {
						periodTransfers[chain][dest][symbol] = roundToTwoDecimalPlaces(amount)
					}
				}
			}
		}(prefix)
	}

	// daily totals
	dailyTransfers := map[string]map[string]map[string]map[string]float64{}
	if daily != "" {
		wg.Add(1)
		go func(prefix string, queryDays int) {
			hours := (24 * queryDays)
			periodInterval := -time.Duration(hours) * time.Hour
			now := time.Now().UTC()
			prev := now.Add(periodInterval)
			start := time.Date(prev.Year(), prev.Month(), prev.Day(), 0, 0, 0, 0, prev.Location())
			defer wg.Done()
			transfers := createTransfersOfInterval(tbl, ctx, prefix, start)
			for date, chains := range transfers {
				dailyTransfers[date] = map[string]map[string]map[string]float64{}
				for chain, dests := range chains {
					dailyTransfers[date][chain] = map[string]map[string]float64{}
					for destChain, tokens := range dests {
						dailyTransfers[date][chain][destChain] = map[string]float64{}
						for symbol, amount := range tokens {
							dailyTransfers[date][chain][destChain][symbol] = roundToTwoDecimalPlaces(amount)
						}
					}
				}
			}
		}(prefix, queryDays)
	}

	wg.Wait()

	result := &transfersResult{
		Last24Hours:        last24HourCount,
		WithinPeriod:       periodTransfers,
		PeriodDurationDays: queryDays,
		Daily:              dailyTransfers,
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
