package publisher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zufrieden/zufrieden.github.com/tools/mymind-sync/internal/hugosite"
	"github.com/zufrieden/zufrieden.github.com/tools/mymind-sync/internal/mymind"
)

type fakeClient struct {
	objects  []mymind.Object
	listErr  error
	tagErr   error
	tagged   map[string][]string
	lastQury string
}

func (f *fakeClient) ListObjects(_ context.Context, query string, _ int) ([]mymind.Object, error) {
	f.lastQury = query
	return f.objects, f.listErr
}

func (f *fakeClient) AddTags(_ context.Context, objectID string, names ...string) error {
	if f.tagErr != nil {
		return f.tagErr
	}
	if f.tagged == nil {
		f.tagged = map[string][]string{}
	}
	f.tagged[objectID] = append(f.tagged[objectID], names...)
	return nil
}

// newTestSite builds a throwaway Hugo site whose archetype matches this repo's,
// with a hugo binary name that cannot resolve so the archetype fallback is used.
func newTestSite(t *testing.T) *hugosite.Site {
	t.Helper()
	root := t.TempDir()
	write(t, filepath.Join(root, "config.toml"), "baseURL = \"https://zufrieden.io/\"\n")
	write(t, filepath.Join(root, "archetypes", "sharing.md"), `---
title: "{{ replace .Name "_" " " | title }}"
date: {{ .Date }}
showDate: true
draft: false
description :
---
`)
	return &hugosite.Site{Root: root, HugoBin: "hugo-not-installed-for-tests"}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testTime(t *testing.T) time.Time {
	t.Helper()
	when, err := time.Parse(time.RFC3339, "2026-08-10T09:30:00+02:00")
	if err != nil {
		t.Fatal(err)
	}
	return when
}

func TestSharingQuery(t *testing.T) {
	mapping, err := Lookup("sharing")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := mapping.SearchQuery(), "tag:share -tag:shared"; got != want {
		t.Errorf("SearchQuery = %q, want %q", got, want)
	}
	if _, err := Lookup("nope"); err == nil {
		t.Error("expected an error for an unknown content type")
	}
}

func TestRunCreatesPageAndTags(t *testing.T) {
	mapping, _ := Lookup("sharing")
	site := newTestSite(t)
	client := &fakeClient{objects: []mymind.Object{{
		ID:     "obj-1",
		Title:  "Creative is 10%. Structure and systems are the rest.",
		Source: &mymind.Source{URL: "https://www.chrbutler.com/how-to-turn-good-design-direction-into-a-good-system"},
		Tags:   []mymind.Tag{{Name: "share"}},
		Notes:  []mymind.Note{{Content: mymind.Content{Body: "Nice article about\ncreative work."}}},
	}}}

	results, err := Run(context.Background(), client, site, mapping, Options{Now: testTime(t)})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Status != StatusCreated {
		t.Fatalf("unexpected results: %+v", results)
	}

	wantPath := "content/sharing/20260810_creative_is_10_structure_and_systems_are_the_rest.md"
	if results[0].Path != wantPath {
		t.Errorf("path = %q, want %q", results[0].Path, wantPath)
	}

	got, err := os.ReadFile(filepath.Join(site.Root, wantPath))
	if err != nil {
		t.Fatal(err)
	}
	want := `---
title: "Creative is 10%. Structure and systems are the rest."
date: 2026-08-10T09:30:00+02:00
showDate: true
draft: false
description: "Nice article about creative work."
---
[https://www.chrbutler.com/how-to-turn-good-design-direction-into-a-good-system](https://www.chrbutler.com/how-to-turn-good-design-direction-into-a-good-system)
`
	if string(got) != want {
		t.Errorf("page mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}

	if tags := client.tagged["obj-1"]; len(tags) != 1 || tags[0] != "shared" {
		t.Errorf("tagged = %v, want [shared]", tags)
	}
}

func TestDescriptionFallsBackToSummary(t *testing.T) {
	mapping, _ := Lookup("sharing")
	page, err := mapping.Build(mymind.Object{
		Title:   "Thing",
		Summary: "  An AI summary.  ",
		Source:  &mymind.Source{URL: "https://example.com"},
	}, testTime(t))
	if err != nil {
		t.Fatal(err)
	}
	if page.Description != "An AI summary." {
		t.Errorf("description = %q", page.Description)
	}
}

func TestBuildRequiresSourceURL(t *testing.T) {
	mapping, _ := Lookup("sharing")

	// A PDF, image or plain note has no link to publish.
	noLink := []struct {
		name string
		obj  mymind.Object
	}{
		{"no source at all", mymind.Object{Title: "A PDF"}},
		{"empty source url", mymind.Object{Title: "A PDF", Source: &mymind.Source{URL: ""}}},
		{"blank source url", mymind.Object{Title: "A PDF", Source: &mymind.Source{URL: "   "}}},
	}
	for _, tc := range noLink {
		if _, err := mapping.Build(tc.obj, testTime(t)); err == nil {
			t.Errorf("%s: expected an error, got none", tc.name)
		}
	}
}

// A PDF saved to mymind and tagged `share` has no source.url. It must be
// skipped, left untagged (so it is retried if a URL is added later), and it
// must not stop the linked objects in the same batch from being published.
func TestObjectWithoutLinkIsSkippedButBatchContinues(t *testing.T) {
	mapping, _ := Lookup("sharing")
	site := newTestSite(t)
	client := &fakeClient{objects: []mymind.Object{
		{
			ID:      "pdf-1",
			Title:   "Vers une Société des Communs",
			Summary: "A PDF with no link attached.",
			Tags:    []mymind.Tag{{Name: "share"}},
		},
		{
			ID:     "link-1",
			Title:  "A proper link",
			Source: &mymind.Source{URL: "https://example.com/article"},
			Tags:   []mymind.Tag{{Name: "share"}},
		},
	}}

	results, err := Run(context.Background(), client, site, mapping, Options{Now: testTime(t)})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	pdf, link := results[0], results[1]
	if pdf.Status != StatusSkipped {
		t.Errorf("PDF status = %s, want skipped", pdf.Status)
	}
	if !strings.Contains(pdf.Reason, "no link") {
		t.Errorf("PDF reason = %q, want it to mention the missing link", pdf.Reason)
	}
	if _, tagged := client.tagged["pdf-1"]; tagged {
		t.Error("the PDF must stay untagged so it is retried once it has a URL")
	}

	if link.Status != StatusCreated {
		t.Errorf("link status = %s, want created", link.Status)
	}
	if !site.Exists(link.Path) {
		t.Errorf("page %s was not written", link.Path)
	}
	if tags := client.tagged["link-1"]; len(tags) != 1 || tags[0] != "shared" {
		t.Errorf("tagged = %v, want [shared]", tags)
	}

	// A skip is not a failure: the run should exit cleanly.
	if Count(results, StatusFailed) != 0 {
		t.Errorf("skipping a PDF should not fail the run: %+v", results)
	}
}

func TestTitleFallsBackToHost(t *testing.T) {
	mapping, _ := Lookup("sharing")
	page, err := mapping.Build(mymind.Object{
		Source: &mymind.Source{URL: "https://www.example.com/a/b"},
	}, testTime(t))
	if err != nil {
		t.Fatal(err)
	}
	if page.Title != "example.com" {
		t.Errorf("title = %q, want example.com", page.Title)
	}
}

func TestAlreadySharedIsSkipped(t *testing.T) {
	mapping, _ := Lookup("sharing")
	site := newTestSite(t)
	client := &fakeClient{objects: []mymind.Object{
		{ID: "a", Title: "Done", Source: &mymind.Source{URL: "https://example.com/a"},
			Tags: []mymind.Tag{{Name: "share"}, {Name: "Shared"}}},
		{ID: "b", Title: "Not tagged", Source: &mymind.Source{URL: "https://example.com/b"},
			Tags: []mymind.Tag{{Name: "reading"}}},
	}}

	results, err := Run(context.Background(), client, site, mapping, Options{Now: testTime(t)})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Status != StatusSkipped {
			t.Errorf("object %s: status = %s, want skipped", r.ObjectID, r.Status)
		}
	}
	if len(client.tagged) != 0 {
		t.Errorf("nothing should have been tagged, got %v", client.tagged)
	}
}

func TestTagFailureRollsBackThePage(t *testing.T) {
	mapping, _ := Lookup("sharing")
	site := newTestSite(t)
	client := &fakeClient{
		tagErr: errors.New("boom"),
		objects: []mymind.Object{{
			ID: "obj-1", Title: "Thing",
			Source: &mymind.Source{URL: "https://example.com/x"},
			Tags:   []mymind.Tag{{Name: "share"}},
		}},
	}

	results, err := Run(context.Background(), client, site, mapping, Options{Now: testTime(t)})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Status != StatusFailed {
		t.Fatalf("unexpected results: %+v", results)
	}
	if site.Exists("content/sharing/20260810_thing.md") {
		t.Error("page should have been rolled back after the tagging failure")
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	mapping, _ := Lookup("sharing")
	site := newTestSite(t)
	client := &fakeClient{objects: []mymind.Object{{
		ID: "obj-1", Title: "Thing",
		Source: &mymind.Source{URL: "https://example.com/x"},
		Tags:   []mymind.Tag{{Name: "share"}},
	}}}

	results, err := Run(context.Background(), client, site, mapping, Options{Now: testTime(t), DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if Count(results, StatusPlanned) != 1 {
		t.Fatalf("unexpected results: %+v", results)
	}
	if site.Exists("content/sharing/20260810_thing.md") {
		t.Error("dry run wrote a page")
	}
	if len(client.tagged) != 0 {
		t.Errorf("dry run tagged objects: %v", client.tagged)
	}
}

func TestSameDayTitleCollisionGetsSuffix(t *testing.T) {
	mapping, _ := Lookup("sharing")
	site := newTestSite(t)
	client := &fakeClient{objects: []mymind.Object{
		{ID: "a", Title: "Thing", Source: &mymind.Source{URL: "https://example.com/a"},
			Tags: []mymind.Tag{{Name: "share"}}},
		{ID: "b", Title: "Thing", Source: &mymind.Source{URL: "https://example.com/b"},
			Tags: []mymind.Tag{{Name: "share"}}},
	}}

	results, err := Run(context.Background(), client, site, mapping, Options{Now: testTime(t)})
	if err != nil {
		t.Fatal(err)
	}
	if Count(results, StatusCreated) != 2 {
		t.Fatalf("unexpected results: %+v", results)
	}
	if results[1].Path != "content/sharing/20260810_thing_2.md" {
		t.Errorf("second path = %q", results[1].Path)
	}
}

func TestQueryOverride(t *testing.T) {
	mapping, _ := Lookup("sharing")
	client := &fakeClient{}
	if _, err := Run(context.Background(), client, newTestSite(t), mapping, Options{
		Now: testTime(t), Query: "tag:share created:2026",
	}); err != nil {
		t.Fatal(err)
	}
	if client.lastQury != "tag:share created:2026" {
		t.Errorf("query = %q", client.lastQury)
	}
}

func TestListErrorAborts(t *testing.T) {
	mapping, _ := Lookup("sharing")
	client := &fakeClient{listErr: errors.New("network down")}
	if _, err := Run(context.Background(), client, newTestSite(t), mapping, Options{Now: testTime(t)}); err == nil {
		t.Fatal("expected the list error to abort the run")
	}
}
