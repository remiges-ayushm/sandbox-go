package bapwebhook

import (
	"encoding/json"
	"log"
	"net/http"

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

func handle(c *gin.Context) {
	var b body
	_ = c.ShouldBindJSON(&b)

	logged, _ := json.MarshalIndent(gin.H{"message": b.Message, "context": b.Context}, "", "  ")
	log.Println(string(logged))

	c.JSON(http.StatusOK, buildAck(b.Context))
}

func OnDiscover(c *gin.Context) { handle(c) }
func OnSelect(c *gin.Context)   { handle(c) }
func OnInit(c *gin.Context)     { handle(c) }
func OnConfirm(c *gin.Context)  { handle(c) }
func OnStatus(c *gin.Context)   { handle(c) }
func OnCancel(c *gin.Context)   { handle(c) }
func OnUpdate(c *gin.Context)   { handle(c) }
func OnRating(c *gin.Context)   { handle(c) }
func OnRate(c *gin.Context)     { handle(c) }
func OnSupport(c *gin.Context)  { handle(c) }
func OnTrack(c *gin.Context)    { handle(c) }
