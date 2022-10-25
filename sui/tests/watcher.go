package main

import (
	"fmt"
	"golang.org/x/net/websocket"
	"log"
	"time"
	"os"
)

func main() {
	origin := "http://localhost/"
	url := "ws://localhost:9001"
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	s := fmt.Sprintf(`{"jsonrpc":"2.0", "id": 1, "method": "sui_subscribeEvent", "params": [{"SenderAddress": "%s"}]}`, os.Getenv("WORM_OWNER"))
	fmt.Printf("Sending: %s.\n", s)

	if _, err := ws.Write([]byte(s)); err != nil {
		log.Fatal(err)
	}
	for {
		var msg = make([]byte, 512)
		var n int
		ws.SetReadDeadline(time.Now().Local().Add(1_000_000_000));
		if n, err = ws.Read(msg); err != nil {
			fmt.Printf("err");
		} else {
			fmt.Printf("Received: %s.\n", msg[:n])
		}
	}
}
