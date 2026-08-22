package httpx

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type Principal struct {
	UserID string
	Role   string
	Kind   string
}

type APIKeyValidator interface {
	Validate(ctx context.Context, apiKey string) bool
}

type Auth struct {
	InternalToken string
	JWTSecret     []byte
	APIKeys       []string
	KeyValidator  APIKeyValidator
}

func (a Auth) Internal(r *http.Request) bool {
	got := strings.TrimSpace(r.Header.Get("X-Internal-Token"))
	if a.InternalToken == "" || got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(a.InternalToken)) == 1
}

func (a Auth) APIKey(r *http.Request) bool {
	got := strings.TrimSpace(r.Header.Get("X-Api-Key"))
	if got == "" {
		return false
	}
	for _, key := range a.APIKeys {
		if key != "" && subtle.ConstantTimeCompare([]byte(got), []byte(key)) == 1 {
			return true
		}
	}
	if a.KeyValidator != nil {
		return a.KeyValidator.Validate(r.Context(), got)
	}
	return false
}

func (a Auth) JWT(r *http.Request) (Principal, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return Principal{}, errors.New("missing bearer")
	}
	token, err := jwt.Parse(strings.TrimPrefix(header, "Bearer "), func(t *jwt.Token) (any, error) {
		return a.JWTSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return Principal{}, errors.New("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return Principal{}, errors.New("invalid claims")
	}
	sub, _ := claims["sub"].(string)
	role, _ := claims["role"].(string)
	if sub == "" {
		return Principal{}, errors.New("missing sub")
	}
	return Principal{UserID: sub, Role: role, Kind: "jwt"}, nil
}
