package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const sessionIssuer = "users-service"

var errUnauthenticated = errors.New("missing or invalid session")

// authenticate returns the user ID from a valid "Authorization: Bearer" session token.
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

func writeUnauthenticated(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, errorBody{
		Error:   "UNAUTHENTICATED",
		Message: "Your session has expired, please sign in again",
	})
}
