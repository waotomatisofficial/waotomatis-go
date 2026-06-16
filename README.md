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
	"fmt"
	"os"

	waotomatis "github.com/waotomatisofficial/waotomatis-go"
)

func main() {
	client := waotomatis.New(os.Getenv("WAO_API_KEY"))

	msg, _ := client.Sessions("sess_123").Messages.SendText(&waotomatis.TextMessage{
		To:   "628123456789",
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
sess.Messages.SendText(&waotomatis.TextMessage{To: "62812...", Text: "hi"},
	waotomatis.WithContext(ctx))
```

## Sending messages

There is one method per message type, each taking a focused, typed input and
posting to its own endpoint. `Media` takes a `Type` (image/video/audio/document/
sticker) and `Interactive` takes a `Type` (button/list/cta_url/flow/product/
product_list) — these are fields, not separate methods.

```go
sess := client.Sessions("sess_123")

// 1. Text (optionally with a link preview).
sess.Messages.SendText(&waotomatis.TextMessage{
	To: "628123456789", Text: "https://waotomatis.com", PreviewURL: true,
})

// 2. Media — by uploaded media id or public link. Type picks the kind.
sess.Messages.SendMedia(&waotomatis.MediaMessage{
	To: "628123456789", Type: waotomatis.MediaImage,
	MediaID: "med_123", Caption: "Invoice",
})
sess.Messages.SendMedia(&waotomatis.MediaMessage{
	To: "628123456789", Type: waotomatis.MediaDocument,
	Link: "https://example.com/invoice.pdf", FileName: "invoice.pdf",
})

// 3. Template — a pre-approved template; Components passes through verbatim.
sess.Messages.SendTemplate(&waotomatis.TemplateMessage{
	To: "628123456789", Name: "order_update", LanguageCode: "en_US",
	Components: []any{
		map[string]any{"type": "body", "parameters": []any{
			map[string]any{"type": "text", "text": "Budi"},
		}},
	},
})

// 4. Interactive — Type selects the subtype; provide the matching fields.
sess.Messages.SendInteractive(&waotomatis.InteractiveMessage{
	To: "628123456789", Type: waotomatis.InteractiveTypeButton,
	BodyText: "Confirm your order?",
	Buttons: []waotomatis.InteractiveButton{
		{ID: "yes", Title: "Yes"}, {ID: "no", Title: "No"},
	},
})

// 5. Reaction — react by wamid; an empty Emoji clears the reaction.
sess.Messages.SendReaction(&waotomatis.ReactionMessage{
	To: "628123456789", MessageID: "wamid.HBg...", Emoji: "👍",
})

// 6. Location — a pinned location.
sess.Messages.SendLocation(&waotomatis.LocationMessage{
	To: "628123456789", Latitude: -6.2088, Longitude: 106.8456,
	Name: "Kantor Pusat", Address: "Jl. Sudirman, Jakarta",
})

// 7. Contacts — one or more contact cards (Name.FormattedName is required).
sess.Messages.SendContacts(&waotomatis.ContactsMessage{
	To: "628123456789",
	Contacts: []waotomatis.ContactCard{{
		Name:   waotomatis.ContactName{FormattedName: "Budi Santoso", FirstName: "Budi"},
		Phones: []waotomatis.ContactPhone{{Phone: "+628123456789", Type: "WORK", WaID: "628123456789"}},
		Org:    &waotomatis.ContactOrg{Company: "WAOtomatis"},
	}},
})

// 8. Carousel — a carousel template with cards.
sess.Messages.SendCarousel(&waotomatis.CarouselMessage{
	To: "628123456789", Name: "promo", LanguageCode: "id",
	BodyParams: []string{"Budi"},
	Cards: []waotomatis.CarouselCard{{
		HeaderImageLink: "https://example.com/card.png",
		BodyParams:      []string{"Diskon 20%"},
		Buttons:         []waotomatis.CarouselButton{{SubType: "quick_reply", Index: 0, Payload: "buy"}},
	}},
})

// Idempotent send: set IdempotencyKey on any input (or pass a per-call
// waotomatis.WithIdempotencyKey, which wins).
sess.Messages.SendText(&waotomatis.TextMessage{
	To: "628123456789", Text: "hi", IdempotencyKey: "order-42",
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

## Templates

Manage the WhatsApp message templates on a session's WABA. `Components` is Meta's
template-component array (`HEADER` / `BODY` / `FOOTER` / `BUTTONS` objects), left
loosely typed (`[]map[string]any`) so the full provider shape passes through.

```go
tpls := client.Sessions("sess_123").Templates

// List (filter + paginate). Pass nil for the first page unfiltered.
list, _ := tpls.List(&waotomatis.TemplateListParams{
	Limit: 50, Status: "APPROVED", Category: "MARKETING",
})
for _, t := range list.Data {
	fmt.Println(t.Name, t.Language, t.Status)
}
// list.Paging carries Meta's cursors (paging.cursors.after) for the next page.

// Get every language version of a template by exact name.
got, _ := tpls.Get("promo_diskon")

// Create (submitted to Meta for approval; status starts PENDING).
res, _ := tpls.Create(&waotomatis.CreateTemplateInput{
	Name:     "promo_diskon",
	Language: "id",
	Category: waotomatis.TemplateMarketing,
	Components: []map[string]any{
		{
			"type": "BODY",
			"text": "Halo {{1}}, ada diskon {{2}}% spesial untukmu!",
			"example": map[string]any{
				"body_text": [][]string{{"Budi", "20"}},
			},
		},
		{
			"type": "BUTTONS",
			"buttons": []map[string]any{
				{"type": "QUICK_REPLY", "text": "Lihat promo"},
			},
		},
	},
})
fmt.Println(res.ID, res.Status) // poll List/Get until status is APPROVED

// Delete by name (removes all language versions).
del, _ := tpls.Delete("promo_diskon")
fmt.Println(del.Success)
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
_, err := client.Sessions("sess_123").Messages.SendText(msg)
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
