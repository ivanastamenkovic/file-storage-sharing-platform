package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	port := getenv("PORT", "8081")

	quota, err := strconv.ParseInt(getenv("DEFAULT_STORAGE_QUOTA_BYTES", "5368709120"), 10, 64)
	if err != nil || quota <= 0 {
		log.Fatalf("invalid DEFAULT_STORAGE_QUOTA_BYTES: %v", err)
	}
	tokenTTL, err := time.ParseDuration(getenv("CONFIRMATION_TOKEN_TTL", "24h"))
	if err != nil || tokenTTL <= 0 {
		log.Fatalf("invalid CONFIRMATION_TOKEN_TTL: %v", err)
	}

	otpTTL, err := time.ParseDuration(getenv("OTP_TTL", "5m"))
	if err != nil || otpTTL <= 0 {
		log.Fatalf("invalid OTP_TTL: %v", err)
	}
	otpMaxAttempts, err := strconv.Atoi(getenv("OTP_MAX_ATTEMPTS", "5"))
	if err != nil || otpMaxAttempts <= 0 {
		log.Fatalf("invalid OTP_MAX_ATTEMPTS: %v", err)
	}
	sessionTTL, err := time.ParseDuration(getenv("SESSION_TTL", "24h"))
	if err != nil || sessionTTL <= 0 {
		log.Fatalf("invalid SESSION_TTL: %v", err)
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be set and at least 32 characters long")
	}
	dummyHash, err := bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("could not prepare password hashing: %v", err)
	}

	client := connectMongo(getenv("MONGO_URI", "mongodb://localhost:27017"))
	defer func() { _ = client.Disconnect(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	db := client.Database(getenv("MONGO_DB", "users"))
	store, err := newUserStore(ctx, db)
	if err != nil {
		log.Fatalf("could not prepare users collection: %v", err)
	}
	challenges, err := newChallengeStore(ctx, db)
	cancel()
	if err != nil {
		log.Fatalf("could not prepare login challenges collection: %v", err)
	}

	mailer := newSMTPMailer(
		getenv("SMTP_HOST", "localhost"),
		getenv("SMTP_PORT", "1025"),
		getenv("SMTP_FROM", "no-reply@filestorage.local"),
		os.Getenv("SMTP_USERNAME"),
		os.Getenv("SMTP_PASSWORD"),
	)

	svc := &registrationService{
		store:        store,
		mailer:       mailer,
		appBaseURL:   strings.TrimRight(getenv("APP_BASE_URL", "http://localhost:5173"), "/"),
		defaultQuota: quota,
		tokenTTL:     tokenTTL,
	}
	auth := &loginService{
		users:          store,
		challenges:     challenges,
		mailer:         mailer,
		otpTTL:         otpTTL,
		otpMaxAttempts: otpMaxAttempts,
		jwtSecret:      []byte(jwtSecret),
		sessionTTL:     sessionTTL,
		dummyHash:      dummyHash,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "users"})
	})
	mux.HandleFunc("POST /register", svc.register)
	mux.HandleFunc("POST /confirm", svc.confirm)
	mux.HandleFunc("POST /login", auth.login)
	mux.HandleFunc("POST /login/verify", auth.verify)

	log.Printf("users service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
