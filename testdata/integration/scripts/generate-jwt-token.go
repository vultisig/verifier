package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	PublicKey string `json:"public_key"`
	TokenID   string `json:"token_id"`
	jwt.RegisteredClaims
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <jwt_secret> <public_key> [token_id]\n", os.Args[0])
		os.Exit(1)
	}

	jwtSecret := os.Args[1]
	publicKey := os.Args[2]
	tokenID := "integration-token-1"
	if len(os.Args) > 3 {
		tokenID = os.Args[3]
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		PublicKey: publicKey,
		TokenID:   tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		log.Fatalf("Failed to create token: %v", err)
	}

	fmt.Println(tokenString)
}
