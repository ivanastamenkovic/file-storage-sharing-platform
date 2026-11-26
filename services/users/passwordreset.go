package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

const passwordResetGenericMessage = "If an account with this email exists, we sent a link to reset the password."

type passwordResetService struct {
	users      *userStore
	tokens     *authTokenStore
	mailer     *smtpMailer
	appBaseURL string
	tokenTTL   time.Duration
	cooldown   time.Duration
}

type passwordResetRequest struct {
	Email string `json:"email"`
}

func (s *passwordResetService) request(w http.ResponseWriter, r *http.Request) {
	var req passwordResetRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodyLen)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "INVALID_BODY", Message: "Request body must be valid JSON"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	addr, err := mail.ParseAddress(email)
	if email == "" || err != nil || addr.Address != email || len(email) > maxEmailLen {
		writeJSON(w, http.StatusBadRequest, validationErrorBody{
			Error:   "VALIDATION_ERROR",
			Message: "Some fields are invalid",
			Fields:  map[string]string{"email": "Email is not valid"},
		})
		return
	}

	go s.process(email)

	writeJSON(w, http.StatusOK, map[string]string{"message": passwordResetGenericMessage})
}

func (s *passwordResetService) process(email string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	user, err := s.users.FindByEmail(ctx, email)
	if errors.Is(err, ErrUserNotFound) {
		log.Printf("password reset requested for an unknown account")
		return
	}
	if err != nil {
		log.Printf("password reset: find user: %v", err)
		return
	}
	if user.Status != StatusActive {
		log.Printf("password reset requested for inactive account %s", user.ID)
		return
	}

	now := time.Now().UTC()
	if s.cooldown > 0 {
		recent, err := s.tokens.CreatedSince(ctx, user.ID, TokenPasswordReset, now.Add(-s.cooldown))
		if err != nil {
			log.Printf("password reset: cooldown check for user %s: %v", user.ID, err)
			return
		}
		if recent {
			log.Printf("password reset for user %s skipped: cooldown", user.ID)
			return
		}
	}

	if err := s.tokens.InvalidateUnused(ctx, user.ID, TokenPasswordReset); err != nil {
		log.Printf("password reset: invalidate old tokens for user %s: %v", user.ID, err)
		return
	}

	token, tokenHash, err := newToken()
	if err != nil {
		log.Printf("password reset: generate token: %v", err)
		return
	}
	if err := s.tokens.Create(ctx, &authToken{
		TokenHash: tokenHash,
		UserID:    user.ID,
		Type:      TokenPasswordReset,
		ExpiresAt: now.Add(s.tokenTTL),
		CreatedAt: now,
	}); err != nil {
		log.Printf("password reset: store token for user %s: %v", user.ID, err)
		return
	}

	link := s.appBaseURL + "/reset-password?token=" + token
	if err := s.mailer.SendPasswordReset(user.Email, user.FirstName, link, s.tokenTTL); err != nil {
		log.Printf("password reset: send email to user %s: %v", user.ID, err)
		return
	}
	log.Printf("password reset link sent to user %s", user.ID)
}