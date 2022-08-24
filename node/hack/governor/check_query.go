// This tool can be used to confirm that the CoinkGecko price query still works after the token list is updated.
// Usage: go run check_query.go

package main

import (
	"github.com/certusone/wormhole/node/pkg/governor"

	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	logger.Info("Testing Coin Gecko query.")
	if err := governor.CheckQuery(logger); err != nil {
		logger.Fatal("Coin Gecko query failed", zap.Error(err))
	}

	logger.Info("Coin Gecko query completed successfully.")
}
