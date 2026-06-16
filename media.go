package waotomatis

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// MediaResource uploads media for a session and downloads inbound media.
// Obtain it via Client.Sessions(id).Media.
type MediaResource struct {
	client    *Client
	sessionID string
}

// UploadOptions tune a raw-bytes upload.
type UploadOptions struct {
	// FileName is the multipart filename (default "upload").
	FileName string
	// MimeType is the part's Content-Type (default "application/octet-stream").
	MimeType string
}

func (r *MediaResource) mediaPath() string {
	return "/v1/sessions/" + url.PathEscape(r.sessionID) + "/media"
}

// Upload uploads raw bytes as multipart (field "file") and returns a MediaID you
// can pass to Message.MediaID. Multipart uploads are never auto-retried.
func (r *MediaResource) Upload(data []byte, o *UploadOptions, opts ...CallOption) (*UploadMediaResult, error) {
	return r.upload(bytes.NewReader(data), o, opts...)
}

// UploadReader uploads from an io.Reader (e.g. a streaming source) as multipart.
func (r *MediaResource) UploadReader(src io.Reader, o *UploadOptions, opts ...CallOption) (*UploadMediaResult, error) {
	return r.upload(src, o, opts...)
}

// UploadFile reads a file from disk and uploads it. The on-disk base name is
// used as the multipart filename unless o.FileName overrides it.
func (r *MediaResource) UploadFile(path string, o *UploadOptions, opts ...CallOption) (*UploadMediaResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, newConnectionError("failed to open file: " + err.Error())
	}
	defer f.Close()
	if o == nil {
		o = &UploadOptions{}
	}
	if o.FileName == "" {
		o.FileName = filepath.Base(path)
	}
	return r.upload(f, o, opts...)
}

func (r *MediaResource) upload(src io.Reader, o *UploadOptions, opts ...CallOption) (*UploadMediaResult, error) {
	if o == nil {
		o = &UploadOptions{}
	}
	fileName := o.FileName
	if fileName == "" {
		fileName = "upload"
	}
	mimeType := o.MimeType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+escapeQuotes(fileName)+`"`)
	h.Set("Content-Type", mimeType)
	part, err := w.CreatePart(h)
	if err != nil {
		return nil, newConnectionError("failed to build multipart body: " + err.Error())
	}
	if _, err := io.Copy(part, src); err != nil {
		return nil, newConnectionError("failed to read upload data: " + err.Error())
	}
	if err := w.Close(); err != nil {
		return nil, newConnectionError("failed to finalize multipart body: " + err.Error())
	}

	var out UploadMediaResult
	err = r.client.doJSON(request{
		method:         "POST",
		path:           r.mediaPath(),
		rawBody:        buf.Bytes(),
		rawContentType: w.FormDataContentType(),
		opts:           opts,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UploadFromURL tells the server to fetch media from a public URL and returns a
// MediaID. mimeType is optional (pass "" to let the server infer it).
func (r *MediaResource) UploadFromURL(mediaURL, mimeType string, opts ...CallOption) (*UploadMediaResult, error) {
	body := map[string]string{"url": mediaURL}
	if mimeType != "" {
		body["mimeType"] = mimeType
	}
	var out UploadMediaResult
	err := r.client.doJSON(request{method: "POST", path: r.mediaPath(), body: body, opts: opts}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Download fetches inbound media bytes by provider media id.
func (r *MediaResource) Download(mediaID string, opts ...CallOption) (*MediaDownload, error) {
	res, err := r.client.do(request{
		method: "GET",
		path:   r.mediaPath() + "/" + url.PathEscape(mediaID),
		opts:   opts,
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, readErr := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, parseErrorBody(res, body)
	}
	if readErr != nil {
		return nil, newConnectionError("failed reading media body: " + readErr.Error())
	}
	mimeType := res.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return &MediaDownload{Data: body, MimeType: mimeType}, nil
}

func escapeQuotes(s string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(s)
}
