// Package p contains an HTTP Cloud Function.
package p

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"cloud.google.com/go/bigtable"
)

type transfersFromResult struct {
	Daily map[string]map[string]float64
	Total float64
}

// an in-memory cache of previously calculated results
var transfersFromCache transfersFromResult
var muTransfersFromCache sync.RWMutex
var transfersFromFilePath = "notional-transferred-from.json"

// finds the daily amount transferred from each chain from the specified start to the present.
func createTransfersFromOfInterval(tbl *bigtable.Table, ctx context.Context, prefix string, start time.Time) {
	if len(transfersFromCache.Daily) == 0 && loadCache {
		loadJsonToInterface(ctx, transfersFromFilePath, &muTransfersFromCache, &transfersFromCache)
	}

	now := time.Now().UTC()
	numPrevDays := int(now.Sub(start).Hours() / 24)

	var intervalsWG sync.WaitGroup
	// there will be a query for each previous day, plus today
	intervalsWG.Add(numPrevDays + 1)

	for daysAgo := 0; daysAgo <= numPrevDays; daysAgo++ {
		go func(tbl *bigtable.Table, ctx context.Context, prefix string, daysAgo int) {
			defer intervalsWG.Done()
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

			muTransfersFromCache.Lock()
			// check to see if there is cache data for this date/query
			if _, ok := transfersFromCache.Daily[dateStr]; ok && useCache(dateStr) {
				// have a cache for this date
				if daysAgo >= 1 {
					// only use the cache for yesterday and older
					muTransfersFromCache.Unlock()
					return
				}
			}
			// no cache for this query, initialize the map
			if transfersFromCache.Daily == nil {
				transfersFromCache.Daily = map[string]map[string]float64{}
			}
			transfersFromCache.Daily[dateStr] = map[string]float64{"*": 0}
			muTransfersFromCache.Unlock()

			queryResult := fetchTransferRowsInInterval(tbl, ctx, prefix, start, end)

			// iterate through the rows and increment the amounts
			for _, row := range queryResult {
				if _, ok := transfersFromCache.Daily[dateStr][row.LeavingChain]; !ok {
					transfersFromCache.Daily[dateStr][row.LeavingChain] = 0
				}
				transfersFromCache.Daily[dateStr]["*"] = transfersFromCache.Daily[dateStr]["*"] + row.Notional
				transfersFromCache.Daily[dateStr][row.LeavingChain] = transfersFromCache.Daily[dateStr][row.LeavingChain] + row.Notional
			}
		}(tbl, ctx, prefix, daysAgo)
	}
	intervalsWG.Wait()

	// having consistent keys in each object is helpful for clients, explorer GUI
	transfersFromCache.Total = 0
	seenChainSet := map[string]bool{}
	for _, chains := range transfersFromCache.Daily {
		for chain, amount := range chains {
			seenChainSet[chain] = true
			if chain == "*" {
				transfersFromCache.Total += amount
			}
		}
	}
	for date, chains := range transfersFromCache.Daily {
		for chain := range seenChainSet {
			if _, ok := chains[chain]; !ok {
				transfersFromCache.Daily[date][chain] = 0
			}
		}
	}

	persistInterfaceToJson(ctx, transfersFromFilePath, &muTransfersFromCache, transfersFromCache)
}

// finds the value that has been transferred from each chain
func ComputeNotionalTransferredFrom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Set CORS headers for the preflight request
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ctx := context.Background()
	createTransfersFromOfInterval(tbl, ctx, "", releaseDay)

	w.WriteHeader(http.StatusOK)
}

func NotionalTransferredFrom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Set CORS headers for the preflight request
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var result transfersFromResult
	loadJsonToInterface(ctx, transfersFromFilePath, &muTransfersFromCache, &result)

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
