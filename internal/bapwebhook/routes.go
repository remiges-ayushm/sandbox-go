package bapwebhook

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the BAP-side webhook routes, equivalent to bapWebhookRoutes() in
// src/bap-webhook/routes.ts.
func RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/on_discover", OnDiscover)
	rg.POST("/on_select", OnSelect)
	rg.POST("/on_init", OnInit)
	rg.POST("/on_confirm", OnConfirm)
	rg.POST("/on_status", OnStatus)
	rg.POST("/on_cancel", OnCancel)
	rg.POST("/on_update", OnUpdate)
	rg.POST("/on_rating", OnRating)
	rg.POST("/on_rate", OnRate)
	rg.POST("/on_support", OnSupport)
	rg.POST("/on_track", OnTrack)
}
