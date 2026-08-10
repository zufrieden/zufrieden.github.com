package mymind

import (
	"encoding/json"
	"strings"
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
	Tags     []Tag    `json:"tags"`
	Notes    []Note   `json:"notes"`
	Created  string   `json:"created"`
	Modified string   `json:"modified"`
}

type Source struct {
	URL string `json:"url"`
}

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
