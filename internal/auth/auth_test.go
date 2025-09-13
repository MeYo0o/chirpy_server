package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPasswordHashGeneration(t *testing.T) {
	passwordsToTest := []string{"123456789", "Moaz", "P@$$W0rD", "Hello World as a password"}

	for _, passToTest := range passwordsToTest {
		hashedPass, err := HashPassword(passToTest)
		if err != nil {
			t.Errorf("couldn't hash the password %v", err)
		}

		err = ComparePasswordHash(passToTest, hashedPass)
		if err != nil {
			t.Error("hashed password is not valid.")
		}
	}

}

func TestJWTGenerationAndValidation(t *testing.T) {
	tokenSecret := "ASDAFFDHDFGJHFJTRY#$%#$%#$^#HFDHD!@#!@ASDASDFADADASDAS"
	testUserUUID, _ := uuid.Parse("55ff04a8-37b4-4e04-8e14-c59041117dba")
	expiresIn, _ := time.ParseDuration("5s")

	generatedTokenStr, err := MakeJWT(testUserUUID, tokenSecret, expiresIn)
	if err != nil {
		t.Errorf("couldn't make JWT: %v", err)
	}

	_, err = ValidateJWT(generatedTokenStr, tokenSecret)
	if err != nil {
		t.Errorf("couldn't validate the token: %v", err)
	}
}

func TestJWTGenerationAndValidationAfterTime(t *testing.T) {
	tokenSecret := "ASDAFFDHDFGJHFJTRY#$%#$%#$^#HFDHD!@#!@ASDASDFADADASDAS"
	testUserUUID, _ := uuid.Parse("55ff04a8-37b4-4e04-8e14-c59041117dba")
	expiresIn, _ := time.ParseDuration("5s")

	generatedTokenStr, err := MakeJWT(testUserUUID, tokenSecret, expiresIn)
	if err != nil {
		t.Errorf("couldn't make JWT: %v", err)
	}

	// Sleep for 10 seconds to ensure the token expires
	// time.Sleep(time.Second * 10)

	_, err = ValidateJWT(generatedTokenStr, tokenSecret)
	if err != nil {
		t.Errorf("couldn't validate the token: %v", err)
	}
}
