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
		if r.URL.Path != "/v1/sessions/sess_123/messages/text" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in map[string]any
		_ = json.Unmarshal(body, &in)
		// "type" must NOT be on the wire for the per-type text endpoint.
		if _, ok := in["type"]; ok {
			t.Errorf("text body should not carry `type`: %s", body)
		}
		if in["to"] != "628123456789" || in["text"] != "Halo" || in["previewUrl"] != true {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"msg_abc123","eventId":"evt_1","status":"sent"}`))
	})

	msg, err := c.Sessions("sess_123").Messages.SendText(&waotomatis.TextMessage{
		To:         "628123456789",
		Text:       "Halo",
		PreviewURL: true,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if msg.ID != "msg_abc123" {
		t.Errorf("id = %q", msg.ID)
	}
}

func TestSendMedia(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/s/messages/media" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in map[string]any
		_ = json.Unmarshal(body, &in)
		if in["type"] != "image" || in["mediaId"] != "med_1" || in["caption"] != "Invoice" {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"msg_m1","eventId":"e","status":"sent"}`))
	})
	msg, err := c.Sessions("s").Messages.SendMedia(&waotomatis.MediaMessage{
		To: "1", Type: waotomatis.MediaImage, MediaID: "med_1", Caption: "Invoice",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if msg.ID != "msg_m1" {
		t.Errorf("id = %q", msg.ID)
	}
}

func TestSendInteractive(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/s/messages/interactive" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in struct {
			Type     string `json:"type"`
			BodyText string `json:"bodyText"`
			Buttons  []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"buttons"`
		}
		if err := json.Unmarshal(body, &in); err != nil {
			t.Fatalf("unmarshal: %v (%s)", err, body)
		}
		if in.Type != "button" || in.BodyText != "Pick one" || len(in.Buttons) != 1 ||
			in.Buttons[0].ID != "yes" || in.Buttons[0].Title != "Yes" {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"msg_i1","eventId":"e","status":"sent"}`))
	})
	msg, err := c.Sessions("s").Messages.SendInteractive(&waotomatis.InteractiveMessage{
		To:       "1",
		Type:     waotomatis.InteractiveTypeButton,
		BodyText: "Pick one",
		Buttons:  []waotomatis.InteractiveButton{{ID: "yes", Title: "Yes"}},
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if msg.ID != "msg_i1" {
		t.Errorf("id = %q", msg.ID)
	}
}

func TestSendReactionEmptyEmojiClears(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/s/messages/reaction" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in map[string]any
		_ = json.Unmarshal(body, &in)
		// Empty emoji must still be emitted (it clears a reaction).
		emoji, ok := in["emoji"]
		if !ok || emoji != "" {
			t.Errorf("emoji must be present and empty: %s", body)
		}
		if in["messageId"] != "wamid.X" {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"msg_r1","eventId":"e","status":"sent"}`))
	})
	_, err := c.Sessions("s").Messages.SendReaction(&waotomatis.ReactionMessage{
		To: "1", MessageID: "wamid.X", Emoji: "",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSendLocation(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/s/messages/location" {
			t.Errorf("path = %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Name      string  `json:"name"`
		}
		_ = json.Unmarshal(body, &in)
		if in.Latitude != -6.2088 || in.Longitude != 106.8456 || in.Name != "Kantor" {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"msg_l1","eventId":"e","status":"sent"}`))
	})
	_, err := c.Sessions("s").Messages.SendLocation(&waotomatis.LocationMessage{
		To: "1", Latitude: -6.2088, Longitude: 106.8456, Name: "Kantor",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSendCarousel(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/s/messages/carousel" {
			t.Errorf("path = %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in struct {
			Name         string `json:"name"`
			LanguageCode string `json:"languageCode"`
			Cards        []struct {
				BodyParams []string `json:"bodyParams"`
			} `json:"cards"`
		}
		_ = json.Unmarshal(body, &in)
		if in.Name != "promo" || in.LanguageCode != "id" || len(in.Cards) != 1 {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"msg_ca1","eventId":"e","status":"sent"}`))
	})
	_, err := c.Sessions("s").Messages.SendCarousel(&waotomatis.CarouselMessage{
		To: "1", Name: "promo", LanguageCode: "id",
		Cards: []waotomatis.CarouselCard{{BodyParams: []string{"Budi"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSendTemplate(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/s/messages/template" {
			t.Errorf("path = %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in struct {
			Name         string `json:"name"`
			LanguageCode string `json:"languageCode"`
		}
		_ = json.Unmarshal(body, &in)
		if in.Name != "order_update" || in.LanguageCode != "en_US" {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"msg_t1","eventId":"e","status":"sent"}`))
	})
	_, err := c.Sessions("s").Messages.SendTemplate(&waotomatis.TemplateMessage{
		To: "1", Name: "order_update", LanguageCode: "en_US",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIdempotencyKeyHeader(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Idempotency-Key"); got != "key-1" {
			t.Errorf("idempotency header = %q", got)
		}
		if r.URL.Path != "/v1/sessions/s/messages/text" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"m","eventId":"e","status":"sent"}`))
	})
	_, err := c.Sessions("s").Messages.SendText(&waotomatis.TextMessage{
		To: "1", Text: "hi", IdempotencyKey: "key-1",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIdempotencyKeyCallOptionWins(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// An explicit WithIdempotencyKey CallOption overrides the input field.
		if got := r.Header.Get("Idempotency-Key"); got != "override" {
			t.Errorf("idempotency header = %q", got)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"m","eventId":"e","status":"sent"}`))
	})
	_, err := c.Sessions("s").Messages.SendText(
		&waotomatis.TextMessage{To: "1", Text: "hi", IdempotencyKey: "from-input"},
		waotomatis.WithIdempotencyKey("override"),
	)
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

func TestSendContacts(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/sess_123/messages/contacts" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in struct {
			Contacts []struct {
				Name struct {
					FormattedName string `json:"formatted_name"`
					FirstName     string `json:"first_name"`
				} `json:"name"`
				Phones []struct {
					Phone string `json:"phone"`
					Type  string `json:"type"`
					WaID  string `json:"wa_id"`
				} `json:"phones"`
				Org *struct {
					Company string `json:"company"`
				} `json:"org"`
			} `json:"contacts"`
		}
		if err := json.Unmarshal(body, &in); err != nil {
			t.Fatalf("unmarshal: %v (%s)", err, body)
		}
		if len(in.Contacts) != 1 {
			t.Fatalf("body = %s", body)
		}
		ct := in.Contacts[0]
		if ct.Name.FormattedName != "Budi Santoso" || ct.Name.FirstName != "Budi" {
			t.Errorf("name = %+v", ct.Name)
		}
		if len(ct.Phones) != 1 || ct.Phones[0].Phone != "+628123456789" ||
			ct.Phones[0].Type != "WORK" || ct.Phones[0].WaID != "628123456789" {
			t.Errorf("phones = %+v", ct.Phones)
		}
		if ct.Org == nil || ct.Org.Company != "WAOtomatis" {
			t.Errorf("org = %+v", ct.Org)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"msg_c1","eventId":"evt_1","status":"sent"}`))
	})

	msg, err := c.Sessions("sess_123").Messages.SendContacts(&waotomatis.ContactsMessage{
		To: "628123456789",
		Contacts: []waotomatis.ContactCard{{
			Name:   waotomatis.ContactName{FormattedName: "Budi Santoso", FirstName: "Budi"},
			Phones: []waotomatis.ContactPhone{{Phone: "+628123456789", Type: "WORK", WaID: "628123456789"}},
			Org:    &waotomatis.ContactOrg{Company: "WAOtomatis"},
		}},
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if msg.ID != "msg_c1" {
		t.Errorf("id = %q", msg.ID)
	}
}

func TestTemplatesList(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/sess_123/templates" || r.Method != http.MethodGet {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("limit") != "50" || q.Get("status") != "APPROVED" || q.Get("category") != "MARKETING" {
			t.Errorf("query = %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"t1","name":"promo","language":"id","status":"APPROVED","category":"MARKETING","components":[{"type":"BODY","text":"hi"}],"quality_score":{"score":"GREEN"}}],"paging":{"cursors":{"after":"c2"}}}`))
	})
	list, err := c.Sessions("sess_123").Templates.List(&waotomatis.TemplateListParams{
		Limit: 50, Status: "APPROVED", Category: "MARKETING",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Data) != 1 || list.Data[0].Name != "promo" || list.Data[0].Status != "APPROVED" {
		t.Errorf("data = %+v", list.Data)
	}
	if len(list.Data[0].Components) != 1 || list.Data[0].Components[0]["type"] != "BODY" {
		t.Errorf("components = %+v", list.Data[0].Components)
	}
	if string(list.Paging) == "" || string(list.Data[0].QualityScore) == "" {
		t.Errorf("paging=%s quality=%s", list.Paging, list.Data[0].QualityScore)
	}
}

func TestTemplatesGet(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/s/templates/promo_diskon" || r.Method != http.MethodGet {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"t1","name":"promo_diskon","language":"id","status":"APPROVED","category":"MARKETING"}]}`))
	})
	got, err := c.Sessions("s").Templates.Get("promo_diskon")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Data) != 1 || got.Data[0].Name != "promo_diskon" {
		t.Errorf("data = %+v", got.Data)
	}
}

func TestTemplatesCreate(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/s/templates" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in struct {
			Name       string           `json:"name"`
			Language   string           `json:"language"`
			Category   string           `json:"category"`
			Components []map[string]any `json:"components"`
		}
		if err := json.Unmarshal(body, &in); err != nil {
			t.Fatalf("unmarshal: %v (%s)", err, body)
		}
		if in.Name != "promo_diskon" || in.Language != "id" || in.Category != "MARKETING" {
			t.Errorf("body = %s", body)
		}
		if len(in.Components) != 1 || in.Components[0]["type"] != "BODY" {
			t.Errorf("components = %+v", in.Components)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"t_new","status":"PENDING","category":"MARKETING"}`))
	})
	res, err := c.Sessions("s").Templates.Create(&waotomatis.CreateTemplateInput{
		Name:     "promo_diskon",
		Language: "id",
		Category: waotomatis.TemplateMarketing,
		Components: []map[string]any{
			{"type": "BODY", "text": "Halo {{1}}"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ID != "t_new" || res.Status != "PENDING" {
		t.Errorf("res = %+v", res)
	}
}

func TestTemplatesDelete(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/s/templates/promo_diskon" || r.Method != http.MethodDelete {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true}`))
	})
	res, err := c.Sessions("s").Templates.Delete("promo_diskon")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success {
		t.Errorf("success = %v", res.Success)
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
