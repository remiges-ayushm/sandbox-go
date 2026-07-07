package app

import (
	"net/http"

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

// recoveryMiddleware mirrors the Express global error fallback in src/app.ts.
func recoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	})
}
