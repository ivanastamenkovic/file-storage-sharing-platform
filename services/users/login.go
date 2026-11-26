package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type loginService struct {
	users          *userStore
	challenges     *challengeStore
	mailer         *smtpMailer
	otpTTL         time.Duration
	otpMaxAttempts int
	jwtSecret      []byte
	sessionTTL     time.Duration
	passwordMaxAge time.Duration
	// Compared against when the email is unknown, so a missing account
	// takes as long to reject as a wrong password.
	dummyHash []byte
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	ChallengeID      string `json:"challengeId"`
	ExpiresInSeconds int    `json:"expiresInSeconds"`
	Message          string `json:"message"`
}

type verifyRequest struct {
	ChallengeID string `json:"challengeId"`
	OTP         string `json:"otp"`
}

type sessionUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type sessionResponse struct {
	Token              string      `json:"token"`
	TokenType          string      `json:"tokenType"`
	ExpiresAt          time.Time   `json:"expiresAt"`
	MustChangePassword bool        `json:"mustChangePassword"`
	User               sessionUser `json:"user"`
}

type sessionClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (s *loginService) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodyLen)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "INVALID_BODY", Message: "Request body must be valid JSON"})
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "VALIDATION_ERROR", Message: "Email and password are required"})
		return
	}

	user, err := s.users.FindByEmail(r.Context(), email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		internalError(w, "find user", err)
		return
	}

	hash := s.dummyHash
	if user != nil {
		hash = []byte(user.PasswordHash)
	}
	passwordOK := bcrypt.CompareHashAndPassword(hash, []byte(req.Password)) == nil
	if user == nil || !passwordOK {
		writeJSON(w, http.StatusUnauthorized, errorBody{
			Error:   "INVALID_CREDENTIALS",
			Message: "Email or password is incorrect",
		})
		return
	}

	// Only reported after a correct password, so it reveals nothing to a stranger.
	if user.Status != StatusActive {
		writeJSON(w, http.StatusForbidden, errorBody{
			Error:   "ACCOUNT_NOT_CONFIRMED",
			Message: "Your account is not confirmed yet. Check your email for the confirmation link.",
		})
		return
	}

	challengeID, _, err := newToken()
	if err != nil {
		internalError(w, "generate challenge", err)
		return
	}
	otp, err := newOTP()
	if err != nil {
		internalError(w, "generate otp", err)
		return
	}

	now := time.Now().UTC()
	if err := s.challenges.Create(r.Context(), &loginChallenge{
		ID:        challengeID,
		UserID:    user.ID,
		OTPHash:   hashOTP(challengeID, otp),
		ExpiresAt: now.Add(s.otpTTL),
		CreatedAt: now,
	}); err != nil {
		internalError(w, "create challenge", err)
		return
	}

	if err := s.mailer.SendLoginCode(user.Email, user.FirstName, otp, s.otpTTL); err != nil {
		log.Printf("send login code to user %s: %v", user.ID, err)
		writeJSON(w, http.StatusBadGateway, errorBody{
			Error:   "EMAIL_DELIVERY_FAILED",
			Message: "We could not send your login code, please try again later",
		})
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		ChallengeID:      challengeID,
		ExpiresInSeconds: int(s.otpTTL.Seconds()),
		Message:          "We sent a one-time code to your email.",
	})
}

func (s *loginService) verify(w http.ResponseWriter, r *http.Request) {
	var req verifyRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodyLen)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "INVALID_BODY", Message: "Request body must be valid JSON"})
		return
	}
	otp := strings.TrimSpace(req.OTP)
	if req.ChallengeID == "" || otp == "" {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "VALIDATION_ERROR", Message: "The one-time code is required"})
		return
	}

	userID, attemptsLeft, err := s.challenges.Consume(r.Context(), req.ChallengeID, otp, time.Now().UTC(), s.otpMaxAttempts)
	switch {
	case errors.Is(err, ErrOTPExpired):
		writeJSON(w, http.StatusGone, errorBody{Error: "OTP_EXPIRED", Message: "This code has expired. Sign in again to get a new one."})
		return
	case errors.Is(err, ErrOTPTooManyAttempt):
		writeJSON(w, http.StatusTooManyRequests, errorBody{Error: "TOO_MANY_ATTEMPTS", Message: "Too many wrong codes. Sign in again to get a new one."})
		return
	case errors.Is(err, ErrOTPInvalid):
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"error":        "INVALID_OTP",
			"message":      "The code is invalid or has already been used",
			"attemptsLeft": attemptsLeft,
		})
		return
	case err != nil:
		internalError(w, "consume challenge", err)
		return
	}

	user, err := s.users.FindByID(r.Context(), userID)
	if err != nil {
		internalError(w, "load user", err)
		return
	}
	if user.Status != StatusActive {
		writeJSON(w, http.StatusForbidden, errorBody{Error: "ACCOUNT_NOT_CONFIRMED", Message: "Your account is not confirmed yet"})
		return
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.sessionTTL)
	jti, _, err := newToken()
	if err != nil {
		internalError(w, "generate jti", err)
		return
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, sessionClaims{
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    sessionIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        jti,
		},
	}).SignedString(s.jwtSecret)
	if err != nil {
		internalError(w, "sign token", err)
		return
	}

	writeJSON(w, http.StatusOK, sessionResponse{
		Token:              token,
		TokenType:          "Bearer",
		ExpiresAt:          expiresAt,
		MustChangePassword: user.PasswordExpired(now, s.passwordMaxAge),
		User: sessionUser{
			ID:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		},
	})
}
