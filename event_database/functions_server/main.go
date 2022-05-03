package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	p "github.com/certusone/wormhole/event_database/cloud_functions"
)

func createAndSubscribe(client *pubsub.Client, topicName, subscriptionName string, handler func(ctx context.Context, m p.PubSubMessage) error) {
	var topic *pubsub.Topic
	var topicErr error
	ctx := context.Background()
	topic, topicErr = client.CreateTopic(ctx, topicName)
	if topicErr != nil {
		log.Printf("pubsub.CreateTopic err: %v", topicErr)
		// already exists
		topic = client.Topic(topicName)
	} else {
		log.Println("created topic:", topicName)
	}

	subConf := pubsub.SubscriptionConfig{Topic: topic}
	_, subErr := client.CreateSubscription(ctx, subscriptionName, subConf)
	if subErr != nil {
		log.Printf("pubsub.CreateSubscription err: %v", subErr)
	} else {
		log.Println("created subscription:", subscriptionName)
	}

	sub := client.Subscription(subscriptionName)

	err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		msg.Ack()
		handler(ctx, p.PubSubMessage{Data: msg.Data})

	})
	if err != nil {
		fmt.Println(fmt.Errorf("receive err: %v", err))
	}
}

var mux = newMux()

// Entry is the cloud function entry point
func Entry(w http.ResponseWriter, r *http.Request) {
	mux.ServeHTTP(w, r)
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/notionaltransferred", p.NotionalTransferred)
	mux.HandleFunc("/notionaltransferredto", p.NotionalTransferredTo)
	mux.HandleFunc("/notionaltransferredfrom", p.NotionalTransferredFrom)
	mux.HandleFunc("/computenotionaltransferredfrom", p.ComputeNotionalTransferredFrom)
	mux.HandleFunc("/notionaltransferredtocumulative", p.NotionalTransferredToCumulative)
	mux.HandleFunc("/notionaltvl", p.TVL)
	mux.HandleFunc("/computenotionaltvl", p.ComputeTVL)
	mux.HandleFunc("/notionaltvlcumulative", p.TvlCumulative)
	mux.HandleFunc("/computenotionaltvlcumulative", p.ComputeTvlCumulative)
	mux.HandleFunc("/addressestransferredto", p.AddressesTransferredTo)
	mux.HandleFunc("/addressestransferredtocumulative", p.AddressesTransferredToCumulative)
	mux.HandleFunc("/totals", p.Totals)
	mux.HandleFunc("/nfts", p.NFTs)
	mux.HandleFunc("/recent", p.Recent)
	mux.HandleFunc("/transaction", p.Transaction)
	mux.HandleFunc("/readrow", p.ReadRow)
	mux.HandleFunc("/findvalues", p.FindValues)

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	return mux
}

func main() {
	var wg sync.WaitGroup

	// http functions
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx := context.Background()
		if err := funcframework.RegisterHTTPFunctionContext(ctx, "/", Entry); err != nil {
			log.Fatalf("funcframework.RegisterHTTPFunctionContext: %v\n", err)
		}
		// Use PORT environment variable, or default to 8080.
		port := "8080"
		if envPort := os.Getenv("PORT"); envPort != "" {
			port = envPort
		}
		if err := funcframework.Start(port); err != nil {
			log.Fatalf("funcframework.Start: %v\n", err)
		}
	}()

	// pubsub functions
	pubsubCtx := context.Background()
	gcpProject := os.Getenv("GCP_PROJECT")

	pubsubClient, err := pubsub.NewClient(pubsubCtx, gcpProject)
	if err != nil {
		fmt.Println(fmt.Errorf("pubsub.NewClient err: %v", err))
	}

	pubsubTopicVAA := os.Getenv("PUBSUB_NEW_VAA_TOPIC")
	pubsubSubscriptionVAA := os.Getenv("PUBSUB_NEW_VAA_SUBSCRIPTION")
	wg.Add(1)
	go createAndSubscribe(pubsubClient, pubsubTopicVAA, pubsubSubscriptionVAA, p.ProcessVAA)
	wg.Done()

	pubsubTopicTransfer := os.Getenv("PUBSUB_TOKEN_TRANSFER_DETAILS_TOPIC")
	pubsubSubscriptionTransfer := os.Getenv("PUBSUB_TOKEN_TRANSFER_DETAILS_SUBSCRIPTION")
	wg.Add(1)
	go createAndSubscribe(pubsubClient, pubsubTopicTransfer, pubsubSubscriptionTransfer, p.ProcessTransfer)
	wg.Done()

	wg.Wait()
	pubsubClient.Close()
}
