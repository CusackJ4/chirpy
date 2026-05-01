package main

import (
	"log"
	"net/http"
)

func main() {
	const filePathRoot = "."
	const port = "8080"

	mux := http.NewServeMux()
	// mux.Handle("/", http.FileServer(http.Dir(filePathRoot)))
	mux.Handle("/app/", http.StripPrefix("/app",
		http.FileServer(http.Dir(filePathRoot))))

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
	})

	//2. Initialize the httpServer field
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())

}

// the first compiles the binary and runs, the second runs without compiling
// go build -o out && ./out
// go run .
