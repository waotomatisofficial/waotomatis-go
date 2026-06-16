package waotomatis

import (
	"net/url"
)

// WebhookResource manages webhook subscriptions for a session. Obtain it via
// Client.Sessions(id).Webhooks.
type WebhookResource struct {
	client    *Client
	sessionID string
}

func (r *WebhookResource) base() string {
	return "/v1/sessions/" + url.PathEscape(r.sessionID) + "/webhooks"
}

// Create registers a webhook. The signing Secret is returned ONCE, here — store
// it to verify deliveries with VerifyWebhook / ConstructEvent.
func (r *WebhookResource) Create(in *CreateWebhookInput, opts ...CallOption) (*Webhook, error) {
	var out Webhook
	err := r.client.doJSON(request{method: "POST", path: r.base(), body: in, opts: opts}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns one page of this session's webhooks.
func (r *WebhookResource) List(params *ListParams, opts ...CallOption) (*Page[Webhook], error) {
	var out Page[Webhook]
	err := r.client.doJSON(request{method: "GET", path: r.base(), query: listQuery(params), opts: opts}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete removes a webhook by id.
func (r *WebhookResource) Delete(webhookID string, opts ...CallOption) (*DeleteResult, error) {
	var out DeleteResult
	err := r.client.doJSON(request{method: "DELETE", path: r.base() + "/" + url.PathEscape(webhookID), opts: opts}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
