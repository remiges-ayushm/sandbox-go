package webhook

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the BPP-side webhook routes, equivalent to webhookRoutes() in
// src/webhook/routes.ts.
func RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/discover", OnDiscover)
	rg.POST("/select", OnSelect)
	rg.POST("/init", OnInit)
	rg.POST("/confirm", OnConfirm)
	rg.POST("/status", OnStatus)
	rg.POST("/cancel", OnCancel)
	rg.POST("/update", OnUpdate)
	rg.POST("/rating", OnRating)
	rg.POST("/rate", OnRate)
	rg.POST("/support", OnSupport)
	rg.POST("/track", OnTrack)

	// Unsolicited triggering routes
	rg.POST("/trigger/on_status", TriggerOnStatus)
	rg.POST("/trigger/on_cancel", TriggerOnCancel)
	rg.POST("/trigger/on_update", TriggerOnUpdate)
}
