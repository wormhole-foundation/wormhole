package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	p "github.com/certusone/wormhole/event_database/cloud_functions"
)

func main() {
	var wg sync.WaitGroup

	// http functions
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx := context.Background()
		if err := funcframework.RegisterHTTPFunctionContext(ctx, "/", p.Entry); err != nil {
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
	wg.Add(1)
	go func() {
		defer wg.Done()
		pubsubCtx := context.Background()
		gcpProject := os.Getenv("GCP_PROJECT")

		client, err := pubsub.NewClient(pubsubCtx, gcpProject)
		if err != nil {
			fmt.Println(fmt.Errorf("pubsub.NewClient err: %v", err))
		}
		defer client.Close()

		pubsubTopic := os.Getenv("PUBSUB_TOPIC")
		pubsubSubscription := os.Getenv("PUBSUB_SUBSCRIPTION")
		var topic *pubsub.Topic
		var topicErr error

		topic, topicErr = client.CreateTopic(pubsubCtx, pubsubTopic)
		if topicErr != nil {
			log.Printf("pubsub.CreateTopic err: %v", topicErr)
			// already exists
			topic = client.Topic(pubsubTopic)
		} else {
			log.Println("created topic:", pubsubTopic)
		}

		subConf := pubsub.SubscriptionConfig{Topic: topic}
		_, subErr := client.CreateSubscription(pubsubCtx, pubsubSubscription, subConf)
		if subErr != nil {
			log.Printf("pubsub.CreateSubscription err: %v", subErr)
		} else {
			log.Println("created subscription:", pubsubSubscription)
		}

		sub := client.Subscription(pubsubSubscription)

		err = sub.Receive(pubsubCtx, func(ctx context.Context, msg *pubsub.Message) {
			msg.Ack()
			p.ProcessVAA(ctx, p.PubSubMessage{Data: msg.Data})

		})
		if err != nil {
			fmt.Println(fmt.Errorf("receive err: %v", err))
		}
	}()

	wg.Wait()
}
