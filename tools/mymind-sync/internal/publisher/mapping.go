package publisher

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/zufrieden/zufrieden.github.com/tools/mymind-sync/internal/hugosite"
	"github.com/zufrieden/zufrieden.github.com/tools/mymind-sync/internal/mymind"
)

// Page is the content this tool wants to end up on the site. Title, Date and
// Description land in the front matter; Body is the markdown underneath it.
type Page struct {
	Title       string
	Date        time.Time
	Description string
	Body        string
	// Extra carries section-specific front matter (the `linking` archetype's
	// `link:` key, for instance).
	Extra []hugosite.Field
}

// Mapping describes how one Hugo section is fed from mymind: which tag marks
// an object for publication, which tag marks it as done, and how an object
// turns into a page.
//
// Adding a new content type is a matter of registering another Mapping.
type Mapping struct {
	// Name is what you pass to -type on the command line.
	Name string
	// Section is the Hugo content section, e.g. "sharing" for content/sharing/.
	Section string
	// SourceTag selects objects to publish.
	SourceTag string
	// DoneTag is written back to mymind once the page exists.
	DoneTag string
	// Build converts an object into a page. Returning an error skips the
	// object without tagging it, so it gets retried on the next run.
	Build func(obj mymind.Object, now time.Time) (Page, error)
}

// SearchQuery is the mymind search that selects unpublished objects.
func (m Mapping) SearchQuery() string {
	return fmt.Sprintf("tag:%s -tag:%s", m.SourceTag, m.DoneTag)
}

var registry = map[string]Mapping{
	sharing.Name: sharing,
}

// Lookup finds a registered mapping by name.
func Lookup(name string) (Mapping, error) {
	m, ok := registry[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return Mapping{}, fmt.Errorf("unknown content type %q (known: %s)", name, strings.Join(Names(), ", "))
	}
	return m, nil
}

// Names lists the registered content types.
func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// sharing maps a mymind object onto content/sharing/:
//
//	title       <- object title
//	date        <- the day it is shared (today)
//	description <- first note, falling back to the AI summary
//	body        <- the source URL as a markdown link
var sharing = Mapping{
	Name:      "sharing",
	Section:   "sharing",
	SourceTag: "share",
	DoneTag:   "shared",
	Build: func(obj mymind.Object, now time.Time) (Page, error) {
		// A sharing page is a link, so objects saved as a PDF, image, file or
		// plain note have nothing to point at. Skip them rather than
		// publishing an empty page — and leave them untagged so they get
		// picked up again if a URL is added later.
		sourceURL := obj.SourceURL()
		if sourceURL == "" {
			return Page{}, fmt.Errorf("no link attached (source.url is empty)")
		}

		description := obj.FirstNote()
		if description == "" {
			description = strings.TrimSpace(obj.Summary)
		}

		return Page{
			Title:       titleFor(obj, sourceURL),
			Date:        now,
			Description: singleLine(description),
			Body:        markdownLink(sourceURL),
		}, nil
	},
}

func titleFor(obj mymind.Object, sourceURL string) string {
	if title := singleLine(obj.Title); title != "" {
		return title
	}
	if parsed, err := url.Parse(sourceURL); err == nil && parsed.Host != "" {
		return strings.TrimPrefix(parsed.Host, "www.")
	}
	return "Untitled"
}

// markdownLink renders the URL as its own label, matching the existing pages.
func markdownLink(rawURL string) string {
	return "[" + rawURL + "](" + rawURL + ")"
}

// singleLine flattens whitespace so a multi-paragraph note still fits on one
// front matter line.
func singleLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
