package main

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	"github.com/CusackJ4/chirpy.git/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries // from db.go
}

func main() {
	const filePathRoot = "."
	const port = "8080"

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Unable to connect to DB")
	}
	dbQueries := database.New(dbConn) // from db.go, type is *database.Queries

	var apiCfg apiConfig // see alternate method
	apiCfg.db = dbQueries

	/*
		Alternate method to initializing 'apiCfg' struct: isntead of initalizing an empty apiCFG, use a struct literal
		apiCfg := apiConfig{
				fileserverHits: atomic.Int32{},
				db: dbQueries
			}
	*/

	mux := http.NewServeMux()
	// mux.Handle("/", http.FileServer(http.Dir(filePathRoot)))
	mux.Handle("/app/",
		// StripPrefix allows any location with '/app/' to be searchable
		apiCfg.middlewareMetricsInc(http.StripPrefix("/app",
			http.FileServer(http.Dir(filePathRoot)))))

	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.hitresetHandler)

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
	})
	mux.HandleFunc("POST /api/validate_chirp", apiCfg.chirplengthHandler)

	//2. Initialize the httpServer field
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())

}

// html template for hit metrics
var metricsTmpl = template.Must(template.New("metrics").Parse(`<!DOCTYPE html>
<html>
<body>
	<h1>Welcome, Chirpy Admin</h1>
	<p>Chirpy has been visited {{.Hits}} times!</p>
</body>
</html>`))

// Middleware method that is used to track hits to the site
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// method that's used to print hits to the site
func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct {
		Hits int32
	}{
		Hits: cfg.fileserverHits.Load(),
	}
	// fmt.Fprint(w, "Hits: ", cfg.fileserverHits.Load())
	if err := metricsTmpl.Execute(w, data); err != nil {
		http.Error(w, "ERROR: Failed to render template", http.StatusInternalServerError)
	}

}

// method that's used to reset # of hits to the site
func (cfg *apiConfig) hitresetHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Counter reset to 0\n"))
}

// method to accept a POST request and send a response. aka validate chirp
func (cfg *apiConfig) chirplengthHandler(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		w.WriteHeader(500)
		return
	}
	// params now has a populated body (string)
	bodyLength := utf8.RuneCountInString(params.Body)

	switch {
	case bodyLength > 140:
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
	case bodyLength <= 140:
		// badword checker / string cleaner
		badwords := []string{"kerfuffle", "sharbert", "fornax"}
		params.Body = stringCleaner(params.Body, badwords)
		//end check

		respondWithJson(w, http.StatusOK, params.Body)
	}
}

type APIResponse[T any] struct {
	Data         T      `json:"data,omitempty"`
	Error        string `json:"error,omitempty"`
	Valid        bool   `json:"valid,omitempty"`
	Cleaned_body T      `json:"cleaned_body,omitempty"`
}

func writeJson(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("Failed to encode JSON: %v", err)
	}

}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	writeJson(w, code, APIResponse[any]{Error: msg})
}

// data T is here because it was asked for in the lesson.
/* func respondWithJson[T any](w http.ResponseWriter, code int, data T) {
	writeJson(w, code, APIResponse[T]{Valid: true})
} */
func respondWithJson[T any](w http.ResponseWriter, code int, data T) {
	writeJson(w, code, APIResponse[T]{Valid: true, Cleaned_body: data})
}

func stringCleaner(body string, badwords []string) string {

	targetMap := make(map[string]struct{}) // empty map for badwords
	for _, target := range badwords {
		// create map of badwords, w/empty structs as values
		targetMap[target] = struct{}{}
	}
	splitBody := strings.Split(body, " ")

	// use i to modify index directly instead of copy ('word' is a copy)
	for i, word := range splitBody {
		lowWord := strings.ToLower(word)
		if _, found := targetMap[lowWord]; found {
			splitBody[i] = "****"
		}
	}

	return strings.Join(splitBody, " ")
}
