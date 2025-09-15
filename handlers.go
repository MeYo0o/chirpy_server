package main

import (
	"database/sql"
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
	chirpUUID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		msg, _ := encodeJson(map[string]any{
			"error": "Invalid chirp ID format",
		})
		rw.WriteHeader(400)
		rw.Write(msg)
		return
	}

	// Get the chirp from database
	chirp, err := cfg.db.GetChirpy(r.Context(), chirpUUID)
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

func (cfg *apiConfig) handlerDeleteChirp(rw http.ResponseWriter, r *http.Request) {
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

	userToken, err := auth.GetBearerToken(r.Header)
	if err != nil || userToken == "" {
		msg, _ := encodeJson(map[string]any{
			"error": "unauthorized: invalid user JWT",
		})
		rw.WriteHeader(401)
		rw.Write(msg)
		return
	}

	userUUID, err := auth.ValidateJWT(userToken, cfg.jwtSecret)
	if err != nil {
		msg, _ := encodeJson(map[string]any{
			"error": fmt.Sprintf("unauthorized: %v", err),
		})
		rw.WriteHeader(401)
		rw.Write(msg)
		return
	}

	// Parse the chirp ID string to UUID
	chirpUUID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		msg, _ := encodeJson(map[string]any{
			"error": "Invalid chirp ID format",
		})
		rw.WriteHeader(400)
		rw.Write(msg)
		return
	}

	_, err = cfg.db.GetChirpyByUserID(r.Context(), database.GetChirpyByUserIDParams{
		ID:     chirpUUID,
		UserID: userUUID,
	})
	if err != nil {
		msg, _ := encodeJson(map[string]any{
			"error": "chirp not found",
		})
		rw.WriteHeader(403)
		rw.Write(msg)
		return
	}

	err = cfg.db.DeleteChirp(r.Context(), database.DeleteChirpParams{
		ID:     chirpUUID,
		UserID: userUUID,
	})
	if err != nil {
		msg, _ := encodeJson(map[string]any{
			"error": "couldn't delete this chirpy",
		})
		rw.WriteHeader(403)
		rw.Write(msg)
		return
	}

	rw.WriteHeader(204)
}

func (cfg *apiConfig) handlerPolkaWebhooks(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	type WebhooksData struct {
		UserID string `json:"user_id"`
	}

	type WebhooksRequest struct {
		Event string       `json:"event"`
		Data  WebhooksData `json:"data"`
	}

	var webhooksReq WebhooksRequest

	// Extract the API key and validate it
	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil || apiKey != cfg.polkaKey {
		rw.WriteHeader(401)
		dat, _ := encodeJson(map[string]any{
			"message": "unauthorized APIKey",
		})
		rw.Write(dat)
		return
	}

	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&webhooksReq)
	if err != nil || webhooksReq.Event == "" || webhooksReq.Data.UserID == "" {
		rw.WriteHeader(400)
		dat, _ := encodeJson(map[string]any{
			"message": "invalid request",
		})
		rw.Write(dat)
		return
	}
	defer r.Body.Close()

	if webhooksReq.Event == "user.upgraded" {
		userUUID, err := uuid.Parse(webhooksReq.Data.UserID)
		if err != nil {
			rw.WriteHeader(403)
			dat, _ := encodeJson(map[string]any{"messages": "couldn't parse that userID"})
			rw.Write(dat)
			return
		}

		_, err = cfg.db.GetUserByID(r.Context(), userUUID)
		if err != nil {
			rw.WriteHeader(404)
			dat, _ := encodeJson(map[string]any{"messages": "user not found"})
			rw.Write(dat)
			return
		}

		err = cfg.db.UpgradeUserToRed(r.Context(), userUUID)
		if err != nil {
			rw.WriteHeader(403)
			dat, _ := encodeJson(map[string]any{"messages": "couldn't upgrade the user"})
			rw.Write(dat)
			return
		}
	}

	rw.WriteHeader(204)
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
		"id":            user.ID,
		"email":         user.Email,
		"is_chirpy_red": user.IsChirpyRed,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
	})
	rw.Write(dat)
}

func (cfg *apiConfig) handlerUpdateUser(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	type UserUpdateRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var userUpdateReq UserUpdateRequest

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&userUpdateReq)
	if err != nil || userUpdateReq.Email == "" || userUpdateReq.Password == "" {
		rw.WriteHeader(400)
		dat, _ := encodeJson(map[string]any{
			"messages": "invalid request",
		})
		rw.Write(dat)
		return
	}

	defer r.Body.Close()

	accessToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		rw.WriteHeader(401)
		dat, _ := encodeJson(map[string]any{
			"messages": "invalid or expired Token",
		})
		rw.Write(dat)
		return
	}

	userUUID, err := auth.ValidateJWT(accessToken, cfg.jwtSecret)
	if err != nil {
		rw.WriteHeader(401)
		dat, _ := encodeJson(map[string]any{
			"messages": "no user is found with that token",
		})
		rw.Write(dat)
		return
	}

	hashedPassword, err := auth.HashPassword(userUpdateReq.Password)
	if err != nil {
		rw.WriteHeader(403)
		dat, _ := encodeJson(map[string]any{
			"messages": "couldn't hash user's password",
		})
		rw.Write(dat)
		return
	}

	updatedUser, err := cfg.db.UpdateUser(r.Context(), database.UpdateUserParams{
		Email:          userUpdateReq.Email,
		HashedPassword: hashedPassword,
		ID:             userUUID,
	})
	if err != nil {
		rw.WriteHeader(403)
		dat, _ := encodeJson(map[string]any{
			"messages": "couldn't update the user",
		})
		rw.Write(dat)
		return
	}

	rw.WriteHeader(200)
	dat, _ := encodeJson(map[string]any{
		"id":            updatedUser.ID,
		"email":         updatedUser.Email,
		"is_chirpy_red": updatedUser.IsChirpyRed,
		"created_at":    updatedUser.CreatedAt,
		"updated_at":    updatedUser.UpdatedAt,
	})
	rw.Write(dat)

}

func (cfg *apiConfig) handlerLoginUser(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	// default expiry time in seconds, unless modified by the client's request
	ExpiresInSeconds := time.Hour * 1
	RefreshTokenExpireIn := time.Hour * 24 * 60

	type LoginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
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

	// Get the user data for password verification
	user, err := cfg.db.GetUserByEmail(r.Context(), loginReq.Email)
	if err != nil {
		rw.WriteHeader(403)
		dat, _ := encodeJson(map[string]any{
			"messages": "user not found",
		})
		rw.Write(dat)
		return
	}

	// Hash the User password
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

	// Password is Valid
	// Generate JWT for the user
	generatedToken, err := auth.MakeJWT(
		user.ID,
		cfg.jwtSecret,
		ExpiresInSeconds,
	)
	if err != nil {
		rw.WriteHeader(403)
		dat, _ := encodeJson(map[string]any{
			"messages": "couldn't generate access token for the user",
		})
		rw.Write(dat)
		return
	}

	// Also Generate A Refresh Token with 60days so the user can stay longer on the platform
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil || refreshToken == "" {
		rw.WriteHeader(403)
		dat, _ := encodeJson(map[string]any{
			"messages": "couldn't generate refresh token for the user",
		})
		rw.Write(dat)
		return
	}
	cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(RefreshTokenExpireIn),
		RevokedAt: sql.NullTime{},
		UserID:    user.ID,
	})

	rw.WriteHeader(200)
	dat, _ := encodeJson(map[string]any{
		"id":            user.ID,
		"email":         user.Email,
		"is_chirpy_red": user.IsChirpyRed,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
		"token":         generatedToken,
		"refresh_token": refreshToken,
	})
	rw.Write(dat)
}

func (cfg *apiConfig) handlerRefreshToken(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil || refreshToken == "" {
		rw.WriteHeader(401)
		dat, _ := encodeJson(map[string]any{
			"messages": "couldn't retrieve refresh token for the user",
		})
		rw.Write(dat)
		return
	}

	// Fetch the refreshToken from the DB & check if it's still valid/not-expired
	foundRefreshToken, err := cfg.db.GetRefreshToken(r.Context(), refreshToken)
	if err != nil || (time.Since(foundRefreshToken.ExpiresAt) > 0) || foundRefreshToken.RevokedAt.Valid {
		rw.WriteHeader(401)
		dat, _ := encodeJson(map[string]any{
			"messages": "either the refresh token is expired or not found",
		})
		rw.Write(dat)
		return
	}

	// Refresh token is still valid => Create an Access Token for the user, as the current one is expired, that's why this RefreshToken api is called in the first place
	user, err := cfg.db.GetUserFromRefreshToken(r.Context(), foundRefreshToken.UserID)
	if err != nil {
		rw.WriteHeader(404)
		dat, _ := encodeJson(map[string]any{
			"messages": "user not found",
		})
		rw.Write(dat)
		return
	}

	JWT, err := auth.MakeJWT(user.UserID, cfg.jwtSecret, time.Hour*1)
	if err != nil {
		rw.WriteHeader(401)
		dat, _ := encodeJson(map[string]any{
			"messages": "couldn't create Access Token for the user",
		})
		rw.Write(dat)
		return
	}

	rw.WriteHeader(200)
	dat, _ := encodeJson(map[string]any{
		"token": JWT,
	})
	rw.Write(dat)
}

func (cfg *apiConfig) handlerRevokeRefreshToken(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")

	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil || refreshToken == "" {
		rw.WriteHeader(401)
		dat, _ := encodeJson(map[string]any{
			"messages": "couldn't retrieve refresh token for the user",
		})
		rw.Write(dat)
		return
	}

	err = cfg.db.UpdateRefreshToken(r.Context(), database.UpdateRefreshTokenParams{
		RevokedAt: sql.NullTime{Time: time.Now(), Valid: true},
		UpdatedAt: time.Now(),
		Token:     refreshToken,
	})
	if err != nil {
		rw.WriteHeader(403)
		dat, _ := encodeJson(map[string]any{
			"messages": "couldn't update the Refresh token record.",
		})
		rw.Write(dat)
		return
	}

	rw.WriteHeader(204)
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
