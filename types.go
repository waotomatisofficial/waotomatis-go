package waotomatis

import "encoding/json"

// This file mirrors the core domain types from the API contract
// (apps/api/openapi.json) and the official TypeScript SDK (packages/sdk/src).
//
// Pointer fields are used for values the server marks nullable or optional, so
// the SDK can distinguish "absent" from a zero value when it matters.

// SessionStatus is the lifecycle state of a connected WhatsApp number.
type SessionStatus string

const (
	SessionPending      SessionStatus = "pending"
	SessionConnecting   SessionStatus = "connecting"
	SessionConnected    SessionStatus = "connected"
	SessionDisconnected SessionStatus = "disconnected"
	SessionLoggedOut    SessionStatus = "logged_out"
)

// ConnectionMode is how a session is connected to WhatsApp.
type ConnectionMode string

const (
	ModeCloudAPI    ConnectionMode = "cloud_api"
	ModeCoexistence ConnectionMode = "coexistence"
)

// MessageType enumerates the kinds of message you can send.
type MessageType string

const (
	TypeText        MessageType = "text"
	TypeImage       MessageType = "image"
	TypeVideo       MessageType = "video"
	TypeAudio       MessageType = "audio"
	TypeDocument    MessageType = "document"
	TypeSticker     MessageType = "sticker"
	TypeTemplate    MessageType = "template"
	TypeInteractive MessageType = "interactive"
	TypeReaction    MessageType = "reaction"
	TypeLocation    MessageType = "location"
	TypeContacts    MessageType = "contacts"
	TypeCarousel    MessageType = "carousel"
)

// Session is a connected WhatsApp number.
type Session struct {
	ID            string         `json:"id"`
	TeamID        *string        `json:"teamId,omitempty"`
	Label         *string        `json:"label,omitempty"`
	Status        SessionStatus  `json:"status"`
	Mode          ConnectionMode `json:"mode"`
	Sandbox       bool           `json:"sandbox"`
	PhoneNumber   *string        `json:"phoneNumber,omitempty"`
	PhoneNumberID *string        `json:"phoneNumberId,omitempty"`
	WabaID        *string        `json:"wabaId,omitempty"`
	ConnectedAt   *string        `json:"connectedAt,omitempty"`
	CreatedAt     string         `json:"createdAt"`
}

// Message is the input for Messages.Send. It is a flat struct (matching the
// server's SendMessageInput) covering every message type; populate only the
// fields relevant to Type. The compiler does not enforce per-type requirements,
// but the server's validation does (text needs Text; media needs MediaID or
// Link; template needs Template; interactive needs Interactive).
type Message struct {
	// To is the recipient in E.164 (with or without a leading +). Required.
	To string `json:"to"`
	// Type is the message kind. Required.
	Type MessageType `json:"type"`

	// Text — for Type == "text".
	Text string `json:"text,omitempty"`
	// PreviewURL renders a link preview for the first URL in Text.
	PreviewURL bool `json:"previewUrl,omitempty"`

	// MediaID references previously-uploaded media (image/video/audio/document/
	// sticker). Provide exactly one of MediaID or Link.
	MediaID string `json:"mediaId,omitempty"`
	// Link is a public URL the server fetches at send time. Alternative to MediaID.
	Link string `json:"link,omitempty"`
	// Caption — for image/video/document.
	Caption string `json:"caption,omitempty"`
	// FileName — for document.
	FileName string `json:"fileName,omitempty"`
	// Voice sends audio as a voice note (PTT) rather than an audio file.
	Voice bool `json:"voice,omitempty"`

	// Template — for Type == "template".
	Template *TemplateInput `json:"template,omitempty"`
	// Interactive — for Type == "interactive".
	Interactive *InteractiveInput `json:"interactive,omitempty"`

	// Reaction — for Type == "reaction".
	Reaction *ReactionInput `json:"reaction,omitempty"`
	// Location — for Type == "location".
	Location *LocationInput `json:"location,omitempty"`
	// Contacts — for Type == "contacts". Each entry is a WhatsApp contact card
	// (requires Name.FormattedName).
	Contacts []ContactCard `json:"contacts,omitempty"`
	// Carousel — for Type == "carousel".
	Carousel *CarouselInput `json:"carousel,omitempty"`

	// ReplyTo quotes a prior message by its provider wamid.
	ReplyTo string `json:"replyTo,omitempty"`

	// IdempotencyKey dedupes retries; the same key returns the original result.
	// It is sent as the Idempotency-Key header, not in the JSON body.
	IdempotencyKey string `json:"-"`
}

// TemplateInput is a pre-approved WhatsApp message template invocation.
type TemplateInput struct {
	Name         string `json:"name"`
	LanguageCode string `json:"languageCode"`
	// Components holds the (provider-shaped) template components. Use any so
	// callers can pass header/body/button parameter objects verbatim.
	Components []any `json:"components,omitempty"`
}

// InteractiveButton is a single reply button (max 3 per message).
type InteractiveButton struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// InteractiveListRow is a row inside a list section.
type InteractiveListRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// InteractiveListSection is a list section (must hold at least one row).
type InteractiveListSection struct {
	Title string               `json:"title,omitempty"`
	Rows  []InteractiveListRow `json:"rows"`
}

// InteractiveProductSection is a product_list section (must hold at least one
// product item).
type InteractiveProductSection struct {
	Title        string                   `json:"title,omitempty"`
	ProductItems []InteractiveProductItem `json:"productItems"`
}

// InteractiveProductItem references a single catalog product by its retailer id.
type InteractiveProductItem struct {
	ProductRetailerID string `json:"productRetailerId"`
}

// InteractiveFlow configures a Flow-type interactive message (the flow CTA and
// its launch parameters).
type InteractiveFlow struct {
	FlowCTA   string `json:"flowCta"`
	FlowID    string `json:"flowId,omitempty"`
	FlowToken string `json:"flowToken,omitempty"`
	// FlowAction is "navigate" or "data_exchange".
	FlowAction        string `json:"flowAction,omitempty"`
	FlowActionPayload any    `json:"flowActionPayload,omitempty"`
	// Mode is "draft" or "published".
	Mode string `json:"mode,omitempty"`
}

// InteractiveInput is interactive content. Type selects the shape:
//
//	"button"        — use Buttons
//	"list"          — use ListButton + Sections
//	"cta_url"       — use BodyText + CTADisplayText + CTAUrl
//	"flow"          — use BodyText + Flow
//	"product"       — use CatalogID + ProductRetailerID
//	"product_list"  — use HeaderText + BodyText + CatalogID + ProductSections
//
// matching the server's superRefine rules. The compiler does not enforce
// per-type requirements; the server's validation does.
type InteractiveInput struct {
	Type       string                   `json:"type"`
	BodyText   string                   `json:"bodyText,omitempty"`
	HeaderText string                   `json:"headerText,omitempty"`
	FooterText string                   `json:"footerText,omitempty"`
	Buttons    []InteractiveButton      `json:"buttons,omitempty"`
	ListButton string                   `json:"listButton,omitempty"`
	Sections   []InteractiveListSection `json:"sections,omitempty"`

	// CTADisplayText and CTAUrl — for Type == "cta_url".
	CTADisplayText string `json:"ctaDisplayText,omitempty"`
	CTAUrl         string `json:"ctaUrl,omitempty"`

	// Flow — for Type == "flow".
	Flow *InteractiveFlow `json:"flow,omitempty"`

	// CatalogID — for Type == "product" or "product_list".
	CatalogID string `json:"catalogId,omitempty"`
	// ProductRetailerID — for Type == "product".
	ProductRetailerID string `json:"productRetailerId,omitempty"`
	// ProductSections — for Type == "product_list".
	ProductSections []InteractiveProductSection `json:"productSections,omitempty"`
}

// ReactionInput reacts to a prior message. Emoji "" clears an existing reaction.
type ReactionInput struct {
	MessageID string `json:"messageId"`
	Emoji     string `json:"emoji"`
}

// LocationInput is a pinned location.
type LocationInput struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

// ContactCard is a WhatsApp contact card (vCard) for Type == "contacts". Only
// Name.FormattedName is required; every other field is optional. Wire keys match
// the WhatsApp Cloud API's snake_case contact shape (formatted_name, wa_id, …).
//
// (Distinct from Contact, which is a cached counterpart returned by the contacts
// list endpoint.)
type ContactCard struct {
	Name      ContactName      `json:"name"`
	Phones    []ContactPhone   `json:"phones,omitempty"`
	Emails    []ContactEmail   `json:"emails,omitempty"`
	Org       *ContactOrg      `json:"org,omitempty"`
	URLs      []ContactURL     `json:"urls,omitempty"`
	Addresses []ContactAddress `json:"addresses,omitempty"`
	// Birthday in YYYY-MM-DD form.
	Birthday string `json:"birthday,omitempty"`
}

// ContactName is the name block of a contact card. FormattedName is required.
type ContactName struct {
	FormattedName string `json:"formatted_name"`
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	MiddleName    string `json:"middle_name,omitempty"`
	Suffix        string `json:"suffix,omitempty"`
	Prefix        string `json:"prefix,omitempty"`
}

// ContactPhone is a phone entry on a contact card. WaID, when set, links the
// number to its WhatsApp account.
type ContactPhone struct {
	Phone string `json:"phone,omitempty"`
	Type  string `json:"type,omitempty"`
	WaID  string `json:"wa_id,omitempty"`
}

// ContactEmail is an email entry on a contact card.
type ContactEmail struct {
	Email string `json:"email,omitempty"`
	Type  string `json:"type,omitempty"`
}

// ContactOrg is the organization block of a contact card.
type ContactOrg struct {
	Company    string `json:"company,omitempty"`
	Department string `json:"department,omitempty"`
	Title      string `json:"title,omitempty"`
}

// ContactURL is a URL entry on a contact card.
type ContactURL struct {
	URL  string `json:"url,omitempty"`
	Type string `json:"type,omitempty"`
}

// ContactAddress is a postal address entry on a contact card.
type ContactAddress struct {
	Street      string `json:"street,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	Zip         string `json:"zip,omitempty"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Type        string `json:"type,omitempty"`
}

// CarouselButton is a button on a carousel card. SubType is "quick_reply" or
// "url".
type CarouselButton struct {
	SubType  string `json:"subType"`
	Index    int    `json:"index"`
	Payload  string `json:"payload,omitempty"`
	URLParam string `json:"urlParam,omitempty"`
}

// CarouselCard is a single card in a carousel template. Provide at most one
// header media field (image id/link or video id/link).
type CarouselCard struct {
	HeaderImageID   string           `json:"headerImageId,omitempty"`
	HeaderImageLink string           `json:"headerImageLink,omitempty"`
	HeaderVideoID   string           `json:"headerVideoId,omitempty"`
	HeaderVideoLink string           `json:"headerVideoLink,omitempty"`
	BodyParams      []string         `json:"bodyParams,omitempty"`
	Buttons         []CarouselButton `json:"buttons,omitempty"`
}

// CarouselInput is a carousel template invocation.
type CarouselInput struct {
	Name         string         `json:"name"`
	LanguageCode string         `json:"languageCode"`
	BodyParams   []string       `json:"bodyParams,omitempty"`
	Cards        []CarouselCard `json:"cards"`
}

// SendMessageResult is returned by Messages.Send.
type SendMessageResult struct {
	ID                string  `json:"id"`
	EventID           string  `json:"eventId"`
	ProviderMessageID *string `json:"providerMessageId,omitempty"`
	Status            string  `json:"status"`
	Idempotent        bool    `json:"idempotent,omitempty"`
}

// UploadMediaResult is returned by Media.Upload / Media.UploadFromURL.
type UploadMediaResult struct {
	MediaID  string `json:"mediaId"`
	MimeType string `json:"mimeType"`
	Size     int64  `json:"size"`
}

// MediaDownload is the result of Media.Download (raw inbound media bytes).
type MediaDownload struct {
	Data     []byte
	MimeType string
}

// SimpleStatus is the { status } envelope returned by Messages.MarkRead.
type SimpleStatus struct {
	Status string `json:"status"`
}

// DeleteResult is the { id, status } envelope returned by Sessions.Delete.
type DeleteResult struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// Chat is a conversation summary (last message + contact name per counterpart).
type Chat struct {
	ChatID        string  `json:"chatId"`
	LastMessageID string  `json:"lastMessageId,omitempty"`
	Direction     string  `json:"direction,omitempty"`
	Type          string  `json:"type,omitempty"`
	Status        string  `json:"status,omitempty"`
	LastMessageAt string  `json:"lastMessageAt"`
	LastText      *string `json:"lastText,omitempty"`
	ContactName   *string `json:"contactName,omitempty"`
}

// ChatMessage is a single message in a chat's history.
type ChatMessage struct {
	ID                string  `json:"id"`
	EventID           string  `json:"eventId"`
	ChatID            string  `json:"chatId"`
	Direction         string  `json:"direction"`
	Type              string  `json:"type"`
	Status            string  `json:"status"`
	ProviderMessageID *string `json:"providerMessageId,omitempty"`
	Payload           any     `json:"payload,omitempty"`
	Timestamp         string  `json:"timestamp"`
}

// Contact is a cached counterpart (profile name keyed by WhatsApp id).
type Contact struct {
	WaID      string  `json:"waId"`
	Name      *string `json:"name,omitempty"`
	CreatedAt string  `json:"createdAt,omitempty"`
	UpdatedAt string  `json:"updatedAt,omitempty"`
}

// EventType is a webhook / realtime event name.
type EventType string

const (
	EventMessageReceived EventType = "message.received"
	EventMessageUpdated  EventType = "message.updated"
	EventSessionStatus   EventType = "session.status"
)

// Webhook is a registered webhook subscription. Secret is returned ONCE, at
// creation time, by Webhooks.Create.
type Webhook struct {
	ID     string      `json:"id"`
	URL    string      `json:"url"`
	Events []EventType `json:"events"`
	Secret string      `json:"secret,omitempty"`
	Active bool        `json:"active"`
}

// CreateWebhookInput registers a webhook for a session.
type CreateWebhookInput struct {
	URL    string      `json:"url"`
	Events []EventType `json:"events"`
}

// Page is the envelope returned by every cursor-paginated list endpoint.
type Page[T any] struct {
	Data    []T     `json:"data"`
	HasMore bool    `json:"hasMore"`
	Cursor  *string `json:"cursor,omitempty"`
}

// ListParams are the common options for cursor-paginated list methods.
type ListParams struct {
	// Cursor is the opaque page cursor returned by a previous page (empty = first).
	Cursor string
	// Limit caps the page size (0 = server default).
	Limit int
}

// WebhookEvent is the envelope shared by webhooks and realtime. The Data shape
// depends on Event; decode it into the matching *Data struct, or leave it as
// raw JSON.
type WebhookEvent struct {
	EventID   string          `json:"eventId"`
	Event     EventType       `json:"event"`
	SessionID string          `json:"sessionId"`
	CreatedAt string          `json:"createdAt"`
	Data      json.RawMessage `json:"data"`
}

// MessageReceivedData is the Data payload for "message.received" events.
type MessageReceivedData struct {
	From              string  `json:"from"`
	ChatID            string  `json:"chatId"`
	Type              string  `json:"type"`
	Text              *string `json:"text,omitempty"`
	Media             any     `json:"media,omitempty"`
	ProviderMessageID string  `json:"providerMessageId"`
	Timestamp         any     `json:"timestamp"`
}

// MessageUpdatedData is the Data payload for "message.updated" events.
type MessageUpdatedData struct {
	ProviderMessageID string `json:"providerMessageId"`
	Status            string `json:"status"`
	Recipient         string `json:"recipient"`
}

// SessionStatusData is the Data payload for "session.status" events.
type SessionStatusData struct {
	Status SessionStatus `json:"status"`
}
