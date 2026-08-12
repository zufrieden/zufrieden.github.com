package publisher

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zufrieden/zufrieden.github.com/tools/mymind-sync/internal/mymind"
)

func TestAssetForReportsNothingWithoutABlob(t *testing.T) {
	if asset := assetFor(mymind.Object{Title: "A bookmark", Source: &mymind.Source{URL: "https://example.com"}}); asset != nil {
		t.Errorf("assetFor = %+v, want nil for an object with no file", asset)
	}
}

// What gets written to disk is derived from the uploaded name, never taken
// from it: the name is user input, and it ends up in a path.
func TestAssetFilename(t *testing.T) {
	cases := []struct {
		name string
		obj  mymind.Object
		want string
	}{
		{
			name: "slugified, with the extension the type implies",
			obj:  mymind.Object{Blob: &mymind.Blob{Name: "Vers une Société des Communs.pdf", Type: "application/pdf"}},
			want: "vers_une_societe_des_communs.pdf",
		},
		{
			name: "jpeg keeps its conventional extension",
			obj:  mymind.Object{Blob: &mymind.Blob{Name: "IMG_4821.JPEG", Type: "image/jpeg"}},
			want: "img_4821.jpg",
		},
		{
			name: "no uploaded name falls back to the title",
			obj:  mymind.Object{Title: "Sunset in Oulu", Blob: &mymind.Blob{Type: "image/png"}},
			want: "sunset_in_oulu.png",
		},
		{
			name: "no name and no title still produces something writable",
			obj:  mymind.Object{Blob: &mymind.Blob{Type: "image/webp"}},
			want: "untitled.webp",
		},
		{
			name: "a directory in the name cannot escape the bundle",
			obj:  mymind.Object{Blob: &mymind.Blob{Name: "../../../etc/passwd", Type: "image/png"}},
			want: "passwd.png",
		},
		{
			name: "an unusable subtype falls back to the uploaded extension",
			obj:  mymind.Object{Blob: &mymind.Blob{Name: "diagram.svg", Type: "application/octet-stream"}},
			want: "diagram.svg",
		},
		{
			name: "no type at all still writes the file, without an extension",
			obj:  mymind.Object{Title: "Mystery", Blob: &mymind.Blob{}},
			want: "mystery",
		},
		{
			name: "mimeType is read as well as type",
			obj:  mymind.Object{Title: "Doc", Blob: blobFromJSON(t, `{"mimeType":"application/pdf"}`)},
			want: "doc.pdf",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			asset := assetFor(tc.obj)
			if asset == nil {
				t.Fatal("assetFor returned nil")
			}
			if got := asset.Filename(); got != tc.want {
				t.Errorf("Filename() = %q, want %q", got, tc.want)
			}
			if strings.ContainsAny(asset.Filename(), `/\`) {
				t.Errorf("Filename() %q must stay inside the bundle", asset.Filename())
			}
		})
	}
}

// Images are shown, everything else is linked. The reference stays relative:
// that is what makes it a page resource Hugo can resolve and process.
func TestAssetMarkdown(t *testing.T) {
	image := assetFor(mymind.Object{Blob: &mymind.Blob{Name: "photo.jpg", Type: "image/jpeg"}})
	if got, want := image.Markdown("A [bracketed] title"), `![A \[bracketed\] title](photo.jpg)`; got != want {
		t.Errorf("image markdown = %q, want %q", got, want)
	}

	pdf := assetFor(mymind.Object{Blob: &mymind.Blob{Name: "paper.pdf", Type: "application/pdf"}})
	if got, want := pdf.Markdown("ignored"), "[paper.pdf](paper.pdf)"; got != want {
		t.Errorf("pdf markdown = %q, want %q", got, want)
	}

	// No uploaded name: the saved filename is the only label there is.
	unnamed := assetFor(mymind.Object{Title: "A talk", Blob: &mymind.Blob{Type: "application/pdf"}})
	if got, want := unnamed.Markdown("ignored"), "[a_talk.pdf](a_talk.pdf)"; got != want {
		t.Errorf("unnamed markdown = %q, want %q", got, want)
	}

	var missing *Asset
	if got := missing.Markdown("nothing"); got != "" {
		t.Errorf("a nil asset should render nothing, got %q", got)
	}
}

func TestWithTypeResolvesTheMediaType(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 32))

	untyped := &Asset{stem: "thing"}
	if got := untyped.withType("image/png; charset=binary", png).Type; got != "image/png" {
		t.Errorf("Type = %q, want image/png with the parameters dropped", got)
	}

	// The listing's type wins: it describes the file, the header only describes
	// the response, and a CDN answering "application/octet-stream" must not turn
	// an image into a link.
	typed := &Asset{Type: "image/jpeg", stem: "thing"}
	if got := typed.withType("application/octet-stream", png).Type; got != "image/jpeg" {
		t.Errorf("Type = %q, want the object's own image/jpeg", got)
	}

	// Neither the object nor the header says anything useful — which is exactly
	// what mymind's media host does — so the bytes decide.
	if got := untyped.withType("application/octet-stream", png).Type; got != "image/png" {
		t.Errorf("Type = %q, want image/png sniffed from the bytes", got)
	}
	pdf := []byte("%PDF-1.7\n" + strings.Repeat("x", 32))
	if got := untyped.withType("", pdf).Type; got != "application/pdf" {
		t.Errorf("Type = %q, want application/pdf sniffed from the bytes", got)
	}

	// Nothing recognisable anywhere: the file is still saved, just without a
	// type to make claims about.
	if got := untyped.withType("   ", []byte{0x01, 0x02, 0x03}).Type; got != "" {
		t.Errorf("Type = %q, want it left empty", got)
	}
}

// blobFromJSON decodes a blob the way a response would, so the test exercises
// the lenient key handling rather than setting the field directly.
func blobFromJSON(t *testing.T, raw string) *mymind.Blob {
	t.Helper()
	var blob mymind.Blob
	if err := json.Unmarshal([]byte(raw), &blob); err != nil {
		t.Fatal(err)
	}
	return &blob
}
