package waotomatis_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	waotomatis "github.com/waotomatisofficial/waotomatis-go"
)

// newTestClient spins up an httptest server and a client pointed at it.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*waotomatis.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := waotomatis.New("test_key", waotomatis.WithBaseURL(srv.URL), waotomatis.WithMaxRetries(0))
	return c, srv
}

func TestSendText(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test_key" {
			t.Errorf("auth header = %q", got)
		}
		if r.URL.Path != "/v1/sessions/sess_123/messages" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in map[string]any
		_ = json.Unmarshal(body, &in)
		if in["to"] != "628123456789" || in["type"] != "text" || in["text"] != "Halo" {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"msg_abc123","eventId":"evt_1","status":"sent"}`))
	})

	// This is the exact shape advertised in Hero.astro for Go.
	msg, err := c.Sessions("sess_123").Messages.Send(&waotomatis.Message{
		To:   "628123456789",
		Type: "text",
		Text: "Halo",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if msg.ID != "msg_abc123" {
		t.Errorf("id = %q", msg.ID)
	}
}

func TestIdempotencyKeyHeader(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Idempotency-Key"); got != "key-1" {
			t.Errorf("idempotency header = %q", got)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"m","eventId":"e","status":"sent"}`))
	})
	_, err := c.Sessions("s").Messages.Send(&waotomatis.Message{
		To: "1", Type: "text", Text: "hi", IdempotencyKey: "key-1",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestErrorMapping(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"error":{"code":"session_not_found","message":"nope","requestId":"req_9"}}`))
	})
	_, err := c.Sessions("missing").Get()
	if err == nil {
		t.Fatal("expected error")
	}
	var nf *waotomatis.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("want *NotFoundError, got %T", err)
	}
	if waotomatis.Code(err) != waotomatis.CodeSessionNotFound {
		t.Errorf("code = %q", waotomatis.Code(err))
	}
	if waotomatis.Status(err) != 404 {
		t.Errorf("status = %d", waotomatis.Status(err))
	}
	if waotomatis.RequestID(err) != "req_9" {
		t.Errorf("requestId = %q", waotomatis.RequestID(err))
	}
}

func TestRateLimitRetryAfter(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"error":{"code":"rate_limited","message":"slow down"}}`))
	})
	_, err := c.Sessions("s").Get()
	var rl *waotomatis.RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("want *RateLimitError, got %T", err)
	}
	if rl.RetryAfter != 7 {
		t.Errorf("retryAfter = %d", rl.RetryAfter)
	}
}

func TestMediaUploadFromURL(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/s/media" {
			t.Errorf("path = %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"url":"https://x/y.png"`) {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"mediaId":"med_1","mimeType":"image/png","size":42}`))
	})
	res, err := c.Sessions("s").Media.UploadFromURL("https://x/y.png", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.MediaID != "med_1" || res.Size != 42 {
		t.Errorf("res = %+v", res)
	}
}

func TestMediaUploadBytes(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("content-type = %q", ct)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		f, hdr, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("form file: %v", err)
		}
		defer f.Close()
		if hdr.Filename != "logo.png" {
			t.Errorf("filename = %q", hdr.Filename)
		}
		data, _ := io.ReadAll(f)
		if string(data) != "PNGDATA" {
			t.Errorf("data = %q", data)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"mediaId":"med_2","mimeType":"image/png","size":7}`))
	})
	res, err := c.Sessions("s").Media.Upload([]byte("PNGDATA"),
		&waotomatis.UploadOptions{FileName: "logo.png", MimeType: "image/png"})
	if err != nil {
		t.Fatal(err)
	}
	if res.MediaID != "med_2" {
		t.Errorf("mediaId = %q", res.MediaID)
	}
}

func TestSessionsList(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "2" {
			t.Errorf("limit = %q", r.URL.Query().Get("limit"))
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"sess_1","status":"connected","mode":"cloud_api","sandbox":false,"createdAt":"now"}],"hasMore":true,"cursor":"c2"}`))
	})
	page, err := c.SessionList().List(&waotomatis.ListParams{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "sess_1" || !page.HasMore {
		t.Errorf("page = %+v", page)
	}
	if page.Cursor == nil || *page.Cursor != "c2" {
		t.Errorf("cursor = %v", page.Cursor)
	}
}

func TestMarkRead(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// wamid with `/` must be escaped into a single path segment.
		if !strings.HasSuffix(r.URL.EscapedPath(), "/read") {
			t.Errorf("path = %s", r.URL.EscapedPath())
		}
		if !strings.Contains(r.URL.Path, "wamid.AB/cd") {
			t.Errorf("decoded path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"status":"read"}`))
	})
	res, err := c.Sessions("s").Messages.MarkRead("wamid.AB/cd")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "read" {
		t.Errorf("status = %q", res.Status)
	}
}

func TestWithContextCancel(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // never respond; wait for cancel
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err := c.Sessions("s").Get(waotomatis.WithContext(ctx))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestVerifyWebhook(t *testing.T) {
	body := []byte(`{"event":"message.received","eventId":"e","sessionId":"s","createdAt":"now","data":{"from":"1","chatId":"c","type":"text","providerMessageId":"p","timestamp":1}}`)
	secret := "shh"
	// Sign exactly as the server does.
	sig := signFixture(body, secret)

	if !waotomatis.VerifyWebhook(body, "sha256="+sig, secret) {
		t.Error("expected valid signature to verify")
	}
	if !waotomatis.VerifyWebhook(body, sig, secret) {
		t.Error("expected valid signature (no prefix) to verify")
	}
	if waotomatis.VerifyWebhook(body, "sha256=deadbeef", secret) {
		t.Error("expected bad signature to fail")
	}
	if waotomatis.VerifyWebhook(body, "", secret) {
		t.Error("empty signature must fail")
	}

	event, err := waotomatis.ConstructEvent(body, "sha256="+sig, secret)
	if err != nil {
		t.Fatalf("constructEvent: %v", err)
	}
	if event.Event != waotomatis.EventMessageReceived {
		t.Errorf("event = %q", event.Event)
	}
	var d waotomatis.MessageReceivedData
	if err := json.Unmarshal(event.Data, &d); err != nil {
		t.Fatal(err)
	}
	if d.From != "1" {
		t.Errorf("from = %q", d.From)
	}

	if _, err := waotomatis.ConstructEvent(body, "sha256=bad", secret); err == nil {
		t.Error("expected bad-signature error")
	} else if waotomatis.Code(err) != waotomatis.CodeUnauthorized {
		t.Errorf("code = %q", waotomatis.Code(err))
	}
}
