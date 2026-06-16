package waotomatis

// This file maps each focused Send* input struct onto its exact per-endpoint
// JSON request body. The wire structs carry the precise camelCase keys from the
// API contract (apps/api/src/routes/messages.ts) and omit the IdempotencyKey
// field (which travels as a header, not in the body).

type textWireBody struct {
	To         string `json:"to"`
	Text       string `json:"text"`
	PreviewURL bool   `json:"previewUrl,omitempty"`
	ReplyTo    string `json:"replyTo,omitempty"`
}

func textWire(in *TextMessage) textWireBody {
	return textWireBody{To: in.To, Text: in.Text, PreviewURL: in.PreviewURL, ReplyTo: in.ReplyTo}
}

type mediaWireBody struct {
	To       string    `json:"to"`
	Type     MediaType `json:"type"`
	MediaID  string    `json:"mediaId,omitempty"`
	Link     string    `json:"link,omitempty"`
	Caption  string    `json:"caption,omitempty"`
	FileName string    `json:"fileName,omitempty"`
	Voice    bool      `json:"voice,omitempty"`
	ReplyTo  string    `json:"replyTo,omitempty"`
}

func mediaWire(in *MediaMessage) mediaWireBody {
	return mediaWireBody{
		To: in.To, Type: in.Type, MediaID: in.MediaID, Link: in.Link,
		Caption: in.Caption, FileName: in.FileName, Voice: in.Voice, ReplyTo: in.ReplyTo,
	}
}

type templateWireBody struct {
	To           string `json:"to"`
	Name         string `json:"name"`
	LanguageCode string `json:"languageCode"`
	Components   []any  `json:"components,omitempty"`
	ReplyTo      string `json:"replyTo,omitempty"`
}

func templateWire(in *TemplateMessage) templateWireBody {
	return templateWireBody{
		To: in.To, Name: in.Name, LanguageCode: in.LanguageCode,
		Components: in.Components, ReplyTo: in.ReplyTo,
	}
}

type interactiveWireBody struct {
	To                string                      `json:"to"`
	Type              InteractiveType             `json:"type"`
	BodyText          string                      `json:"bodyText,omitempty"`
	HeaderText        string                      `json:"headerText,omitempty"`
	FooterText        string                      `json:"footerText,omitempty"`
	Buttons           []InteractiveButton         `json:"buttons,omitempty"`
	ListButton        string                      `json:"listButton,omitempty"`
	Sections          []InteractiveListSection    `json:"sections,omitempty"`
	CTADisplayText    string                      `json:"ctaDisplayText,omitempty"`
	CTAUrl            string                      `json:"ctaUrl,omitempty"`
	Flow              *InteractiveFlow            `json:"flow,omitempty"`
	CatalogID         string                      `json:"catalogId,omitempty"`
	ProductRetailerID string                      `json:"productRetailerId,omitempty"`
	ProductSections   []InteractiveProductSection `json:"productSections,omitempty"`
	ReplyTo           string                      `json:"replyTo,omitempty"`
}

func interactiveWire(in *InteractiveMessage) interactiveWireBody {
	return interactiveWireBody{
		To: in.To, Type: in.Type, BodyText: in.BodyText, HeaderText: in.HeaderText,
		FooterText: in.FooterText, Buttons: in.Buttons, ListButton: in.ListButton,
		Sections: in.Sections, CTADisplayText: in.CTADisplayText, CTAUrl: in.CTAUrl,
		Flow: in.Flow, CatalogID: in.CatalogID, ProductRetailerID: in.ProductRetailerID,
		ProductSections: in.ProductSections, ReplyTo: in.ReplyTo,
	}
}

type reactionWireBody struct {
	To        string `json:"to"`
	MessageID string `json:"messageId"`
	// Emoji is always emitted (no omitempty) so "" can clear a reaction.
	Emoji string `json:"emoji"`
}

func reactionWire(in *ReactionMessage) reactionWireBody {
	return reactionWireBody{To: in.To, MessageID: in.MessageID, Emoji: in.Emoji}
}

type locationWireBody struct {
	To        string  `json:"to"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
	ReplyTo   string  `json:"replyTo,omitempty"`
}

func locationWire(in *LocationMessage) locationWireBody {
	return locationWireBody{
		To: in.To, Latitude: in.Latitude, Longitude: in.Longitude,
		Name: in.Name, Address: in.Address, ReplyTo: in.ReplyTo,
	}
}

type contactsWireBody struct {
	To       string        `json:"to"`
	Contacts []ContactCard `json:"contacts"`
	ReplyTo  string        `json:"replyTo,omitempty"`
}

func contactsWire(in *ContactsMessage) contactsWireBody {
	return contactsWireBody{To: in.To, Contacts: in.Contacts, ReplyTo: in.ReplyTo}
}

type carouselWireBody struct {
	To           string         `json:"to"`
	Name         string         `json:"name"`
	LanguageCode string         `json:"languageCode"`
	BodyParams   []string       `json:"bodyParams,omitempty"`
	Cards        []CarouselCard `json:"cards"`
	ReplyTo      string         `json:"replyTo,omitempty"`
}

func carouselWire(in *CarouselMessage) carouselWireBody {
	return carouselWireBody{
		To: in.To, Name: in.Name, LanguageCode: in.LanguageCode,
		BodyParams: in.BodyParams, Cards: in.Cards, ReplyTo: in.ReplyTo,
	}
}
