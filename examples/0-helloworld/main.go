package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/augustoroman/sandwich"
)

func main() {
	mux := sandwich.TheUsual()
	mux.Get("/", func(w http.ResponseWriter) {
		fmt.Fprintf(w, "Hello world!")
	})
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
