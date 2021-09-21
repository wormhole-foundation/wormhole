package p

import (
	"net/http"
	"strings"
	"sync"

	"cloud.google.com/go/bigtable"
)

// shared code for the various functions, primarily response formatting.

// client is a global Bigtable client, to avoid initializing a new client for
// every request.
var client *bigtable.Client
var clientOnce sync.Once

var columnFamilies = []string{"MessagePublication", "Signatures", "VAAState", "QuorumState"}

type (
	Summary struct {
		EmitterChain        string
		EmitterAddress      string
		Sequence            string
		InitiatingTxID      string
		Payload             []byte
		GuardiansThatSigned []string
		SignedVAABytes      []byte
		QuorumTime          string
	}
)

func makeSummary(row bigtable.Row) *Summary {
	summary := &Summary{}
	if _, ok := row[columnFamilies[0]]; ok {

		for _, item := range row[columnFamilies[0]] {
			switch item.Column {
			case "MessagePublication:InitiatingTxID":
				summary.InitiatingTxID = string(item.Value)
			case "MessagePublication:Payload":
				summary.Payload = item.Value
			case "MessagePublication:EmitterChain":
				summary.EmitterChain = string(item.Value)
			case "MessagePublication:EmitterAddress":
				summary.EmitterAddress = string(item.Value)
			case "MessagePublication:Sequence":
				summary.Sequence = string(item.Value)
			}
		}
	}
	if _, ok := row[columnFamilies[1]]; ok {
		for _, item := range row[columnFamilies[1]] {
			column := strings.Split(item.Column, ":")
			summary.GuardiansThatSigned = append(summary.GuardiansThatSigned, column[1])
		}
	}
	if _, ok := row[columnFamilies[3]]; ok {

		for _, item := range row[columnFamilies[3]] {
			if item.Column == "QuorumState:SignedVAA" {
				summary.SignedVAABytes = item.Value
				summary.QuorumTime = item.Timestamp.Time().String()
			}
		}
	}
	return summary
}

var mux = newMux()

// Entry is the cloud function entry point
func Entry(w http.ResponseWriter, r *http.Request) {
	mux.ServeHTTP(w, r)
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/totals", Totals)
	mux.HandleFunc("/recent", Recent)
	mux.HandleFunc("/transaction", Transaction)
	mux.HandleFunc("/readrow", ReadRow)

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	return mux
}
