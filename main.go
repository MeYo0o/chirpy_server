package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	serverIp := ""
	serverPort := 8080

	chirpyServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", serverIp, serverPort),
		Handler: mux,
	}

	mux.Handle("/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	mux.Handle("/app/assets/logo.png", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	mux.HandleFunc("/healthz", handlerHealth)

	log.Printf("Serving files from %s on port: %d\n", serverIp, serverPort)
	log.Fatal(chirpyServer.ListenAndServe())
}

func handlerHealth(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Add("Content-Type", "text/plain; charset=utf-8")

	if _, err := rw.Write([]byte("OK")); err != nil {
		log.Println("couldn't write headers:", err)
	}

	rw.WriteHeader(200)

}
