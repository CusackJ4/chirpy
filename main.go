package main

import (
	"log"
	"net/http"
)

func main() {

	const port = "8080"
	mux := http.NewServeMux()

	//2. Initialize the httpServer field
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())

}
