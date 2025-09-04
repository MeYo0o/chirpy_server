package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {
	mux := http.NewServeMux()
	apiCfg := apiConfig{}

	serverIp := ""
	serverPort := 8080
	chirpyServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", serverIp, serverPort),
		Handler: mux,
	}

	// File Server related
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handlerHome))
	mux.HandleFunc("/app/assets/logo.png", handlerLogo)

	// API related
	mux.HandleFunc("GET /api/healthz", handlerHealth)
	mux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerResetMetrics)

	log.Printf("Serving files from %s on port: %d\n", serverIp, serverPort)
	log.Fatal(chirpyServer.ListenAndServe())
}
