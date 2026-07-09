package main

import (
	"log/slog"
	"os"

	"github.com/beckn/sandbox-go/internal/app"
	"github.com/beckn/sandbox-go/internal/logging"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	logging.Init()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3002"
	}

	engine := app.New()
	slog.Info("server starting", "port", port)
	if err := engine.Run(":" + port); err != nil {
		slog.Error("server exited", "error", err)
		os.Exit(1)
	}
}
