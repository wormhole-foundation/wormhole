// Package p contains an HTTP Cloud Function.
package p

import (
	"bytes"
	"context"
	"encoding/binary"
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

// warmCache keeps some data around between invocations, so that we don't have
// to do a full table scan with each request.
// https://cloud.google.com/functions/docs/bestpractices/tips#use_global_variables_to_reuse_objects_in_future_invocations
// TODO - make a struct for cache
var warmTransfersCache = map[string]map[string]map[string]map[string]map[string]float64{}

func fetchTransferRowsInInterval(tbl *bigtable.Table, ctx context.Context, prefix string, start, end time.Time) ([]TransferData, error) {
	rows := []TransferData{}
	err := tbl.ReadRows(ctx, bigtable.PrefixRange(prefix), func(row bigtable.Row) bool {

		t := &TransferData{}
		if _, ok := row[transferDetailsFam]; ok {
			for _, item := range row[transferDetailsFam] {
				switch item.Column {
				case "TokenTransferDetails:Amount":
					amount, _ := strconv.ParseFloat(string(item.Value), 64)
					t.TokenAmount = amount
				case "TokenTransferDetails:NotionalUSD":
					reader := bytes.NewReader(item.Value)
					var notionalFloat float64
					if err := binary.Read(reader, binary.BigEndian, &notionalFloat); err != nil {
						log.Fatalf("failed to read NotionalUSD of row: %v. err %v ", row.Key(), err)
					}
					t.Notional = notionalFloat
				case "TokenTransferDetails:OriginSymbol":
					t.TokenSymbol = string(item.Value)
				}
			}

			if _, ok := row[transferPayloadFam]; ok {
				for _, item := range row[transferPayloadFam] {
					switch item.Column {
					case "TokenTransferPayload:OriginChain":
						t.OriginChain = string(item.Value)
					case "TokenTransferPayload:TargetChain":
						t.DestinationChain = string(item.Value)
					}
				}
			}

			t.LeavingChain = row.Key()[:1]

			rows = append(rows, *t)
		}

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
				bigtable.FamilyFilter(fmt.Sprintf("%v|%v", columnFamilies[2], columnFamilies[5])),
				bigtable.ColumnFilter("Amount|NotionalUSD|OriginSymbol|OriginChain|TargetChain"),
				bigtable.LatestNFilter(1),
			),
			bigtable.BlockAllFilter(),
		),
	))
	if err != nil {
		fmt.Println("failed reading rows to create RowList.", err)
		return nil, err
	}
	return rows, err
}

func createTransfersOfInterval(tbl *bigtable.Table, ctx context.Context, prefix string, numPrevDays int) (map[string]map[string]map[string]map[string]float64, error) {
	var mu sync.RWMutex
	results := map[string]map[string]map[string]map[string]float64{}

	now := time.Now().UTC()

	var intervalsWG sync.WaitGroup
	// there will be a query for each previous day, plus today
	intervalsWG.Add(numPrevDays + 1)

	// create the unique identifier for this query, for cache
	cachePrefix := prefix
	if prefix == "" {
		cachePrefix = "*"
	}

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

			mu.Lock()
			// initialize the map for this date in the result set
			results[dateStr] = map[string]map[string]map[string]float64{"*": {"*": {"*": 0}}}
			// check to see if there is cache data for this date/query
			if dateCache, ok := warmTransfersCache[dateStr]; ok {
				// have a cache for this date

				if val, ok := dateCache[cachePrefix]; ok {
					// have a cache for this query
					if daysAgo >= 1 {
						// only use the cache for yesterday and older
						results[dateStr] = val
						mu.Unlock()
						intervalsWG.Done()
						return
					}
				} else {
					// no cache for this query
					warmTransfersCache[dateStr][cachePrefix] = map[string]map[string]map[string]float64{}
				}
			} else {
				// no cache for this date, initialize the map
				warmTransfersCache[dateStr] = map[string]map[string]map[string]map[string]float64{}
				warmTransfersCache[dateStr][cachePrefix] = map[string]map[string]map[string]float64{}
			}
			mu.Unlock()

			var result []TransferData
			var fetchErr error

			defer intervalsWG.Done()

			result, fetchErr = fetchTransferRowsInInterval(tbl, ctx, prefix, start, end)

			if fetchErr != nil {
				log.Fatalf("fetchTransferRowsInInterval returned an error: %v\n", fetchErr)
			}

			// iterate through the rows and increment the count
			for _, row := range result {
				if _, ok := results[dateStr][row.LeavingChain]; !ok {
					results[dateStr][row.LeavingChain] = map[string]map[string]float64{"*": {"*": 0}}
				}
				if _, ok := results[dateStr][row.LeavingChain][row.DestinationChain]; !ok {
					results[dateStr][row.LeavingChain][row.DestinationChain] = map[string]float64{"*": 0}
				}

				// add to the total count
				results[dateStr][row.LeavingChain]["*"]["*"] = results[dateStr][row.LeavingChain]["*"]["*"] + row.Notional
				results[dateStr][row.LeavingChain][row.DestinationChain]["*"] = results[dateStr][row.LeavingChain][row.DestinationChain]["*"] + row.Notional
				// add to the count for chain/symbol
				results[dateStr][row.LeavingChain][row.DestinationChain][row.TokenSymbol] = results[dateStr][row.LeavingChain][row.DestinationChain][row.TokenSymbol] + row.Notional

			}
			// set the result in the cache
			warmTransfersCache[dateStr][cachePrefix] = results[dateStr]
		}(tbl, ctx, prefix, daysAgo)
	}

	intervalsWG.Wait()

	// not sure if having consistent keys is helpful or not, commenting out until this endpoint is consumed by GUIs
	// // create a set of all the keys from all dates/chains/symbols, to ensure the result objects all have the same keys
	// seenKeySet := map[string]bool{}
	// for date, tokens := range results {
	// 	for leaving, dests := range tokens {
	// 		for dest := range dests {
	// 			for key := range results[date][leaving][dest] {
	// 				seenKeySet[key] = true
	// 			}
	// 		}
	// 	}
	// }
	// // ensure each chain object has all the same symbol keys:
	// for date := range results {
	// 	for leaving := range results[date] {
	// 		for dest := range results[date][leaving] {
	// 			for token := range seenKeySet {
	// 				if _, ok := results[date][leaving][token]; !ok {
	// 					// add the missing key to the map
	// 					results[date][leaving][dest][token] = 0
	// 				}
	// 			}
	// 		}
	// 	}
	// }

	return results, nil
}

// returns the count of the rows in the query response
func transfersForInterval(tbl *bigtable.Table, ctx context.Context, prefix string, start, end time.Time) (map[string]map[string]map[string]float64, error) {
	// query for all rows in time range, return result count
	results, fetchErr := fetchTransferRowsInInterval(tbl, ctx, prefix, start, end)
	if fetchErr != nil {
		log.Printf("fetchRowsInInterval returned an error: %v", fetchErr)
		return nil, fetchErr
	}
	var total = float64(0)
	for _, item := range results {
		total = total + item.Notional
	}

	result := map[string]map[string]map[string]float64{"*": {"*": {"*": total}}}

	// iterate through the rows and increment the count for each index
	for _, row := range results {
		if _, ok := result[row.LeavingChain]; !ok {
			result[row.LeavingChain] = map[string]map[string]float64{"*": {"*": 0}}
		}
		if _, ok := result[row.LeavingChain][row.DestinationChain]; !ok {
			result[row.LeavingChain][row.DestinationChain] = map[string]float64{"*": 0}
		}
		// add to total amount
		result[row.LeavingChain]["*"]["*"] = result[row.LeavingChain]["*"]["*"] + row.Notional
		result[row.LeavingChain][row.DestinationChain]["*"] = result[row.LeavingChain][row.DestinationChain]["*"] + row.Notional
		// add to symbol amount
		result[row.LeavingChain][row.DestinationChain][row.TokenSymbol] = result[row.LeavingChain][row.DestinationChain][row.TokenSymbol] + row.Notional
	}
	return result, nil
}

// get number of recent transactions in the last 24 hours, and daily for a period
// optionally group by a EmitterChain or EmitterAddress
// optionally query for recent rows of a given EmitterChain or EmitterAddress
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

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// total of last 24 hours
	var last24HourCount map[string]map[string]map[string]float64
	if last24Hours != "" {
		wg.Add(1)
		go func(prefix string) {
			var err error
			last24HourInterval := -time.Duration(24) * time.Hour
			now := time.Now().UTC()
			start := now.Add(last24HourInterval)
			defer wg.Done()
			last24HourCount, err = transfersForInterval(tbl, ctx, prefix, start, now)
			for chain, dests := range last24HourCount {
				for dest, tokens := range dests {
					for symbol, amount := range tokens {
						last24HourCount[chain][dest][symbol] = roundToTwoDecimalPlaces(amount)
					}
				}
			}
			if err != nil {
				log.Printf("failed getting count for 24h interval, err: %v", err)
			}
		}(prefix)
	}

	// total of the last numDays
	var periodCount map[string]map[string]map[string]float64
	if forPeriod != "" {
		wg.Add(1)
		go func(prefix string) {
			var err error
			hours := (24 * queryDays)
			periodInterval := -time.Duration(hours) * time.Hour

			now := time.Now().UTC()
			prev := now.Add(periodInterval)
			start := time.Date(prev.Year(), prev.Month(), prev.Day(), 0, 0, 0, 0, prev.Location())

			defer wg.Done()
			periodCount, err = transfersForInterval(tbl, ctx, prefix, start, now)
			for chain, dests := range periodCount {
				for dest, tokens := range dests {
					for symbol, amount := range tokens {
						periodCount[chain][dest][symbol] = roundToTwoDecimalPlaces(amount)
					}
				}
			}
			if err != nil {
				log.Printf("failed getting count for numDays interval, err: %v\n", err)
			}
		}(prefix)
	}

	// daily totals
	var dailyTotals map[string]map[string]map[string]map[string]float64
	if daily != "" {
		wg.Add(1)
		go func(prefix string, queryDays int) {
			var err error
			defer wg.Done()
			dailyTotals, err = createTransfersOfInterval(tbl, ctx, prefix, queryDays)
			for date, chains := range dailyTotals {
				for chain, dests := range chains {
					for destChain, tokens := range dests {
						for symbol, amount := range tokens {
							dailyTotals[date][chain][destChain][symbol] = roundToTwoDecimalPlaces(amount)
						}
					}
				}
			}
			if err != nil {
				log.Fatalf("failed getting createCountsOfInterval err %v", err)
			}
		}(prefix, queryDays)
	}

	wg.Wait()

	result := &transfersResult{
		Last24Hours:        last24HourCount,
		WithinPeriod:       periodCount,
		PeriodDurationDays: queryDays,
		Daily:              dailyTotals,
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
