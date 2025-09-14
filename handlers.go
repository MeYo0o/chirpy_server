package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/MeYo0o/chirpy_server/internal/auth"
	"github.com/MeYo0o/chirpy_server/internal/database"
	"github.com/google/uuid"
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

func (cfg *apiConfig) handlerGetChirps(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	chirps, err := cfg.db.GetChirps(r.Context())
	if err != nil {
		msg, _ := encodeJson(map[string]any{
			"messages": fmt.Sprintf("couldn't fetch users: %v", err),
		})
		rw.WriteHeader(403)
		rw.Write(msg)
		return
	}

	chirpsResponseJson := make([]map[string]string, len(chirps))
	for i, chirpy := range chirps {
		chirpsResponseJson[i] = map[string]string{
			"id":         chirpy.UserID.String(),
			"created_at": chirpy.CreatedAt.String(),
			"updated_at": chirpy.UpdatedAt.String(),
			"body":       chirpy.Body,
			"user_id":    chirpy.UserID.String(),
		}
	}

	rw.WriteHeader(200)
	data, _ := json.Marshal(chirpsResponseJson)
	rw.Write(data)

}

func (cfg *apiConfig) handlerGetSingleChirp(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	// Extract chirpID from the URL path
	chirpIDStr := r.PathValue("chirpID")
	if chirpIDStr == "" {
		msg, _ := encodeJson(map[string]any{
			"error": "Missing chirp ID",
		})
		rw.WriteHeader(400)
		rw.Write(msg)
		return
	}

	// Parse the chirp ID string to UUID
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		msg, _ := encodeJson(map[string]any{
			"error": "Invalid chirp ID format",
		})
		rw.WriteHeader(400)
		rw.Write(msg)
		return
	}

	// Get the chirp from database
	chirp, err := cfg.db.GetChirpy(r.Context(), chirpID)
	if err != nil {
		msg, _ := encodeJson(map[string]any{
			"error": "Chirp not found",
		})
		rw.WriteHeader(404)
		rw.Write(msg)
		return
	}

	// Return the chirp
	dat, _ := encodeJson(map[string]any{
		"id":         chirp.ID,
		"body":       chirp.Body,
		"user_id":    chirp.UserID,
		"created_at": chirp.CreatedAt,
		"updated_at": chirp.UpdatedAt,
	})
	rw.WriteHeader(200)
	rw.Write(dat)
}

func (cfg *apiConfig) handlerCreateChirp(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	type ChirpyPostReq struct {
		Body string `json:"body"`
	}
	chirpyPostReq := ChirpyPostReq{}

	msg := make([]byte, 0)

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&chirpyPostReq)
	if err != nil {
		msg, _ = encodeJson(map[string]any{
			"error": "Something went wrong",
		})
		rw.WriteHeader(400)
		rw.Write(msg)
		return
	}
	defer r.Body.Close()

	fmt.Println(r.Body)

	// check if the request is valid
	if chirpyPostReq.Body == "" {
		msg, _ = encodeJson(map[string]any{
			"error": "invalid request",
		})
		rw.WriteHeader(400)
		rw.Write(msg)
		return
	}

	// to this point, we have a valid chirpValidateParams
	if len(chirpyPostReq.Body) > 140 {
		msg, _ = encodeJson(map[string]any{
			"error": "Something went wrong",
		})
		rw.WriteHeader(400)
		rw.Write(msg)
		return
	}

	userToken, err := auth.GetBearerToken(r.Header)
	if err != nil || userToken == "" {
		msg, _ = encodeJson(map[string]any{
			"error": "unauthorized: invalid user JWT",
		})
		rw.WriteHeader(401)
		rw.Write(msg)
		return
	}

	userUUID, err := auth.ValidateJWT(userToken, cfg.jwtSecret)
	if err != nil {
		msg, _ = encodeJson(map[string]any{
			"error": fmt.Sprintf("unauthorized: %v", err),
		})
		rw.WriteHeader(401)
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

	keywordsToCheck := strings.Split(chirpyPostReq.Body, " ")

	for i, keyword := range keywordsToCheck {
		for _, preventedKeyword := range preventedKeywords {
			if keyword == preventedKeyword {
				keywordsToCheck[i] = "****"
			}
		}
	}

	cleanedPost := strings.Join(keywordsToCheck, " ")

	chirpyPost, err := cfg.db.CreateChirpy(r.Context(), database.CreateChirpyParams{
		ID:        uuid.New(),
		Body:      cleanedPost,
		UserID:    userUUID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		msg, _ := encodeJson(map[string]any{
			"messages": fmt.Sprintf("couldn't create a chirpy: %v", err),
		})
		rw.WriteHeader(403)
		rw.Write(msg)
		return
	}

	dat, _ := encodeJson(map[string]any{
		"id":         chirpyPost.ID,
		"body":       chirpyPost.Body,
		"user_id":    chirpyPost.UserID,
		"created_at": chirpyPost.CreatedAt,
		"updated_at": chirpyPost.UpdatedAt,
	})
	rw.WriteHeader(201)
	rw.Write(dat)
}

func encodeJson(params map[string]any) ([]byte, error) {
	return json.Marshal(params)
}

func (cfg *apiConfig) handlerCreateUser(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	type emailRequest struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	var emailReq emailRequest

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&emailReq)
	if err != nil || emailReq.Email == "" || emailReq.Password == "" {
		rw.WriteHeader(400)
		dat, _ := encodeJson(map[string]any{
			"messages": "invalid request",
		})
		rw.Write(dat)
		return
	}

	defer r.Body.Close()

	hashedPassword, err := auth.HashPassword(emailReq.Password)
	if err != nil {
		rw.WriteHeader(403)
		dat, _ := encodeJson(map[string]any{
			"messages": "couldn't generate hashed password",
		})
		rw.Write(dat)
		return
	}

	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		ID:             uuid.New(),
		Email:          emailReq.Email,
		HashedPassword: hashedPassword,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})
	if err != nil {
		rw.WriteHeader(401)
		dat, _ := encodeJson(map[string]any{
			"messages": "couldn't create the user",
		})
		rw.Write(dat)
		return
	}

	rw.WriteHeader(201)
	dat, _ := encodeJson(map[string]any{
		"id":         user.ID,
		"email":      user.Email,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	})
	rw.Write(dat)
}

func (cfg *apiConfig) handlerLoginUser(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	// default expiry time in seconds, unless modified by the client's request
	ExpiresInSeconds := time.Hour * 1

	type LoginRequest struct {
		Email            string `json:"email"`
		Password         string `json:"password"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
	}

	var loginReq LoginRequest

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&loginReq)
	if err != nil || loginReq.Email == "" || loginReq.Password == "" {
		rw.WriteHeader(400)
		dat, _ := encodeJson(map[string]any{
			"messages": "invalid request",
		})
		rw.Write(dat)
		return
	}

	defer r.Body.Close()

	// check if there is a passed "expiry time" via client's request, if so , set it to the default value
	if loginReq.ExpiresInSeconds != 0 {
		ExpiresInSeconds = time.Second * time.Duration(loginReq.ExpiresInSeconds)
	}

	user, err := cfg.db.GetUserByEmail(r.Context(), loginReq.Email)
	if err != nil {
		rw.WriteHeader(403)
		dat, _ := encodeJson(map[string]any{
			"messages": "user not found",
		})
		rw.Write(dat)
		return
	}

	HashedPassByteSli := []byte(user.HashedPassword)
	err = auth.ComparePasswordHash(loginReq.Password, string(HashedPassByteSli))
	if err != nil {
		rw.WriteHeader(401)
		dat, _ := encodeJson(map[string]any{
			"messages": "password is not correct",
		})
		rw.Write(dat)
		return
	}

	generatedToken, err := auth.MakeJWT(
		user.ID,
		cfg.jwtSecret,
		ExpiresInSeconds,
	)
	if err != nil {
		rw.WriteHeader(403)
		dat, _ := encodeJson(map[string]any{
			"messages": "couldn't generate a token for the user",
		})
		rw.Write(dat)
		return
	}

	rw.WriteHeader(200)
	dat, _ := encodeJson(map[string]any{
		"id":         user.ID,
		"email":      user.Email,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
		"token":      generatedToken,
	})
	rw.Write(dat)
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
	rw.Header().Add("Content-Type", "application/json")

	// Continue only if this is "local/dev" environment
	if cfg.platform != "dev" {
		rw.WriteHeader(403)
		dat, _ := encodeJson(map[string]any{
			"messages": "this api is forbidden",
		})
		rw.Write(dat)
		return
	}

	err := cfg.db.DeleteUsers(r.Context())
	if err != nil {
		dat, _ := encodeJson(map[string]any{
			"messages": "couldn't delete users",
		})
		rw.Write(dat)
		rw.WriteHeader(500)
		return
	}

	cfg.fileserverHits.Store(0)
	rw.WriteHeader(200)
}
