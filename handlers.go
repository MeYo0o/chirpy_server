package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
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
	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(200)
	if _, err := rw.Write([]byte("OK")); err != nil {
		log.Println("couldn't write headers of Health API:", err)
	}
}

func (cfg *apiConfig) handlerGetChirps(rw http.ResponseWriter, r *http.Request) {
	var chirps []database.Chirp
	var err error
	var sort string = "asc"

	// [Optional] when providing author_id/user_id && sort
	urlValues := r.URL.Query()

	// we set the sort with the optional provided sort if it's valid
	optionalSort := strings.ToLower(urlValues.Get("sort"))
	if optionalSort != "" && (optionalSort == "asc" || optionalSort == "desc") {
		sort = optionalSort
	}

	// we get all chirps/posts of that provided author_id/user_id
	authorID := urlValues.Get("author_id")
	if authorID != "" {
		userUUID, err := validateUUID(authorID, "author_id")
		if err != nil {
			writeErrorResponse(rw, 404, "user not found")
			return
		}

		chirps, err = cfg.db.GetChirpsByUserID(r.Context(), userUUID)
		if err != nil {
			writeErrorResponse(rw, 403, fmt.Sprintf("couldn't fetch chirps: %v", err))
			return
		}
	} else {
		chirps, err = cfg.db.GetChirps(r.Context())
		if err != nil {
			writeErrorResponse(rw, 403, fmt.Sprintf("couldn't fetch chirps: %v", err))
			return
		}
	}

	// sort chirps slice in memory in terms of "created_at"
	slices.SortFunc(chirps, func(a, b database.Chirp) int {
		if sort == "asc" {
			return a.CreatedAt.Compare(b.CreatedAt)
		}

		return b.CreatedAt.Compare(a.CreatedAt)
	})

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

	writeSuccessResponse(rw, 200, map[string]any{
		"chirps": chirpsResponseJson,
	})
}

func (cfg *apiConfig) handlerGetSingleChirp(rw http.ResponseWriter, r *http.Request) {
	// Extract chirpID from the URL path
	chirpIDStr := r.PathValue("chirpID")
	chirpUUID, err := validateUUID(chirpIDStr, "chirp ID")
	if err != nil {
		writeErrorResponse(rw, 400, err.Error())
		return
	}

	// Get the chirp from database
	chirp, err := cfg.db.GetChirpy(r.Context(), chirpUUID)
	if err != nil {
		writeErrorResponse(rw, 404, "Chirp not found")
		return
	}

	// Return the chirp
	writeSuccessResponse(rw, 200, map[string]any{
		"id":         chirp.ID,
		"body":       chirp.Body,
		"user_id":    chirp.UserID,
		"created_at": chirp.CreatedAt,
		"updated_at": chirp.UpdatedAt,
	})
}

func (cfg *apiConfig) handlerDeleteChirp(rw http.ResponseWriter, r *http.Request) {
	// Extract chirpID from the URL path
	chirpIDStr := r.PathValue("chirpID")
	chirpUUID, err := validateUUID(chirpIDStr, "chirp ID")
	if err != nil {
		writeErrorResponse(rw, 400, err.Error())
		return
	}

	// Validate JWT and get user UUID
	userUUID, err := cfg.validateJWTFromRequest(r)
	if err != nil {
		writeErrorResponse(rw, 401, err.Error())
		return
	}

	// Check if chirp exists and belongs to user
	_, err = cfg.db.GetChirpyByUserID(r.Context(), database.GetChirpyByUserIDParams{
		ID:     chirpUUID,
		UserID: userUUID,
	})
	if err != nil {
		writeErrorResponse(rw, 403, "chirp not found")
		return
	}

	// Delete the chirp
	err = cfg.db.DeleteChirp(r.Context(), database.DeleteChirpParams{
		ID:     chirpUUID,
		UserID: userUUID,
	})
	if err != nil {
		writeErrorResponse(rw, 403, "couldn't delete this chirpy")
		return
	}

	writeEmptyResponse(rw, 204)
}

func (cfg *apiConfig) handlerPolkaWebhooks(rw http.ResponseWriter, r *http.Request) {
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
		writeErrorResponse(rw, 401, "unauthorized APIKey")
		return
	}

	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&webhooksReq)
	if err != nil || webhooksReq.Event == "" || webhooksReq.Data.UserID == "" {
		writeErrorResponse(rw, 400, "invalid request")
		return
	}
	defer r.Body.Close()

	if webhooksReq.Event == "user.upgraded" {
		userUUID, err := validateUUID(webhooksReq.Data.UserID, "user ID")
		if err != nil {
			writeErrorResponse(rw, 403, "couldn't parse that userID")
			return
		}

		_, err = cfg.db.GetUserByID(r.Context(), userUUID)
		if err != nil {
			writeErrorResponse(rw, 404, "user not found")
			return
		}

		err = cfg.db.UpgradeUserToRed(r.Context(), userUUID)
		if err != nil {
			writeErrorResponse(rw, 403, "couldn't upgrade the user")
			return
		}
	}

	writeEmptyResponse(rw, 204)
}

func (cfg *apiConfig) handlerCreateChirp(rw http.ResponseWriter, r *http.Request) {
	type ChirpyPostReq struct {
		Body string `json:"body"`
	}
	chirpyPostReq := ChirpyPostReq{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&chirpyPostReq)
	if err != nil {
		writeErrorResponse(rw, 400, "Something went wrong")
		return
	}
	defer r.Body.Close()

	// Validate chirp body
	if err := validateChirpBody(chirpyPostReq.Body); err != nil {
		writeErrorResponse(rw, 400, err.Error())
		return
	}

	// Validate JWT and get user UUID
	userUUID, err := cfg.validateJWTFromRequest(r)
	if err != nil {
		writeErrorResponse(rw, 401, err.Error())
		return
	}

	// Clean the post content
	cleanedPost := cleanChirpContent(chirpyPostReq.Body)

	chirpyPost, err := cfg.db.CreateChirpy(r.Context(), database.CreateChirpyParams{
		ID:        uuid.New(),
		Body:      cleanedPost,
		UserID:    userUUID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		writeErrorResponse(rw, 403, fmt.Sprintf("couldn't create a chirpy: %v", err))
		return
	}

	writeSuccessResponse(rw, 201, map[string]any{
		"id":         chirpyPost.ID,
		"body":       chirpyPost.Body,
		"user_id":    chirpyPost.UserID,
		"created_at": chirpyPost.CreatedAt,
		"updated_at": chirpyPost.UpdatedAt,
	})
}

// Response helper functions
func encodeJson(params map[string]any) ([]byte, error) {
	return json.Marshal(params)
}

// writeJSONResponse writes a JSON response with the given status code and data
func writeJSONResponse(rw http.ResponseWriter, statusCode int, data map[string]any) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(statusCode)

	response, err := encodeJson(data)
	if err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		return
	}

	if _, err := rw.Write(response); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

// writeErrorResponse writes an error response with the given status code and message
func writeErrorResponse(rw http.ResponseWriter, statusCode int, message string) {
	writeJSONResponse(rw, statusCode, map[string]any{
		"error": message,
	})
}

// writeSuccessResponse writes a success response with the given status code and data
func writeSuccessResponse(rw http.ResponseWriter, statusCode int, data map[string]any) {
	writeJSONResponse(rw, statusCode, data)
}

// writeEmptyResponse writes an empty response with the given status code
func writeEmptyResponse(rw http.ResponseWriter, statusCode int) {
	rw.WriteHeader(statusCode)
}

// Authentication helper functions
func (cfg *apiConfig) validateJWTFromRequest(r *http.Request) (uuid.UUID, error) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil || token == "" {
		return uuid.Nil, fmt.Errorf("unauthorized: invalid user JWT")
	}

	userUUID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		return uuid.Nil, fmt.Errorf("unauthorized: %v", err)
	}

	return userUUID, nil
}

// Request validation helper functions
func validateRequiredFields(fields map[string]string) error {
	for field, value := range fields {
		if value == "" {
			return fmt.Errorf("missing required field: %s", field)
		}
	}
	return nil
}

func validateChirpBody(body string) error {
	if body == "" {
		return fmt.Errorf("chirp body cannot be empty")
	}
	if len(body) > 140 {
		return fmt.Errorf("chirp body too long")
	}
	return nil
}

func validateUUID(uuidStr, fieldName string) (uuid.UUID, error) {
	if uuidStr == "" {
		return uuid.Nil, fmt.Errorf("missing %s", fieldName)
	}

	parsedUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s format", fieldName)
	}

	return parsedUUID, nil
}

func cleanChirpContent(body string) string {
	// prevent these words - has to be without "!"
	preventedKeywords := []string{"kerfuffle", "Kerfuffle", "sharbert", "Sharbert", "fornax", "Fornax"}

	keywordsToCheck := strings.Split(body, " ")

	for i, keyword := range keywordsToCheck {
		for _, preventedKeyword := range preventedKeywords {
			if keyword == preventedKeyword {
				keywordsToCheck[i] = "****"
			}
		}
	}

	return strings.Join(keywordsToCheck, " ")
}

func (cfg *apiConfig) handlerCreateUser(rw http.ResponseWriter, r *http.Request) {
	type emailRequest struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	var emailReq emailRequest

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&emailReq)
	if err != nil {
		writeErrorResponse(rw, 400, "invalid request")
		return
	}
	defer r.Body.Close()

	// Validate required fields
	if err := validateRequiredFields(map[string]string{
		"email":    emailReq.Email,
		"password": emailReq.Password,
	}); err != nil {
		writeErrorResponse(rw, 400, "invalid request")
		return
	}

	hashedPassword, err := auth.HashPassword(emailReq.Password)
	if err != nil {
		writeErrorResponse(rw, 403, "couldn't generate hashed password")
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
		writeErrorResponse(rw, 401, "couldn't create the user")
		return
	}

	writeSuccessResponse(rw, 201, map[string]any{
		"id":            user.ID,
		"email":         user.Email,
		"is_chirpy_red": user.IsChirpyRed,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
	})
}

func (cfg *apiConfig) handlerUpdateUser(rw http.ResponseWriter, r *http.Request) {
	type UserUpdateRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var userUpdateReq UserUpdateRequest

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&userUpdateReq)
	if err != nil {
		writeErrorResponse(rw, 400, "invalid request")
		return
	}
	defer r.Body.Close()

	// Validate required fields
	if err := validateRequiredFields(map[string]string{
		"email":    userUpdateReq.Email,
		"password": userUpdateReq.Password,
	}); err != nil {
		writeErrorResponse(rw, 400, "invalid request")
		return
	}

	// Validate JWT and get user UUID
	userUUID, err := cfg.validateJWTFromRequest(r)
	if err != nil {
		writeErrorResponse(rw, 401, err.Error())
		return
	}

	hashedPassword, err := auth.HashPassword(userUpdateReq.Password)
	if err != nil {
		writeErrorResponse(rw, 403, "couldn't hash user's password")
		return
	}

	updatedUser, err := cfg.db.UpdateUser(r.Context(), database.UpdateUserParams{
		Email:          userUpdateReq.Email,
		HashedPassword: hashedPassword,
		ID:             userUUID,
	})
	if err != nil {
		writeErrorResponse(rw, 403, "couldn't update the user")
		return
	}

	writeSuccessResponse(rw, 200, map[string]any{
		"id":            updatedUser.ID,
		"email":         updatedUser.Email,
		"is_chirpy_red": updatedUser.IsChirpyRed,
		"created_at":    updatedUser.CreatedAt,
		"updated_at":    updatedUser.UpdatedAt,
	})
}

func (cfg *apiConfig) handlerLoginUser(rw http.ResponseWriter, r *http.Request) {
	// default expiry time, unless modified by the client's request
	ExpiresIn := time.Hour * 1
	RefreshTokenExpireIn := time.Hour * 24 * 60

	type LoginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var loginReq LoginRequest

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&loginReq)
	if err != nil {
		writeErrorResponse(rw, 400, "invalid request")
		return
	}
	defer r.Body.Close()

	// Validate required fields
	if err := validateRequiredFields(map[string]string{
		"email":    loginReq.Email,
		"password": loginReq.Password,
	}); err != nil {
		writeErrorResponse(rw, 400, "invalid request")
		return
	}

	// Get the user data for password verification
	user, err := cfg.db.GetUserByEmail(r.Context(), loginReq.Email)
	if err != nil {
		writeErrorResponse(rw, 403, "user not found")
		return
	}

	// Hash the User password
	HashedPassByteSli := []byte(user.HashedPassword)
	err = auth.ComparePasswordHash(loginReq.Password, string(HashedPassByteSli))
	if err != nil {
		writeErrorResponse(rw, 401, "password is not correct")
		return
	}

	// Password is Valid
	// Generate JWT for the user
	generatedToken, err := auth.MakeJWT(
		user.ID,
		cfg.jwtSecret,
		ExpiresIn,
	)
	if err != nil {
		writeErrorResponse(rw, 403, "couldn't generate access token for the user")
		return
	}

	// Also Generate A Refresh Token with 60days so the user can stay longer on the platform
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil || refreshToken == "" {
		writeErrorResponse(rw, 403, "couldn't generate refresh token for the user")
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

	writeSuccessResponse(rw, 200, map[string]any{
		"id":            user.ID,
		"email":         user.Email,
		"is_chirpy_red": user.IsChirpyRed,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
		"token":         generatedToken,
		"refresh_token": refreshToken,
	})
}

func (cfg *apiConfig) handlerRefreshToken(rw http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil || refreshToken == "" {
		writeErrorResponse(rw, 401, "couldn't retrieve refresh token for the user")
		return
	}

	// Fetch the refreshToken from the DB & check if it's still valid/not-expired
	foundRefreshToken, err := cfg.db.GetRefreshToken(r.Context(), refreshToken)
	if err != nil || (time.Since(foundRefreshToken.ExpiresAt) > 0) || foundRefreshToken.RevokedAt.Valid {
		writeErrorResponse(rw, 401, "either the refresh token is expired or not found")
		return
	}

	// Refresh token is still valid => Create an Access Token for the user, as the current one is expired, that's why this RefreshToken api is called in the first place
	user, err := cfg.db.GetUserFromRefreshToken(r.Context(), foundRefreshToken.UserID)
	if err != nil {
		writeErrorResponse(rw, 404, "user not found")
		return
	}

	JWT, err := auth.MakeJWT(user.UserID, cfg.jwtSecret, time.Hour*1)
	if err != nil {
		writeErrorResponse(rw, 401, "couldn't create Access Token for the user")
		return
	}

	writeSuccessResponse(rw, 200, map[string]any{
		"token": JWT,
	})
}

func (cfg *apiConfig) handlerRevokeRefreshToken(rw http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil || refreshToken == "" {
		writeErrorResponse(rw, 401, "couldn't retrieve refresh token for the user")
		return
	}

	err = cfg.db.UpdateRefreshToken(r.Context(), database.UpdateRefreshTokenParams{
		RevokedAt: sql.NullTime{Time: time.Now(), Valid: true},
		UpdatedAt: time.Now(),
		Token:     refreshToken,
	})
	if err != nil {
		writeErrorResponse(rw, 403, "couldn't update the Refresh token record.")
		return
	}

	writeEmptyResponse(rw, 204)
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
	// Continue only if this is "local/dev" environment
	if cfg.platform != "dev" {
		writeErrorResponse(rw, 403, "this api is forbidden")
		return
	}

	err := cfg.db.DeleteUsers(r.Context())
	if err != nil {
		writeErrorResponse(rw, 500, "couldn't delete users")
		return
	}

	cfg.fileserverHits.Store(0)
	writeEmptyResponse(rw, 200)
}
