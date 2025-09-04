package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

var handlerHome http.Handler = http.StripPrefix("/app", http.FileServer(http.Dir(".")))

func handlerLogo(rw http.ResponseWriter, r *http.Request) {
	http.ServeFile(rw, r, "./assets/logo.png")
}

func handlerHealth(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "text/plain; charset=utf-8")
	if _, err := rw.Write([]byte("OK")); err != nil {
		log.Println("couldn't write headers of Health API:", err)
	}
	rw.WriteHeader(200)
}

func handlerValidateChirp(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	type chirpValidate struct {
		Body string `json:"body"`
	}
	chirpyPost := chirpValidate{}

	msg := make([]byte, 0)

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&chirpyPost)
	if err != nil {
		msg, _ = encodeJson(map[string]any{
			"error": "Something went wrong",
		})
		rw.WriteHeader(400)
		rw.Write(msg)
		return
	}

	// to this point, we have a valid chirpValidateParams
	if len(chirpyPost.Body) > 140 {
		msg, _ = encodeJson(map[string]any{
			"error": "Something went wrong",
		})
		rw.WriteHeader(400)
		rw.Write(msg)
		return
	}

	// prevent these words - has to be without "!"
	/*
		kerfuffle
		sharbert
		fornax
	*/

	preventedKeywords := []string{"kerfuffle", "Kerfuffle", "sharbert", "Sharbert", "fornax", "Fornax"}

	keywordsToCheck := strings.Split(chirpyPost.Body, " ")

	for i, keyword := range keywordsToCheck {
		for _, preventedKeyword := range preventedKeywords {
			if keyword == preventedKeyword {
				keywordsToCheck[i] = "****"
			}
		}
	}

	cleanedPost := strings.Join(keywordsToCheck, " ")
	msg, _ = encodeJson(map[string]any{
		"cleaned_body": cleanedPost,
	})

	fmt.Println(string(msg))
	rw.WriteHeader(200)
	rw.Write(msg)

}

func encodeJson(params map[string]any) ([]byte, error) {
	return json.Marshal(params)
}

func (cfg *apiConfig) handlerMetrics(rw http.ResponseWriter, r *http.Request) {
	currentVisits := cfg.fileserverHits.Load()
	currentVisitsResp := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, currentVisits)

	rw.Header().Add("Content-Type", "text/html; charset=utf-8")
	if _, err := rw.Write([]byte(currentVisitsResp)); err != nil {
		log.Println("couldn't write headers of Metrics API:", err)
	}

	rw.WriteHeader(200)
}

func (cfg *apiConfig) handlerResetMetrics(rw http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	rw.WriteHeader(200)
}
