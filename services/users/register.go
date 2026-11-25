package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

type registrationService struct {
	store        *userStore
	mailer       *smtpMailer
	appBaseURL   string
	defaultQuota int64
	tokenTTL     time.Duration
}

type registerRequest struct {
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

type registerResponse struct {
	ID                string `json:"id"`
	Email             string `json:"email"`
	Status            string `json:"status"`
	StorageQuotaBytes int64  `json:"storageQuotaBytes"`
	Message           string `json:"message"`
}

type confirmRequest struct {
	Token string `json:"token"`
}

type validationErrorBody struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields"`
}

const (
	maxNameLen        = 100
	maxEmailLen       = 254
	minPasswordLen    = 8
	maxPasswordBytes  = 72 // bcrypt only uses the first 72 bytes
	maxRequestBodyLen = 1 << 14
)

func validateRegistration(req *registerRequest) map[string]string {
	fields := map[string]string{}

	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	switch {
	case req.FirstName == "":
		fields["firstName"] = "First name is required"
	case len(req.FirstName) > maxNameLen:
		fields["firstName"] = "First name is too long"
	}
	switch {
	case req.LastName == "":
		fields["lastName"] = "Last name is required"
	case len(req.LastName) > maxNameLen:
		fields["lastName"] = "Last name is too long"
	}

	if req.Email == "" {
		fields["email"] = "Email is required"
	} else if addr, err := mail.ParseAddress(req.Email); err != nil || addr.Address != req.Email || len(req.Email) > maxEmailLen {
		fields["email"] = "Email is not valid"
	}

	if req.Password == "" {
		fields["password"] = "Password is required"
	} else if msg := passwordWeakness(req.Password); msg != "" {
		fields["password"] = msg
	}

	if req.ConfirmPassword == "" {
		fields["confirmPassword"] = "Please confirm your password"
	} else if req.Password != req.ConfirmPassword {
		fields["confirmPassword"] = "Passwords do not match"
	}

	return fields
}

// passwordWeakness returns a user-facing reason the password is too weak, or "".
func passwordWeakness(p string) string {
	if len([]rune(p)) < minPasswordLen {
		return "Password must be at least 8 characters long"
	}
	if len(p) > maxPasswordBytes {
		return "Password must be at most 72 bytes long"
	}
	var upper, lower, digit, special bool
	for _, r := range p {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r) || unicode.IsSpace(r):
			special = true
		}
	}
	if !upper || !lower || !digit || !special {
		return "Password must contain an uppercase letter, a lowercase letter, a digit and a special character"
	}
	return ""
}

func newToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // UUID v4
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:], nil
}

func internalError(w http.ResponseWriter, msg string, err error) {
	log.Printf("%s: %v", msg, err)
	writeJSON(w, http.StatusInternalServerError, errorBody{
		Error:   "INTERNAL_ERROR",
		Message: "Something went wrong, please try again later",
	})
}

func (s *registrationService) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodyLen))
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "INVALID_BODY", Message: "Request body must be valid JSON"})
		return
	}

	if fields := validateRegistration(&req); len(fields) > 0 {
		writeJSON(w, http.StatusBadRequest, validationErrorBody{
			Error:   "VALIDATION_ERROR",
			Message: "Some fields are invalid",
			Fields:  fields,
		})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		internalError(w, "hash password", err)
		return
	}
	id, err := newID()
	if err != nil {
		internalError(w, "generate id", err)
		return
	}
	token, tokenHash, err := newToken()
	if err != nil {
		internalError(w, "generate token", err)
		return
	}

	now := time.Now().UTC()
	user := &User{
		ID:                id,
		FirstName:         req.FirstName,
		LastName:          req.LastName,
		Email:             req.Email,
		PasswordHash:      string(passwordHash),
		Status:            StatusPending,
		StorageQuotaBytes: s.defaultQuota,
		StorageUsedBytes:  0,
		CreatedAt:         now,
		Confirmation:      &confirmation{TokenHash: tokenHash, ExpiresAt: now.Add(s.tokenTTL)},
	}

	if err := s.store.Create(r.Context(), user); err != nil {
		if errors.Is(err, ErrEmailTaken) {
			writeJSON(w, http.StatusConflict, validationErrorBody{
				Error:   "EMAIL_TAKEN",
				Message: "An account with this email already exists",
				Fields:  map[string]string{"email": "An account with this email already exists"},
			})
			return
		}
		internalError(w, "create user", err)
		return
	}

	link := s.appBaseURL + "/confirm?token=" + token
	if err := s.mailer.SendConfirmation(user.Email, user.FirstName, link, s.tokenTTL); err != nil {
		log.Printf("send confirmation to user %s: %v", user.ID, err)
		// Roll back so the person can register again with the same email.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if delErr := s.store.Delete(ctx, user.ID); delErr != nil {
			log.Printf("rollback user %s: %v", user.ID, delErr)
		}
		writeJSON(w, http.StatusBadGateway, errorBody{
			Error:   "EMAIL_DELIVERY_FAILED",
			Message: "We could not send the confirmation email, please try again later",
		})
		return
	}

	writeJSON(w, http.StatusCreated, registerResponse{
		ID:                user.ID,
		Email:             user.Email,
		Status:            user.Status,
		StorageQuotaBytes: user.StorageQuotaBytes,
		Message:           "Registration successful. Check your email to confirm your account.",
	})
}

func (s *registrationService) confirm(w http.ResponseWriter, r *http.Request) {
	var req confirmRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodyLen))
	if err := dec.Decode(&req); err != nil || strings.TrimSpace(req.Token) == "" {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "INVALID_BODY", Message: "A confirmation token is required"})
		return
	}

	err := s.store.Activate(r.Context(), hashToken(strings.TrimSpace(req.Token)), time.Now().UTC())
	switch {
	case errors.Is(err, ErrInvalidToken):
		writeJSON(w, http.StatusBadRequest, errorBody{
			Error:   "INVALID_TOKEN",
			Message: "This confirmation link is invalid or has already been used",
		})
	case errors.Is(err, ErrTokenExpired):
		writeJSON(w, http.StatusGone, errorBody{
			Error:   "TOKEN_EXPIRED",
			Message: "This confirmation link has expired",
		})
	case err != nil:
		internalError(w, "activate user", err)
	default:
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  StatusActive,
			"message": "Your account is confirmed. You can now sign in.",
		})
	}
}
