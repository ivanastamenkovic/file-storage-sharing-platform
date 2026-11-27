package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const sessionIssuer = "users-service"

var errUnauthenticated = errors.New("unauthenticated")

type sessionClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func authenticate(r *http.Request, secret []byte) (string, error) {
	raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	raw = strings.TrimSpace(raw)
	if !ok || raw == "" {
		return "", errUnauthenticated
	}

	var claims sessionClaims
	_, err := jwt.ParseWithClaims(raw, &claims,
		func(*jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(sessionIssuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || claims.Subject == "" {
		return "", errUnauthenticated
	}
	return claims.Subject, nil
}

func stripIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("X-User-Id")
		next.ServeHTTP(w, r)
	})
}

func requireAuth(secret []byte, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := authenticate(r, secret)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, errorBody{
				Error:   "UNAUTHENTICATED",
				Message: "Your session has expired, please sign in again",
			})
			return
		}
		r.Header.Set("X-User-Id", userID)
		next.ServeHTTP(w, r)
	})
}
