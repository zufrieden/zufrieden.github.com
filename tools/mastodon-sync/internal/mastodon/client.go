// Package mastodon reads an account's public posts from its Mastodon RSS feed,
// e.g. https://social.coop/@zufrieden.rss.
//
// The feed needs no credentials, which is the whole reason this tool has no
// config: everything it reads is already public. The trade-off is that the feed
// only carries the most recent posts (20, at the time of writing) and is
// read-only — nothing can be marked as "synced" at the source, so the
// idempotency record has to live in the Hugo site.
package mastodon

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultUA identifies the tool to the instance. Mastodon rate-limits
	// anonymous feed reads, so being identifiable is the polite thing to do.
	DefaultUA = "zufrieden-mastodon-sync/1.0"

	// maxDownloadBytes caps a single attachment. Mastodon's own limits are well
	// under this; the cap is here so a misbehaving URL cannot exhaust memory.
	maxDownloadBytes = 64 << 20 // 64 MiB
)

type Client struct {
	feedURL   string
	userAgent string
	http      *http.Client
}

type Option func(*Client)

func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }
func WithUserAgent(ua string) Option       { return func(c *Client) { c.userAgent = ua } }

func New(feedURL string, opts ...Option) (*Client, error) {
	trimmed := strings.TrimSpace(feedURL)
	if trimmed == "" {
		return nil, fmt.Errorf("mastodon: feed URL is empty")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("mastodon: feed URL %q: %w", trimmed, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("mastodon: feed URL %q must be http or https", trimmed)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("mastodon: feed URL %q has no host", trimmed)
	}

	c := &Client{
		feedURL:   trimmed,
		userAgent: DefaultUA,
		http:      &http.Client{Timeout: 60 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// FeedURL is the feed being read, for logging.
func (c *Client) FeedURL() string { return c.feedURL }

// Posts returns the feed's posts, newest first, capped at limit (<= 0 means no
// cap). Posts the feed describes too poorly to publish — no usable status id —
// are dropped.
func (c *Client) Posts(ctx context.Context, limit int) ([]Post, error) {
	raw, err := c.get(ctx, c.feedURL)
	if err != nil {
		return nil, err
	}

	posts, err := parseFeed(raw)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(posts) > limit {
		posts = posts[:limit]
	}
	return posts, nil
}

// Download fetches an attachment and returns its bytes and Content-Type.
func (c *Client) Download(ctx context.Context, rawURL string) ([]byte, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, "", fmt.Errorf("mastodon: attachment URL %q: %w", rawURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, "", fmt.Errorf("mastodon: attachment URL %q must be http or https", rawURL)
	}

	body, contentType, err := c.getWithType(ctx, parsed.String())
	if err != nil {
		return nil, "", err
	}
	return body, contentType, nil
}

func (c *Client) get(ctx context.Context, target string) ([]byte, error) {
	body, _, err := c.getWithType(ctx, target)
	return body, err
}

func (c *Client) getWithType(ctx context.Context, target string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("mastodon: GET %s: %w", target, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("mastodon: GET %s: HTTP %d %s", target, resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	// LimitReader with one spare byte, so an oversized body is reported rather
	// than silently truncated into a corrupt file.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("mastodon: reading %s: %w", target, err)
	}
	if len(body) > maxDownloadBytes {
		return nil, "", fmt.Errorf("mastodon: %s is larger than %d bytes", target, maxDownloadBytes)
	}
	return body, resp.Header.Get("Content-Type"), nil
}

// rssFeed maps only what this tool reads. Mastodon's feed carries a good deal
// more (channel image, webfeeds icon, media ratings) that is of no use here.
type rssFeed struct {
	XMLName xml.Name  `xml:"rss"`
	Items   []rssItem `xml:"channel>item"`
}

type rssItem struct {
	GUID        string     `xml:"guid"`
	Link        string     `xml:"link"`
	PubDate     string     `xml:"pubDate"`
	Description string     `xml:"description"`
	Media       []rssMedia `xml:"http://search.yahoo.com/mrss/ content"`
}

type rssMedia struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr"`
	Medium string `xml:"medium,attr"`
}

// pubDateLayouts covers the RFC 822 date shapes an RSS feed may use. Mastodon
// writes the first one; the others are cheap tolerance for other instances.
var pubDateLayouts = []string{
	time.RFC1123Z, // Sun, 09 Aug 2026 15:58:45 +0000
	time.RFC1123,  // Sun, 09 Aug 2026 15:58:45 GMT
	time.RFC822Z,
	time.RFC822,
	time.RFC3339,
}

func parseFeed(raw []byte) ([]Post, error) {
	var feed rssFeed
	if err := xml.Unmarshal(raw, &feed); err != nil {
		return nil, fmt.Errorf("mastodon: parsing feed: %w", err)
	}

	posts := make([]Post, 0, len(feed.Items))
	for _, item := range feed.Items {
		post, ok := item.toPost()
		if !ok {
			continue
		}
		posts = append(posts, post)
	}
	return posts, nil
}

// toPost converts a feed item, reporting false for one that cannot be
// published: without a status id there is no way to record that it was, so it
// would be republished on every run.
func (i rssItem) toPost() (Post, bool) {
	permalink := strings.TrimSpace(i.GUID)
	id := statusID(permalink)
	if id == "" {
		permalink = strings.TrimSpace(i.Link)
		id = statusID(permalink)
	}
	if id == "" {
		return Post{}, false
	}

	post := Post{
		ID:        id,
		URL:       permalink,
		Published: parsePubDate(i.PubDate),
		HTML:      strings.TrimSpace(i.Description),
	}
	for _, media := range i.Media {
		if u := strings.TrimSpace(media.URL); u != "" {
			post.Media = append(post.Media, Media{
				URL:    u,
				Type:   strings.TrimSpace(media.Type),
				Medium: strings.TrimSpace(media.Medium),
			})
		}
	}
	return post, true
}

// parsePubDate returns the zero time when the date is missing or unreadable;
// callers fall back to the run time rather than dropping the post.
func parsePubDate(raw string) time.Time {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}
	}
	for _, layout := range pubDateLayouts {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed
		}
	}
	return time.Time{}
}
