package publisher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zufrieden/zufrieden.github.com/tools/internal/hugosite"
	"github.com/zufrieden/zufrieden.github.com/tools/mastodon-sync/internal/mastodon"
)

type fakeClient struct {
	posts        []mastodon.Post
	postsErr     error
	downloadErr  error
	lastLimit    int
	downloaded   []string
	contentTypes map[string]string
}

func (f *fakeClient) Posts(_ context.Context, limit int) ([]mastodon.Post, error) {
	f.lastLimit = limit
	return f.posts, f.postsErr
}

func (f *fakeClient) Download(_ context.Context, url string) ([]byte, string, error) {
	if f.downloadErr != nil {
		return nil, "", f.downloadErr
	}
	f.downloaded = append(f.downloaded, url)
	return []byte("bytes-of-" + url), f.contentTypes[url], nil
}

// newTestSite builds a throwaway Hugo site whose sharing archetype matches this
// repo's, with a hugo binary name that cannot resolve so the archetype fallback
// is used instead of shelling out.
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
	when, err := time.Parse(time.RFC3339, "2026-08-11T09:30:00+02:00")
	if err != nil {
		t.Fatal(err)
	}
	return when
}

func published(t *testing.T, raw string) time.Time {
	t.Helper()
	when, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatal(err)
	}
	return when
}

func TestRunCreatesPage(t *testing.T) {
	site := newTestSite(t)
	client := &fakeClient{posts: []mastodon.Post{{
		ID:        "116380224548967393",
		URL:       "https://social.coop/@zufrieden/116380224548967393",
		Published: published(t, "2026-04-10T11:41:37Z"),
		HTML: `<p>Better citizens of the web.<br /><a href="https://thehistoryoftheweb.com/prepping-for-the-endgame/">` +
			`<span class="invisible">https://</span><span class="ellipsis">thehistoryoftheweb.com/preppin</span>` +
			`<span class="invisible">g-for-the-endgame/</span></a></p>`,
	}}}

	results, err := Run(context.Background(), client, site, Options{Now: testTime(t)})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Status != StatusCreated {
		t.Fatalf("unexpected results: %+v", results)
	}

	wantPath := "content/sharing/20260410_better_citizens_of_the_web.md"
	if results[0].Path != wantPath {
		t.Errorf("path = %q, want %q", results[0].Path, wantPath)
	}

	got, err := os.ReadFile(filepath.Join(site.Root, wantPath))
	if err != nil {
		t.Fatal(err)
	}
	want := `---
title: "Better citizens of the web."
date: 2026-04-10T13:41:37+02:00
showDate: true
draft: false
description: "Better citizens of the web. https://thehistoryoftheweb.com/prepping-for-the-endgame/"
mastodon_id: "116380224548967393"
mastodon_url: "https://social.coop/@zufrieden/116380224548967393"
---
Better citizens of the web.\
[https://thehistoryoftheweb.com/prepping-for-the-endgame/](https://thehistoryoftheweb.com/prepping-for-the-endgame/)
`
	if string(got) != want {
		t.Errorf("page mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// The whole idempotency story: the id written into the page is read back on the
// next run, so a second run over the same feed publishes nothing.
func TestSecondRunPublishesNothing(t *testing.T) {
	site := newTestSite(t)
	posts := []mastodon.Post{{
		ID:        "117066375180847363",
		URL:       "https://social.coop/@zufrieden/117066375180847363",
		Published: published(t, "2026-08-09T15:58:45Z"),
		HTML:      `<p>Sunset in the Baltic Sea</p>`,
	}}

	first, err := Run(context.Background(), &fakeClient{posts: posts}, site, Options{Now: testTime(t)})
	if err != nil {
		t.Fatal(err)
	}
	if Count(first, StatusCreated) != 1 {
		t.Fatalf("first run: %+v", first)
	}

	second, err := Run(context.Background(), &fakeClient{posts: posts}, site, Options{Now: testTime(t)})
	if err != nil {
		t.Fatal(err)
	}
	if Count(second, StatusCreated) != 0 || Count(second, StatusSkipped) != 1 {
		t.Fatalf("second run: %+v", second)
	}
	if !strings.Contains(second[0].Reason, "already published as content/sharing/20260809_sunset_in_the_baltic_sea.md") {
		t.Errorf("reason = %q", second[0].Reason)
	}

	// And exactly one page exists.
	entries, err := os.ReadDir(filepath.Join(site.Root, "content", "sharing"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("content/sharing holds %d entries, want 1", len(entries))
	}
}

// A page written by hand (or by an earlier tool) carrying the id is respected
// just the same — the front matter is the record, whatever wrote it.
func TestHandWrittenPageSuppressesAPost(t *testing.T) {
	site := newTestSite(t)
	write(t, filepath.Join(site.Root, "content", "sharing", "i_did_this_myself.md"), `---
title: "I did this myself"
mastodon_id: "117066375180847363"
---
`)

	client := &fakeClient{posts: []mastodon.Post{{
		ID:   "117066375180847363",
		URL:  "https://social.coop/@zufrieden/117066375180847363",
		HTML: `<p>Sunset in the Baltic Sea</p>`,
	}}}
	results, err := Run(context.Background(), client, site, Options{Now: testTime(t)})
	if err != nil {
		t.Fatal(err)
	}
	if Count(results, StatusSkipped) != 1 {
		t.Fatalf("unexpected results: %+v", results)
	}
	if results[0].Reason != "already published as content/sharing/i_did_this_myself.md" {
		t.Errorf("reason = %q", results[0].Reason)
	}
}

func TestMediaIsDownloadedAndLinked(t *testing.T) {
	site := newTestSite(t)
	client := &fakeClient{posts: []mastodon.Post{{
		ID:        "115401564163879776",
		URL:       "https://social.coop/@zufrieden/115401564163879776",
		Published: published(t, "2025-10-19T15:35:25Z"),
		HTML:      `<p><a href="https://social.coop/tags/silentSunday">#<span>silentSunday</span></a></p>`,
		Media: []mastodon.Media{
			{URL: "https://media.example.com/a/9af616f96e0a58fc.jpeg", Type: "image/jpeg", Medium: "image"},
			{URL: "https://media.example.com/a/949df8ec189138c8.jpeg", Type: "image/jpeg", Medium: "image"},
		},
	}}}

	results, err := Run(context.Background(), client, site, Options{Now: testTime(t)})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Status != StatusCreated {
		t.Fatalf("unexpected results: %+v", results)
	}

	for i, want := range []string{
		"static/images/mastodon/115401564163879776/1.jpeg",
		"static/images/mastodon/115401564163879776/2.jpeg",
	} {
		if !site.Exists(want) {
			t.Errorf("%s was not saved", want)
		}
		got, err := os.ReadFile(filepath.Join(site.Root, filepath.FromSlash(want)))
		if err != nil {
			t.Fatal(err)
		}
		if wantBody := "bytes-of-" + client.posts[0].Media[i].URL; string(got) != wantBody {
			t.Errorf("%s holds %q, want %q", want, got, wantBody)
		}
	}

	page, err := os.ReadFile(filepath.Join(site.Root, results[0].Path))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"![#silentSunday](/images/mastodon/115401564163879776/1.jpeg)",
		"![#silentSunday](/images/mastodon/115401564163879776/2.jpeg)",
	} {
		if !strings.Contains(string(page), want) {
			t.Errorf("page is missing %q:\n%s", want, page)
		}
	}
}

// A post is published whole or not at all: if an attachment cannot be fetched
// the page is not written, nothing is left behind, and — because the id is only
// recorded on a page — the next run tries again.
func TestFailedDownloadLeavesNothingBehind(t *testing.T) {
	site := newTestSite(t)
	client := &fakeClient{
		downloadErr: errors.New("404 from the media CDN"),
		posts: []mastodon.Post{{
			ID:        "115401564163879776",
			Published: published(t, "2025-10-19T15:35:25Z"),
			HTML:      `<p>Two shots</p>`,
			Media:     []mastodon.Media{{URL: "https://media.example.com/a/gone.jpeg", Type: "image/jpeg"}},
		}},
	}

	results, err := Run(context.Background(), client, site, Options{Now: testTime(t)})
	if err != nil {
		t.Fatal(err)
	}
	if Count(results, StatusFailed) != 1 {
		t.Fatalf("unexpected results: %+v", results)
	}
	if site.Exists("content/sharing/20251019_two_shots.md") {
		t.Error("the page was written despite the failed download")
	}
	if site.Exists("static/images/mastodon/115401564163879776") {
		t.Error("attachment directory was left behind")
	}
}

// Debris from a run killed between downloading attachments and writing the page
// must not wedge every later run.
func TestLeftoverMediaIsCleared(t *testing.T) {
	site := newTestSite(t)
	write(t, filepath.Join(site.Root, "static", "images", "mastodon", "115401564163879776", "1.jpeg"), "truncated")

	client := &fakeClient{posts: []mastodon.Post{{
		ID:        "115401564163879776",
		Published: published(t, "2025-10-19T15:35:25Z"),
		HTML:      `<p>Two shots</p>`,
		Media:     []mastodon.Media{{URL: "https://media.example.com/a/x.jpeg", Type: "image/jpeg"}},
	}}}

	results, err := Run(context.Background(), client, site, Options{Now: testTime(t)})
	if err != nil {
		t.Fatal(err)
	}
	if Count(results, StatusCreated) != 1 {
		t.Fatalf("unexpected results: %+v", results)
	}
	got, err := os.ReadFile(filepath.Join(site.Root, "static/images/mastodon/115401564163879776/1.jpeg"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "bytes-of-https://media.example.com/a/x.jpeg" {
		t.Errorf("the truncated leftover survived: %q", got)
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	site := newTestSite(t)
	client := &fakeClient{posts: []mastodon.Post{{
		ID:        "117066375180847363",
		Published: published(t, "2026-08-09T15:58:45Z"),
		HTML:      `<p>Sunset in the Baltic Sea</p>`,
		Media:     []mastodon.Media{{URL: "https://media.example.com/a/x.jpeg", Type: "image/jpeg"}},
	}}}

	results, err := Run(context.Background(), client, site, Options{Now: testTime(t), DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if Count(results, StatusPlanned) != 1 {
		t.Fatalf("unexpected results: %+v", results)
	}
	if want := "content/sharing/20260809_sunset_in_the_baltic_sea.md"; results[0].Path != want {
		t.Errorf("path = %q, want %q", results[0].Path, want)
	}
	if site.Exists(results[0].Path) {
		t.Error("dry run wrote a page")
	}
	if site.Exists("static/images/mastodon/117066375180847363") {
		t.Error("dry run downloaded attachments")
	}
	if len(client.downloaded) != 0 {
		t.Errorf("dry run fetched %v", client.downloaded)
	}
}

// Two posts on the same day whose text slugifies identically must not overwrite
// each other, and the suffix should follow the order they were posted in.
//
// The dry run has to agree with the real run here: it writes nothing, so it can
// only tell the two apart by remembering the paths it has already handed out.
func TestSameDayCollisionGetsSuffix(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		site := newTestSite(t)
		// Feed order is newest first.
		client := &fakeClient{posts: []mastodon.Post{
			{ID: "2", Published: published(t, "2026-08-09T18:00:00Z"), HTML: `<p>Same words</p>`},
			{ID: "1", Published: published(t, "2026-08-09T09:00:00Z"), HTML: `<p>Same words</p>`},
		}}

		results, err := Run(context.Background(), client, site, Options{Now: testTime(t), DryRun: dryRun})
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 2 {
			t.Fatalf("dryRun=%v: unexpected results: %+v", dryRun, results)
		}
		if results[0].PostID != "1" || results[0].Path != "content/sharing/20260809_same_words.md" {
			t.Errorf("dryRun=%v: first = %q at %q, want post 1 unsuffixed", dryRun, results[0].PostID, results[0].Path)
		}
		if results[1].PostID != "2" || results[1].Path != "content/sharing/20260809_same_words_2.md" {
			t.Errorf("dryRun=%v: second = %q at %q, want post 2 suffixed", dryRun, results[1].PostID, results[1].Path)
		}
	}
}

// One bad post must not stop the rest of the batch.
func TestOneFailureDoesNotStopTheBatch(t *testing.T) {
	site := newTestSite(t)
	client := &fakeClient{
		downloadErr: errors.New("media CDN is down"),
		posts: []mastodon.Post{
			{ID: "2", Published: published(t, "2026-08-09T18:00:00Z"), HTML: `<p>Has a picture</p>`,
				Media: []mastodon.Media{{URL: "https://media.example.com/a/x.jpeg"}}},
			{ID: "1", Published: published(t, "2026-08-09T09:00:00Z"), HTML: `<p>Text only</p>`},
		},
	}

	results, err := Run(context.Background(), client, site, Options{Now: testTime(t)})
	if err != nil {
		t.Fatal(err)
	}
	if Count(results, StatusCreated) != 1 || Count(results, StatusFailed) != 1 {
		t.Fatalf("unexpected results: %+v", results)
	}
	if !site.Exists("content/sharing/20260809_text_only.md") {
		t.Error("the text-only post should still have been published")
	}
}

func TestLimitIsPassedThrough(t *testing.T) {
	client := &fakeClient{}
	if _, err := Run(context.Background(), client, newTestSite(t), Options{Now: testTime(t), Limit: 5}); err != nil {
		t.Fatal(err)
	}
	if client.lastLimit != 5 {
		t.Errorf("limit = %d, want 5", client.lastLimit)
	}
}

func TestFeedErrorAborts(t *testing.T) {
	client := &fakeClient{postsErr: errors.New("network down")}
	if _, err := Run(context.Background(), client, newTestSite(t), Options{Now: testTime(t)}); err == nil {
		t.Fatal("expected the feed error to abort the run")
	}
}

// A post the feed gave no date for still gets published, dated by the run.
func TestMissingDateFallsBackToNow(t *testing.T) {
	site := newTestSite(t)
	client := &fakeClient{posts: []mastodon.Post{{ID: "1", HTML: `<p>No date</p>`}}}

	results, err := Run(context.Background(), client, site, Options{Now: testTime(t)})
	if err != nil {
		t.Fatal(err)
	}
	if want := "content/sharing/20260811_no_date.md"; results[0].Path != want {
		t.Errorf("path = %q, want %q", results[0].Path, want)
	}
}

// The post's timestamp arrives in UTC, so a post made late in the evening Zurich
// time must not be filed under the previous day.
func TestDateUsesTheRunTimezone(t *testing.T) {
	site := newTestSite(t)
	client := &fakeClient{posts: []mastodon.Post{{
		ID:        "1",
		Published: published(t, "2026-08-09T22:30:00Z"),
		HTML:      `<p>Late evening</p>`,
	}}}

	results, err := Run(context.Background(), client, site, Options{
		Now: testTime(t).In(time.FixedZone("CEST", 2*60*60)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := "content/sharing/20260810_late_evening.md"; results[0].Path != want {
		t.Errorf("path = %q, want %q", results[0].Path, want)
	}
}

func TestTitleFor(t *testing.T) {
	cases := []struct {
		name string
		post mastodon.Post
		want string
	}{{
		name: "the first line is the title",
		post: mastodon.Post{HTML: `<p>Sunset in the Baltic Sea</p>`},
		want: "Sunset in the Baltic Sea",
	}, {
		name: "a leading quote marker is dropped",
		post: mastodon.Post{HTML: `<p>&gt; A quoted remark</p>`},
		want: "A quoted remark",
	}, {
		name: "a bare URL is not a title",
		post: mastodon.Post{HTML: `<p><a href="https://example.com/x">https://example.com/x</a><br />What I think of it</p>`},
		want: "What I think of it",
	}, {
		// The commonest shape in the feed: a remark with the link inline.
		name: "an inline URL is dropped from the title",
		post: mastodon.Post{HTML: `<p>Yes <a href="https://hachyderm.io/@thomasfuchs/115270807475513595">` +
			`<span class="invisible">https://</span><span class="ellipsis">hachyderm.io/@thomasfuchs</span>` +
			`<span class="invisible">/115270807475513595</span></a></p>`},
		want: "Yes",
	}, {
		name: "a URL leading the line does not hide the comment after it",
		post: mastodon.Post{HTML: `<p><a href="https://medienbaecker.com/articles/animate-native-lazy-loading">` +
			`https://medienbaecker.com/articles/animate-native-lazy-loading</a> nice trick for lazy images</p>`},
		want: "nice trick for lazy images",
	}, {
		name: "a fully quoted post loses its quote marks",
		post: mastodon.Post{HTML: `<p>&quot;We Are Still the Web&quot;</p>`},
		want: "We Are Still the Web",
	}, {
		name: "typographic quotes count too",
		post: mastodon.Post{HTML: `<p>&#8220;The majority of the infrastructure&#8221;</p>`},
		want: "The majority of the infrastructure",
	}, {
		name: "a quote marker and a quote mark together",
		post: mastodon.Post{HTML: `<p>&gt; &quot;Both markers&quot;</p>`},
		want: "Both markers",
	}, {
		name: "a hashtag-only post keeps its hashtag",
		post: mastodon.Post{HTML: `<p><a href="https://social.coop/tags/silentSunday">#<span>silentSunday</span></a></p>`},
		want: "#silentSunday",
	}, {
		name: "a long post is cut on a word boundary and marked",
		post: mastodon.Post{HTML: `<p>If your app uses 10x more resources than last year for the same functionality, that's regression, not progress.</p>`},
		want: "If your app uses 10x more resources than last year for the same…",
	}, {
		name: "a picture with no words is a photo",
		post: mastodon.Post{Media: []mastodon.Media{{URL: "https://x/y.jpeg", Medium: "image"}}},
		want: "Photo",
	}, {
		name: "a video with no words is a video",
		post: mastodon.Post{Media: []mastodon.Media{{URL: "https://x/y.mp4", Medium: "video"}}},
		want: "Video",
	}, {
		name: "a video identified only by its MIME type",
		post: mastodon.Post{Media: []mastodon.Media{{URL: "https://x/y", Type: "video/mp4"}}},
		want: "Video",
	}, {
		name: "nothing at all still gets a title",
		post: mastodon.Post{},
		want: "Post",
	}, {
		name: "a post that is only a link",
		post: mastodon.Post{HTML: `<p><a href="https://example.com/x">https://example.com/x</a></p>`},
		want: "Post",
	}}

	for _, tc := range cases {
		if got := titleFor(tc.post); got != tc.want {
			t.Errorf("%s: titleFor = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"short", 20, "short"},
		{"exactly twenty chars", 20, "exactly twenty chars"},
		{"a sentence that runs on for a while here", 20, "a sentence that…"},
		// Nowhere sensible to break: cut mid-word rather than return the lot.
		{strings.Repeat("x", 40), 20, strings.Repeat("x", 20) + "…"},
		// Multi-byte text is counted in characters, not bytes.
		{"éééééééééééééééééééééééé", 10, "éééééééééé…"},
	}
	for _, tc := range cases {
		if got := truncate(tc.in, tc.max); got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.in, tc.max, got, tc.want)
		}
	}
}

// A photo post's alt text comes from the derived title, and brackets in it must
// not close the markdown label early.
func TestImageAltTextIsEscaped(t *testing.T) {
	site := newTestSite(t)
	client := &fakeClient{posts: []mastodon.Post{{
		ID:        "1",
		Published: published(t, "2026-08-09T09:00:00Z"),
		HTML:      `<p>Photo [draft]</p>`,
		Media:     []mastodon.Media{{URL: "https://media.example.com/a/x.jpeg", Type: "image/jpeg"}},
	}}}

	results, err := Run(context.Background(), client, site, Options{Now: testTime(t)})
	if err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(site.Root, results[0].Path))
	if err != nil {
		t.Fatal(err)
	}
	if want := `![Photo \[draft\]](/images/mastodon/1/1.jpeg)`; !strings.Contains(string(page), want) {
		t.Errorf("page is missing %q:\n%s", want, page)
	}
}

// The feed sometimes omits the attachment type; the response's Content-Type
// then decides the extension.
func TestExtensionFallsBackToContentType(t *testing.T) {
	site := newTestSite(t)
	client := &fakeClient{
		contentTypes: map[string]string{"https://media.example.com/a/opaque": "image/png"},
		posts: []mastodon.Post{{
			ID:        "1",
			Published: published(t, "2026-08-09T09:00:00Z"),
			HTML:      `<p>Opaque</p>`,
			Media:     []mastodon.Media{{URL: "https://media.example.com/a/opaque"}},
		}},
	}

	if _, err := Run(context.Background(), client, site, Options{Now: testTime(t)}); err != nil {
		t.Fatal(err)
	}
	if !site.Exists("static/images/mastodon/1/1.png") {
		t.Error("the attachment was not saved as 1.png")
	}
}
