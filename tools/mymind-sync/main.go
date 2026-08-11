// Command mymind-sync publishes tagged mymind objects to this Hugo site.
//
// It looks for objects carrying a trigger tag but not the matching done tag,
// creates the corresponding Hugo page, and then writes the done tag back to
// mymind so the object is not picked up again:
//
//	#share -> content/sharing/, then tagged "shared"
//	#keep  -> content/linking/, then tagged "keeped"
//
//	mymind-sync -dry-run
//	mymind-sync -type linking -limit 10
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata" // embed the tz database so -timezone works on bare runners

	"github.com/zufrieden/zufrieden.github.com/tools/internal/hugosite"
	"github.com/zufrieden/zufrieden.github.com/tools/mymind-sync/internal/config"
	"github.com/zufrieden/zufrieden.github.com/tools/mymind-sync/internal/mymind"
	"github.com/zufrieden/zufrieden.github.com/tools/mymind-sync/internal/publisher"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		contentTypes = flag.String("type", strings.Join(publisher.Names(), ","),
			"comma-separated content types to publish ("+strings.Join(publisher.Names(), ", ")+")")
		repoPath = flag.String("repo", ".", "path inside the Hugo site; the site root is discovered from here")
		envPath  = flag.String("env", ".env", "path to a .env file with credentials (optional; real env vars win)")
		query    = flag.String("query", "", "override the mymind search query")
		limit    = flag.Int("limit", 50, "maximum number of objects to process in one run")
		timezone = flag.String("timezone", "Europe/Zurich", "timezone used for the page date and filename prefix")
		hugoBin  = flag.String("hugo", "hugo", "hugo executable used to scaffold pages from the archetype")
		dryRun   = flag.Bool("dry-run", false, "report what would be published without writing or tagging")
		verbose  = flag.Bool("v", false, "verbose logging")
	)
	flag.Parse()

	mappings, err := publisher.LookupAll(*contentTypes)
	if err != nil {
		return err
	}

	location, err := time.LoadLocation(*timezone)
	if err != nil {
		return fmt.Errorf("timezone %q: %w", *timezone, err)
	}

	cfg, err := config.Load(*envPath)
	if err != nil {
		return err
	}

	site, err := hugosite.Discover(*repoPath, *hugoBin)
	if err != nil {
		return err
	}

	client, err := mymind.New(cfg.BaseURL, cfg.KeyID, cfg.PrivateKey)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logf := func(string, ...any) {}
	if *verbose {
		logf = func(format string, args ...any) { fmt.Fprintf(os.Stderr, "  "+format+"\n", args...) }
	}

	fmt.Printf("mymind-sync: site %s\n", site.Root)
	if *dryRun {
		fmt.Println("dry run: nothing will be written or tagged")
	}

	// A -query override only makes sense for a single content type; each
	// mapping otherwise builds its own `tag:x -tag:y` search.
	if *query != "" && len(mappings) > 1 {
		return fmt.Errorf("-query applies to one content type at a time; add -type <name>")
	}

	var all []publisher.Result
	for _, mapping := range mappings {
		fmt.Printf("\n#%s -> content/%s\n", mapping.SourceTag, mapping.Section)

		results, err := publisher.Run(ctx, client, site, mapping, publisher.Options{
			Limit:  *limit,
			DryRun: *dryRun,
			Now:    time.Now().In(location),
			Query:  *query,
			Logf:   logf,
		})
		if err != nil {
			return fmt.Errorf("%s: %w", mapping.Name, err)
		}

		report(results)
		all = append(all, results...)
	}

	created := publisher.Count(all, publisher.StatusCreated) + publisher.Count(all, publisher.StatusPlanned)
	if err := writeGitHubOutput(created); err != nil {
		return err
	}

	if failed := publisher.Count(all, publisher.StatusFailed); failed > 0 {
		return fmt.Errorf("%d object(s) failed to publish", failed)
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
			fmt.Printf("  - skipped %q: %s\n", displayTitle(r), r.Reason)
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
	return r.ObjectID
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
