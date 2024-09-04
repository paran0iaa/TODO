package services

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func GetEnv(envVar string) string {
	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file")
	}

	if val, ok := os.LookupEnv(envVar); ok {
		return val
	}
	log.Println("Error getting env var")

	return envVar
}
