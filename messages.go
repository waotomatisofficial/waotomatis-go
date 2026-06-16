package waotomatis

import (
	"net/url"
	"strconv"
)

// SessionResource is the entry point for everything you can do with one session.
// Obtain it via Client.Sessions(id):
//
//	sess := client.Sessions("sess_123")
//	sess.Messages.Send(ctx, &waotomatis.Message{...})
//	sess.Media.UploadFromURL(ctx, "https://...")
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
}

func newSessionResource(c *Client, id string) *SessionResource {
	return &SessionResource{
		client:   c,
		ID:       id,
		Messages: &MessageResource{client: c, sessionID: id},
		Media:    &MediaResource{client: c, sessionID: id},
		Webhooks: &WebhookResource{client: c, sessionID: id},
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
type MessageResource struct {
	client    *Client
	sessionID string
}

// Send delivers a message. msg.IdempotencyKey, when set, is sent as the
// Idempotency-Key header (an explicit WithIdempotencyKey CallOption wins).
//
//	msg, err := client.Sessions("sess_123").Messages.Send(&waotomatis.Message{
//	    To:   "628123456789",
//	    Type: "text",
//	    Text: "Halo dari WAOtomatis 👋",
//	})
//
// Pass waotomatis.WithContext(ctx) to attach a context for cancellation.
func (r *MessageResource) Send(msg *Message, opts ...CallOption) (*SendMessageResult, error) {
	if msg.IdempotencyKey != "" {
		// Prepend so a caller-supplied WithIdempotencyKey still overrides it.
		opts = append([]CallOption{WithIdempotencyKey(msg.IdempotencyKey)}, opts...)
	}
	var out SendMessageResult
	err := r.client.doJSON(request{
		method: "POST",
		path:   "/v1/sessions/" + url.PathEscape(r.sessionID) + "/messages",
		body:   msg,
		opts:   opts,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// MarkRead marks an inbound message (by its provider wamid) as read.
func (r *MessageResource) MarkRead(providerMessageID string, opts ...CallOption) (*SimpleStatus, error) {
	var out SimpleStatus
	// wamids contain `/`, `+`, `=` — escape so the path routes correctly.
	path := "/v1/sessions/" + url.PathEscape(r.sessionID) + "/messages/" + url.PathEscape(providerMessageID) + "/read"
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
