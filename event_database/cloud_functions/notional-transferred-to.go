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
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/bigtable"
)

type amountsResult struct {
	Last24Hours        map[string]map[string]float64
	WithinPeriod       map[string]map[string]float64
	PeriodDurationDays int
	Daily              map[string]map[string]map[string]float64
}

// an in-memory cache of previously calculated results
var warmTransfersToCache = map[string]map[string]map[string]map[string]float64{}
var muWarmTransfersToCache sync.RWMutex
var warmTransfersToCacheFilePath = "notional-transferred-to-cache.json"

type TransferData struct {
	TokenSymbol       string
	TokenName         string
	TokenAddress      string
	TokenAmount       float64
	CoinGeckoCoinId   string
	OriginChain       string
	LeavingChain      string
	DestinationChain  string
	Notional          float64
	TokenPrice        float64
	TokenDecimals     int
	TransferTimestamp string
}

// finds all the TokenTransfer rows within the specified period
func fetchTransferRowsInInterval(tbl *bigtable.Table, ctx context.Context, prefix string, start, end time.Time) []TransferData {
	if len(tokenAllowlist) == 0 {
		log.Fatal("tokenAllowlist is empty")
	}
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
				case "TokenTransferDetails:TokenPriceUSD":
					reader := bytes.NewReader(item.Value)
					var tokenPriceFloat float64
					if err := binary.Read(reader, binary.BigEndian, &tokenPriceFloat); err != nil {
						log.Fatalf("failed to read TokenPriceUSD of row: %v. err %v ", row.Key(), err)
					}
					t.TokenPrice = tokenPriceFloat
				case "TokenTransferDetails:OriginSymbol":
					t.TokenSymbol = string(item.Value)
				case "TokenTransferDetails:OriginName":
					t.TokenName = string(item.Value)
				case "TokenTransferDetails:OriginTokenAddress":
					t.TokenAddress = string(item.Value)
				case "TokenTransferDetails:CoinGeckoCoinId":
					t.CoinGeckoCoinId = string(item.Value)
				case "TokenTransferDetails:Decimals":
					t.TokenDecimals, _ = strconv.Atoi(string(item.Value))
				case "TokenTransferDetails:TransferTimestamp":
					t.TransferTimestamp = string(item.Value)
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

			keyParts := strings.Split(row.Key(), ":")
			t.LeavingChain = keyParts[0]

			transferDateStr := t.TransferTimestamp[0:10]
			if isTokenAllowed(t.OriginChain, t.TokenAddress) && isTokenActive(t.OriginChain, t.TokenAddress, transferDateStr) {
				rows = append(rows, *t)
			}
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
				bigtable.FamilyFilter(fmt.Sprintf("%v|%v", transferPayloadFam, transferDetailsFam)),
				bigtable.ColumnFilter("Amount|NotionalUSD|OriginSymbol|OriginName|OriginChain|TargetChain|CoinGeckoCoinId|OriginTokenAddress|TokenPriceUSD|Decimals|TransferTimestamp"),
				bigtable.LatestNFilter(1),
			),
			bigtable.BlockAllFilter(),
		),
	))
	if err != nil {
		log.Fatalln("failed reading rows to create RowList.", err)
	}
	return rows
}

// finds the daily amount of each symbol transferred to each chain, from the specified start to the present.
func amountsTransferredToInInterval(tbl *bigtable.Table, ctx context.Context, prefix string, start time.Time) map[string]map[string]map[string]float64 {
	if _, ok := warmTransfersToCache["*"]; !ok && loadCache {
		loadJsonToInterface(ctx, warmTransfersToCacheFilePath, &muWarmTransfersToCache, &warmTransfersToCache)
	}

	results := map[string]map[string]map[string]float64{}

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

			muWarmTransfersToCache.Lock()
			// initialize the map for this date in the result set
			results[dateStr] = map[string]map[string]float64{"*": {"*": 0}}
			// check to see if there is cache data for this date/query
			if dates, ok := warmTransfersToCache[cachePrefix]; ok {
				// have a cache for this query

				if dateCache, ok := dates[dateStr]; ok && len(dateCache) > 1 && useCache(dateStr) {
					// have a cache for this date
					if daysAgo >= 1 {
						// only use the cache for yesterday and older
						results[dateStr] = dateCache
						muWarmTransfersToCache.Unlock()
						intervalsWG.Done()
						return
					}
				}
			} else {
				// no cache for this query, initialize the map
				warmTransfersToCache[cachePrefix] = map[string]map[string]map[string]float64{}
			}
			muWarmTransfersToCache.Unlock()

			defer intervalsWG.Done()

			queryResult := fetchTransferRowsInInterval(tbl, ctx, prefix, start, end)

			// iterate through the rows and increment the count
			for _, row := range queryResult {
				if _, ok := results[dateStr][row.DestinationChain]; !ok {
					results[dateStr][row.DestinationChain] = map[string]float64{"*": 0}
				}
				// add to the total count for the dest chain
				results[dateStr][row.DestinationChain]["*"] = results[dateStr][row.DestinationChain]["*"] + row.Notional
				// add to total for the day
				results[dateStr]["*"]["*"] = results[dateStr]["*"]["*"] + row.Notional
				// add to the symbol's daily total
				results[dateStr]["*"][row.TokenSymbol] = results[dateStr]["*"][row.TokenSymbol] + row.Notional
				// add to the count for chain/symbol
				results[dateStr][row.DestinationChain][row.TokenSymbol] = results[dateStr][row.DestinationChain][row.TokenSymbol] + row.Notional
			}
			if daysAgo >= 1 {
				// set the result in the cache
				muWarmTransfersToCache.Lock()
				if cacheData, ok := warmTransfersToCache[cachePrefix][dateStr]; !ok || len(cacheData) <= 1 || !useCache(dateStr) {
					// cache does not have this date, persist it for other instances.
					warmTransfersToCache[cachePrefix][dateStr] = results[dateStr]
					cacheNeedsUpdate = true
				}
				muWarmTransfersToCache.Unlock()
			}
		}(tbl, ctx, prefix, daysAgo)
	}

	intervalsWG.Wait()

	if cacheNeedsUpdate {
		persistInterfaceToJson(ctx, warmTransfersToCacheFilePath, &muWarmTransfersToCache, warmTransfersToCache)
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
				results[date][chain] = map[string]float64{"*": 0}
				muResult.Unlock()
			}
		}
	}

	return results
}

func transferredToSinceDate(tbl *bigtable.Table, ctx context.Context, prefix string, start time.Time) map[string]map[string]float64 {
	result := map[string]map[string]float64{"*": {"*": 0}}

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

	return result
}

// returns the count of the rows in the query response
func transfersToForInterval(tbl *bigtable.Table, ctx context.Context, prefix string, start, end time.Time) map[string]map[string]float64 {
	// query for all rows in time range, return result count
	queryResults := fetchTransferRowsInInterval(tbl, ctx, prefix, start, end)

	result := map[string]map[string]float64{"*": {"*": 0}}

	// iterate through the rows and increment the count for each index
	for _, row := range queryResults {
		if _, ok := result[row.DestinationChain]; !ok {
			result[row.DestinationChain] = map[string]float64{"*": 0}
		}
		// add to total amount
		result[row.DestinationChain]["*"] = result[row.DestinationChain]["*"] + row.Notional
		// add to total per symbol
		result["*"][row.TokenSymbol] = result["*"][row.TokenSymbol] + row.Notional
		// add to symbol amount
		result[row.DestinationChain][row.TokenSymbol] = result[row.DestinationChain][row.TokenSymbol] + row.Notional
		// add to all chains/all symbols total
		result["*"]["*"] = result["*"]["*"] + row.Notional
	}
	return result
}

// finds the value that has been transferred to each chain, by symbol.
func NotionalTransferredTo(w http.ResponseWriter, r *http.Request) {
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
	last24HourCount := map[string]map[string]float64{}
	if last24Hours != "" {
		wg.Add(1)
		go func(prefix string) {
			last24HourInterval := -time.Duration(24) * time.Hour
			now := time.Now().UTC()
			start := now.Add(last24HourInterval)
			defer wg.Done()
			transfers := transfersToForInterval(tbl, ctx, prefix, start, now)
			for chain, tokens := range transfers {
				last24HourCount[chain] = map[string]float64{}
				for symbol, amount := range tokens {
					last24HourCount[chain][symbol] = roundToTwoDecimalPlaces(amount)
				}
			}
		}(prefix)
	}

	// total of the last numDays
	periodTransfers := map[string]map[string]float64{}
	if forPeriod != "" {
		wg.Add(1)
		go func(prefix string) {
			hours := (24 * queryDays)
			periodInterval := -time.Duration(hours) * time.Hour

			now := time.Now().UTC()
			prev := now.Add(periodInterval)
			start := time.Date(prev.Year(), prev.Month(), prev.Day(), 0, 0, 0, 0, prev.Location())

			defer wg.Done()
			// periodCount, err = transferredToSince(tbl, ctx, prefix, start)
			// periodCount, err = transfersToForInterval(tbl, ctx, prefix, start, now)
			transfers := transferredToSinceDate(tbl, ctx, prefix, start)
			for chain, tokens := range transfers {
				periodTransfers[chain] = map[string]float64{}
				for symbol, amount := range tokens {
					periodTransfers[chain][symbol] = roundToTwoDecimalPlaces(amount)
				}
			}
		}(prefix)
	}

	// daily totals
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
			transfers := amountsTransferredToInInterval(tbl, ctx, prefix, start)
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

	result := &amountsResult{
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
