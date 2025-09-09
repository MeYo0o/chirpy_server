package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/MeYo0o/chirpy_server/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
}

func main() {
	// environment loading (.env)
	godotenv.Load()

	// Get the current platform
	platform := os.Getenv("PLATFORM")

	// GET db_url
	dbURL := os.Getenv("DB_URL")

	// Open DB connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalln("couldn't open DB:", err)
	}

	// DB Queries instance => for doing SQLC queries
	dbQueries := database.New(db)

	cfg := apiConfig{
		db:       dbQueries,
		platform: platform,
	}

	mux := http.NewServeMux()
	serverIp := ""
	serverPort := 8080
	chirpyServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", serverIp, serverPort),
		Handler: mux,
	}

	// File Server related
	mux.Handle("/app/", cfg.middlewareMetricsInc(handlerHome))
	mux.HandleFunc("/app/assets/logo.png", handlerLogo)

	// API related
	mux.HandleFunc("GET /api/healthz", handlerHealth)
	mux.HandleFunc("POST /api/users", cfg.handlerCreateUser)
	mux.HandleFunc("GET /api/chirps", cfg.handlerGetChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", cfg.handlerGetSingleChirp)
	mux.HandleFunc("POST /api/chirps", cfg.handlerCreateChirp)
	mux.HandleFunc("GET /admin/metrics", cfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", cfg.handlerResetMetrics)

	log.Printf("Serving files from %s on port: %d\n", serverIp, serverPort)
	log.Fatal(chirpyServer.ListenAndServe())
}
