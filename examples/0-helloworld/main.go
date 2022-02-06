package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/augustoroman/sandwich"
)

func main() {
	mw := sandwich.TheUsual()
	http.Handle("/", mw.Then(func(w http.ResponseWriter) {
		fmt.Fprintf(w, "Hello world!")
	}))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
