package waotomatis

import (
	"net/url"
	"strconv"
)

// SessionResource is the entry point for everything you can do with one session.
// Obtain it via Client.Sessions(id):
//
//	sess := client.Sessions("sess_123")
//	sess.Messages.SendText(&waotomatis.TextMessage{To: "62812...", Text: "hi"})
//	sess.Media.UploadFromURL("https://...")
type SessionResource struct {
	client *Client
	// ID is the session id this resource is scoped to.
	ID string

	// Messages sends messages and marks inbound ones as read.
	Messages *MessageResource
	// Media uploads media (by URL or raw bytes) and downloads inbound media.
	Media *MediaResource
	// Webhooks manages webhook subscriptions for this session.
	Webhooks *WebhookResource
	// Templates manages WhatsApp message templates on this session's WABA.
	Templates *TemplateResource
}

func newSessionResource(c *Client, id string) *SessionResource {
	return &SessionResource{
		client:    c,
		ID:        id,
		Messages:  &MessageResource{client: c, sessionID: id},
		Media:     &MediaResource{client: c, sessionID: id},
		Webhooks:  &WebhookResource{client: c, sessionID: id},
		Templates: &TemplateResource{client: c, sessionID: id},
	}
}

// Get fetches this session's current state.
func (r *SessionResource) Get(opts ...CallOption) (*Session, error) {
	var out Session
	err := r.client.doJSON(request{method: "GET", path: "/v1/sessions/" + url.PathEscape(r.ID), opts: opts}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete disconnects this session and drops its stored token.
func (r *SessionResource) Delete(opts ...CallOption) (*DeleteResult, error) {
	var out DeleteResult
	err := r.client.doJSON(request{method: "DELETE", path: "/v1/sessions/" + url.PathEscape(r.ID), opts: opts}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SessionsService is the non-scoped sessions sub-API: list and get/delete by id.
// Obtain it via Client.SessionList().
type SessionsService struct {
	client *Client
}

// List returns one page of the org's sessions. Pass &ListParams{} (or nil) for
// the first page; follow Page.Cursor for the next.
func (s *SessionsService) List(params *ListParams, opts ...CallOption) (*Page[Session], error) {
	var out Page[Session]
	err := s.client.doJSON(request{method: "GET", path: "/v1/sessions", query: listQuery(params), opts: opts}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Get fetches a single session by id.
func (s *SessionsService) Get(id string, opts ...CallOption) (*Session, error) {
	return s.client.Sessions(id).Get(opts...)
}

// Delete disconnects a session by id.
func (s *SessionsService) Delete(id string, opts ...CallOption) (*DeleteResult, error) {
	return s.client.Sessions(id).Delete(opts...)
}

// MessageResource sends messages on a session. Obtain it via
// Client.Sessions(id).Messages.
//
// There is one method per message type, each posting to its own endpoint with a
// focused, typed input struct:
//
//	SendText        → POST /messages/text
//	SendMedia       → POST /messages/media        (Type = image|video|audio|document|sticker)
//	SendTemplate    → POST /messages/template
//	SendInteractive → POST /messages/interactive  (Type = button|list|cta_url|flow|product|product_list)
//	SendReaction    → POST /messages/reaction
//	SendLocation    → POST /messages/location
//	SendContacts    → POST /messages/contacts
//	SendCarousel    → POST /messages/carousel
type MessageResource struct {
	client    *Client
	sessionID string
}

// messagesBase is the per-type messages route prefix for this session.
func (r *MessageResource) messagesBase() string {
	return "/v1/sessions/" + url.PathEscape(r.sessionID) + "/messages"
}

// send posts a per-type message body to /messages/<suffix> and decodes the
// result. The input's IdempotencyKey (when non-empty) is sent as the
// Idempotency-Key header; an explicit WithIdempotencyKey CallOption still wins.
func (r *MessageResource) send(suffix string, body any, idempotencyKey string, opts []CallOption) (*SendMessageResult, error) {
	if idempotencyKey != "" {
		// Prepend so a caller-supplied WithIdempotencyKey still overrides it.
		opts = append([]CallOption{WithIdempotencyKey(idempotencyKey)}, opts...)
	}
	var out SendMessageResult
	err := r.client.doJSON(request{
		method: "POST",
		path:   r.messagesBase() + "/" + suffix,
		body:   body,
		opts:   opts,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SendText sends a plain text message (optionally with a link preview).
//
//	msg, err := client.Sessions("sess_123").Messages.SendText(&waotomatis.TextMessage{
//	    To:   "628123456789",
//	    Text: "Halo dari WAOtomatis 👋",
//	})
//
// Pass waotomatis.WithContext(ctx) to attach a context for cancellation.
func (r *MessageResource) SendText(in *TextMessage, opts ...CallOption) (*SendMessageResult, error) {
	return r.send("text", textWire(in), in.IdempotencyKey, opts)
}

// SendMedia sends a media message. Set Type to one of image/video/audio/
// document/sticker and provide exactly one of MediaID or Link.
func (r *MessageResource) SendMedia(in *MediaMessage, opts ...CallOption) (*SendMessageResult, error) {
	return r.send("media", mediaWire(in), in.IdempotencyKey, opts)
}

// SendTemplate sends a pre-approved message template. Components carries Meta's
// (provider-shaped) template component objects verbatim.
func (r *MessageResource) SendTemplate(in *TemplateMessage, opts ...CallOption) (*SendMessageResult, error) {
	return r.send("template", templateWire(in), in.IdempotencyKey, opts)
}

// SendInteractive sends an interactive message. Set Type to one of button/list/
// cta_url/flow/product/product_list and provide the matching fields (the server
// validates per-subtype requirements).
func (r *MessageResource) SendInteractive(in *InteractiveMessage, opts ...CallOption) (*SendMessageResult, error) {
	return r.send("interactive", interactiveWire(in), in.IdempotencyKey, opts)
}

// SendReaction reacts to a prior message with an emoji. Emoji "" clears an
// existing reaction. Reactions cannot quote a reply.
func (r *MessageResource) SendReaction(in *ReactionMessage, opts ...CallOption) (*SendMessageResult, error) {
	return r.send("reaction", reactionWire(in), in.IdempotencyKey, opts)
}

// SendLocation shares a location pin.
func (r *MessageResource) SendLocation(in *LocationMessage, opts ...CallOption) (*SendMessageResult, error) {
	return r.send("location", locationWire(in), in.IdempotencyKey, opts)
}

// SendContacts shares one or more contact cards. Each card requires
// Name.FormattedName.
func (r *MessageResource) SendContacts(in *ContactsMessage, opts ...CallOption) (*SendMessageResult, error) {
	return r.send("contacts", contactsWire(in), in.IdempotencyKey, opts)
}

// SendCarousel sends a carousel template (a named, language-tagged template with
// cards).
func (r *MessageResource) SendCarousel(in *CarouselMessage, opts ...CallOption) (*SendMessageResult, error) {
	return r.send("carousel", carouselWire(in), in.IdempotencyKey, opts)
}

// MarkRead marks an inbound message (by its provider wamid) as read.
func (r *MessageResource) MarkRead(providerMessageID string, opts ...CallOption) (*SimpleStatus, error) {
	var out SimpleStatus
	// wamids contain `/`, `+`, `=` — escape so the path routes correctly.
	path := r.messagesBase() + "/" + url.PathEscape(providerMessageID) + "/read"
	err := r.client.doJSON(request{method: "POST", path: path, opts: opts}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// listQuery builds the query map for a cursor-paginated request.
func listQuery(p *ListParams) map[string]string {
	if p == nil {
		return nil
	}
	q := map[string]string{}
	if p.Cursor != "" {
		q["cursor"] = p.Cursor
	}
	if p.Limit > 0 {
		q["limit"] = strconv.Itoa(p.Limit)
	}
	return q
}
