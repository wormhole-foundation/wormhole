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

type addressesResult struct {
	Last24HoursAmounts  map[string]map[string]float64
	Last24HoursCounts   map[string]int
	WithinPeriodAmounts map[string]map[string]float64
	WithinPeriodCounts  map[string]int
	PeriodDurationDays  int
	DailyAmounts        map[string]map[string]map[string]float64
	DailyCounts         map[string]map[string]int
}

// an in-memory cache of previously calculated results
var warmAddressesCache = map[string]map[string]map[string]map[string]float64{}
var muWarmAddressesCache sync.RWMutex
var warmAddressesCacheFilePath = "addresses-transferred-to-cache.json"

type AddressData struct {
	TokenSymbol        string
	TokenAmount        float64
	OriginChain        string
	LeavingChain       string
	DestinationChain   string
	DestinationAddress string
	Notional           float64
}

func fetchAddressRowsInInterval(tbl *bigtable.Table, ctx context.Context, prefix string, start, end time.Time) []AddressData {
	rows := []AddressData{}
	err := tbl.ReadRows(ctx, bigtable.PrefixRange(prefix), func(row bigtable.Row) bool {

		t := &AddressData{}
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
					case "TokenTransferPayload:TargetAddress":
						t.DestinationAddress = string(item.Value)
					}
				}
				t.DestinationAddress = transformHexAddressToNative(chainIdStringToType(t.DestinationChain), t.DestinationAddress)
			}

			keyParts := strings.Split(row.Key(), ":")
			t.LeavingChain = keyParts[0]

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
				bigtable.ColumnFilter("Amount|NotionalUSD|OriginSymbol|OriginChain|TargetChain|TargetAddress"),
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

// finds unique addresses tokens have been sent to, for each day since the start time passed in.
func createAddressesOfInterval(tbl *bigtable.Table, ctx context.Context, prefix string, start time.Time) map[string]map[string]map[string]float64 {
	if _, ok := warmAddressesCache["*"]; !ok && loadCache {
		loadJsonToInterface(ctx, warmAddressesCacheFilePath, &muWarmAddressesCache, &warmAddressesCache)
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

			muWarmAddressesCache.Lock()
			// initialize the map for this date in the result set
			results[dateStr] = map[string]map[string]float64{}
			// check to see if there is cache data for this date/query
			if dates, ok := warmAddressesCache[cachePrefix]; ok {
				// have a cache for this query

				if dateCache, ok := dates[dateStr]; ok && useCache(dateStr) {
					// have a cache for this date
					if daysAgo >= 1 {
						// only use the cache for yesterday and older
						results[dateStr] = dateCache
						muWarmAddressesCache.Unlock()
						intervalsWG.Done()
						return
					}
				}
			} else {
				// no cache for this query, initialize the map
				warmAddressesCache[cachePrefix] = map[string]map[string]map[string]float64{}
			}
			muWarmAddressesCache.Unlock()

			defer intervalsWG.Done()

			queryResult := fetchAddressRowsInInterval(tbl, ctx, prefix, start, end)

			// iterate through the rows and increment the count
			for _, row := range queryResult {
				if _, ok := results[dateStr][row.DestinationChain]; !ok {
					results[dateStr][row.DestinationChain] = map[string]float64{}
				}
				results[dateStr][row.DestinationChain][row.DestinationAddress] = results[dateStr][row.DestinationChain][row.DestinationAddress] + row.Notional

			}

			if daysAgo >= 1 {
				// set the result in the cache
				muWarmAddressesCache.Lock()
				if _, ok := warmAddressesCache[cachePrefix][dateStr]; !ok || !useCache(dateStr) {
					// cache does not have this date, persist it for other instances.
					warmAddressesCache[cachePrefix][dateStr] = results[dateStr]
					cacheNeedsUpdate = true
				}
				muWarmAddressesCache.Unlock()
			}
		}(tbl, ctx, prefix, daysAgo)
	}

	intervalsWG.Wait()

	if cacheNeedsUpdate {
		persistInterfaceToJson(ctx, warmAddressesCacheFilePath, &muWarmAddressesCache, warmAddressesCache)
	}

	// create a set of all the keys from all dates/chains, to ensure the result objects all have the same keys
	seenChainSet := map[string]bool{}
	for _, chains := range results {
		for leaving := range chains {
			seenChainSet[leaving] = true
		}
	}
	// ensure each chain object has all the same symbol keys:
	for date := range results {
		for chain := range seenChainSet {
			// check that date has all the chains
			if _, ok := results[date][chain]; !ok {
				results[date][chain] = map[string]float64{}
			}
		}
	}

	return results
}

// finds all the unique addresses that have received tokens since a particular moment.
func addressesTransferredToSinceDate(tbl *bigtable.Table, ctx context.Context, prefix string, start time.Time) map[string]map[string]float64 {

	result := map[string]map[string]float64{}

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

	return result
}

// returns addresses that received tokens within the specified time range
func addressesForInterval(tbl *bigtable.Table, ctx context.Context, prefix string, start, end time.Time) map[string]map[string]float64 {
	// query for all rows in time range, return result count
	queryResult := fetchAddressRowsInInterval(tbl, ctx, prefix, start, end)

	result := map[string]map[string]float64{}

	// iterate through the rows and increment the count for each index
	for _, row := range queryResult {
		if _, ok := result[row.DestinationChain]; !ok {
			result[row.DestinationChain] = map[string]float64{}
		}
		result[row.DestinationChain][row.DestinationAddress] = result[row.DestinationChain][row.DestinationAddress] + row.Notional
	}
	return result
}

// find the addresses tokens have been transferred to
func AddressesTransferredTo(w http.ResponseWriter, r *http.Request) {
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

	var numDays, forChain, forAddress, daily, last24Hours, forPeriod, counts, amounts string

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
		counts = queryParams.Get("counts")
		amounts = queryParams.Get("amounts")

	case http.MethodPost:
		// declare request body properties
		var d struct {
			NumDays     string `json:"numDays"`
			ForChain    string `json:"forChain"`
			ForAddress  string `json:"forAddress"`
			Daily       string `json:"daily"`
			Last24Hours string `json:"last24Hours"`
			ForPeriod   string `json:"forPeriod"`
			Counts      string `json:"counts"`
			Amounts     string `json:"amounts"`
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
		counts = d.Counts
		amounts = d.Amounts

	default:
		http.Error(w, "405 - Method Not Allowed", http.StatusMethodNotAllowed)
		log.Println("Method Not Allowed")
		return
	}

	if daily == "" && last24Hours == "" && forPeriod == "" {
		// none of the options were set, so set one
		last24Hours = "true"
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

	// total of last 24 hours
	last24HourAmounts := map[string]map[string]float64{}
	last24HourCounts := map[string]int{}
	if last24Hours != "" {
		wg.Add(1)
		go func(prefix string) {

			last24HourInterval := -time.Duration(24) * time.Hour
			now := time.Now().UTC()
			start := now.Add(last24HourInterval)
			defer wg.Done()
			last24HourAddresses := addressesForInterval(tbl, ctx, prefix, start, now)
			if amounts != "" {
				for chain, addresses := range last24HourAddresses {
					last24HourAmounts[chain] = map[string]float64{}
					for address, amount := range addresses {
						last24HourAmounts[chain][address] = roundToTwoDecimalPlaces(amount)
					}
				}
			}

			if counts != "" {
				for chain, addresses := range last24HourAddresses {
					// need to sum all the chains to get the total count of addresses,
					// since addresses are not unique across chains.
					numAddresses := len(addresses)
					last24HourCounts[chain] = numAddresses
					last24HourCounts["*"] = last24HourCounts["*"] + numAddresses
				}
			}
		}(prefix)
	}

	// total of the last numDays
	addressesDailyAmounts := map[string]map[string]float64{}
	addressesDailyCounts := map[string]int{}
	if forPeriod != "" {
		wg.Add(1)
		go func(prefix string) {
			hours := (24 * queryDays)
			periodInterval := -time.Duration(hours) * time.Hour

			now := time.Now().UTC()
			prev := now.Add(periodInterval)
			start := time.Date(prev.Year(), prev.Month(), prev.Day(), 0, 0, 0, 0, prev.Location())

			defer wg.Done()
			// periodAmounts, err := addressesTransferredToSince(tbl, ctx, prefix, start)
			periodAmounts := addressesTransferredToSinceDate(tbl, ctx, prefix, start)

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
					addressesDailyCounts[chain] = numAddresses
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
			dailyTotals := createAddressesOfInterval(tbl, ctx, prefix, start)

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

	result := &addressesResult{
		Last24HoursAmounts:  last24HourAmounts,
		Last24HoursCounts:   last24HourCounts,
		WithinPeriodAmounts: addressesDailyAmounts,
		WithinPeriodCounts:  addressesDailyCounts,
		PeriodDurationDays:  queryDays,
		DailyAmounts:        dailyAmounts,
		DailyCounts:         dailyCounts,
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)

}
