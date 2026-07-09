package logging

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"

	"github.com/gin-gonic/gin"
)

// RequestIDHeader is the header used to read/propagate a request's correlation ID
// across inbound requests and outbound callback calls.
const RequestIDHeader = "X-Request-Id"

const (
	requestIDContextKey = "request_id"
	loggerContextKey    = "request_logger"
)

// NewRequestID generates a random hex-encoded request identifier.
func NewRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf)
}

// SetRequestLogger stashes the request ID and its scoped logger on the gin context.
func SetRequestLogger(c *gin.Context, id string, logger *slog.Logger) {
	c.Set(requestIDContextKey, id)
	c.Set(loggerContextKey, logger)
}

// RequestID returns the request ID stored on the gin context, if any.
func RequestID(c *gin.Context) string {
	if v, ok := c.Get(requestIDContextKey); ok {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// FromContext returns the request-scoped logger stashed on the gin context, falling
// back to the process default logger when absent.
func FromContext(c *gin.Context) *slog.Logger {
	if v, ok := c.Get(loggerContextKey); ok {
		if logger, ok := v.(*slog.Logger); ok {
			return logger
		}
	}
	if id := RequestID(c); id != "" {
		return slog.Default().With("request_id", id)
	}
	return slog.Default()
}
