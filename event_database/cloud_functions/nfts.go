// Package p contains an HTTP Cloud Function.
package p

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"cloud.google.com/go/bigtable"
)

// warmNFTCache keeps some data around between invocations, so that we don't have
// to do a full table scan with each request.
// https://cloud.google.com/functions/docs/bestpractices/tips#use_global_variables_to_reuse_objects_in_future_invocations
var warmNFTCache = map[string]map[string]map[string]int{}
var muWarmNFTCache sync.RWMutex
var warmNFTCacheFilePath = "nft-cache.json"

func fetchNFTRowsInInterval(tbl *bigtable.Table, ctx context.Context, prefix string, start, end time.Time) ([]bigtable.Row, error) {
	rows := []bigtable.Row{}

	err := tbl.ReadRows(ctx, bigtable.PrefixRange(prefix), func(row bigtable.Row) bool {

		rows = append(rows, row)

		return true

	}, bigtable.RowFilter(
		bigtable.ConditionFilter(
			bigtable.ChainFilters(
				bigtable.FamilyFilter(columnFamilies[1]),
				bigtable.CellsPerRowLimitFilter(1),        // only the first cell in column
				bigtable.TimestampRangeFilter(start, end), // within time range
				bigtable.StripValueFilter(),               // no columns/values, just the row.Key()
			),
			bigtable.ChainFilters(
				bigtable.FamilyFilter(columnFamilies[4]),
				bigtable.ColumnFilter("PayloadId"),
				bigtable.ValueFilter("1"),
			),
			bigtable.BlockAllFilter(),
		)))
	return rows, err
}

func createNFTCountsOfInterval(tbl *bigtable.Table, ctx context.Context, prefix string, numPrevDays int, keySegments int) (map[string]map[string]int, error) {
	if _, ok := warmNFTCache["2021-09-13"]; !ok && loadCache {
		loadJsonToInterface(ctx, warmNFTCacheFilePath, &muWarmNFTCache, &warmNFTCache)
	}

	results := map[string]map[string]int{}

	now := time.Now().UTC()

	var intervalsWG sync.WaitGroup
	// there will be a query for each previous day, plus today
	intervalsWG.Add(numPrevDays + 1)

	// create the unique identifier for this query, for cache
	cachePrefix := prefix
	if prefix == "" {
		cachePrefix = "*"
	}
	cachePrefix = fmt.Sprintf("%v-%v", cachePrefix, keySegments)
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

			muWarmNFTCache.Lock()
			// initialize the map for this date in the result set
			results[dateStr] = map[string]int{"*": 0}
			// check to see if there is cache data for this date/query
			if dateCache, ok := warmNFTCache[dateStr]; ok && useCache(dateStr) {
				// have a cache for this date

				if val, ok := dateCache[cachePrefix]; ok {
					// have a cache for this query
					if daysAgo >= 1 {
						// only use the cache for yesterday and older
						results[dateStr] = val
						muWarmNFTCache.Unlock()
						intervalsWG.Done()
						return
					}
				} else {
					// no cache for this query
					warmNFTCache[dateStr][cachePrefix] = map[string]int{}
				}
			} else {
				// no cache for this date, initialize the map
				warmNFTCache[dateStr] = map[string]map[string]int{}
				warmNFTCache[dateStr][cachePrefix] = map[string]int{}
			}
			muWarmNFTCache.Unlock()

			var result []bigtable.Row
			var fetchErr error

			defer intervalsWG.Done()
			result, fetchErr = fetchNFTRowsInInterval(tbl, ctx, prefix, start, end)

			if fetchErr != nil {
				log.Fatalf("fetchNFTRowsInInterval returned an error: %v", fetchErr)
			}

			// iterate through the rows and increment the count
			for _, row := range result {
				countBy := makeGroupKey(keySegments, row.Key())
				if keySegments != 0 {
					// increment the total count
					results[dateStr]["*"] = results[dateStr]["*"] + 1
				}
				results[dateStr][countBy] = results[dateStr][countBy] + 1
			}

			if cacheData, ok := warmNFTCache[dateStr][cachePrefix]; !ok || len(cacheData) <= 1 {
				// set the result in the cache
				muWarmNFTCache.Lock()
				warmNFTCache[dateStr][cachePrefix] = results[dateStr]
				muWarmNFTCache.Unlock()
				cacheNeedsUpdate = true
			}
		}(tbl, ctx, prefix, daysAgo)
	}

	intervalsWG.Wait()

	if cacheNeedsUpdate {
		persistInterfaceToJson(ctx, warmNFTCacheFilePath, &muWarmNFTCache, warmNFTCache)
	}

	// create a set of all the keys from all dates, to ensure the result objects all have the same keys
	seenKeySet := map[string]bool{}
	for _, v := range results {
		for key := range v {
			seenKeySet[key] = true
		}
	}
	// ensure each date object has the same keys:
	for date := range results {
		for key := range seenKeySet {
			if _, ok := results[date][key]; !ok {
				// add the missing key to the map
				results[date][key] = 0
			}
		}
	}

	return results, nil
}

// returns the count of the rows in the query response
func nftMessageCountForInterval(tbl *bigtable.Table, ctx context.Context, prefix string, start, end time.Time, keySegments int) (map[string]int, error) {
	// query for all rows in time range, return result count
	results, fetchErr := fetchNFTRowsInInterval(tbl, ctx, prefix, start, end)
	if fetchErr != nil {
		log.Printf("fetchRowsInInterval returned an error: %v", fetchErr)
		return nil, fetchErr
	}

	result := map[string]int{"*": len(results)}

	// iterate through the rows and increment the count for each index
	if keySegments != 0 {
		for _, row := range results {
			countBy := makeGroupKey(keySegments, row.Key())
			result[countBy] = result[countBy] + 1
		}
	}
	return result, nil
}

// get number of recent transactions in the last 24 hours, and daily for a period
// optionally group by a EmitterChain or EmitterAddress
// optionally query for recent rows of a given EmitterChain or EmitterAddress
func NFTs(w http.ResponseWriter, r *http.Request) {
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

	var last24Hours, numDays, groupBy, forChain, forAddress string

	// allow GET requests with querystring params, or POST requests with json body.
	switch r.Method {
	case http.MethodGet:
		queryParams := r.URL.Query()
		last24Hours = queryParams.Get("last24Hours")
		numDays = queryParams.Get("numDays")
		groupBy = queryParams.Get("groupBy")
		forChain = queryParams.Get("forChain")
		forAddress = queryParams.Get("forAddress")

		readyCheck := queryParams.Get("readyCheck")
		if readyCheck != "" {
			// for running in devnet
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, html.EscapeString("ready"))
			return
		}

	case http.MethodPost:
		// declare request body properties
		var d struct {
			Last24Hours string `json:"last24Hours"`
			NumDays     string `json:"numDays"`
			GroupBy     string `json:"groupBy"`
			ForChain    string `json:"forChain"`
			ForAddress  string `json:"forAddress"`
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
		numDays = d.NumDays
		groupBy = d.GroupBy
		forChain = d.ForChain
		forAddress = d.ForAddress

	default:
		http.Error(w, "405 - Method Not Allowed", http.StatusMethodNotAllowed)
		log.Println("Method Not Allowed")
		return
	}

	// default query period is all time
	queryDays := int(time.Now().UTC().Sub(releaseDay).Hours() / 24)

	// if the request included numDays, set the query period to that
	if numDays != "" {
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
		if groupBy == "" {
			// if the request is forChain, and groupBy is empty, set it to groupBy chain
			groupBy = "chain"
		}
		if forAddress != "" {
			// if the request is forAddress, always groupBy address
			groupBy = "address"
			prefix = forChain + ":" + forAddress
		}
	}

	// use the groupBy value to determine how many segements of the rowkey should be used.
	keySegments := 0
	if groupBy == "chain" {
		keySegments = 1
	}
	if groupBy == "address" {
		keySegments = 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// total of last 24 hours
	var last24HourCount map[string]int
	if last24Hours != "" {
		wg.Add(1)
		go func(prefix string, keySegments int) {
			var err error
			last24HourInterval := -time.Duration(24) * time.Hour
			now := time.Now().UTC()
			start := now.Add(last24HourInterval)
			defer wg.Done()
			last24HourCount, err = nftMessageCountForInterval(tbl, ctx, prefix, start, now, keySegments)
			if err != nil {
				log.Printf("failed getting count for interval, err: %v", err)
			}
		}(prefix, keySegments)
	}

	periodTotals := map[string]int{}
	var dailyTotals map[string]map[string]int
	wg.Add(1)
	go func(prefix string, keySegments int, queryDays int) {
		var err error
		defer wg.Done()
		dailyTotals, err = createNFTCountsOfInterval(tbl, ctx, prefix, queryDays, keySegments)
		if err != nil {
			log.Fatalf("failed getting createNFTCountsOfInterval err %v", err)
		}
		// sum all the days to create a map with totals for the query period
		for _, vals := range dailyTotals {
			for chain, amount := range vals {
				periodTotals[chain] += amount
			}
		}
	}(prefix, keySegments, queryDays)

	wg.Wait()

	result := &totalsResult{
		LastDayCount:           last24HourCount,
		TotalCount:             periodTotals,
		TotalCountDurationDays: queryDays,
		DailyTotals:            dailyTotals,
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
