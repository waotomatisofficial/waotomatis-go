package waotomatis

import (
	"net/url"
	"strconv"
)

// TemplateResource manages WhatsApp message templates on a session's WABA.
// Obtain it via Client.Sessions(id).Templates.
//
//	tpls := client.Sessions("sess_123").Templates
//	list, _ := tpls.List(nil)
//	tpls.Create(&waotomatis.CreateTemplateInput{ ... })
type TemplateResource struct {
	client    *Client
	sessionID string
}

func (r *TemplateResource) base() string {
	return "/v1/sessions/" + url.PathEscape(r.sessionID) + "/templates"
}

// List returns message templates on this session's WABA. Pass &TemplateListParams{}
// (or nil) for the first page unfiltered; follow the cursor in TemplateList.Paging
// (paging.cursors.after) via params.After for the next page.
func (r *TemplateResource) List(params *TemplateListParams, opts ...CallOption) (*TemplateList, error) {
	var out TemplateList
	err := r.client.doJSON(request{method: "GET", path: r.base(), query: templateListQuery(params), opts: opts}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Get returns every language version of the template matching the exact name.
func (r *TemplateResource) Get(name string, opts ...CallOption) (*TemplateList, error) {
	var out TemplateList
	err := r.client.doJSON(request{method: "GET", path: r.base() + "/" + url.PathEscape(name), opts: opts}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Create submits a new template to Meta for approval. Approval is asynchronous —
// poll List / Get and watch Template.Status (PENDING → APPROVED/REJECTED).
func (r *TemplateResource) Create(in *CreateTemplateInput, opts ...CallOption) (*TemplateCreateResult, error) {
	var out TemplateCreateResult
	err := r.client.doJSON(request{method: "POST", path: r.base(), body: in, opts: opts}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete removes a template by name (all of its language versions).
func (r *TemplateResource) Delete(name string, opts ...CallOption) (*TemplateDeleteResult, error) {
	var out TemplateDeleteResult
	err := r.client.doJSON(request{method: "DELETE", path: r.base() + "/" + url.PathEscape(name), opts: opts}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// templateListQuery builds the query map for Templates.List.
func templateListQuery(p *TemplateListParams) map[string]string {
	if p == nil {
		return nil
	}
	q := map[string]string{}
	if p.Limit > 0 {
		q["limit"] = strconv.Itoa(p.Limit)
	}
	if p.After != "" {
		q["after"] = p.After
	}
	if p.Name != "" {
		q["name"] = p.Name
	}
	if p.Language != "" {
		q["language"] = p.Language
	}
	if p.Status != "" {
		q["status"] = p.Status
	}
	if p.Category != "" {
		q["category"] = p.Category
	}
	return q
}
