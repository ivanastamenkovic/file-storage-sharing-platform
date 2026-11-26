package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type passwordChangeService struct {
	users     *userStore
	jwtSecret []byte
	// A password must be at least minAge old before it can be changed again.
	minAge time.Duration
	// A password older than maxAge has to be changed; zero disables expiry.
	maxAge time.Duration
}

type passwordChangeRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}

type passwordStatusResponse struct {
	ChangedAt   time.Time  `json:"changedAt"`
	CanChangeAt time.Time  `json:"canChangeAt"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	MustChange  bool       `json:"mustChange"`
}

func (s *passwordChangeService) status(u *User, now time.Time) passwordStatusResponse {
	setAt := u.PasswordSetAt()
	res := passwordStatusResponse{
		ChangedAt:   setAt,
		CanChangeAt: setAt.Add(s.minAge),
		MustChange:  u.PasswordExpired(now, s.maxAge),
	}
	// A forced change must never be blocked by the minimum age.
	if u.MustChangePassword {
		res.CanChangeAt = setAt
	}
	if s.maxAge > 0 {
		expiresAt := setAt.Add(s.maxAge)
		res.ExpiresAt = &expiresAt
	}
	return res
}

// currentUser loads the signed-in user, writing a 401 response when there is none.
func (s *passwordChangeService) currentUser(w http.ResponseWriter, r *http.Request) (*User, bool) {
	userID, err := authenticate(r, s.jwtSecret)
	if err != nil {
		writeUnauthenticated(w)
		return nil, false
	}
	user, err := s.users.FindByID(r.Context(), userID)
	if errors.Is(err, ErrUserNotFound) {
		writeUnauthenticated(w)
		return nil, false
	}
	if err != nil {
		internalError(w, "load user", err)
		return nil, false
	}
	if user.Status != StatusActive {
		writeUnauthenticated(w)
		return nil, false
	}
	return user, true
}

func (s *passwordChangeService) getStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.status(user, time.Now().UTC()))
}

func (s *passwordChangeService) change(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUser(w, r)
	if !ok {
		return
	}

	var req passwordChangeRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodyLen)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "INVALID_BODY", Message: "Request body must be valid JSON"})
		return
	}

	now := time.Now().UTC()
	status := s.status(user, now)
	if now.Before(status.CanChangeAt) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error":       "PASSWORD_TOO_RECENT",
			"message":     "Your password was changed less than a day ago. You can change it again later.",
			"canChangeAt": status.CanChangeAt,
		})
		return
	}

	fields := map[string]string{}
	if req.CurrentPassword == "" {
		fields["currentPassword"] = "Current password is required"
	}
	if req.NewPassword == "" {
		fields["newPassword"] = "New password is required"
	} else if msg := passwordWeakness(req.NewPassword); msg != "" {
		fields["newPassword"] = msg
	} else if req.NewPassword == req.CurrentPassword {
		fields["newPassword"] = "New password must be different from the current one"
	}
	if req.ConfirmPassword == "" {
		fields["confirmPassword"] = "Please confirm your new password"
	} else if req.NewPassword != req.ConfirmPassword {
		fields["confirmPassword"] = "Passwords do not match"
	}
	if len(fields) > 0 {
		writeJSON(w, http.StatusBadRequest, validationErrorBody{
			Error:   "VALIDATION_ERROR",
			Message: "Some fields are invalid",
			Fields:  fields,
		})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)) != nil {
		writeJSON(w, http.StatusBadRequest, validationErrorBody{
			Error:   "INVALID_CURRENT_PASSWORD",
			Message: "Current password is incorrect",
			Fields:  map[string]string{"currentPassword": "Current password is incorrect"},
		})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		internalError(w, "hash password", err)
		return
	}
	if err := s.users.SetPassword(r.Context(), user.ID, string(passwordHash), now); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			writeUnauthenticated(w)
			return
		}
		internalError(w, "set password", err)
		return
	}

	log.Printf("password changed for user %s", user.ID)
	user.PasswordChangedAt = now
	user.MustChangePassword = false
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Your password has been changed.",
		"status":  s.status(user, now),
	})
}
