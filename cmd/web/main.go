package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	log.Println("Starting server on :4000")
	err := http.ListenAndServe(":4000", mux)
	log.Fatal(err)
}
