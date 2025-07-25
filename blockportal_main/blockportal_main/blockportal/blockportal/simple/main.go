package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	const port = ":8081" // Changed from 8080 to 8081

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, World!")
		log.Printf("Request received: %s %s", r.Method, r.URL)
	})

	log.Printf("Starting test server on %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v\nIf port is in use, try killing the process or changing the port number", err)
	}
}
