package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	MongoURL      string
	AccessSecret  string
	RefreshSecret string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from system environment")
	}


	cfg := &Config{
		Port: getEnv("PORT", "8000"),
		MongoURL: os.Getenv("MONGO_URL"),
		AccessSecret: os.Getenv("ACCESS_SECRET"),
		RefreshSecret: os.Getenv("REFRESH_SECRET"),
	}

	if cfg.MongoURL == "" || cfg.AccessSecret == "" || cfg.RefreshSecret == ""{
		log.Fatal("Missing required environment variables (MONGO_URI, ACCESS_SECRET, REFRESH_SECRET)")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}