package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocql/gocql"
)

var schema string

var userIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

type notification struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	IsRead    bool      `json:"isRead"`
	CreatedAt time.Time `json:"createdAt"`
}

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

func connect() *gocql.Session {
	hosts := strings.Split(getenv("CASSANDRA_HOSTS", "localhost"), ",")
	port, err := strconv.Atoi(getenv("CASSANDRA_PORT", "9042"))
	if err != nil {
		log.Fatalf("invalid CASSANDRA_PORT: %v", err)
	}

	const maxAttempts = 30
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		cluster := gocql.NewCluster(hosts...)
		cluster.Port = port
		cluster.Timeout = 10 * time.Second
		cluster.ConnectTimeout = 10 * time.Second

		session, err := cluster.CreateSession()
		if err == nil {
			log.Printf("connected to cassandra (attempt %d)", attempt)
			return session
		}
		log.Printf("cassandra not ready (attempt %d/%d): %v", attempt, maxAttempts, err)
		time.Sleep(5 * time.Second)
	}
	log.Fatal("could not connect to cassandra")
	return nil
}

func applySchema(session *gocql.Session) {
	for _, stmt := range strings.Split(schema, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if err := session.Query(stmt).Exec(); err != nil {
			log.Fatalf("schema error: %v", err)
		}
	}
	log.Print("cassandra schema is ready")
}

func listNotifications(session *gocql.Session) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-Id")
		if !userIDPattern.MatchString(userID) {
			writeJSON(w, http.StatusUnauthorized, errorBody{
				Error:   "UNAUTHORIZED",
				Message: "Missing or invalid user identity",
			})
			return
		}

		iter := session.Query(
			`SELECT id, type, message, is_read, created_at
			 FROM notifications.notifications WHERE user_id = ?`,
			userID,
		).WithContext(r.Context()).Iter()

		result := []notification{}
		var (
			id gocql.UUID
			n  notification
		)
		for iter.Scan(&id, &n.Type, &n.Message, &n.IsRead, &n.CreatedAt) {
			n.ID = id.String()
			result = append(result, n)
		}
		if err := iter.Close(); err != nil {
			log.Printf("failed to load notifications: %v", err)
			writeJSON(w, http.StatusInternalServerError, errorBody{
				Error:   "INTERNAL_ERROR",
				Message: "Could not load notifications",
			})
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

func main() {
	port := getenv("PORT", "8083")

	session := connect()
	defer session.Close()
	applySchema(session)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "notifications"})
	})
	mux.HandleFunc("GET /{$}", listNotifications(session))

	log.Printf("notifications service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
