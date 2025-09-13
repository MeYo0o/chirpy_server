package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	passByteSli := []byte(password)
	hashByteSli, err := bcrypt.GenerateFromPassword(passByteSli, 10)
	if err != nil {
		return "", fmt.Errorf("couldn't hash the password.")
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
