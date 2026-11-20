package util

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims defines standard and custom token claims
type JWTClaims struct {
	UserID    string  `json:"sub"`
	Name      string  `json:"name,omitempty"`
	Email     string  `json:"email"`
	Role      string  `json:"role"`
	HotelID   *string `json:"hotel_id,omitempty"`
	HotelName string  `json:"hotel_name,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken generates a signed JWT token
func GenerateToken(userID, name, email, role string, hotelID *string, hotelName string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "super_secure_jwt_secret_key_change_in_production_2024!"
	}

	expiryHours := 24
	if envExp := os.Getenv("TOKEN_EXPIRY_HOURS"); envExp != "" {
		if val, err := strconv.Atoi(envExp); err == nil && val > 0 {
			expiryHours = val
		}
	}

	claims := JWTClaims{
		UserID:    userID,
		Name:      name,
		Email:     email,
		Role:      role,
		HotelID:   hotelID,
		HotelName: hotelName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken parses and validates a token string
func ValidateToken(tokenString string) (*JWTClaims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "super_secure_jwt_secret_key_change_in_production_2024!"
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	return claims, nil
}
