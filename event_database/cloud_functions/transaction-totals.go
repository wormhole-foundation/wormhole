// Package p contains an HTTP Cloud Function.
package p

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/bigtable"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type txTotals struct {
	DailyTotals map[string]map[string]int
}

var txTotalsResult txTotals
var txTotalsMutex sync.RWMutex
var txTotalsResultPath = "transaction-totals.json"

func fetchRowKeys(tbl *bigtable.Table, ctx context.Context, start, end time.Time) []string {
	rowKeys := []string{}
	chainIds := tvlChainIDs
	chainIds = append(chainIds, vaa.ChainIDPythNet)
	for _, chainId := range chainIds {
		err := tbl.ReadRows(ctx, bigtable.PrefixRange(chainIDRowPrefix(chainId)), func(row bigtable.Row) bool {
			rowKeys = append(rowKeys, row.Key())
			return true
		}, bigtable.RowFilter(
			bigtable.ChainFilters(
				bigtable.FamilyFilter(quorumStateFam),     // VAAs that have reached quorum
				bigtable.CellsPerRowLimitFilter(1),        // only the first cell in each column
				bigtable.TimestampRangeFilter(start, end), // within time range
				bigtable.StripValueFilter(),               // no columns/values, just the row.Key()
			)))
		if err != nil {
			log.Fatalf("fetchRowsInInterval returned an error: %v", err)
		}
	}
	return rowKeys
}

func updateTxTotalsResult(tbl *bigtable.Table, ctx context.Context, numPrevDays int) {
	if txTotalsResult.DailyTotals == nil {
		txTotalsResult.DailyTotals = map[string]map[string]int{}
		if loadCache {
			loadJsonToInterface(ctx, txTotalsResultPath, &txTotalsMutex, &txTotalsResult.DailyTotals)
		}
	}

	now := time.Now().UTC()

	var intervalsWG sync.WaitGroup
	// there will be a query for each previous day, plus today
	intervalsWG.Add(numPrevDays + 1)

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
			end := time.Date(year, month, day, 23, 59, 59, 999999999, loc)

			dateStr := start.Format("2006-01-02")

			txTotalsMutex.Lock()
			if daysAgo >= 1 {
				if _, ok := txTotalsResult.DailyTotals[dateStr]; ok && useCache(dateStr) {
					txTotalsMutex.Unlock()
					intervalsWG.Done()
					return
				}
			}
			txTotalsMutex.Unlock()

			defer intervalsWG.Done()
			result := fetchRowKeys(tbl, ctx, start, end)

			// iterate through the rows and increment the counts
			countsByDay := map[string]int{}
			countsByDay["*"] = 0
			for _, rowKey := range result {
				chainId := strings.Split(rowKey, ":")[0]
				if _, ok := countsByDay[chainId]; !ok {
					countsByDay[chainId] = 1
				} else {
					countsByDay[chainId] = countsByDay[chainId] + 1
				}
				countsByDay["*"] = countsByDay["*"] + 1
			}

			txTotalsMutex.Lock()
			txTotalsResult.DailyTotals[dateStr] = countsByDay
			txTotalsMutex.Unlock()

		}(tbl, ctx, daysAgo)
	}

	intervalsWG.Wait()

	// create a set of all the keys from all dates, to ensure the result objects all have the same keys
	seenKeySet := map[string]bool{}
	for _, v := range txTotalsResult.DailyTotals {
		for chainId := range v {
			seenKeySet[chainId] = true
		}
	}
	// ensure each date object has the same keys:
	for date := range txTotalsResult.DailyTotals {
		for chainId := range seenKeySet {
			if _, ok := txTotalsResult.DailyTotals[date][chainId]; !ok {
				// add the missing key to the map
				txTotalsResult.DailyTotals[date][chainId] = 0
			}
		}
	}

	persistInterfaceToJson(ctx, txTotalsResultPath, &txTotalsMutex, txTotalsResult.DailyTotals)
}

func ComputeTransactionTotals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	queryDays := int(time.Now().UTC().Sub(releaseDay).Hours() / 24)

	ctx := context.Background()

	var err error
	updateTxTotalsResult(tbl, ctx, queryDays)
	if err != nil {
		log.Fatalf("failed getting createCountsOfInterval err %v", err)
	}

	jsonBytes, err := json.Marshal(txTotalsResult)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Println(err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func TransactionTotals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ctx := context.Background()

	var cachedResult txTotals
	cachedResult.DailyTotals = map[string]map[string]int{}
	loadJsonToInterface(ctx, txTotalsResultPath, &txTotalsMutex, &cachedResult.DailyTotals)

	jsonBytes, err := json.Marshal(cachedResult)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Println(err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}
