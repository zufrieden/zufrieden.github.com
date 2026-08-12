package mymind

import (
	"cmp"
	"encoding/json"
	"path"
	"strings"
	"time"
)

// Object is a mymind saved item. Only the fields this tool needs are mapped;
// the API returns quite a few more.
type Object struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Summary  string   `json:"summary"`
	URL      string   `json:"url"`
	Source   *Source  `json:"source"`
	Content  *Content `json:"content"`
	Blob     *Blob    `json:"blob"`
	Tags     []Tag    `json:"tags"`
	Notes    []Note   `json:"notes"`
	Created  string   `json:"created"`
	Modified string   `json:"modified"`
}

type Source struct {
	URL string `json:"url"`
}

// Blob is the file an object was made from — the image or PDF that was
// uploaded, as opposed to a bookmarked page, which has a source URL instead.
// Its bytes are fetched separately, from GET /objects/:id/blob.
//
// https://access.mymind.com/api/types#blobreference
type Blob struct {
	// Path is the location under https://mymind.media.
	Path string `json:"path"`
	// Type is the MIME type, e.g. "image/jpeg" or "application/pdf".
	Type string `json:"type"`
	// Name is the filename it was uploaded under, when mymind kept one.
	Name string `json:"name"`
	// URL is the fully-qualified location, when the API includes one.
	URL string `json:"url"`
	// Width and Height are the pixel dimensions of an image.
	Width  int `json:"width"`
	Height int `json:"height"`
}

// UnmarshalJSON also accepts `mimeType` for the MIME type. The API is in beta
// and its own examples disagree with its type reference on this key; reading
// both means a file is still recognised as an image either way.
func (b *Blob) UnmarshalJSON(data []byte) error {
	type alias Blob
	var raw struct {
		alias
		MimeType    string `json:"mimeType"`
		ContentType string `json:"contentType"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*b = Blob(raw.alias)
	if b.Type == "" {
		b.Type = cmp.Or(raw.MimeType, raw.ContentType)
	}
	return nil
}

// MediaType is the blob's MIME type, lowercased and without the parameters a
// header can carry ("image/jpeg; charset=binary").
func (b *Blob) MediaType() string {
	if b == nil {
		return ""
	}
	base, _, _ := strings.Cut(b.Type, ";")
	return strings.ToLower(strings.TrimSpace(base))
}

// UploadedName is the name the file was uploaded under, or "" when mymind kept
// none. Directory components are stripped, but it is still only a suggestion:
// the caller decides what is safe to write to disk.
//
// The blob's `path` is deliberately not a fallback — it is an opaque key on
// mymind's media host, which makes a poor filename and a worse link label.
func (b *Blob) UploadedName() string {
	if b == nil {
		return ""
	}
	name := strings.TrimSpace(b.Name)
	if name == "" {
		return ""
	}
	return path.Base(name)
}

// HasBlob reports whether the object carries a file of its own.
func (o Object) HasBlob() bool { return o.Blob != nil }

type Tag struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

// Content is normally {"type": ..., "body": ...} but tolerates a bare string.
type Content struct {
	Type string `json:"type,omitempty"`
	Body string `json:"body,omitempty"`
}

func (c *Content) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		c.Body = s
		return nil
	}
	type alias Content
	var a alias
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	*c = Content(a)
	return nil
}

// Note is normally {"id": ..., "content": {...}} but tolerates a bare string
// or a flat {"text": ...} / {"body": ...} shape.
type Note struct {
	ID      string  `json:"id,omitempty"`
	Content Content `json:"content"`
}

func (n *Note) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		n.Content = Content{Body: s}
		return nil
	}
	var raw struct {
		ID      string   `json:"id"`
		Content *Content `json:"content"`
		Text    string   `json:"text"`
		Body    string   `json:"body"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	n.ID = raw.ID
	switch {
	case raw.Content != nil:
		n.Content = *raw.Content
	case raw.Text != "":
		n.Content = Content{Body: raw.Text}
	case raw.Body != "":
		n.Content = Content{Body: raw.Body}
	}
	return nil
}

// SourceURL returns the object's originating URL, preferring source.url.
func (o Object) SourceURL() string {
	if o.Source != nil && strings.TrimSpace(o.Source.URL) != "" {
		return strings.TrimSpace(o.Source.URL)
	}
	return strings.TrimSpace(o.URL)
}

// CreatedAt parses the object's `created` timestamp, which the API documents as
// an RFC 3339 instant in UTC (https://access.mymind.com/api/types#timestamp).
// The second return value is false when it is missing or unparseable, so
// callers can fall back rather than publishing a zero date.
func (o Object) CreatedAt() (time.Time, bool) {
	trimmed := strings.TrimSpace(o.Created)
	if trimmed == "" {
		return time.Time{}, false
	}
	// RFC3339 also accepts fractional seconds, so it covers the nano variant.
	t, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// FirstNote returns the body of the first attached note, if any. The mymind
// app itself only ever shows the first one.
func (o Object) FirstNote() string {
	for _, n := range o.Notes {
		if b := strings.TrimSpace(n.Content.Body); b != "" {
			return b
		}
	}
	return ""
}

// HasTag reports whether the object carries the given tag, case-insensitively.
func (o Object) HasTag(name string) bool {
	for _, t := range o.Tags {
		if strings.EqualFold(strings.TrimSpace(t.Name), strings.TrimSpace(name)) {
			return true
		}
	}
	return false
}

// TagNames lists the object's tags, for logging.
func (o Object) TagNames() []string {
	names := make([]string, 0, len(o.Tags))
	for _, t := range o.Tags {
		names = append(names, t.Name)
	}
	return names
}
