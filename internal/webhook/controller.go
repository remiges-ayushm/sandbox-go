package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/beckn/sandbox-go/internal/fixtures"
	"github.com/beckn/sandbox-go/internal/logging"
	"github.com/gin-gonic/gin"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

func getPersona() string {
	return os.Getenv("PERSONA")
}

func firstNonEmptyString(values ...interface{}) string {
	for _, v := range values {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func getCallbackURL(context map[string]interface{}, action string) (string, error) {
	if base := os.Getenv("BPP_CALLBACK_ENDPOINT"); base != "" {
		return fmt.Sprintf("%s/on_%s", strings.TrimRight(base, "/"), action), nil
	}

	bppURL := firstNonEmptyString(context["bpp_uri"], context["bpp_url"], context["bppUri"], context["bppUrl"])
	parsed, err := url.Parse(bppURL)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s://%s/bpp/caller/on_%s", parsed.Scheme, parsed.Host, action), nil
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

func buildResponseContext(context map[string]interface{}, action string) map[string]interface{} {
	result := make(map[string]interface{}, len(context)+1)
	for k, v := range context {
		result[k] = v
	}
	result["action"] = "on_" + action

	now := time.Now().UTC().Format(time.RFC3339)
	if _, ok := context["timestamp"]; ok {
		result["timestamp"] = now
	} else if _, ok := context["time_stamp"]; ok {
		result["time_stamp"] = now
	}

	return result
}

func mergeContext(template interface{}, context map[string]interface{}, action string) map[string]interface{} {
	result := map[string]interface{}{}
	if m, ok := template.(map[string]interface{}); ok {
		for k, v := range m {
			result[k] = v
		}
	}
	result["context"] = buildResponseContext(context, action)
	return result
}

func postJSON(logger *slog.Logger, requestID string, url string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		logger.Error("failed to marshal outbound callback payload", "url", url, "error", err)
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		logger.Error("failed to build outbound callback request", "url", url, "error", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if requestID != "" {
		req.Header.Set(logging.RequestIDHeader, requestID)
	}

	logger.Info("outbound callback request", "method", http.MethodPost, "url", url, "body", json.RawMessage(body))

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Error("outbound callback request failed", "url", url, "error", err)
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	logFields := []any{"url", url, "status", resp.StatusCode, "body", loggableCallbackBody(respBody)}
	if resp.StatusCode >= http.StatusBadRequest {
		logger.Error("outbound callback response", logFields...)
	} else {
		logger.Info("outbound callback response", logFields...)
	}
	return nil
}

func loggableCallbackBody(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	if json.Valid(b) {
		return json.RawMessage(b)
	}
	return string(b)
}

// jsonataActions mirrors the TS controller: only these actions run their fixture template
// through applyJsonata before merging in the response context.
var jsonataActions = map[string]bool{
	"select":  true,
	"init":    true,
	"confirm": true,
	"status":  true,
	"update":  true,
	"cancel":  true,
}

// respond builds the fire-and-forget on_<action> handler shared by all 11 webhook actions:
// it reads/transforms the fixture template in a goroutine and POSTs it to the BPP callback,
// while immediately returning a synchronous ACK, matching the un-awaited IIFE pattern in
// src/webhook/controller.ts.
func respond(action string) gin.HandlerFunc {
	label := strings.ToUpper(action[:1]) + action[1:]
	useJsonata := jsonataActions[action]

	return func(c *gin.Context) {
		logger := logging.FromContext(c)
		requestID := logging.RequestID(c)

		var body map[string]interface{}
		if err := c.ShouldBindJSON(&body); err != nil {
			logger.Warn("failed to bind request body as JSON", "action", action, "error", err)
		}
		headers := c.Request.Header.Clone()
		context, _ := body["context"].(map[string]interface{})

		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("on_"+action+" callback panic", "action", action, "panic", fmt.Sprintf("%v", r))
				}
			}()

			template := fixtures.ReadRequestResponse(body, "on_"+action, getPersona(), headers)
			if useJsonata {
				template = fixtures.ApplyJsonata(template, body)
			}
			responsePayload := mergeContext(template, context, action)

			callbackURL, err := getCallbackURL(context, action)
			if err != nil {
				logger.Error("failed to resolve callback URL", "action", label, "error", err)
				return
			}
			logger.Info("triggering callback", "action", label, "callback_url", callbackURL)

			if err := postJSON(logger, requestID, callbackURL, responsePayload); err != nil {
				logger.Error("callback delivery failed", "action", label, "callback_url", callbackURL, "error", err)
			}
		}()

		c.JSON(http.StatusOK, buildAck(context))
	}
}

// triggerRespond builds the synchronous /trigger/on_<action> handler: it awaits the callback
// POST of the raw {context, message} before responding, matching the async/await TS handlers.
func triggerRespond(action string) gin.HandlerFunc {
	label := strings.ToUpper(action[:1]) + action[1:]

	return func(c *gin.Context) {
		logger := logging.FromContext(c)
		requestID := logging.RequestID(c)

		var body map[string]interface{}
		if err := c.ShouldBindJSON(&body); err != nil {
			logger.Warn("failed to bind request body as JSON", "action", action, "error", err)
		}
		context, _ := body["context"].(map[string]interface{})
		message := body["message"]

		callbackURL, err := getCallbackURL(context, action)
		if err != nil {
			logger.Error("failed to resolve callback URL", "action", label, "error", err)
		} else {
			logger.Info("triggering callback", "action", label, "callback_url", callbackURL)
			if err := postJSON(logger, requestID, callbackURL, gin.H{"context": context, "message": message}); err != nil {
				logger.Error("callback delivery failed", "action", label, "callback_url", callbackURL, "error", err)
			}
		}

		c.JSON(http.StatusOK, buildAck(context))
	}
}

var (
	OnDiscover = respond("discover")
	OnSelect   = respond("select")
	OnInit     = respond("init")
	OnConfirm  = respond("confirm")
	OnStatus   = respond("status")
	OnUpdate   = respond("update")
	OnRating   = respond("rating")
	OnRate     = respond("rate")
	OnSupport  = respond("support")
	OnTrack    = respond("track")
	OnCancel   = respond("cancel")

	TriggerOnStatus = triggerRespond("status")
	TriggerOnCancel = triggerRespond("cancel")
	TriggerOnUpdate = triggerRespond("update")
)
