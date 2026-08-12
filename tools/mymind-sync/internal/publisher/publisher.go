// Package publisher turns tagged mymind objects into Hugo pages and marks them
// as published back in mymind.
package publisher

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/zufrieden/zufrieden.github.com/tools/internal/hugosite"
	"github.com/zufrieden/zufrieden.github.com/tools/mymind-sync/internal/mymind"
)

// Fetcher is the slice of the mymind client the publisher needs.
type Fetcher interface {
	ListObjects(ctx context.Context, query string, limit int) ([]mymind.Object, error)
	Blob(ctx context.Context, objectID string) ([]byte, string, error)
	AddTags(ctx context.Context, objectID string, names ...string) error
}

type Status string

const (
	StatusCreated Status = "created"
	StatusSkipped Status = "skipped"
	StatusPlanned Status = "planned" // dry run only
	StatusFailed  Status = "failed"
)

type Result struct {
	ObjectID string
	Title    string
	Path     string
	Status   Status
	Reason   string
	Err      error
}

type Options struct {
	// Limit caps how many objects are fetched per run. <= 0 means no cap.
	Limit int
	// DryRun reports what would happen without writing files or tagging.
	DryRun bool
	// Now is the run time. A page is dated by the object's mymind `created`
	// timestamp; Now only supplies the timezone that timestamp is read in, and
	// the date itself for an object that has no usable one.
	Now time.Time
	// Query overrides the mapping's default mymind search.
	Query string
	// Logf receives progress lines. Optional.
	Logf func(format string, args ...any)
}

// Run publishes every object matching the mapping and returns one result per
// object considered. A per-object failure is recorded rather than aborting the
// whole run; the error return is reserved for failures that stop everything
// (such as the list call itself).
func Run(ctx context.Context, client Fetcher, site *hugosite.Site, mapping Mapping, opts Options) ([]Result, error) {
	logf := opts.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}

	query := opts.Query
	if query == "" {
		query = mapping.SearchQuery()
	}

	logf("querying mymind: %s", query)
	objects, err := client.ListObjects(ctx, query, opts.Limit)
	if err != nil {
		return nil, err
	}
	logf("%d object(s) returned", len(objects))

	results := make([]Result, 0, len(objects))
	for _, obj := range objects {
		if err := ctx.Err(); err != nil {
			return results, err
		}
		results = append(results, publishOne(ctx, client, site, mapping, opts, obj, logf))
	}
	return results, nil
}

func publishOne(
	ctx context.Context,
	client Fetcher,
	site *hugosite.Site,
	mapping Mapping,
	opts Options,
	obj mymind.Object,
	logf func(string, ...any),
) Result {
	result := Result{ObjectID: obj.ID, Title: obj.Title}

	// The search query should already have filtered these out, but the tags on
	// the object are the authority — never republish something already done.
	if obj.HasTag(mapping.DoneTag) {
		result.Status, result.Reason = StatusSkipped, "already tagged "+mapping.DoneTag
		return result
	}
	if !obj.HasTag(mapping.SourceTag) {
		result.Status, result.Reason = StatusSkipped, "missing tag "+mapping.SourceTag
		return result
	}

	var asset *Asset
	if mapping.Files {
		asset = assetFor(obj)
	}

	// The file comes down before the page is built, because the response is
	// what settles a MIME type mymind did not report in the listing — and the
	// type decides whether the page shows the file or links to it. A failure
	// here fails the object rather than publishing a page missing the very
	// thing it is about; nothing has been tagged, so the next run retries.
	var data []byte
	if asset != nil && !opts.DryRun {
		body, contentType, err := client.Blob(ctx, obj.ID)
		if err != nil {
			result.Status, result.Err = StatusFailed, fmt.Errorf("downloading the file for %s: %w", obj.ID, err)
			return result
		}
		data, asset = body, asset.withType(contentType, body)
	}

	page, err := mapping.Build(obj, asset, opts.Now)
	if err != nil {
		result.Status, result.Reason, result.Err = StatusSkipped, err.Error(), err
		return result
	}
	result.Title = page.Title

	// A page carrying a file is a leaf bundle, so the file sits beside it as a
	// page resource — the only form Hugo can process images in.
	relPath := fmt.Sprintf("content/%s/%s", mapping.Section, mapping.Filename(page))
	if asset != nil {
		relPath = hugosite.BundlePath(relPath)
	}
	relPath = site.AvailablePath(relPath)
	result.Path = relPath

	if opts.DryRun {
		result.Status = StatusPlanned
		if asset != nil {
			logf("would create %s (%s, with %s)", relPath, obj.ID, asset.Filename())
		} else {
			logf("would create %s (%s)", relPath, obj.ID)
		}
		return result
	}

	// Everything written for this object, newest first, so a rollback undoes it
	// in one call: for a bundle that is the whole directory, file included.
	rollback := func() { _ = site.RemoveAll(path.Dir(relPath)) }
	if asset == nil {
		rollback = func() { _ = site.RemoveAll(relPath) }
	}

	if asset != nil {
		filePath := path.Join(path.Dir(relPath), asset.Filename())
		if _, err := site.WriteFile(filePath, data); err != nil {
			rollback()
			result.Status, result.Err = StatusFailed, err
			return result
		}
		logf("saved %s (%d bytes)", filePath, len(data))
	}

	if _, err := site.CreatePage(relPath, frontMatterFields(page), page.Body); err != nil {
		rollback()
		result.Status, result.Err = StatusFailed, err
		return result
	}
	logf("created %s", relPath)

	// Tag last: the page is the deliverable, the tag is the bookkeeping. If
	// tagging fails we roll the page back so the next run retries cleanly
	// instead of publishing a second copy under a "_2" suffix.
	if err := client.AddTags(ctx, obj.ID, mapping.DoneTag); err != nil {
		rollback()
		result.Status = StatusFailed
		result.Err = fmt.Errorf("tagging %s as %q failed, rolled back %s: %w", obj.ID, mapping.DoneTag, relPath, err)
		return result
	}
	logf("tagged %s as %q", obj.ID, mapping.DoneTag)

	result.Status = StatusCreated
	return result
}

func frontMatterFields(page Page) []hugosite.Field {
	fields := []hugosite.Field{
		{Key: "title", Value: hugosite.YAMLString(page.Title)},
		{Key: "date", Value: hugosite.YAMLRaw(page.Date.Format(time.RFC3339))},
		{Key: "description", Value: hugosite.YAMLString(page.Description)},
	}
	return append(fields, page.Extra...)
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
