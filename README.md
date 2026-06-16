# WAOtomatis Go SDK

Official Go SDK for [WAOtomatis](https://waotomatis.com) — headless WhatsApp
(WhatsApp Business / Cloud API) for developers. Send messages, upload media, and
verify webhooks against the WAOtomatis REST API.

- Zero dependencies — standard library only (`net/http`).
- Idiomatic: an exported `Client`, functional options, `context.Context` on every
  call, and typed errors you can branch on with `errors.As`.
- Mirrors the official TypeScript SDK's surface, error model, and webhook HMAC.

## Install

```sh
go get github.com/waotomatisofficial/waotomatis-go
```

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"os"

	waotomatis "github.com/waotomatisofficial/waotomatis-go"
)

func main() {
	client := waotomatis.New(os.Getenv("WAO_API_KEY"))

	msg, _ := client.Sessions("sess_123").Messages.Send(context.Background(), &waotomatis.Message{
		To:   "628123456789",
		Type: "text",
		Text: "Halo dari WAOtomatis 👋",
	})

	fmt.Println(msg.ID) // msg_abc123
}
```

## Configuration

`New` takes the API key positionally, then functional options:

```go
client := waotomatis.New(
	apiKey,
	waotomatis.WithBaseURL("https://api.waotomatis.com"), // default
	waotomatis.WithTimeout(30*time.Second),               // per-request, default 60s
	waotomatis.WithMaxRetries(2),                         // transient failures, default 2
	waotomatis.WithDefaultHeader("X-Org-Id", "org_123"),  // sent on every request
	waotomatis.WithHTTPClient(&http.Client{ /* proxy, transport */ }),
)
```

Transient failures (HTTP 408/429/5xx and network/timeout errors) are retried
with exponential backoff + jitter, honoring `Retry-After`, but only for
idempotent verbs or requests carrying an idempotency key. Multipart uploads are
never auto-retried.

### Context

Calls do not take a `context.Context` positionally (so the advertised snippet
stays terse). Attach one per call with `waotomatis.WithContext(ctx)` for
cancellation and deadlines; without it, calls use `context.Background()`:

```go
sess.Messages.Send(&waotomatis.Message{To: "62812...", Type: "text", Text: "hi"},
	waotomatis.WithContext(ctx))
```

## Sending messages

```go
sess := client.Sessions("sess_123")

// Text with a link preview.
sess.Messages.Send(&waotomatis.Message{
	To: "628123456789", Type: waotomatis.TypeText,
	Text: "https://waotomatis.com", PreviewURL: true,
})

// Media by uploaded media id.
sess.Messages.Send(&waotomatis.Message{
	To: "628123456789", Type: waotomatis.TypeImage,
	MediaID: "med_123", Caption: "Invoice",
})

// Media by public link.
sess.Messages.Send(&waotomatis.Message{
	To: "628123456789", Type: waotomatis.TypeDocument,
	Link: "https://example.com/invoice.pdf", FileName: "invoice.pdf",
})

// A pinned location.
sess.Messages.Send(&waotomatis.Message{
	To: "628123456789", Type: waotomatis.TypeLocation,
	Location: &waotomatis.LocationInput{
		Latitude: -6.2088, Longitude: 106.8456,
		Name: "Kantor Pusat", Address: "Jl. Sudirman, Jakarta",
	},
})

// A contact card (vCard). Only Name.FormattedName is required.
sess.Messages.Send(&waotomatis.Message{
	To: "628123456789", Type: waotomatis.TypeContacts,
	Contacts: []waotomatis.ContactCard{{
		Name:   waotomatis.ContactName{FormattedName: "Budi Santoso", FirstName: "Budi"},
		Phones: []waotomatis.ContactPhone{{Phone: "+628123456789", Type: "WORK", WaID: "628123456789"}},
		Org:    &waotomatis.ContactOrg{Company: "WAOtomatis"},
	}},
})

// Idempotent send (also accepts a per-call waotomatis.WithIdempotencyKey).
sess.Messages.Send(&waotomatis.Message{
	To: "628123456789", Type: waotomatis.TypeText, Text: "hi",
	IdempotencyKey: "order-42",
})

// Mark an inbound message read by its provider wamid.
sess.Messages.MarkRead("wamid.HBg...")
```

## Sessions

```go
page, _ := client.SessionList().List(&waotomatis.ListParams{Limit: 50})
for _, s := range page.Data {
	fmt.Println(s.ID, s.Status)
}
// page.HasMore + page.Cursor drive the next page.

s, _ := client.SessionList().Get("sess_123")
```

## Media

```go
m := client.Sessions("sess_123").Media

// Upload by URL (server fetches it).
res, _ := m.UploadFromURL("https://example.com/cat.png", "image/png")

// Upload raw bytes.
res, _ = m.Upload(pngBytes, &waotomatis.UploadOptions{
	FileName: "cat.png", MimeType: "image/png",
})

// Upload a file from disk.
res, _ = m.UploadFile("/path/to/cat.png", nil)

fmt.Println(res.MediaID) // pass to Message.MediaID

// Download inbound media bytes.
dl, _ := m.Download("med_inbound_123")
os.WriteFile("cat.png", dl.Data, 0o644)
```

## Webhooks

The server signs every delivery with `X-Wao-Signature: sha256=<hex>`, an
HMAC-SHA256 over the **exact raw request body**. Verify it before trusting the
payload — never re-marshal a parsed body first.

```go
func handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	sig := r.Header.Get(waotomatis.WebhookSignatureHeader) // "X-Wao-Signature"

	event, err := waotomatis.ConstructEvent(body, sig, webhookSecret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	switch event.Event {
	case waotomatis.EventMessageReceived:
		var d waotomatis.MessageReceivedData
		_ = json.Unmarshal(event.Data, &d)
		fmt.Println("from", d.From, "text", d.Text)
	case waotomatis.EventMessageUpdated:
		// ...
	case waotomatis.EventSessionStatus:
		// ...
	}
	w.WriteHeader(http.StatusOK)
}
```

If you only need the boolean check, use `waotomatis.VerifyWebhook(body, sig, secret)`.

Register a webhook (the signing `Secret` is returned **once**, here):

```go
wh, _ := client.Sessions("sess_123").Webhooks.Create(&waotomatis.CreateWebhookInput{
	URL:    "https://yourapp.com/wao/webhook",
	Events: []waotomatis.EventType{waotomatis.EventMessageReceived},
})
secret := wh.Secret // store it
```

## Errors

Every failure is a typed error wrapping a `*waotomatis.Error`
(`{ Code, Message, RequestID, Status }`). Branch with `errors.As`, or use the
`Code` / `Status` / `RequestID` helpers.

```go
_, err := client.Sessions("sess_123").Messages.Send(msg)
if err != nil {
	var rl *waotomatis.RateLimitError
	if errors.As(err, &rl) {
		time.Sleep(time.Duration(rl.RetryAfter) * time.Second)
	}
	if waotomatis.Code(err) == waotomatis.CodeSessionDisconnected {
		// reconnect the session...
	}
	log.Printf("waotomatis error: code=%s status=%d request_id=%s",
		waotomatis.Code(err), waotomatis.Status(err), waotomatis.RequestID(err))
}
```

Typed errors: `AuthenticationError` (401), `PermissionError` (403),
`NotFoundError` (404), `ValidationError` (409/422), `RateLimitError` (429),
`APIError` (5xx), `TimeoutError`, and `ConnectionError`.

## License

[MIT](./LICENSE)
