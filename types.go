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

// MediaType is the kind of media in a MediaMessage (the `type` wire field on
// POST /messages/media).
type MediaType string

const (
	MediaImage    MediaType = "image"
	MediaVideo    MediaType = "video"
	MediaAudio    MediaType = "audio"
	MediaDocument MediaType = "document"
	MediaSticker  MediaType = "sticker"
)

// InteractiveType is the subtype of an InteractiveMessage (the `type` wire field
// on POST /messages/interactive).
type InteractiveType string

const (
	InteractiveTypeButton      InteractiveType = "button"
	InteractiveTypeList        InteractiveType = "list"
	InteractiveTypeCTAUrl      InteractiveType = "cta_url"
	InteractiveTypeFlow        InteractiveType = "flow"
	InteractiveTypeProduct     InteractiveType = "product"
	InteractiveTypeProductList InteractiveType = "product_list"
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

// Each Send* method on MessageResource takes one of the focused input structs
// below. They mirror the per-endpoint request bodies in the API contract: every
// JSON field is exactly the camelCase wire key. The compiler does not enforce
// per-type requirements (e.g. media needs MediaID or Link; interactive needs the
// fields matching its subtype) — the server's validation does.
//
// IdempotencyKey, when set, is sent as the Idempotency-Key header (not in the
// JSON body); an explicit WithIdempotencyKey CallOption still wins.

// TextMessage is the input for MessageResource.SendText.
type TextMessage struct {
	// To is the recipient in E.164 (with or without a leading +). Required.
	To string
	// Text is the message body. Required.
	Text string
	// PreviewURL renders a link preview for the first URL in Text.
	PreviewURL bool
	// ReplyTo quotes a prior message by its provider wamid.
	ReplyTo string
	// IdempotencyKey dedupes retries (sent as the Idempotency-Key header).
	IdempotencyKey string
}

// MediaMessage is the input for MessageResource.SendMedia. Provide exactly one
// of MediaID or Link.
type MediaMessage struct {
	// To is the recipient in E.164. Required.
	To string
	// Type is the media kind (image/video/audio/document/sticker). Required.
	Type MediaType
	// MediaID references previously-uploaded media. Provide MediaID or Link.
	MediaID string
	// Link is a public URL the server fetches at send time. Alternative to MediaID.
	Link string
	// Caption — for image/video/document.
	Caption string
	// FileName — for document.
	FileName string
	// Voice sends audio as a voice note (PTT) rather than an audio file.
	Voice bool
	// ReplyTo quotes a prior message by its provider wamid.
	ReplyTo string
	// IdempotencyKey dedupes retries (sent as the Idempotency-Key header).
	IdempotencyKey string
}

// TemplateMessage is the input for MessageResource.SendTemplate — a pre-approved
// WhatsApp message template invocation.
type TemplateMessage struct {
	// To is the recipient in E.164. Required.
	To string
	// Name is the approved template name. Required.
	Name string
	// LanguageCode is a BCP-47 code (e.g. "en_US", "id"). Required.
	LanguageCode string
	// Components holds the (provider-shaped) template components. Use any so
	// callers can pass header/body/button parameter objects verbatim.
	Components []any
	// ReplyTo quotes a prior message by its provider wamid.
	ReplyTo string
	// IdempotencyKey dedupes retries (sent as the Idempotency-Key header).
	IdempotencyKey string
}

// InteractiveMessage is the input for MessageResource.SendInteractive. Type
// selects the subtype; provide the matching fields:
//
//	"button"        — BodyText + Buttons
//	"list"          — BodyText + ListButton + Sections
//	"cta_url"       — BodyText + CTADisplayText + CTAUrl
//	"flow"          — BodyText + Flow
//	"product"       — CatalogID + ProductRetailerID
//	"product_list"  — CatalogID + ProductSections
//
// matching the server's per-subtype rules (which it enforces).
type InteractiveMessage struct {
	// To is the recipient in E.164. Required.
	To string
	// Type is the interactive subtype. Required.
	Type InteractiveType

	BodyText   string
	HeaderText string
	FooterText string
	Buttons    []InteractiveButton
	ListButton string
	Sections   []InteractiveListSection

	// CTADisplayText and CTAUrl — for Type == "cta_url".
	CTADisplayText string
	CTAUrl         string

	// Flow — for Type == "flow".
	Flow *InteractiveFlow

	// CatalogID — for Type == "product" or "product_list".
	CatalogID string
	// ProductRetailerID — for Type == "product".
	ProductRetailerID string
	// ProductSections — for Type == "product_list".
	ProductSections []InteractiveProductSection

	// ReplyTo quotes a prior message by its provider wamid.
	ReplyTo string
	// IdempotencyKey dedupes retries (sent as the Idempotency-Key header).
	IdempotencyKey string
}

// ReactionMessage is the input for MessageResource.SendReaction. Emoji "" clears
// an existing reaction. Reactions cannot quote a reply (no ReplyTo).
type ReactionMessage struct {
	// To is the recipient in E.164. Required.
	To string
	// MessageID is the provider wamid of the message to react to. Required.
	MessageID string
	// Emoji is the reaction; "" removes a previously-sent reaction.
	Emoji string
	// IdempotencyKey dedupes retries (sent as the Idempotency-Key header).
	IdempotencyKey string
}

// LocationMessage is the input for MessageResource.SendLocation — a pinned
// location.
type LocationMessage struct {
	// To is the recipient in E.164. Required.
	To string
	// Latitude in decimal degrees. Required.
	Latitude float64
	// Longitude in decimal degrees. Required.
	Longitude float64
	// Name is an optional place name.
	Name string
	// Address is an optional street address.
	Address string
	// ReplyTo quotes a prior message by its provider wamid.
	ReplyTo string
	// IdempotencyKey dedupes retries (sent as the Idempotency-Key header).
	IdempotencyKey string
}

// ContactsMessage is the input for MessageResource.SendContacts. Each entry is a
// WhatsApp contact card (requires Name.FormattedName).
type ContactsMessage struct {
	// To is the recipient in E.164. Required.
	To string
	// Contacts is the list of contact cards (at least one). Required.
	Contacts []ContactCard
	// ReplyTo quotes a prior message by its provider wamid.
	ReplyTo string
	// IdempotencyKey dedupes retries (sent as the Idempotency-Key header).
	IdempotencyKey string
}

// CarouselMessage is the input for MessageResource.SendCarousel — a carousel
// template invocation.
type CarouselMessage struct {
	// To is the recipient in E.164. Required.
	To string
	// Name is the carousel template name. Required.
	Name string
	// LanguageCode is a BCP-47 code (e.g. "en_US", "id"). Required.
	LanguageCode string
	// BodyParams fills the message bubble body params.
	BodyParams []string
	// Cards are the carousel cards (at least one). Required.
	Cards []CarouselCard
	// ReplyTo quotes a prior message by its provider wamid.
	ReplyTo string
	// IdempotencyKey dedupes retries (sent as the Idempotency-Key header).
	IdempotencyKey string
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

// ContactCard is a WhatsApp contact card (vCard) for SendContacts. Only
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

// SendMessageResult is returned by every MessageResource.Send* method.
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

// TemplateCategory is the category of a message template.
type TemplateCategory string

const (
	TemplateMarketing      TemplateCategory = "MARKETING"
	TemplateUtility        TemplateCategory = "UTILITY"
	TemplateAuthentication TemplateCategory = "AUTHENTICATION"
)

// Template is a WhatsApp message template as returned by Meta.
//
// Components is Meta's template-component array (HEADER / BODY / FOOTER /
// BUTTONS objects); it is left loosely typed ([]map[string]any) so the full
// provider shape passes through verbatim. QualityScore is provider-defined and
// kept as raw JSON — decode it yourself if you need it.
type Template struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Language   string           `json:"language"`
	Status     string           `json:"status"`
	Category   string           `json:"category"`
	Components []map[string]any `json:"components,omitempty"`
	// QualityScore is Meta's provider-defined quality block (raw JSON).
	QualityScore json.RawMessage `json:"quality_score,omitempty"`
}

// TemplateList is the { data, paging? } envelope returned by Templates.List and
// Templates.Get. Paging carries Meta's cursors (paging.cursors.before/after) and
// is kept as raw JSON.
type TemplateList struct {
	Data   []Template      `json:"data"`
	Paging json.RawMessage `json:"paging,omitempty"`
}

// TemplateListParams are the filter + pagination options for Templates.List.
// All fields are optional; the zero value lists the first page unfiltered.
type TemplateListParams struct {
	// Limit caps the page size (1–100; 0 = server default).
	Limit int
	// After is Meta's pagination cursor (from paging.cursors.after).
	After string
	// Name filters to an exact template name.
	Name string
	// Language filters by BCP-47 code (e.g. "en_US", "id").
	Language string
	// Status filters by APPROVED | PENDING | REJECTED | PAUSED | DISABLED.
	Status string
	// Category filters by MARKETING | UTILITY | AUTHENTICATION.
	Category string
}

// CreateTemplateInput submits a new template to Meta for approval. Components
// follows Meta's template component schema (HEADER / BODY / FOOTER / BUTTONS)
// and is left loosely typed so the provider shape passes through verbatim.
type CreateTemplateInput struct {
	// Name is lowercase letters, digits & underscores (e.g. "order_update").
	Name string `json:"name"`
	// Language is a BCP-47 code (e.g. "en_US", "id").
	Language string `json:"language"`
	// Category is one of the TemplateCategory constants.
	Category TemplateCategory `json:"category"`
	// Components is Meta's component array (at least one entry).
	Components []map[string]any `json:"components"`
	// AllowCategoryChange lets Meta re-categorize the template if needed.
	AllowCategoryChange bool `json:"allowCategoryChange,omitempty"`
}

// TemplateCreateResult is the { id, status, category } envelope returned by
// Templates.Create. Approval is asynchronous — poll Templates.List / Get and
// watch Status (PENDING → APPROVED/REJECTED).
type TemplateCreateResult struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Category string `json:"category"`
}

// TemplateDeleteResult is the { success } envelope returned by Templates.Delete.
type TemplateDeleteResult struct {
	Success bool `json:"success"`
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
