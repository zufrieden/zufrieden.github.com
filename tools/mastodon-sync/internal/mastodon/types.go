package mastodon

import (
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

// Post is one status from the account's RSS feed.
//
// A Mastodon post has no title — the text *is* the content — so there is no
// Title field to map. Callers derive one.
type Post struct {
	// ID is the status id, taken from the guid permalink. It never changes,
	// including when the post is edited, which is what makes it usable as the
	// "have I published this already?" key.
	ID string
	// URL is the permalink to the post on the instance.
	URL string
	// Published is when the post went out.
	Published time.Time
	// HTML is the status body as Mastodon renders it: <p>, <br>, and <a> with
	// the link text split across <span>s.
	HTML string
	// Media lists the attachments, in the order the feed gives them.
	Media []Media
}

// Media is one attachment on a post.
type Media struct {
	// URL points at the instance's media CDN.
	URL string
	// Type is the MIME type, e.g. "image/jpeg". The feed does not always set it.
	Type string
	// Medium is the feed's coarse category, e.g. "image" or "video".
	Medium string
}

// Filename is the name the attachment should be saved under: its position in
// the post plus an extension inferred from the URL, falling back to the MIME
// type. Mastodon's own filenames are content hashes, which say nothing useful.
func (m Media) Filename(position int) string {
	name := strconv.Itoa(position)
	if ext := m.extension(); ext != "" {
		name += ext
	}
	return name
}

func (m Media) extension() string {
	if parsed, err := url.Parse(m.URL); err == nil {
		if ext := path.Ext(parsed.Path); isSafeExtension(ext) {
			return strings.ToLower(ext)
		}
	}
	// The URL had nothing usable; fall back to the MIME subtype.
	_, subtype, ok := strings.Cut(m.Type, "/")
	if !ok {
		return ""
	}
	if ext := "." + strings.ToLower(strings.TrimSpace(subtype)); isSafeExtension(ext) {
		return ext
	}
	return ""
}

// isSafeExtension keeps the extension to plain lowercase/uppercase letters and
// digits. It is what stops a hostile or malformed media URL from steering the
// saved filename somewhere it should not go.
func isSafeExtension(ext string) bool {
	if len(ext) < 2 || len(ext) > 6 || ext[0] != '.' {
		return false
	}
	for _, r := range ext[1:] {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

// statusID pulls the status id out of a permalink, e.g.
// "https://social.coop/@zufrieden/117066375180847363" -> "117066375180847363".
func statusID(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	id := path.Base(strings.TrimSuffix(parsed.Path, "/"))
	if id == "." || id == "/" || id == "" {
		return ""
	}
	// The id ends up in a file path, so refuse anything that is not the plain
	// numeric id Mastodon issues.
	for _, r := range id {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return id
}
