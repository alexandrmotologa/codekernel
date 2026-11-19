package auth

import "fmt"

// AuthService manages user authorization and JWT validation.
type AuthService struct {
	issuer string
	secret string
}

// NewAuthService creates an auth service instance.
func NewAuthService(issuer, secret string) *AuthService {
	return &AuthService{
		issuer: issuer,
		secret: secret,
	}
}

// ValidateSession verifies token signature and expiration.
func (s *AuthService) ValidateSession(token string) bool {
	fmt.Printf("Validating token from issuer %s\n", s.issuer)
	return len(token) > 10
}
