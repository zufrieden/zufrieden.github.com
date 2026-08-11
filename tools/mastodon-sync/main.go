// Command mastodon-sync publishes this account's Mastodon posts to the Hugo
// site as content/sharing/ pages.
//
// It reads the account's public RSS feed, skips every post whose status id
// already appears in a page's front matter, and creates a page (plus a copy of
// any attachments under static/) for the rest:
//
//	https://social.coop/@zufrieden.rss -> content/sharing/, tagged mastodon_id
//
//	mastodon-sync -dry-run
//	mastodon-sync -limit 5 -v
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // embed the tz database so -timezone works on bare runners

	"github.com/zufrieden/zufrieden.github.com/tools/internal/hugosite"
	"github.com/zufrieden/zufrieden.github.com/tools/mastodon-sync/internal/mastodon"
	"github.com/zufrieden/zufrieden.github.com/tools/mastodon-sync/internal/publisher"
)

// defaultFeed is this site's own account. The feed is public, which is why this
// tool needs no credentials at all.
const defaultFeed = "https://social.coop/@zufrieden.rss"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		feedURL  = flag.String("feed", defaultFeed, "Mastodon account RSS feed to read")
		repoPath = flag.String("repo", ".", "path inside the Hugo site; the site root is discovered from here")
		limit    = flag.Int("limit", 0, "maximum number of posts to consider, newest first (0 = the whole feed)")
		timezone = flag.String("timezone", "Europe/Zurich", "timezone the post date is read in, for the page date and filename prefix")
		hugoBin  = flag.String("hugo", "hugo", "hugo executable used to scaffold pages from the archetype")
		dryRun   = flag.Bool("dry-run", false, "report what would be published without writing anything")
		verbose  = flag.Bool("v", false, "verbose logging")
	)
	flag.Parse()

	location, err := time.LoadLocation(*timezone)
	if err != nil {
		return fmt.Errorf("timezone %q: %w", *timezone, err)
	}

	site, err := hugosite.Discover(*repoPath, *hugoBin)
	if err != nil {
		return err
	}

	client, err := mastodon.New(*feedURL)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logf := func(string, ...any) {}
	if *verbose {
		logf = func(format string, args ...any) { fmt.Fprintf(os.Stderr, "  "+format+"\n", args...) }
	}

	fmt.Printf("mastodon-sync: site %s\n", site.Root)
	if *dryRun {
		fmt.Println("dry run: nothing will be written or downloaded")
	}
	fmt.Printf("\n%s -> content/%s\n", client.FeedURL(), publisher.Section)

	results, err := publisher.Run(ctx, client, site, publisher.Options{
		Limit:  *limit,
		DryRun: *dryRun,
		Now:    time.Now().In(location),
		Logf:   logf,
	})
	if err != nil {
		return err
	}

	report(results)

	created := publisher.Count(results, publisher.StatusCreated) + publisher.Count(results, publisher.StatusPlanned)
	if err := writeGitHubOutput(created); err != nil {
		return err
	}

	if failed := publisher.Count(results, publisher.StatusFailed); failed > 0 {
		return fmt.Errorf("%d post(s) failed to publish", failed)
	}
	return nil
}

func report(results []publisher.Result) {
	for _, r := range results {
		switch r.Status {
		case publisher.StatusCreated:
			fmt.Printf("  + %s\n", r.Path)
		case publisher.StatusPlanned:
			fmt.Printf("  ~ %s (dry run)\n", r.Path)
		case publisher.StatusSkipped:
			fmt.Printf("  - skipped %s: %s\n", r.PostID, r.Reason)
		case publisher.StatusFailed:
			fmt.Printf("  ! failed %q: %v\n", displayTitle(r), r.Err)
		}
	}

	fmt.Printf("done: %d created, %d skipped, %d failed",
		publisher.Count(results, publisher.StatusCreated),
		publisher.Count(results, publisher.StatusSkipped),
		publisher.Count(results, publisher.StatusFailed),
	)
	if planned := publisher.Count(results, publisher.StatusPlanned); planned > 0 {
		fmt.Printf(", %d planned", planned)
	}
	fmt.Println()
}

func displayTitle(r publisher.Result) string {
	if r.Title != "" {
		return r.Title
	}
	return r.PostID
}

// writeGitHubOutput exposes the number of new pages so the workflow can decide
// whether there is anything to commit.
func writeGitHubOutput(created int) error {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		return nil
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := fmt.Fprintf(file, "created=%d\n", created)
	return errors.Join(writeErr, file.Close())
}
