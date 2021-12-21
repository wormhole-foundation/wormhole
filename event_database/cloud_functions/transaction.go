// Package p contains an HTTP Cloud Function.
package p

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"

	"cloud.google.com/go/bigtable"
)

// fetch a single row by transaction identifier
func Transaction(w http.ResponseWriter, r *http.Request) {
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

	var transactionID string

	// allow GET requests with querystring params, or POST requests with json body.
	switch r.Method {
	case http.MethodGet:
		queryParams := r.URL.Query()
		transactionID = queryParams.Get("id")

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
			ID string `json:"id"`
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

		transactionID = d.ID

	default:
		http.Error(w, "405 - Method Not Allowed", http.StatusMethodNotAllowed)
		log.Println("Method Not Allowed")
		return
	}

	if transactionID == "" {
		fmt.Fprint(w, "id cannot be blank")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}

	var result bigtable.Row
	readErr := tbl.ReadRows(r.Context(), bigtable.PrefixRange(""), func(row bigtable.Row) bool {

		result = row
		return true

	}, bigtable.RowFilter(bigtable.ValueFilter(transactionID)))

	if readErr != nil {
		log.Fatalf("failed to read rows: %v", readErr)
	}

	if result == nil {
		http.NotFound(w, r)
		log.Printf("did not find row with transaction ID %v", transactionID)
		return
	}

	key := result.Key()
	row, err := tbl.ReadRow(r.Context(), key, bigtable.RowFilter(bigtable.LatestNFilter(1)))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Fatalf("Could not read row with key %s: %v", key, err)
	}

	details := makeDetails(row)
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
