package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/ivanastamenkovic/file-storage-sharing-platform/services/files/internal/handler"
	"github.com/ivanastamenkovic/file-storage-sharing-platform/services/files/internal/repository"
	"github.com/ivanastamenkovic/file-storage-sharing-platform/services/files/internal/service"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	port := getenv("PORT", "8082")
	mongoURI := getenv("MONGO_URI", "mongodb://localhost:27017")
	mongoDB := getenv("MONGO_DB", "files")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("mongo connect failed: %v", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalf("mongo ping failed: %v", err)
	}

	folderRepo, err := repository.NewFolderRepository(ctx, client.Database(mongoDB))
	if err != nil {
		log.Fatalf("folder repository init failed: %v", err)
	}
	folderService := service.NewFolderService(folderRepo)
	folderHandler := handler.NewFolderHandler(folderService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "files"})
	})
	folderHandler.Register(mux)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("files service listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}
