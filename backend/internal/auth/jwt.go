package auth 

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(userID, role, secret string, expiryHours int) (string, time.Time, string, error) {
	expiresAt := time.Now().Add(time.Duration(expiryHours) * time.Hour)
	jti := uuid.NewString()
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},

	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	return signed, expiresAt, jti, err

}

func ValidateToken(tokenStr, secret string) (*Claims,error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret),nil
	})
	if err != nil {
		return nil,err 
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil 
}

