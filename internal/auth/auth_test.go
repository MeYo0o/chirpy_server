package auth

import "testing"

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
