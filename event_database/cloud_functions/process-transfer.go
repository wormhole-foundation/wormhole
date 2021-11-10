package p

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/certusone/wormhole/node/pkg/vaa"
)

// ProcessTransfer is triggered by a PubSub message, once a TokenTransferPayload is written to a row.
func ProcessTransfer(ctx context.Context, m PubSubMessage) error {
	data := string(m.Data)
	if data == "" {
		return fmt.Errorf("no data to process in message")
	}

	log.Printf("ProcessTransfer got message!")
	signedVaa, err := vaa.Unmarshal(m.Data)
	if err != nil {
		log.Println("failed Unmarshaling VAA")
		return err
	}
	jsonVaa, _ := json.MarshalIndent(signedVaa, "", "  ")
	log.Printf("ProcessTransfer Unmarshaled VAA: %q\n", string(jsonVaa))

	return nil
}
