package bapwebhook

import (
	"net/http"

	"github.com/beckn/sandbox-go/internal/logging"
	"github.com/gin-gonic/gin"
)

type body struct {
	Context map[string]interface{} `json:"context"`
	Message map[string]interface{} `json:"message"`
}

func messageID(context map[string]interface{}) string {
	if v, ok := context["messageId"].(string); ok && v != "" {
		return v
	}
	if v, ok := context["message_id"].(string); ok && v != "" {
		return v
	}
	return ""
}

func buildAck(context map[string]interface{}) gin.H {
	return gin.H{"message": gin.H{"status": "ACK", "messageId": messageID(context)}}
}

func handle(action string, c *gin.Context) {
	logger := logging.FromContext(c)

	var b body
	if err := c.ShouldBindJSON(&b); err != nil {
		logger.Warn("failed to bind request body as JSON", "action", action, "error", err)
	}

	logger.Info("received callback", "action", action, "message", b.Message, "context", b.Context)

	c.JSON(http.StatusOK, buildAck(b.Context))
}

func OnDiscover(c *gin.Context) { handle("on_discover", c) }
func OnSelect(c *gin.Context)   { handle("on_select", c) }
func OnInit(c *gin.Context)     { handle("on_init", c) }
func OnConfirm(c *gin.Context)  { handle("on_confirm", c) }
func OnStatus(c *gin.Context)   { handle("on_status", c) }
func OnCancel(c *gin.Context)   { handle("on_cancel", c) }
func OnUpdate(c *gin.Context)   { handle("on_update", c) }
func OnRating(c *gin.Context)   { handle("on_rating", c) }
func OnRate(c *gin.Context)     { handle("on_rate", c) }
func OnSupport(c *gin.Context)  { handle("on_support", c) }
func OnTrack(c *gin.Context)    { handle("on_track", c) }
