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
	pubsubCtx := context.Background()
	gcpProject := os.Getenv("GCP_PROJECT")

	pubsubClient, err := pubsub.NewClient(pubsubCtx, gcpProject)
	if err != nil {
		fmt.Println(fmt.Errorf("pubsub.NewClient err: %v", err))
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		pubsubTopic := os.Getenv("PUBSUB_NEW_VAA_TOPIC")
		pubsubSubscription := os.Getenv("PUBSUB_NEW_VAA_SUBSCRIPTION")

		createAndSubscribe(pubsubClient, pubsubTopic, pubsubSubscription, p.ProcessVAA)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		pubsubTopic := os.Getenv("PUBSUB_TOKEN_TRANSFER_DETAILS_TOPIC")
		pubsubSubscription := os.Getenv("PUBSUB_TOKEN_TRANSFER_DETAILS_SUBSCRIPTION")

		createAndSubscribe(pubsubClient, pubsubTopic, pubsubSubscription, p.ProcessTransfer)
	}()

	wg.Wait()
	pubsubClient.Close()
}
