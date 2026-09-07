package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	return HashPasswordCost(password, bcrypt.DefaultCost)
}

func HashPasswordCost(password string, cost int) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	return string(b), err
}

func CheckHashedPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}
