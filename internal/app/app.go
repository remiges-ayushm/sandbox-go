package app

import (
	"net/http"

	"github.com/beckn/sandbox-go/internal/bapwebhook"
	"github.com/beckn/sandbox-go/internal/webhook"
	"github.com/gin-gonic/gin"
)

const jsonBodyLimitBytes = 5 << 20 // 5mb

// New builds the Gin engine, mirroring createApp() in src/app.ts.
func New() *gin.Engine {
	engine := gin.New()

	engine.Use(requestIDMiddleware())
	engine.Use(loggingMiddleware())
	engine.Use(recoveryMiddleware())
	engine.Use(corsMiddleware())
	engine.Use(helmetMiddleware())
	engine.Use(bodyLimitMiddleware(jsonBodyLimitBytes))

	api := engine.Group("/api")
	webhook.RegisterRoutes(api.Group("/webhook"))
	bapwebhook.RegisterRoutes(api.Group("/bap-webhook"))
	api.Any("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "OK!"})
	})

	return engine
}
