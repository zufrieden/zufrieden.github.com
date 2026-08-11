package publisher

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/zufrieden/zufrieden.github.com/tools/internal/hugosite"
	"github.com/zufrieden/zufrieden.github.com/tools/mastodon-sync/internal/mastodon"
)

const (
	// Section is the Hugo content section these pages land in.
	Section = "sharing"

	// FrontMatterKey records the Mastodon status id on the generated page. It
	// is what makes a run idempotent: the published pages are the record of
	// what has been published, so there is no state file to keep in step.
	//
	// The name mirrors the `tweet_id` the imported tweets in content/sharing
	// already carry.
	FrontMatterKey = "mastodon_id"

	// URLKey holds the permalink, so the page can point back at the post.
	URLKey = "mastodon_url"

	// mediaDir is where attachments are saved, keyed by status id. Under
	// static/, so Hugo serves it from the site root.
	mediaDir = "static/images/mastodon"

	// maxTitleLen caps the title derived from the post text. Long enough to
	// carry a whole short post, short enough to work as a heading and a link
	// label.
	maxTitleLen = 70
)

// Page is what a post becomes on the site.
type Page struct {
	Title       string
	Date        time.Time
	Description string
	Body        string
	// Extra carries the Mastodon provenance keys, which the sharing archetype
	// does not declare.
	Extra []hugosite.Field
}

// Filename is the page's filename, matching the date-prefixed convention of the
// other pages in content/sharing/.
func (p Page) Filename() string {
	return p.Date.Format("20060102") + "_" + hugosite.Slugify(p.Title) + ".md"
}

// build turns a post into a page. mediaPaths are the site-absolute URLs of the
// attachments already saved for this post, in order.
//
// Where a mymind sharing page is a link with a note attached, a Mastodon post
// *is* the content — so the text goes in the body as markdown (keeping its real
// links) and a flattened copy goes in the description, which is all the section
// list template shows per item.
func build(post mastodon.Post, mediaPaths []string, now time.Time) Page {
	title := titleFor(post)
	return Page{
		Title:       title,
		Date:        pageDate(post, now),
		Description: singleLine(post.Text()),
		Body:        body(post, title, mediaPaths),
		Extra: []hugosite.Field{
			// Quoted: a bare 18-digit id is a YAML integer, and comparing it in
			// a template against a string would then silently never match.
			{Key: FrontMatterKey, Value: hugosite.YAMLString(post.ID)},
			{Key: URLKey, Value: hugosite.YAMLString(post.URL)},
		},
	}
}

// pageDate is when the post went out, in the run's timezone. Mastodon reports
// UTC, so the conversion is what keeps a post made late in the evening from
// being filed under the previous day. A post with no readable date falls back
// to the run time.
func pageDate(post mastodon.Post, now time.Time) time.Time {
	if post.Published.IsZero() {
		return now
	}
	return post.Published.In(now.Location())
}

// titleFor derives a heading from a post that has none.
//
// The first line that says something is the best available summary: the leading
// line of a post is where people put the point. Links are dropped from the
// candidate, because a post about a link usually names it inline ("Yes
// https://…") and the URL is already in the body — in a heading it is noise.
func titleFor(post mastodon.Post) string {
	for _, line := range strings.Split(post.Text(), "\n") {
		if candidate := stripURLs(trimQuoteMarkers(line)); candidate != "" {
			return truncate(candidate, maxTitleLen)
		}
	}
	// Nothing but attachments, or nothing at all.
	return mediaTitle(post)
}

// trimQuoteMarkers removes the punctuation used to mark a line as somebody
// else's words. A post that is entirely a quotation would otherwise get a
// heading opening with a stray '>' or an unclosed quote mark.
func trimQuoteMarkers(line string) string {
	line = strings.TrimSpace(line)
	for {
		trimmed := strings.TrimSpace(strings.TrimPrefix(line, ">"))
		trimmed = strings.Trim(trimmed, `"“”`)
		trimmed = strings.TrimSpace(trimmed)
		if trimmed == line {
			return line
		}
		line = trimmed
	}
}

// stripURLs drops whole-word links from a line and collapses the gap they leave.
func stripURLs(line string) string {
	fields := strings.Fields(line)
	kept := make([]string, 0, len(fields))
	for _, field := range fields {
		if strings.HasPrefix(field, "http://") || strings.HasPrefix(field, "https://") {
			continue
		}
		kept = append(kept, field)
	}
	return strings.Join(kept, " ")
}

// mediaTitle names a post that has no text of its own.
func mediaTitle(post mastodon.Post) string {
	for _, media := range post.Media {
		switch {
		case strings.EqualFold(media.Medium, "video"), strings.HasPrefix(media.Type, "video/"):
			return "Video"
		case strings.EqualFold(media.Medium, "audio"), strings.HasPrefix(media.Type, "audio/"):
			return "Audio"
		}
	}
	if len(post.Media) > 0 {
		return "Photo"
	}
	return "Post"
}

// truncate shortens text to at most max runes, preferring a word boundary and
// marking the cut so the heading does not read as a mangled sentence.
func truncate(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}

	cut := string(runes[:max])
	if idx := strings.LastIndexAny(cut, " \t"); idx > max/2 {
		cut = cut[:idx]
	}
	return strings.TrimRight(strings.TrimSpace(cut), ",;:.!?-–—") + "…"
}

// body is the page's markdown: the post itself, then its attachments.
func body(post mastodon.Post, title string, mediaPaths []string) string {
	blocks := make([]string, 0, 2)
	if markdown := post.Markdown(); markdown != "" {
		blocks = append(blocks, markdown)
	}

	// The feed carries no alt text, so the derived title stands in — it is the
	// post's own words about what it is showing, which beats an empty alt.
	images := make([]string, 0, len(mediaPaths))
	for _, mediaPath := range mediaPaths {
		images = append(images, "!["+escapeLabel(title)+"]("+mediaPath+")")
	}
	if len(images) > 0 {
		blocks = append(blocks, strings.Join(images, "\n"))
	}
	return strings.Join(blocks, "\n\n")
}

// escapeLabel keeps brackets in a title from closing a markdown label early.
func escapeLabel(label string) string {
	return strings.NewReplacer(`[`, `\[`, `]`, `\]`).Replace(label)
}

// mediaRelPath is where an attachment is saved inside the site, e.g.
// "static/images/mastodon/117066375180847363/1.jpeg".
func mediaRelPath(postID, filename string) string {
	return path.Join(mediaDir, postID, filename)
}

// mediaURL is how the page refers to a saved attachment. Hugo serves everything
// under static/ from the site root, so the "static" prefix is dropped.
func mediaURL(relPath string) string {
	return "/" + strings.TrimPrefix(relPath, "static/")
}

// postMediaDir is the whole directory of one post's attachments.
func postMediaDir(postID string) string {
	return path.Join(mediaDir, postID)
}

// frontMatterFields is the front matter to write over the archetype's.
func frontMatterFields(page Page) []hugosite.Field {
	fields := []hugosite.Field{
		{Key: "title", Value: hugosite.YAMLString(page.Title)},
		{Key: "date", Value: hugosite.YAMLRaw(page.Date.Format(time.RFC3339))},
		{Key: "description", Value: hugosite.YAMLString(page.Description)},
	}
	return append(fields, page.Extra...)
}

// singleLine flattens the post so a multi-paragraph one still fits on a single
// front matter line.
func singleLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// pagePath is the page's path inside the site.
func pagePath(page Page) string {
	return fmt.Sprintf("content/%s/%s", Section, page.Filename())
}
