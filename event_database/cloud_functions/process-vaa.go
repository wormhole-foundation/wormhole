package p

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/certusone/wormhole/node/pkg/vaa"
)

type PubSubMessage struct {
	Data []byte `json:"data"`
}

// ProcessVAA is triggered by PubSub messages
func ProcessVAA(ctx context.Context, m PubSubMessage) error {
	data := string(m.Data)
	if data == "" {
		log.Print("no data in message.")

	} else {
		log.Printf("ProcessVAA got message!")
		signedVaa, err := vaa.Unmarshal(m.Data)
		if err != nil {
			fmt.Println("failed Unmarshaling VAA")
		}
		jsonVaa, _ := json.MarshalIndent(signedVaa, "", "  ")
		fmt.Printf("ProcessVAA Unmarshaled VAA: %q\n", string(jsonVaa))

		// TODO:
		// decode payload
		// save payload to bigtable
		// publish pubsub message for token transfer messages
	}
	return nil
}
