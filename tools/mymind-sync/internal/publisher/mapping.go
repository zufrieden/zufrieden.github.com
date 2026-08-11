package publisher

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/zufrieden/zufrieden.github.com/tools/internal/hugosite"
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
	// Filename builds the page's filename (no directory). Sections do not
	// agree: sharing prefixes the date, linking does not.
	Filename func(page Page) string
	// Build converts an object into a page. now is the run time, used only as
	// the fallback date for an object mymind gave no `created` timestamp for.
	// Returning an error skips the object without tagging it, so it gets
	// retried on the next run.
	Build func(obj mymind.Object, now time.Time) (Page, error)
}

// SearchQuery is the mymind search that selects unpublished objects.
func (m Mapping) SearchQuery() string {
	return fmt.Sprintf("tag:%s -tag:%s", m.SourceTag, m.DoneTag)
}

var registry = map[string]Mapping{
	sharing.Name: sharing,
	linking.Name: linking,
}

// Lookup finds a registered mapping by name.
func Lookup(name string) (Mapping, error) {
	m, ok := registry[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return Mapping{}, fmt.Errorf("unknown content type %q (known: %s)", name, strings.Join(Names(), ", "))
	}
	return m, nil
}

// LookupAll resolves a comma-separated list of content types, preserving the
// order given and ignoring repeats.
func LookupAll(names string) ([]Mapping, error) {
	var mappings []Mapping
	seen := map[string]bool{}
	for _, name := range strings.Split(names, ",") {
		if strings.TrimSpace(name) == "" {
			continue
		}
		m, err := Lookup(name)
		if err != nil {
			return nil, err
		}
		if seen[m.Name] {
			continue
		}
		seen[m.Name] = true
		mappings = append(mappings, m)
	}
	if len(mappings) == 0 {
		return nil, fmt.Errorf("no content type given (known: %s)", strings.Join(Names(), ", "))
	}
	return mappings, nil
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
//	date        <- the day it was saved in mymind (object.created)
//	description <- first note, falling back to the AI summary
//	body        <- the source URL as a markdown link
var sharing = Mapping{
	Name:      "sharing",
	Section:   "sharing",
	SourceTag: "share",
	DoneTag:   "shared",
	Filename: func(page Page) string {
		return page.Date.Format("20060102") + "_" + hugosite.Slugify(page.Title) + ".md"
	},
	Build: func(obj mymind.Object, now time.Time) (Page, error) {
		// A sharing page is a link, so objects saved as a PDF, image, file or
		// plain note have nothing to point at. Skip them rather than
		// publishing an empty page — and leave them untagged so they get
		// picked up again if a URL is added later.
		sourceURL := obj.SourceURL()
		if sourceURL == "" {
			return Page{}, fmt.Errorf("no link attached (source.url is empty)")
		}

		// A sharing page carries no tags, so no entity has a tag page to
		// point at, and sharing/single.html renders the description escaped:
		// plain text is what belongs here.
		return Page{
			Title:       titleFor(obj, sourceURL),
			Date:        pageDate(obj, now),
			Description: plainDescription(obj),
			Body:        markdownLink(sourceURL),
		}, nil
	},
}

// linking maps a mymind object onto content/linking/:
//
//	title       <- object title
//	date        <- the day it was saved in mymind (object.created)
//	tags        <- the object's mymind tags
//	link        <- source.url, in the front matter rather than the body
//	description <- first note, falling back to the AI summary
//
// Unlike sharing, the filename carries no date prefix — matching the existing
// pages in content/linking/ — and the body is left empty, because the whole
// point of a linking page is the `link:` key.
var linking = Mapping{
	Name:      "linking",
	Section:   "linking",
	SourceTag: "keep",
	DoneTag:   "keeped",
	Filename: func(page Page) string {
		return hugosite.Slugify(page.Title) + ".md"
	},
	Build: func(obj mymind.Object, now time.Time) (Page, error) {
		sourceURL := obj.SourceURL()
		if sourceURL == "" {
			return Page{}, fmt.Errorf("no link attached (source.url is empty)")
		}

		tags := tagsFor(obj, "keeped")

		return Page{
			Title:       titleFor(obj, sourceURL),
			Date:        pageDate(obj, now),
			Description: htmlDescription(obj, tags),
			Extra: []hugosite.Field{
				{Key: "tags", Value: hugosite.YAMLStringSlice(tags)},
				{Key: "link", Value: hugosite.YAMLString(sourceURL)},
			},
		}, nil
	},
}

// pageDate is when the page says it happened: the moment the object was saved
// in mymind, not the moment this tool got around to publishing it. The two
// differ whenever a run is late, and it is the save date that the reader cares
// about.
//
// The timestamp comes back in UTC, so it is moved into the run's timezone
// before anyone reads a day off it — otherwise anything saved between midnight
// and 02:00 Zurich time would be dated the day before in the filename prefix.
// An object with no usable `created` falls back to the run time.
func pageDate(obj mymind.Object, now time.Time) time.Time {
	created, ok := obj.CreatedAt()
	if !ok {
		return now
	}
	return created.In(now.Location())
}

// tagsFor lists the object's mymind tags for the front matter, dropping the
// bookkeeping tag this tool adds after publishing. The trigger tag is kept —
// the existing linking pages carry it.
func tagsFor(obj mymind.Object, exclude string) []string {
	tags := make([]string, 0, len(obj.Tags))
	for _, name := range obj.TagNames() {
		name = strings.TrimSpace(name)
		if name == "" || strings.EqualFold(name, exclude) {
			continue
		}
		tags = append(tags, name)
	}
	return tags
}

// wikiLink matches mymind's entity markup, e.g. "[[Artificial Intelligence]]"
// or "[[target|label]]".
var wikiLink = regexp.MustCompile(`\[\[([^\[\]]*)\]\]`)

// rawDescription is the source text for every section: the first note,
// falling back to the AI summary.
func rawDescription(obj mymind.Object) string {
	if note := obj.FirstNote(); note != "" {
		return note
	}
	return obj.Summary
}

// wikiLabel is the text a "[[…]]" match should display.
func wikiLabel(match string) string {
	label := strings.TrimSpace(match[2 : len(match)-2])
	// "[[target|label]]" displays the label.
	if _, piped, ok := strings.Cut(label, "|"); ok {
		label = strings.TrimSpace(piped)
	}
	return label
}

// plainDescription strips mymind's entity markup and returns plain text.
//
// For sections whose template renders `{{ .Description }}` unmodified — Go
// templates escape that, so the text must NOT be pre-escaped or it would come
// out double-escaped.
func plainDescription(obj mymind.Object) string {
	return singleLine(wikiLink.ReplaceAllStringFunc(rawDescription(obj), wikiLabel))
}

// htmlDescription escapes mymind's text and turns entities matching one of
// tags into links to their tag page.
//
// For sections whose template renders `{{ .Description | safeHTML }}`.
// Matching against the page's *own* tags is what keeps the links honest: Hugo
// only generates /tags/x/ for terms some page actually carries, so linking a
// bare entity like "[[Large Language Models]]" would 404.
func htmlDescription(obj mymind.Object, tags []string) string {
	return singleLine(resolveWikiLinks(rawDescription(obj), tags))
}

// resolveWikiLinks escapes everything that came from mymind and splices in
// anchors for the entities that have a tag page. Escaping is not optional:
// the description is rendered with safeHTML, so unescaped input would go
// straight into the page.
func resolveWikiLinks(raw string, linkableTags []string) string {
	slugs := make(map[string]string, len(linkableTags))
	for _, tag := range linkableTags {
		if slug := hugosite.URLize(tag); slug != "" {
			slugs[slug] = slug
		}
	}

	var b strings.Builder
	end := 0
	for _, m := range wikiLink.FindAllStringSubmatchIndex(raw, -1) {
		b.WriteString(html.EscapeString(raw[end:m[0]]))
		end = m[1]

		label := wikiLabel(raw[m[0]:m[1]])
		if label == "" {
			continue
		}
		if slug, ok := slugs[hugosite.URLize(label)]; ok {
			b.WriteString(`<a href="/tags/` + slug + `">` + html.EscapeString(label) + `</a>`)
			continue
		}
		b.WriteString(html.EscapeString(label))
	}
	b.WriteString(html.EscapeString(raw[end:]))
	return b.String()
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
