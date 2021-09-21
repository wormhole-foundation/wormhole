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
	"os"
	"sort"
	"strconv"
	"strings"

	"cloud.google.com/go/bigtable"
)

// query for last of each rowKey prefix
func getLatestOfEachEmitterAddress(tbl *bigtable.Table, ctx context.Context, prefix string, keySegments int) map[string]string {

	mostRecentByKeySegment := map[string]string{}
	err := tbl.ReadRows(ctx, bigtable.PrefixRange(prefix), func(row bigtable.Row) bool {

		keyParts := strings.Split(row.Key(), ":")
		groupByKey := strings.Join(keyParts[:2], ":")
		mostRecentByKeySegment[groupByKey] = row.Key()

		return true
		// TODO - add filter to only return rows created within the last 30(?) days
	}, bigtable.RowFilter(bigtable.StripValueFilter()))

	if err != nil {
		log.Fatalf("failed to read recent rows: %v", err)
	}
	return mostRecentByKeySegment
}

func fetchMostRecentRows(tbl *bigtable.Table, ctx context.Context, prefix string, keySegments int, numRowsToFetch int) (map[string][]bigtable.Row, error) {
	// returns { key: []bigtable.Row }, key either being "*", "chainID", "chainID:address"

	latest := getLatestOfEachEmitterAddress(tbl, ctx, prefix, keySegments)

	// key/value pairs are the start/stop rowKeys for range queries
	rangePairs := map[string]string{}

	for _, highestSequenceKey := range latest {
		rowKeyParts := strings.Split(highestSequenceKey, ":")
		// convert the sequence part of the rowkey from a string to an int, so it can be used for math
		highSequence, _ := strconv.Atoi(rowKeyParts[2])
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

	// create bibtable client and open table
	clientOnce.Do(func() {
		// Declare a separate err variable to avoid shadowing client.
		var err error
		project := os.Getenv("GCP_PROJECT")
		instance := os.Getenv("BIGTABLE_INSTANCE")
		client, err = bigtable.NewClient(context.Background(), project, instance)
		if err != nil {
			http.Error(w, "Error initializing client", http.StatusInternalServerError)
			log.Printf("bigtable.NewClient: %v", err)
			return
		}
	})
	tbl := client.Open("v2Events")

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
			return v[i]["MessagePublication"][0].Timestamp > v[j]["MessagePublication"][0].Timestamp
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
