package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gocql/gocql"
)

var schema string

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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

func main() {
	port := getenv("PORT", "8083")

	session := connect()
	defer session.Close()
	applySchema(session)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "notifications"})
	})

	log.Printf("notifications service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
