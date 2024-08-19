package token

import (
	"crypto/rsa"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"log"
	"os"
)

var (
	verifyKey *rsa.PublicKey
)

func InitVerifyKey(verifyKeyFilePath string) {
	// Read the private key
	privateKeyData, err := os.ReadFile(verifyKeyFilePath)
	if err != nil {
		log.Fatalf("Error reading private key: %v", err)
	}

	// Parse the public key
	verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(privateKeyData)
	if err != nil {
		log.Fatalf("Error parsing private key: %v", err)
	}
}
func VerifyToken(tokenString string) (jwt.MapClaims, error) {
	if verifyKey == nil {
		return nil, fmt.Errorf("verify key is not initialized")
	}

	token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return verifyKey, nil
	})

	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	} else {
		return nil, fmt.Errorf("failed to parse claims")
	}
}
