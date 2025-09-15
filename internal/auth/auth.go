package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	passByteSli := []byte(password)
	hashByteSli, err := bcrypt.GenerateFromPassword(passByteSli, 10)
	if err != nil {
		return "", fmt.Errorf("couldn't hash the password: %v", err)
	}
	hashedPassStr := string(hashByteSli)

	return hashedPassStr, nil
}

func ComparePasswordHash(password, hash string) error {
	hashByteSli := []byte(hash)
	passByteSli := []byte(password)

	err := bcrypt.CompareHashAndPassword(hashByteSli, passByteSli)
	if err != nil {
		return fmt.Errorf("password is not correct")
	}

	return nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer: "chirpy",
		IssuedAt: &jwt.NumericDate{
			Time: time.Now().UTC(),
		},
		ExpiresAt: &jwt.NumericDate{
			Time: time.Time.Add(time.Now().UTC(), expiresIn),
		},
		Subject: userID.String(),
	})

	return jwtToken.SignedString([]byte(tokenSecret))
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {

	jwtToken, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(t *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("token is invalid or expired: %v", err)
	}

	expiresAt, err := jwtToken.Claims.GetExpirationTime()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("couldn't extract the Expiration Date from this token: %v", err)
	}

	if time.Now().UTC().Sub(expiresAt.Time) > 0 {
		return uuid.UUID{}, fmt.Errorf("token is expired: %v", err)
	}

	userID, err := jwtToken.Claims.GetSubject()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("couldn't extract userID from this token: %v", err)
	}

	return uuid.Parse(userID)
}

func GetBearerToken(headers http.Header) (string, error) {
	bearerToken := headers.Get("Authorization")

	if bearerToken == "" {
		return "", fmt.Errorf("couldn't extract Bearer token")
	}

	token := strings.Replace(bearerToken, "Bearer ", "", 1)

	return token, nil

}

func MakeRefreshToken() (string, error) {
	tokenByteSli := make([]byte, 32)
	_, err := rand.Read(tokenByteSli)
	if err != nil {
		return "", fmt.Errorf("couldn't make refresh token for the user: %w", err)
	}

	return hex.EncodeToString(tokenByteSli), nil
}

func GetAPIKey(header http.Header) (string, error) {
	apiKeyToken := header.Get("Authorization")

	if apiKeyToken == "" {
		return "", fmt.Errorf("couldn't extract the Authorization header")
	}

	apiKey := strings.Replace(apiKeyToken, "ApiKey ", "", 1)

	return apiKey, nil

}
