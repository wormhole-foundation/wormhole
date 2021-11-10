package p

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"cloud.google.com/go/bigtable"
	"cloud.google.com/go/pubsub"
	"github.com/certusone/wormhole/node/pkg/vaa"
)

// shared code for the various functions, primarily response formatting.

// client is a global Bigtable client, to avoid initializing a new client for
// every request.
var client *bigtable.Client
var clientOnce sync.Once
var tbl *bigtable.Table

var pubsubClient *pubsub.Client
var pubSubTokenTransferDetailsTopic *pubsub.Topic

// init runs during cloud function initialization. So, this will only run during an
// an instance's cold start.
// https://cloud.google.com/functions/docs/bestpractices/networking#accessing_google_apis
func init() {
	clientOnce.Do(func() {
		// Declare a separate err variable to avoid shadowing client.
		var err error
		project := os.Getenv("GCP_PROJECT")
		instance := os.Getenv("BIGTABLE_INSTANCE")
		client, err = bigtable.NewClient(context.Background(), project, instance)
		if err != nil {
			// http.Error(w, "Error initializing client", http.StatusInternalServerError)
			log.Printf("bigtable.NewClient error: %v", err)

			return
		}

		var pubsubErr error
		pubsubClient, pubsubErr = pubsub.NewClient(context.Background(), project)
		if pubsubErr != nil {
			log.Printf("pubsub.NewClient error: %v", pubsubErr)
			return
		}
	})
	tbl = client.Open("v2Events")

	// create the topic that will be published to after decoding token transfer payloads
	tokenTransferDetailsTopic := os.Getenv("PUBSUB_TOKEN_TRANSFER_DETAILS_TOPIC")
	pubSubTokenTransferDetailsTopic = pubsubClient.Topic(tokenTransferDetailsTopic)
}

var columnFamilies = []string{
	"MessagePublication",
	"QuorumState",
	"TokenTransferPayload",
	"AssetMetaPayload",
	"NFTTransferPayload",
	"TokenTransferDetails",
	"ChainDetails",
}

type (
	Summary struct {
		EmitterChain   string
		EmitterAddress string
		Sequence       string
		InitiatingTxID string
		Payload        []byte
		SignedVAABytes []byte
		QuorumTime     string
	}
	// Details is a Summary, with the VAA decoded as SignedVAA
	Details struct {
		SignedVAA      *vaa.VAA
		EmitterChain   string
		EmitterAddress string
		Sequence       string
		InitiatingTxID string
		Payload        []byte
		SignedVAABytes []byte
		QuorumTime     string
	}
)

func chainIdStringToType(chainId string) vaa.ChainID {
	switch chainId {
	case "1":
		return vaa.ChainIDSolana
	case "2":
		return vaa.ChainIDEthereum
	case "3":
		return vaa.ChainIDTerra
	case "4":
		return vaa.ChainIDBSC
	case "5":
		return vaa.ChainIDPolygon
	}
	return vaa.ChainIDUnset
}

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
	} else {
		// Some rows have a QuorumState, but no MessagePublication,
		// so populate Summary values from the rowKey.
		keyParts := strings.Split(row.Key(), ":")
		chainId := chainIdStringToType(keyParts[0])
		summary.EmitterChain = chainId.String()
		summary.EmitterAddress = keyParts[1]
		seq := strings.TrimLeft(keyParts[2], "0")
		if seq == "" {
			seq = "0"
		}
		summary.Sequence = seq
	}
	if _, ok := row[columnFamilies[1]]; ok {
		item := row[columnFamilies[1]][0]
		summary.SignedVAABytes = item.Value
		summary.QuorumTime = item.Timestamp.Time().String()
	}
	return summary
}

func makeDetails(row bigtable.Row) *Details {
	sum := makeSummary(row)
	deets := &Details{
		EmitterChain:   sum.EmitterChain,
		EmitterAddress: sum.EmitterAddress,
		Sequence:       sum.Sequence,
		InitiatingTxID: sum.InitiatingTxID,
		Payload:        sum.Payload,
		SignedVAABytes: sum.SignedVAABytes,
		QuorumTime:     sum.QuorumTime,
	}
	if _, ok := row[columnFamilies[1]]; ok {
		item := row[columnFamilies[1]][0]
		deets.SignedVAA, _ = vaa.Unmarshal(item.Value)
	}
	return deets
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
