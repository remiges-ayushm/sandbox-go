package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/beckn/sandbox-go/internal/logging"
	"github.com/gin-gonic/gin"
)

const helmetCSP = "default-src 'self';base-uri 'self';font-src 'self' https: data:;" +
	"form-action 'self';frame-ancestors 'self';img-src 'self' data:;object-src 'none';" +
	"script-src 'self';script-src-attr 'none';style-src 'self' https: 'unsafe-inline';" +
	"upgrade-insecure-requests"

// corsMiddleware mirrors the npm `cors()` package's default configuration (no options passed).
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,HEAD,PUT,PATCH,POST,DELETE")
		if reqHeaders := c.Request.Header.Get("Access-Control-Request-Headers"); reqHeaders != "" {
			c.Header("Access-Control-Allow-Headers", reqHeaders)
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// helmetMiddleware replicates the default header set emitted by the npm `helmet()` v8 package.
func helmetMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-DNS-Prefetch-Control", "off")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Strict-Transport-Security", "max-age=15552000; includeSubDomains")
		h.Set("X-Download-Options", "noopen")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Permitted-Cross-Domain-Policies", "none")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-XSS-Protection", "0")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		h.Set("Origin-Agent-Cluster", "?1")
		h.Set("Content-Security-Policy", helmetCSP)
		c.Next()
	}
}

// bodyLimitMiddleware mirrors express.json({ limit: "5mb" }).
func bodyLimitMiddleware(limitBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limitBytes)
		c.Next()
	}
}

// recoveryMiddleware mirrors the Express global error fallback in src/app.ts, and logs
// the recovered panic + stack trace before responding.
func recoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logging.FromContext(c).Error("panic recovered",
			"error", fmt.Sprintf("%v", recovered),
			"stack", string(debug.Stack()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	})
}

// requestIDMiddleware reuses an inbound X-Request-Id header if present, otherwise
// generates one, and stashes a request-scoped logger on the gin context so every log
// line for this request can be correlated together.
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(logging.RequestIDHeader)
		if id == "" {
			id = logging.NewRequestID()
		}
		logger := slog.Default().With("request_id", id)
		logging.SetRequestLogger(c, id, logger)
		c.Header(logging.RequestIDHeader, id)
		c.Next()
	}
}

// bodyLogWriter wraps gin's ResponseWriter to capture the response body for logging
// without affecting what's actually written to the client.
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// loggableBody renders a captured body for structured logging: valid JSON is embedded
// as raw JSON (readable in log output), anything else falls back to a plain string.
func loggableBody(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	if json.Valid(b) {
		return json.RawMessage(b)
	}
	return string(b)
}

// loggingMiddleware logs every inbound request and its outgoing response, including
// full bodies, method/path, status code, and latency, correlated by request_id.
func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		logger := logging.FromContext(c)

		var reqBody []byte
		if c.Request.Body != nil {
			reqBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewReader(reqBody))
		}

		logger.Info("incoming request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"query", c.Request.URL.RawQuery,
			"client_ip", c.ClientIP(),
			"body", loggableBody(reqBody),
		)

		blw := &bodyLogWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = blw

		c.Next()

		fields := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"body", loggableBody(blw.body.Bytes()),
		}

		switch {
		case c.Writer.Status() >= http.StatusInternalServerError:
			logger.Error("outgoing response", fields...)
		case c.Writer.Status() >= http.StatusBadRequest:
			logger.Warn("outgoing response", fields...)
		default:
			logger.Info("outgoing response", fields...)
		}
	}
}
