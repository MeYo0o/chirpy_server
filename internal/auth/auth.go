package auth

import (
	"fmt"
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
