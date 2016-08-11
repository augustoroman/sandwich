package main

import (
	"fmt"
	"github.com/augustoroman/sandwich"
	"log"
	"net/http"
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
