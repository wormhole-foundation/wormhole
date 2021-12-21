// Package p contains an HTTP Cloud Function.
package p

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

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

	var columnFamily, columnName, value, emitterChain, emitterAddress string

	// allow GET requests with querystring params, or POST requests with json body.
	switch r.Method {
	case http.MethodGet:
		queryParams := r.URL.Query()
		columnFamily = queryParams.Get("columnFamily")
		columnName = queryParams.Get("columnName")
		value = queryParams.Get("value")
		emitterChain = queryParams.Get("emitterChain")
		emitterAddress = queryParams.Get("emitterAddress")

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
	default:
		http.Error(w, "405 - Method Not Allowed", http.StatusMethodNotAllowed)
		log.Println("Method Not Allowed")
		return
	}

	if columnFamily != "TokenTransferPayload" &&
		columnFamily != "AssetMetaPayload" &&
		columnFamily != "NFTTransferPayload" &&
		columnFamily != "TokenTransferDetails" {
		fmt.Fprint(w, "columnFamily must be one of: ['TokenTransferPayload', 'AssetMetaPayload', 'NFTTransferPayload', 'TokenTransferDetails']")
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

	results := []bigtable.Row{}
	err := tbl.ReadRows(r.Context(), bigtable.PrefixRange(prefix), func(row bigtable.Row) bool {
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

	details := []Details{}
	for _, result := range results {
		detail := makeDetails(result)
		details = append(details, *detail)
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
