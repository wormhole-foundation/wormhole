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
	"time"

	"cloud.google.com/go/bigtable"
)

// fetch rows by matching payload value
func FindValues(w http.ResponseWriter, r *http.Request) {
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

	var columnFamily, columnName, value, emitterChain, emitterAddress, vaaBytes, numRows string

	// allow GET requests with querystring params, or POST requests with json body.
	switch r.Method {
	case http.MethodGet:
		queryParams := r.URL.Query()
		columnFamily = queryParams.Get("columnFamily")
		columnName = queryParams.Get("columnName")
		value = queryParams.Get("value")
		emitterChain = queryParams.Get("emitterChain")
		emitterAddress = queryParams.Get("emitterAddress")
		vaaBytes = queryParams.Get("vaaBytes")
		numRows = queryParams.Get("numRows")

		// check for empty values
		if columnFamily == "" || columnName == "" || value == "" {
			fmt.Fprint(w, "query params ['columnFamily', 'columnName', 'value'] cannot be empty")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	case http.MethodPost:
		// declare request body properties
		var d struct {
			ColumnFamily   string `json:"columnFamily"`
			ColumnName     string `json:"columnName"`
			Value          string `json:"value"`
			EmitterChain   string `json:"emitterChain"`
			EmitterAddress string `json:"emitterAddress"`
			VAABytes       string `json:"vaaBytes"`
			NumRows        string `json:"numRows"`
		}

		// deserialize request body
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			switch err {
			case io.EOF:
				fmt.Fprint(w, "request body required")
				return
			default:
				log.Printf("json.NewDecoder: %v", err)
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		}

		// check for empty values
		if d.ColumnFamily == "" || d.ColumnName == "" || d.Value == "" {
			fmt.Fprint(w, "body values ['columnFamily', 'columnName', 'value'] cannot be empty")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		columnFamily = d.ColumnFamily
		columnName = d.ColumnName
		value = d.Value
		emitterChain = d.EmitterChain
		emitterAddress = d.EmitterAddress
		vaaBytes = d.VAABytes
		numRows = d.NumRows
	default:
		http.Error(w, "405 - Method Not Allowed", http.StatusMethodNotAllowed)
		log.Println("Method Not Allowed")
		return
	}

	var resultCount uint64
	if numRows == "" {
		resultCount = 0
	} else {
		var convErr error
		resultCount, convErr = strconv.ParseUint(numRows, 10, 64)
		if convErr != nil {
			fmt.Fprint(w, "numRows must be an integer")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}

	if columnFamily != "TokenTransferPayload" &&
		columnFamily != "AssetMetaPayload" &&
		columnFamily != "NFTTransferPayload" &&
		columnFamily != "TokenTransferDetails" &&
		columnFamily != "ChainDetails" {
		fmt.Fprint(w, "columnFamily must be one of: ['TokenTransferPayload', 'AssetMetaPayload', 'NFTTransferPayload', 'TokenTransferDetails', 'ChainDetails']")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	prefix := ""
	if emitterChain != "" {
		prefix = emitterChain
		if emitterAddress != "" {
			prefix = emitterChain + emitterAddress
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	results := []bigtable.Row{}
	err := tbl.ReadRows(ctx, bigtable.PrefixRange(prefix), func(row bigtable.Row) bool {
		results = append(results, row)
		return true
	}, bigtable.RowFilter(
		bigtable.ConditionFilter(
			bigtable.ChainFilters(
				bigtable.FamilyFilter(columnFamily),
				bigtable.ColumnFilter(columnName),
				bigtable.ValueFilter(value),
			),
			bigtable.ChainFilters(
				bigtable.PassAllFilter(),
				bigtable.LatestNFilter(1),
			),
			bigtable.BlockAllFilter(),
		)))

	if err != nil {
		http.Error(w, "Error reading rows", http.StatusInternalServerError)
		log.Printf("tbl.ReadRows(): %v", err)
		return
	}

	if resultCount > 0 {
		// means do not limit, cause you'd never query 0 rows.
		// if the result set is limited to a number, sort the results
		// and return the n latest.

		// sort the results to be newest first
		sort.Slice(results, func(i, j int) bool {
			// bigtable rows dont have timestamps, use a cell timestamp all rows will have.
			var iTimestamp bigtable.Timestamp
			var jTimestamp bigtable.Timestamp
			// rows may have: only MessagePublication, only QuorumState, or both.
			// find a timestamp for each row, try to use MessagePublication, if it exists:
			if len(results[i]["MessagePublication"]) >= 1 {
				iTimestamp = results[i]["MessagePublication"][0].Timestamp
			} else if len(results[i]["QuorumState"]) >= 1 {
				iTimestamp = results[i]["QuorumState"][0].Timestamp
			}
			if len(results[j]["MessagePublication"]) >= 1 {
				jTimestamp = results[j]["MessagePublication"][0].Timestamp
			} else if len(results[j]["QuorumState"]) >= 1 {
				jTimestamp = results[j]["QuorumState"][0].Timestamp
			}
			return iTimestamp > jTimestamp
		})

		// trim the result down to the requested amount
		num := uint64(len(results))
		if num > resultCount {
			results = results[:resultCount]
		} else {
			results = results[:]
		}
	}

	details := []Details{}
	for _, result := range results {
		detail := makeDetails(result)
		// create a slimmer version of the details struct
		slimDetails := Details{
			Summary: Summary{
				EmitterChain:    detail.EmitterChain,
				EmitterAddress:  detail.EmitterAddress,
				Sequence:        detail.Sequence,
				InitiatingTxID:  detail.InitiatingTxID,
				Payload:         detail.Payload,
				QuorumTime:      detail.QuorumTime,
				TransferDetails: detail.TransferDetails,
			},
			TokenTransferPayload: detail.TokenTransferPayload,
			AssetMetaPayload:     detail.AssetMetaPayload,
			NFTTransferPayload:   detail.NFTTransferPayload,
			ChainDetails:         detail.ChainDetails,
		}
		if vaaBytes != "" {
			slimDetails.SignedVAABytes = detail.SignedVAABytes
		}
		details = append(details, slimDetails)
	}
	jsonBytes, err := json.Marshal(details)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Println(err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}
