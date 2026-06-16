package waotomatis

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// WebhookSignatureHeader is the HTTP header the server signs deliveries with.
// Its value is "sha256=<hex>".
const WebhookSignatureHeader = "X-Wao-Signature"

// VerifyWebhook reports whether signature matches an HMAC-SHA256 of rawBody
// using secret. It mirrors the server's scheme (and packages/sdk/src/webhook.ts):
// the signature is computed over the EXACT raw request body — never re-marshal a
// parsed object first. The "sha256=" prefix on signature is optional.
//
//	body, _ := io.ReadAll(r.Body)
//	if !waotomatis.VerifyWebhook(body, r.Header.Get(waotomatis.WebhookSignatureHeader), secret) {
//	    w.WriteHeader(http.StatusUnauthorized)
//	    return
//	}
func VerifyWebhook(rawBody []byte, signature, secret string) bool {
	if signature == "" {
		return false
	}
	provided := strings.TrimPrefix(signature, "sha256=")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	// hmac.Equal is constant-time; compare the lowercase hex strings as bytes.
	return hmac.Equal([]byte(expected), []byte(strings.ToLower(provided)))
}

// ConstructEvent verifies the signature AND parses rawBody into a WebhookEvent.
// It returns an *Error with code "unauthorized" on a bad signature, or
// "validation_failed" on an unparseable / event-less body. Decode event.Data
// into the matching *Data struct based on event.Event.
//
//	event, err := waotomatis.ConstructEvent(body, sig, secret)
//	if err != nil { http.Error(w, "bad signature", 401); return }
//	if event.Event == waotomatis.EventMessageReceived {
//	    var d waotomatis.MessageReceivedData
//	    _ = json.Unmarshal(event.Data, &d)
//	}
func ConstructEvent(rawBody []byte, signature, secret string) (*WebhookEvent, error) {
	if !VerifyWebhook(rawBody, signature, secret) {
		return nil, &Error{Code: CodeUnauthorized, Message: "Invalid webhook signature.", Status: 401}
	}
	var event WebhookEvent
	if err := json.Unmarshal(rawBody, &event); err != nil {
		return nil, &Error{Code: CodeValidationFailed, Message: "Webhook body is not valid JSON.", Status: 422}
	}
	if event.Event == "" {
		return nil, &Error{Code: CodeValidationFailed, Message: "Webhook body is missing an `event`.", Status: 422}
	}
	return &event, nil
}
