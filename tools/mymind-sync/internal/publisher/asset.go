package publisher

import (
	"net/http"
	"path"
	"strings"

	"github.com/zufrieden/zufrieden.github.com/tools/internal/hugosite"
	"github.com/zufrieden/zufrieden.github.com/tools/mymind-sync/internal/mymind"
)

// Asset is the file an object carries, saved into the page's bundle so Hugo
// sees it as a page resource. That is the whole reason these pages are bundles
// rather than plain files: image processing only works on resources, and
// nothing under static/ is one.
type Asset struct {
	// Type is the MIME type, e.g. "image/jpeg" or "application/pdf".
	Type string
	// Label is the name the file was uploaded under, used as the link text for
	// anything that is not an image. It may be empty.
	Label string
	// stem is the slugified basename, without extension.
	stem string
}

// assetFor describes the file to save for an object, or nil when there is
// nothing attached — a bookmarked page, or a plain note.
func assetFor(obj mymind.Object) *Asset {
	if !obj.HasBlob() {
		return nil
	}

	// The name it was uploaded under, falling back to the object's title for a
	// file mymind kept no name for. Slugify handles the rest, including the
	// length cap and the empty case.
	label := obj.Blob.UploadedName()
	named := strings.TrimSuffix(label, path.Ext(label))
	if named == "" {
		named = obj.Title
	}

	return &Asset{Type: obj.Blob.MediaType(), Label: label, stem: hugosite.Slugify(named)}
}

// IsImage reports whether the file should be shown rather than linked.
func (a *Asset) IsImage() bool {
	return a != nil && strings.HasPrefix(a.Type, "image/")
}

// Filename is what the file is saved as inside the bundle. The extension comes
// from the MIME type rather than from the uploaded name, because the type is
// what the server actually served and the name is user input.
func (a *Asset) Filename() string {
	if a == nil {
		return ""
	}
	return a.stem + extensionFor(a.Type, a.Label)
}

// Markdown is how the page refers to the file: images are shown, everything
// else is a link labelled with the name it was uploaded under. The reference is
// relative, which is what makes it resolvable as a page resource — and which
// the image render hook then turns into a resized, responsive <img>.
func (a *Asset) Markdown(alt string) string {
	if a == nil {
		return ""
	}
	if a.IsImage() {
		return "![" + escapeLabel(alt) + "](" + a.Filename() + ")"
	}
	label := a.Label
	if label == "" {
		label = a.Filename()
	}
	return "[" + escapeLabel(label) + "](" + a.Filename() + ")"
}

// withType settles the MIME type once the file is in hand, because the type is
// what decides the extension and whether the file is shown or linked.
//
// The object's own type wins, then the download's header, and failing both the
// bytes are sniffed. That last step earns its place: mymind's media host serves
// uploads as "application/octet-stream", so an object the listing gave no type
// for would otherwise leave a perfectly good image looking like an anonymous
// file to link rather than show.
func (a *Asset) withType(contentType string, data []byte) *Asset {
	if a == nil || isUsefulType(a.Type) {
		return a
	}

	// A response header carries parameters the type itself does not:
	// "image/png; charset=binary".
	header, _, _ := strings.Cut(contentType, ";")
	resolved := strings.ToLower(strings.TrimSpace(header))
	if !isUsefulType(resolved) {
		sniffed, _, _ := strings.Cut(http.DetectContentType(data), ";")
		resolved = strings.ToLower(strings.TrimSpace(sniffed))
	}
	if !isUsefulType(resolved) {
		return a
	}

	updated := *a
	updated.Type = resolved
	return &updated
}

// isUsefulType rejects the types that say nothing about the file.
// "application/octet-stream" is what a CDN answers when it has not been told.
func isUsefulType(mediaType string) bool {
	switch mediaType {
	case "", "application/octet-stream", "binary/octet-stream":
		return false
	}
	return strings.Contains(mediaType, "/")
}

// knownExtensions maps the MIME types whose extension is not simply the
// subtype: the ones where the subtype is unusable ("svg+xml") or where the
// conventional extension differs ("jpeg").
var knownExtensions = map[string]string{
	"image/jpeg":      ".jpg",
	"image/svg+xml":   ".svg",
	"application/pdf": ".pdf",
	"text/plain":      ".txt",
	"video/quicktime": ".mov",
}

// extensionFor picks the file extension: the MIME type's conventional one, then
// the subtype, then whatever the uploaded name ended in. It returns "" rather
// than guess, so a file with no usable type is at least saved under a name that
// cannot mislead.
func extensionFor(mediaType, label string) string {
	if ext, ok := knownExtensions[mediaType]; ok {
		return ext
	}
	if _, subtype, ok := strings.Cut(mediaType, "/"); ok {
		if ext := "." + strings.ToLower(strings.TrimSpace(subtype)); isSafeExtension(ext) {
			return ext
		}
	}
	if ext := strings.ToLower(path.Ext(label)); isSafeExtension(ext) {
		return ext
	}
	return ""
}

// isSafeExtension keeps the extension to plain letters and digits. It is what
// stops a hostile or malformed filename from steering the saved file somewhere
// it should not go.
func isSafeExtension(ext string) bool {
	if len(ext) < 2 || len(ext) > 6 || ext[0] != '.' {
		return false
	}
	for _, r := range ext[1:] {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

// escapeLabel keeps brackets in a label from closing a markdown link early.
func escapeLabel(label string) string {
	return strings.NewReplacer(`[`, `\[`, `]`, `\]`).Replace(label)
}
