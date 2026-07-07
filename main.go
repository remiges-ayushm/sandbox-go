package main

import (
	"log"
	"os"

	"github.com/beckn/sandbox-go/internal/app"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3002"
	}

	engine := app.New()
	log.Printf("Server is running on port %s", port)
	if err := engine.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
