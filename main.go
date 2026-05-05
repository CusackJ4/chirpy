package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

func main() {
	const filePathRoot = "."
	const port = "8080"
	var apiCfg apiConfig

	mux := http.NewServeMux()
	// mux.Handle("/", http.FileServer(http.Dir(filePathRoot)))
	mux.Handle("/app/",
		// StripPrefix allows any location with '/app/' to be searchable
		apiCfg.middlewareMetricsInc(http.StripPrefix("/app",
			http.FileServer(http.Dir(filePathRoot)))))

	mux.HandleFunc("/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("/reset", apiCfg.hitresetHandler)
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

type apiConfig struct {
	fileserverHits atomic.Int32
}

// Middleware that is used to track hits to the site
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// method that's used to print hits to the site
func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, "Hits: ", cfg.fileserverHits.Load())
}

// method that's used to reset # of hits to the site
func (cfg *apiConfig) hitresetHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Counter reset to 0\n"))
}
