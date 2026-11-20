package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/shravastee-thakur/Go-Auth/internal/config"
	"github.com/shravastee-thakur/Go-Auth/internal/db"
)

func main() {
	// 1. Load config
	cfg := config.Load()

	// 2. Connect to MongoDB Atlas
	client := db.ConnectDB(cfg.MongoURL)
	defer client.Disconnect(context.Background())

	// 3. Setup Chi router (like const app = express())
	r := chi.NewRouter()

	// 4. Attach global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// 5. Define a test route
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status}":"ok"`))
	})


	// 6. Start the HTTP server
	log.Printf("Server running on port %s\n", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}