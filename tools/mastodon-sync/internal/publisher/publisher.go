// Package publisher turns Mastodon posts into Hugo pages in content/sharing/.
//
// Unlike a source that can be written back to, an RSS feed offers nowhere to
// record "this one is done". So the record lives in the site: every generated
// page carries the status id in its front matter, and a run starts by reading
// those ids back. The page and its own idempotency marker are therefore the
// same artifact, committed together — they cannot drift apart the way a
// separate state file can.
package publisher

import (
	"context"
	"fmt"
	"time"

	"github.com/zufrieden/zufrieden.github.com/tools/internal/hugosite"
	"github.com/zufrieden/zufrieden.github.com/tools/mastodon-sync/internal/mastodon"
)

// Fetcher is the slice of the mastodon client the publisher needs.
type Fetcher interface {
	Posts(ctx context.Context, limit int) ([]mastodon.Post, error)
	Download(ctx context.Context, url string) ([]byte, string, error)
}

type Status string

const (
	StatusCreated Status = "created"
	StatusSkipped Status = "skipped"
	StatusPlanned Status = "planned" // dry run only
	StatusFailed  Status = "failed"
)

type Result struct {
	PostID string
	Title  string
	Path   string
	Status Status
	Reason string
	Err    error
}

type Options struct {
	// Limit caps how many of the feed's posts are considered, newest first.
	// <= 0 means all of them.
	Limit int
	// DryRun reports what would happen without writing files or downloading
	// attachments.
	DryRun bool
	// Now is the run time. A page is dated by the post's own timestamp; Now
	// supplies the timezone that timestamp is read in, and the date itself for a
	// post the feed gave no readable one.
	Now time.Time
	// Logf receives progress lines. Optional.
	Logf func(format string, args ...any)
}

// Run publishes every post in the feed that the site does not already have, and
// returns one result per post considered. A per-post failure is recorded rather
// than aborting the whole run; the error return is reserved for failures that
// stop everything, such as fetching the feed.
func Run(ctx context.Context, client Fetcher, site *hugosite.Site, opts Options) ([]Result, error) {
	logf := opts.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}

	published, err := site.FrontMatterValues(Section, FrontMatterKey)
	if err != nil {
		return nil, fmt.Errorf("reading published %s values: %w", FrontMatterKey, err)
	}
	logf("%d post(s) already published in content/%s", len(published), Section)

	posts, err := client.Posts(ctx, opts.Limit)
	if err != nil {
		return nil, err
	}
	logf("%d post(s) in the feed", len(posts))

	results := make([]Result, 0, len(posts))
	// Nothing is written during a dry run, so the paths handed out have to be
	// remembered for the report to be accurate.
	claimed := map[string]bool{}
	// The feed is newest-first; publish oldest-first so that same-day pages get
	// their "_2" suffixes in the order the posts were actually made.
	for i := len(posts) - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return results, err
		}
		results = append(results, publishOne(ctx, client, site, opts, posts[i], published, claimed, logf))
	}
	return results, nil
}

func publishOne(
	ctx context.Context,
	client Fetcher,
	site *hugosite.Site,
	opts Options,
	post mastodon.Post,
	published map[string]string,
	claimed map[string]bool,
	logf func(string, ...any),
) Result {
	result := Result{PostID: post.ID}

	if existing, done := published[post.ID]; done {
		result.Status, result.Reason = StatusSkipped, "already published as "+existing
		return result
	}

	if opts.DryRun {
		// Nothing is downloaded, so the page is built without attachment paths.
		page := build(post, nil, opts.Now)
		result.Title = page.Title
		result.Path = site.AvailablePathExcluding(pagePath(page), claimed)
		claimed[result.Path] = true
		result.Status = StatusPlanned
		logf("would create %s (%s, %d attachment(s))", result.Path, post.ID, len(post.Media))
		return result
	}

	mediaPaths, cleanupMedia, err := downloadMedia(ctx, client, site, post, logf)
	if err != nil {
		result.Title = titleFor(post)
		result.Status, result.Err = StatusFailed, err
		return result
	}

	page := build(post, mediaPaths, opts.Now)
	result.Title = page.Title
	result.Path = site.AvailablePath(pagePath(page))

	// The page is the last thing written, so there is nothing left to roll it
	// back for — unlike mymind-sync, no follow-up call can fail after this.
	if _, err := site.CreatePage(result.Path, frontMatterFields(page), page.Body); err != nil {
		cleanupMedia()
		result.Status, result.Err = StatusFailed, err
		return result
	}
	logf("created %s", result.Path)

	// Guard against the same post appearing twice in one feed, which would
	// otherwise produce a "_2" duplicate.
	published[post.ID] = result.Path

	result.Status = StatusCreated
	return result
}

// downloadMedia saves a post's attachments and returns the URLs the page should
// use, together with a function removing them again.
//
// A failure here fails the post rather than publishing it without its pictures:
// the status id is only recorded on a page that exists, so the next run retries
// from scratch.
func downloadMedia(
	ctx context.Context,
	client Fetcher,
	site *hugosite.Site,
	post mastodon.Post,
	logf func(string, ...any),
) (urls []string, cleanup func(), err error) {
	if len(post.Media) == 0 {
		return nil, func() {}, nil
	}

	var written []func()
	cleanup = func() {
		for _, remove := range written {
			remove()
		}
	}

	// This post has no page (or it would have been skipped), so anything already
	// sitting in its attachment directory is debris from a run that was killed
	// before it could clean up. Left in place it would block every retry.
	dir := postMediaDir(post.ID)
	if site.Exists(dir) {
		logf("clearing %s left by an interrupted run", dir)
		if err := site.RemoveAll(dir); err != nil {
			return nil, func() {}, fmt.Errorf("clearing %s: %w", dir, err)
		}
	}

	for i, media := range post.Media {
		body, contentType, err := client.Download(ctx, media.URL)
		if err != nil {
			cleanup()
			return nil, func() {}, err
		}
		// The feed does not always set the type; the response does.
		if media.Type == "" {
			media.Type = contentType
		}

		relPath := mediaRelPath(post.ID, media.Filename(i+1))
		remove, err := site.WriteFile(relPath, body)
		if err != nil {
			cleanup()
			return nil, func() {}, err
		}
		written = append(written, remove)
		logf("saved %s (%d bytes)", relPath, len(body))

		urls = append(urls, mediaURL(relPath))
	}
	return urls, cleanup, nil
}

// Count tallies results by status.
func Count(results []Result, status Status) int {
	n := 0
	for _, r := range results {
		if r.Status == status {
			n++
		}
	}
	return n
}
