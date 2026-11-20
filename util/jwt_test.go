package util

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWT_GenerateAndValidateToken(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret_key_12345!")
	os.Setenv("TOKEN_EXPIRY_HOURS", "2")

	userID := "user-uuid-100"
	email := "admin@mumbaihotel.in"
	role := "hotel_admin"
	hotelID := "hotel-uuid-mumbai"
	hotelName := "Grand Horizon Hotel"

	// 1. Generate token
	tokenStr, err := GenerateToken(userID, "Arjun Sharma", email, role, &hotelID, hotelName)
	if err != nil {
		t.Fatalf("expected no error generating token, got: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("expected non-empty token string")
	}

	// 2. Validate token
	claims, err := ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("expected valid token, got error: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("expected Email %s, got %s", email, claims.Email)
	}
	if claims.Role != role {
		t.Errorf("expected Role %s, got %s", role, claims.Role)
	}
	if claims.HotelID == nil || *claims.HotelID != hotelID {
		t.Errorf("expected HotelID %s, got %v", hotelID, claims.HotelID)
	}
	if claims.HotelName != hotelName {
		t.Errorf("expected HotelName %s, got %s", hotelName, claims.HotelName)
	}
}

func TestJWT_TamperedTokenRejected(t *testing.T) {
	os.Setenv("JWT_SECRET", "correct_secret_key")

	tokenStr, _ := GenerateToken("user-1", "Test User", "user@test.com", "hotel_admin", nil, "")

	// Tamper token by changing characters
	tamperedToken := tokenStr + "tampered"
	_, err := ValidateToken(tamperedToken)
	if err == nil {
		t.Fatal("expected error validating tampered token, got nil")
	}
}

func TestJWT_ExpiredTokenRejected(t *testing.T) {
	secret := "test_secret_for_expiry"
	os.Setenv("JWT_SECRET", secret)

	// Create intentionally expired claims
	claims := JWTClaims{
		UserID: "user-expired",
		Email:  "expired@test.com",
		Role:   "hotel_admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hr ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to create expired test token: %v", err)
	}

	// Validation must fail
	_, err = ValidateToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for expired token, but got nil")
	}
}
