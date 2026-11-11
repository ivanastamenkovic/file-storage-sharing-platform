package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func proxyTo(target, prefix string) http.Handler {
	u, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid upstream url %q: %v", target, err)
	}
	p := httputil.NewSingleHostReverseProxy(u)
	p.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("upstream error for %s: %v", prefix, err)
		writeJSON(w, http.StatusBadGateway, errorBody{
			Error:   "SERVICE_UNAVAILABLE",
			Message: "The requested service is currently unavailable",
		})
	}
	return http.StripPrefix(prefix, p)
}

func main() {
	port := getenv("PORT", "8080")
	usersURL := getenv("USERS_SERVICE_URL", "http://localhost:8081")
	filesURL := getenv("FILES_SERVICE_URL", "http://localhost:8082")
	notificationsURL := getenv("NOTIFICATIONS_SERVICE_URL", "http://localhost:8083")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "gateway"})
	})
	mux.Handle("/api/users/", proxyTo(usersURL, "/api/users"))
	mux.Handle("/api/files/", proxyTo(filesURL, "/api/files"))
	mux.Handle("/api/notifications/", proxyTo(notificationsURL, "/api/notifications"))

	log.Printf("gateway listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
