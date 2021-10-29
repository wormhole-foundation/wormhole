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
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigtable"
)

// warmCache keeps some data around between invocations, so that we don't have
// to do a full table scan with each request.
// https://cloud.google.com/functions/docs/bestpractices/tips#use_global_variables_to_reuse_objects_in_future_invocations
var warmCache = map[string]map[string]string{}
var lastCacheReset = time.Now()

// query for last of each rowKey prefix
func getLatestOfEachEmitterAddress(tbl *bigtable.Table, ctx context.Context, prefix string, keySegments int) map[string]string {
	// get cache data for query
	cachePrefix := prefix
	if prefix == "" {
		cachePrefix = "*"
	}

	var rowSet bigtable.RowSet
	rowSet = bigtable.PrefixRange(prefix)
	now := time.Now()
	oneHourAgo := now.Add(-time.Duration(1) * time.Hour)
	if oneHourAgo.Before(lastCacheReset) {
		// cache is less than one hour old, use it
		if cached, ok := warmCache[cachePrefix]; ok {
			// use the highest possible sequence number as the range end.
			maxSeq := "9999999999999999"
			rowSets := bigtable.RowRangeList{}
			for k, v := range cached {
				start := fmt.Sprintf("%v:%v", k, v)
				end := fmt.Sprintf("%v:%v", k, maxSeq)
				rowSets = append(rowSets, bigtable.NewRange(start, end))
			}
			if len(rowSets) >= 1 {
				rowSet = rowSets
			}
		}
	} else {
		// cache is more than hour old, don't use it, reset it
		warmCache = map[string]map[string]string{}
		lastCacheReset = now
	}

	// create a time range for query: last 30 days
	thirtyDays := -time.Duration(24*30) * time.Hour
	prev := now.Add(thirtyDays)
	start := time.Date(prev.Year(), prev.Month(), prev.Day(), 0, 0, 0, 0, prev.Location())
	end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, maxNano, now.Location())

	mostRecentByKeySegment := map[string]string{}
	err := tbl.ReadRows(ctx, rowSet, func(row bigtable.Row) bool {

		keyParts := strings.Split(row.Key(), ":")
		groupByKey := strings.Join(keyParts[:2], ":")
		mostRecentByKeySegment[groupByKey] = keyParts[2]

		return true
	}, bigtable.RowFilter(
		bigtable.ChainFilters(
			bigtable.CellsPerRowLimitFilter(1),
			bigtable.TimestampRangeFilter(start, end),
			bigtable.StripValueFilter(),
		)))

	if err != nil {
		log.Fatalf("failed to read recent rows: %v", err)
	}
	// update the cache with the latest rows
	warmCache[cachePrefix] = mostRecentByKeySegment
	return mostRecentByKeySegment
}

func fetchMostRecentRows(tbl *bigtable.Table, ctx context.Context, prefix string, keySegments int, numRowsToFetch int) (map[string][]bigtable.Row, error) {
	// returns { key: []bigtable.Row }, key either being "*", "chainID", "chainID:address"

	latest := getLatestOfEachEmitterAddress(tbl, ctx, prefix, keySegments)

	// key/value pairs are the start/stop rowKeys for range queries
	rangePairs := map[string]string{}

	for prefixGroup, highestSequence := range latest {
		rowKeyParts := strings.Split(prefixGroup, ":")
		// convert the sequence part of the rowkey from a string to an int, so it can be used for math
		highSequence, _ := strconv.Atoi(highestSequence)
		lowSequence := highSequence - numRowsToFetch
		// create a rowKey to use as the start of the range query
		rangeQueryStart := fmt.Sprintf("%v:%v:%016d", rowKeyParts[0], rowKeyParts[1], lowSequence)
		// create a rowKey with the highest seen sequence + 1, because range end is exclusive
		rangeQueryEnd := fmt.Sprintf("%v:%v:%016d", rowKeyParts[0], rowKeyParts[1], highSequence+1)
		rangePairs[rangeQueryStart] = rangeQueryEnd
	}

	rangeList := bigtable.RowRangeList{}
	for k, v := range rangePairs {
		rangeList = append(rangeList, bigtable.NewRange(k, v))
	}

	results := map[string][]bigtable.Row{}

	err := tbl.ReadRows(ctx, rangeList, func(row bigtable.Row) bool {

		var groupByKey string
		if keySegments == 0 {
			groupByKey = "*"
		} else {
			keyParts := strings.Split(row.Key(), ":")
			groupByKey = strings.Join(keyParts[:keySegments], ":")
		}
		results[groupByKey] = append(results[groupByKey], row)
		return true
	})
	if err != nil {
		log.Printf("failed reading row ranges. err: %v", err)
		return nil, err
	}

	return results, nil
}

// fetch recent rows.
// optionally group by a EmitterChain or EmitterAddress
// optionally query for recent rows of a given EmitterChain or EmitterAddress
func Recent(w http.ResponseWriter, r *http.Request) {
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

	var numRows, groupBy, forChain, forAddress string

	// allow GET requests with querystring params, or POST requests with json body.
	switch r.Method {
	case http.MethodGet:
		queryParams := r.URL.Query()
		numRows = queryParams.Get("numRows")
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
			NumRows    string `json:"numRows"`
			GroupBy    string `json:"groupBy"`
			ForChain   string `json:"forChain"`
			ForAddress string `json:"forAddress"`
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

		numRows = d.NumRows
		groupBy = d.GroupBy
		forChain = d.ForChain
		forAddress = d.ForAddress

	default:
		http.Error(w, "405 - Method Not Allowed", http.StatusMethodNotAllowed)
		log.Println("Method Not Allowed")
		return
	}

	var resultCount int
	if numRows == "" {
		resultCount = 30
	} else {
		var convErr error
		resultCount, convErr = strconv.Atoi(numRows)
		if convErr != nil {
			fmt.Fprint(w, "numRows must be an integer")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}

	// use the groupBy value to determine how many segements of the rowkey should be used for indexing results.
	keySegments := 0
	if groupBy == "chain" {
		keySegments = 1
	}
	if groupBy == "address" {
		keySegments = 2
	}

	// create the rowkey prefix for querying, and the keySegments to use for indexing results.
	prefix := ""
	if forChain != "" {
		prefix = forChain
		if groupBy == "" {
			// groupBy was not set, but forChain was, so set the keySegments to index by chain
			keySegments = 1
		}
		if forAddress != "" {
			prefix = forChain + ":" + forAddress
			if groupBy == "" {
				// groupBy was not set, but forAddress was, so set the keySegments to index by address
				keySegments = 2
			}
		}
	}

	recent, err := fetchMostRecentRows(tbl, r.Context(), prefix, keySegments, resultCount)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Println(err.Error())
		return
	}
	res := map[string][]*Summary{}

	for k, v := range recent {
		sort.Slice(v, func(i, j int) bool {
			// bigtable rows dont have timestamps, use a cell timestamp all rows will have.
			var iTimestamp bigtable.Timestamp
			var jTimestamp bigtable.Timestamp
			// rows may have: only MessagePublication, only QuorumState, or both.
			// find a timestamp for each row, try to use MessagePublication, if it exists:
			if len(v[i]["MessagePublication"]) >= 1 {
				iTimestamp = v[i]["MessagePublication"][0].Timestamp
			} else if len(v[i]["QuorumState"]) >= 1 {
				iTimestamp = v[i]["QuorumState"][0].Timestamp
			}
			if len(v[j]["MessagePublication"]) >= 1 {
				jTimestamp = v[j]["MessagePublication"][0].Timestamp
			} else if len(v[j]["QuorumState"]) >= 1 {
				jTimestamp = v[j]["QuorumState"][0].Timestamp
			}
			return iTimestamp > jTimestamp
		})
		// trim the result down to the requested amount now that sorting is complete
		num := len(v)
		var rows []bigtable.Row
		if num > resultCount {
			rows = v[:resultCount]
		} else {
			rows = v[:]
		}

		res[k] = make([]*Summary, len(rows))
		for i, r := range rows {
			res[k][i] = makeSummary(r)
		}
	}

	jsonBytes, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Println(err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}
