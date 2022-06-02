package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

func main() {
	var port int
	var code int
	var message string

	flag.IntVar(&port, "port", 8980, "port to start the server on.")
	flag.IntVar(&code, "code", 400, "response code to use.")
	flag.StringVar(&message, "message", "this is all that happens", "response message to display.")

	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		fmt.Fprintf(w, "%s\n", message)
	})

	fmt.Printf("Starting server at port %d\nResponse code (%d)\nmessage (%s)\n", port, code, message)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
	}
}
