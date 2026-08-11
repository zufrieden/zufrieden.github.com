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
	write(t, filepath.Join(root, "archetypes", "linking.md"), `---
title: "{{ replace .Name "_" " " | title }}"
date: {{ .Date }}
showDate: false
draft: false
tags: []
link: https://
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

// The generated page must match content/linking/a_brief_history_of_javascript.md:
// underscored slug with no date prefix, tags carried over from mymind, the URL
// in the front matter rather than the body, and an empty body.
func TestLinkingCreatesPageAndTags(t *testing.T) {
	mapping, err := Lookup("linking")
	if err != nil {
		t.Fatal(err)
	}
	site := newTestSite(t)
	client := &fakeClient{objects: []mymind.Object{{
		ID:      "obj-1",
		Title:   "A Brief History of Javascript",
		Summary: "This article discusses the development and evolution of JavaScript.",
		Source:  &mymind.Source{URL: "https://deno.com/blog/history-of-javascript"},
		Tags:    []mymind.Tag{{Name: "keep"}, {Name: "js"}},
	}}}

	results, err := Run(context.Background(), client, site, mapping, Options{Now: testTime(t)})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Status != StatusCreated {
		t.Fatalf("unexpected results: %+v", results)
	}

	wantPath := "content/linking/a_brief_history_of_javascript.md"
	if results[0].Path != wantPath {
		t.Errorf("path = %q, want %q", results[0].Path, wantPath)
	}

	got, err := os.ReadFile(filepath.Join(site.Root, wantPath))
	if err != nil {
		t.Fatal(err)
	}
	want := `---
title: "A Brief History of Javascript"
date: 2026-08-10T09:30:00+02:00
showDate: false
draft: false
tags: ["keep","js"]
link: "https://deno.com/blog/history-of-javascript"
description: "This article discusses the development and evolution of JavaScript."
---
`
	if string(got) != want {
		t.Errorf("page mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}

	if tags := client.tagged["obj-1"]; len(tags) != 1 || tags[0] != "keeped" {
		t.Errorf("tagged = %v, want [keeped]", tags)
	}
}

func TestLinkingQueryAndDoneTag(t *testing.T) {
	mapping, _ := Lookup("linking")
	if got, want := mapping.SearchQuery(), "tag:keep -tag:keeped"; got != want {
		t.Errorf("SearchQuery = %q, want %q", got, want)
	}
	if mapping.Section != "linking" {
		t.Errorf("Section = %q, want linking", mapping.Section)
	}
}

// The bookkeeping tag must never leak into the page's own tags list, even if
// mymind hands it back on a later fetch.
func TestLinkingTagsExcludeDoneTag(t *testing.T) {
	mapping, _ := Lookup("linking")
	page, err := mapping.Build(mymind.Object{
		Title:  "Thing",
		Source: &mymind.Source{URL: "https://example.com"},
		Tags:   []mymind.Tag{{Name: "keep"}, {Name: "Keeped"}, {Name: " js "}, {Name: ""}},
	}, testTime(t))
	if err != nil {
		t.Fatal(err)
	}

	var tags string
	for _, f := range page.Extra {
		if f.Key == "tags" {
			tags = f.Value
		}
	}
	if want := `["keep","js"]`; tags != want {
		t.Errorf("tags = %s, want %s", tags, want)
	}
}

func TestLinkingWithNoTagsRendersEmptyArray(t *testing.T) {
	mapping, _ := Lookup("linking")
	page, err := mapping.Build(mymind.Object{
		Title:  "Thing",
		Source: &mymind.Source{URL: "https://example.com"},
	}, testTime(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range page.Extra {
		if f.Key == "tags" && f.Value != "[]" {
			t.Errorf("tags = %s, want []", f.Value)
		}
	}
}

func TestLinkingRequiresSourceURL(t *testing.T) {
	mapping, _ := Lookup("linking")
	if _, err := mapping.Build(mymind.Object{Title: "A PDF"}, testTime(t)); err == nil {
		t.Fatal("expected an error when the object has no link")
	}
}

// Both sections are independent: a `share` object must not land in linking and
// a `keep` object must not land in sharing.
func TestSectionsDoNotCrossContaminate(t *testing.T) {
	objects := []mymind.Object{
		{ID: "s", Title: "Shared thing", Source: &mymind.Source{URL: "https://example.com/s"},
			Tags: []mymind.Tag{{Name: "share"}}},
		{ID: "k", Title: "Kept thing", Source: &mymind.Source{URL: "https://example.com/k"},
			Tags: []mymind.Tag{{Name: "keep"}}},
	}

	for _, tc := range []struct {
		contentType string
		wantPath    string
		wantTag     string
		otherID     string
	}{
		{"sharing", "content/sharing/20260810_shared_thing.md", "shared", "k"},
		{"linking", "content/linking/kept_thing.md", "keeped", "s"},
	} {
		mapping, _ := Lookup(tc.contentType)
		site := newTestSite(t)
		// The fake returns everything; the mapping must filter on its own tag.
		client := &fakeClient{objects: objects}

		results, err := Run(context.Background(), client, site, mapping, Options{Now: testTime(t)})
		if err != nil {
			t.Fatal(err)
		}
		if Count(results, StatusCreated) != 1 {
			t.Errorf("%s: expected exactly 1 page, got %+v", tc.contentType, results)
		}
		if !site.Exists(tc.wantPath) {
			t.Errorf("%s: %s was not created", tc.contentType, tc.wantPath)
		}
		if _, tagged := client.tagged[tc.otherID]; tagged {
			t.Errorf("%s: tagged object %q, which belongs to the other section", tc.contentType, tc.otherID)
		}
	}
}

func TestLookupAll(t *testing.T) {
	mappings, err := LookupAll("sharing,linking")
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) != 2 || mappings[0].Name != "sharing" || mappings[1].Name != "linking" {
		t.Errorf("LookupAll returned %v, want [sharing linking] in order", names(mappings))
	}

	// Order preserved, repeats and blanks ignored.
	mappings, err = LookupAll(" linking , sharing ,linking, ")
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) != 2 || mappings[0].Name != "linking" {
		t.Errorf("LookupAll returned %v, want [linking sharing]", names(mappings))
	}

	if _, err := LookupAll("sharing,nope"); err == nil {
		t.Error("expected an error for an unknown content type")
	}
	if _, err := LookupAll("  ,  "); err == nil {
		t.Error("expected an error when no content type is given")
	}
}

func names(mappings []Mapping) []string {
	out := make([]string, 0, len(mappings))
	for _, m := range mappings {
		out = append(out, m.Name)
	}
	return out
}

// An entity with no matching tag must lose its brackets and stay plain text —
// Hugo only generates /tags/x/ for terms a page actually carries, so linking
// one of these would 404.
func TestWikiLinksWithoutATagBecomePlainText(t *testing.T) {
	cases := map[string]string{
		"The integration of [[Artificial Intelligence]] into software": "The integration of Artificial Intelligence into software",
		"[[AI]] and [[Large Language Models]] tend toward consensus":   "AI and Large Language Models tend toward consensus",
		"pipe form [[the-target|the label]] here":                      "pipe form the label here",
		"padded [[  Spaced Entity  ]] here":                            "padded Spaced Entity here",
		// The gap this leaves is collapsed later by singleLine — asserted in
		// TestDescriptionAppliesToBothSections.
		"an empty [[]] pair":                              "an empty  pair",
		"a single [bracket] survives":                     "a single [bracket] survives",
		"a markdown [link](https://example.com) survives": "a markdown [link](https://example.com) survives",
	}
	for in, want := range cases {
		if got := resolveWikiLinks(in, nil); got != want {
			t.Errorf("resolveWikiLinks(%q, nil)\n  got  %q\n  want %q", in, got, want)
		}
	}
}

// An entity matching one of the page's tags becomes an anchor to its tag page.
func TestWikiLinksMatchingATagBecomeAnchors(t *testing.T) {
	tags := []string{"Artificial intelligence", "ai", "UX design", "keep"}

	cases := map[string]string{
		// Case differs from the tag; both urlize the same way.
		"about [[Artificial Intelligence]] here": `about <a href="/tags/artificial-intelligence">Artificial Intelligence</a> here`,
		"[[AI]] wins":                            `<a href="/tags/ai">AI</a> wins`,
		"[[UX Design]] matters":                  `<a href="/tags/ux-design">UX Design</a> matters`,
		// Not a tag on this page: plain text.
		"[[Large Language Models]] tend to average": "Large Language Models tend to average",
		// Mixed in one string.
		"[[AI]] and [[judgment]]": `<a href="/tags/ai">AI</a> and judgment`,
	}
	for in, want := range cases {
		if got := resolveWikiLinks(in, tags); got != want {
			t.Errorf("resolveWikiLinks(%q)\n  got  %q\n  want %q", in, got, want)
		}
	}
}

// The description is rendered with safeHTML, so anything mymind sends has to
// be escaped; only our own anchors may be raw.
func TestDescriptionEscapesMymindHTML(t *testing.T) {
	got := resolveWikiLinks(`5 < 6 & "quoted" <script>alert(1)</script> [[AI]]`, []string{"ai"})
	want := `5 &lt; 6 &amp; &#34;quoted&#34; &lt;script&gt;alert(1)&lt;/script&gt; <a href="/tags/ai">AI</a>`
	if got != want {
		t.Errorf("\n  got  %q\n  want %q", got, want)
	}

	// An entity label is escaped too, even when it becomes a link.
	if got := resolveWikiLinks(`[[A & B]]`, []string{"A & B"}); !strings.Contains(got, "A &amp; B") {
		t.Errorf("label was not escaped: %q", got)
	}
}

func TestDescriptionAppliesToBothSections(t *testing.T) {
	// Also pins the whitespace contract: the gap an empty [[]] leaves behind,
	// the newline, and the double space all collapse to single spaces.
	obj := mymind.Object{
		Title:   "Thing",
		Summary: "About [[Artificial Intelligence]] [[]] and\ngood  design.",
		Source:  &mymind.Source{URL: "https://example.com"},
	}
	want := "About Artificial Intelligence and good design."

	for _, name := range []string{"sharing", "linking"} {
		mapping, _ := Lookup(name)
		page, err := mapping.Build(obj, testTime(t))
		if err != nil {
			t.Fatal(err)
		}
		if page.Description != want {
			t.Errorf("%s: description = %q, want %q", name, page.Description, want)
		}
	}
}

// sharing/single.html renders {{ .Description }}, which Go templates escape.
// Pre-escaping here as well would double-escape: "R&D" would reach the page as
// "R&amp;amp;D".
func TestSharingDescriptionIsPlainText(t *testing.T) {
	mapping, _ := Lookup("sharing")
	page, err := mapping.Build(mymind.Object{
		Title:   "Thing",
		Summary: `R&D and "quotes" and 5 < 6, plus [[AI]]`,
		Source:  &mymind.Source{URL: "https://example.com"},
		Tags:    []mymind.Tag{{Name: "share"}, {Name: "ai"}},
	}, testTime(t))
	if err != nil {
		t.Fatal(err)
	}
	want := `R&D and "quotes" and 5 < 6, plus AI`
	if page.Description != want {
		t.Errorf("description = %q, want %q", page.Description, want)
	}
	if strings.Contains(page.Description, "&amp;") || strings.Contains(page.Description, "<a ") {
		t.Errorf("sharing description must be plain text, got %q", page.Description)
	}
}

// A note takes priority over the summary, and is cleaned the same way.
func TestNoteWinsOverSummaryAndIsCleaned(t *testing.T) {
	mapping, _ := Lookup("linking")
	page, err := mapping.Build(mymind.Object{
		Title:   "Thing",
		Summary: "the AI summary",
		Notes:   []mymind.Note{{Content: mymind.Content{Body: "my own [[note]] here"}}},
		Source:  &mymind.Source{URL: "https://example.com"},
	}, testTime(t))
	if err != nil {
		t.Fatal(err)
	}
	if want := "my own note here"; page.Description != want {
		t.Errorf("description = %q, want %q", page.Description, want)
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

// A page is dated by when the object was saved in mymind, not by when the run
// happened — so a run that is a month late still files the page under the day
// it was kept.
func TestPageDateComesFromCreated(t *testing.T) {
	for _, tc := range []struct {
		contentType string
		wantPath    string
	}{
		{"sharing", "content/sharing/20260704_thing.md"},
		{"linking", "content/linking/thing.md"},
	} {
		mapping, _ := Lookup(tc.contentType)
		site := newTestSite(t)
		client := &fakeClient{objects: []mymind.Object{{
			ID: "obj-1", Title: "Thing",
			Source:  &mymind.Source{URL: "https://example.com/x"},
			Tags:    []mymind.Tag{{Name: mapping.SourceTag}},
			Created: "2026-07-04T18:20:00Z",
		}}}

		results, err := Run(context.Background(), client, site, mapping, Options{Now: testTime(t)})
		if err != nil {
			t.Fatal(err)
		}
		if Count(results, StatusCreated) != 1 {
			t.Fatalf("%s: unexpected results: %+v", tc.contentType, results)
		}
		if results[0].Path != tc.wantPath {
			t.Errorf("%s: path = %q, want %q", tc.contentType, results[0].Path, tc.wantPath)
		}

		got, err := os.ReadFile(filepath.Join(site.Root, results[0].Path))
		if err != nil {
			t.Fatal(err)
		}
		// 18:20 UTC read in the run's +02:00 zone.
		if want := "date: 2026-07-04T20:20:00+02:00\n"; !strings.Contains(string(got), want) {
			t.Errorf("%s: front matter missing %q:\n%s", tc.contentType, want, got)
		}
	}
}

// The timestamp arrives in UTC, so anything saved late in the evening Zurich
// time would be filed under the previous day if the zone were ignored.
func TestPageDateUsesTheRunTimezone(t *testing.T) {
	mapping, _ := Lookup("sharing")
	site := newTestSite(t)
	client := &fakeClient{objects: []mymind.Object{{
		ID: "obj-1", Title: "Thing",
		Source:  &mymind.Source{URL: "https://example.com/x"},
		Tags:    []mymind.Tag{{Name: "share"}},
		Created: "2026-07-04T22:30:00Z",
	}}}

	results, err := Run(context.Background(), client, site, mapping, Options{
		Now: testTime(t).In(time.FixedZone("CEST", 2*60*60)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := "content/sharing/20260705_thing.md"; results[0].Path != want {
		t.Errorf("path = %q, want %q", results[0].Path, want)
	}
}

// An object mymind gave no usable timestamp for still gets published, dated by
// the run instead.
func TestPageDateFallsBackToNow(t *testing.T) {
	mapping, _ := Lookup("sharing")
	for name, created := range map[string]string{"missing": "", "unparseable": "4 July 2026"} {
		site := newTestSite(t)
		client := &fakeClient{objects: []mymind.Object{{
			ID: "obj-1", Title: "Thing",
			Source:  &mymind.Source{URL: "https://example.com/x"},
			Tags:    []mymind.Tag{{Name: "share"}},
			Created: created,
		}}}

		results, err := Run(context.Background(), client, site, mapping, Options{Now: testTime(t)})
		if err != nil {
			t.Fatal(err)
		}
		if want := "content/sharing/20260810_thing.md"; results[0].Path != want {
			t.Errorf("%s: path = %q, want %q", name, results[0].Path, want)
		}
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
