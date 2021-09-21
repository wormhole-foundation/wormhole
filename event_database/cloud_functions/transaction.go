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
	row, err := tbl.ReadRow(r.Context(), key)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Fatalf("Could not read row with key %s: %v", key, err)
	}

	summary := makeSummary(row)
	jsonBytes, err := json.Marshal(summary)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Println(err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}
