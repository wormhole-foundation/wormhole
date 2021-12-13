package main

import (
	"context"
	"log"
	"os"
	"sync"

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

	wg.Wait()
}
