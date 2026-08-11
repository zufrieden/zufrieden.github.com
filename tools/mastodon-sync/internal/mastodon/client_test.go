package mastodon

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sampleFeed is trimmed from a real https://social.coop/@zufrieden.rss response,
// keeping every shape the tool has to cope with: an attachment, Mastodon's
// three-span URL rendering, a hashtag link, <br>, and multiple paragraphs.
const sampleFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:webfeeds="http://webfeeds.org/rss/1.0" xmlns:media="http://search.yahoo.com/mrss/">
  <channel>
    <title>Marc Friederich</title>
    <link>https://social.coop/@zufrieden</link>
    <item>
      <guid isPermaLink="true">https://social.coop/@zufrieden/117066375180847363</guid>
      <link>https://social.coop/@zufrieden/117066375180847363</link>
      <pubDate>Sun, 09 Aug 2026 15:58:45 +0000</pubDate>
      <description>&lt;p&gt;Sunset in the Baltic Sea&lt;/p&gt;</description>
      <media:content url="https://media.example.com/files/original/2f8b3a8e174b5e81.jpeg" type="image/jpeg" fileSize="341748" medium="image">
        <media:rating scheme="urn:simple">nonadult</media:rating>
      </media:content>
    </item>
    <item>
      <guid isPermaLink="true">https://social.coop/@zufrieden/116380224548967393</guid>
      <link>https://social.coop/@zufrieden/116380224548967393</link>
      <pubDate>Fri, 10 Apr 2026 11:41:37 +0000</pubDate>
      <description>&lt;p&gt;&amp;gt; Better citizens of the web.&lt;br /&gt;&lt;a href="https://thehistoryoftheweb.com/prepping-for-the-endgame/" target="_blank" rel="nofollow noopener" translate="no"&gt;&lt;span class="invisible"&gt;https://&lt;/span&gt;&lt;span class="ellipsis"&gt;thehistoryoftheweb.com/preppin&lt;/span&gt;&lt;span class="invisible"&gt;g-for-the-endgame/&lt;/span&gt;&lt;/a&gt;&lt;/p&gt;</description>
    </item>
    <item>
      <guid isPermaLink="true">https://social.coop/@zufrieden/115401564163879776</guid>
      <link>https://social.coop/@zufrieden/115401564163879776</link>
      <pubDate>Sun, 19 Oct 2025 15:35:25 +0000</pubDate>
      <description>&lt;p&gt;&lt;a href="https://social.coop/tags/silentSunday" class="mention hashtag" rel="tag"&gt;#&lt;span&gt;silentSunday&lt;/span&gt;&lt;/a&gt;&lt;/p&gt;&lt;p&gt;Two shots&lt;/p&gt;</description>
      <media:content url="https://media.example.com/files/original/9af616f96e0a58fc.jpeg" type="image/jpeg" medium="image"/>
      <media:content url="https://media.example.com/files/original/949df8ec189138c8.jpeg" type="image/jpeg" medium="image"/>
    </item>
  </channel>
</rss>
`

func TestParseFeed(t *testing.T) {
	posts, err := parseFeed([]byte(sampleFeed))
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 3 {
		t.Fatalf("got %d posts, want 3", len(posts))
	}

	first := posts[0]
	if first.ID != "117066375180847363" {
		t.Errorf("ID = %q", first.ID)
	}
	if first.URL != "https://social.coop/@zufrieden/117066375180847363" {
		t.Errorf("URL = %q", first.URL)
	}
	if got, want := first.Published.UTC().Format("2006-01-02T15:04:05Z"), "2026-08-09T15:58:45Z"; got != want {
		t.Errorf("Published = %s, want %s", got, want)
	}
	if first.Text() != "Sunset in the Baltic Sea" {
		t.Errorf("Text = %q", first.Text())
	}
	if len(first.Media) != 1 {
		t.Fatalf("got %d attachments, want 1", len(first.Media))
	}
	if first.Media[0].URL != "https://media.example.com/files/original/2f8b3a8e174b5e81.jpeg" {
		t.Errorf("media URL = %q", first.Media[0].URL)
	}
	if first.Media[0].Type != "image/jpeg" || first.Media[0].Medium != "image" {
		t.Errorf("media type/medium = %q/%q", first.Media[0].Type, first.Media[0].Medium)
	}

	if n := len(posts[2].Media); n != 2 {
		t.Errorf("third post has %d attachments, want 2", n)
	}
}

// Mastodon splits a URL across three spans so its CSS can shorten it. Joining
// the span text is what recovers the full URL for the plain-text rendering,
// while the href is what the markdown link must point at.
func TestParseFeedRendersMastodonLinks(t *testing.T) {
	posts, err := parseFeed([]byte(sampleFeed))
	if err != nil {
		t.Fatal(err)
	}

	link := posts[1]
	wantText := "> Better citizens of the web.\n" +
		"https://thehistoryoftheweb.com/prepping-for-the-endgame/"
	if got := link.Text(); got != wantText {
		t.Errorf("Text =\n%q\nwant\n%q", got, wantText)
	}
	wantMarkdown := "> Better citizens of the web.\\\n" +
		"[https://thehistoryoftheweb.com/prepping-for-the-endgame/](https://thehistoryoftheweb.com/prepping-for-the-endgame/)"
	if got := link.Markdown(); got != wantMarkdown {
		t.Errorf("Markdown =\n%q\nwant\n%q", got, wantMarkdown)
	}

	hashtag := posts[2]
	if got, want := hashtag.Text(), "#silentSunday\n\nTwo shots"; got != want {
		t.Errorf("Text = %q, want %q", got, want)
	}
	wantHashtag := "[#silentSunday](https://social.coop/tags/silentSunday)\n\nTwo shots"
	if got := hashtag.Markdown(); got != wantHashtag {
		t.Errorf("Markdown = %q, want %q", got, wantHashtag)
	}
}

func TestRender(t *testing.T) {
	cases := []struct {
		name         string
		html         string
		wantText     string
		wantMarkdown string
	}{{
		name:         "entities are decoded",
		html:         `<p>Tabs &amp; spaces &lt;script&gt; &quot;quoted&quot; &#39;apos&#39;</p>`,
		wantText:     `Tabs & spaces <script> "quoted" 'apos'`,
		wantMarkdown: `Tabs & spaces <script> "quoted" 'apos'`,
	}, {
		name:         "paragraphs are separated by a blank line",
		html:         `<p>One</p><p>Two</p><p>Three</p>`,
		wantText:     "One\n\nTwo\n\nThree",
		wantMarkdown: "One\n\nTwo\n\nThree",
	}, {
		name:         "line breaks are kept",
		html:         `<p>One<br />Two</p>`,
		wantText:     "One\nTwo",
		wantMarkdown: "One\\\nTwo",
	}, {
		name:         "leading whitespace is trimmed so it cannot become a code block",
		html:         `<p>Quote</p><p>     indented four or more</p>`,
		wantText:     "Quote\n\nindented four or more",
		wantMarkdown: "Quote\n\nindented four or more",
	}, {
		name:         "a mention keeps its label and links to the profile",
		html:         `<p>Hi <span class="h-card"><a href="https://social.coop/@someone" class="u-url mention">@<span>someone</span></a></span>!</p>`,
		wantText:     "Hi @someone!",
		wantMarkdown: "Hi [@someone](https://social.coop/@someone)!",
	}, {
		name:         "brackets in a label are escaped",
		html:         `<p><a href="https://example.com/x">see [this]</a></p>`,
		wantText:     "see [this]",
		wantMarkdown: `[see \[this\]](https://example.com/x)`,
	}, {
		name:         "a destination with parentheses is angle-bracketed",
		html:         `<p><a href="https://en.wikipedia.org/wiki/Go_(programming_language)">Go</a></p>`,
		wantText:     "Go",
		wantMarkdown: `[Go](<https://en.wikipedia.org/wiki/Go_(programming_language)>)`,
	}, {
		name:         "a link with no text falls back to the bare URL",
		html:         `<p><a href="https://example.com/x"></a></p>`,
		wantText:     "https://example.com/x",
		wantMarkdown: "<https://example.com/x>",
	}, {
		name:         "an unclosed anchor does not swallow the rest",
		html:         `<p><a href="https://example.com">label`,
		wantText:     "label",
		wantMarkdown: "[label](https://example.com)",
	}, {
		name:         "a stray less-than is text",
		html:         `<p>3 < 4 and 5 > 4</p>`,
		wantText:     "3 < 4 and 5 > 4",
		wantMarkdown: "3 < 4 and 5 > 4",
	}, {
		name:         "unknown tags are dropped but their text kept",
		html:         `<p>An <em>emphasis</em> and <strong>strength</strong></p>`,
		wantText:     "An emphasis and strength",
		wantMarkdown: "An emphasis and strength",
	}, {
		name:         "single-quoted attributes are read",
		html:         `<p><a href='https://example.com/x'>x</a></p>`,
		wantText:     "x",
		wantMarkdown: "[x](https://example.com/x)",
	}, {
		name:         "an empty post renders empty",
		html:         ``,
		wantText:     "",
		wantMarkdown: "",
	}}

	for _, tc := range cases {
		post := Post{HTML: tc.html}
		if got := post.Text(); got != tc.wantText {
			t.Errorf("%s: Text = %q, want %q", tc.name, got, tc.wantText)
		}
		if got := post.Markdown(); got != tc.wantMarkdown {
			t.Errorf("%s: Markdown = %q, want %q", tc.name, got, tc.wantMarkdown)
		}
	}
}

// A post whose guid is not a status permalink cannot be recorded as published,
// so it must be dropped rather than republished on every run.
func TestParseFeedDropsPostsWithoutAStatusID(t *testing.T) {
	feed := `<rss version="2.0"><channel>
	  <item><guid>not-a-url</guid><link></link><description>x</description></item>
	  <item><guid isPermaLink="true">https://social.coop/@zufrieden/</guid><description>y</description></item>
	  <item><guid isPermaLink="true">https://social.coop/@zufrieden/123</guid><description>z</description></item>
	</channel></rss>`

	posts, err := parseFeed([]byte(feed))
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 || posts[0].ID != "123" {
		t.Fatalf("got %+v, want just the post with id 123", posts)
	}
}

// The guid is the preferred source of the id, but the link is a usable fallback.
func TestParseFeedFallsBackToLink(t *testing.T) {
	feed := `<rss version="2.0"><channel><item>
	  <guid>tag:social.coop,2026:objectId=99:objectType=Status</guid>
	  <link>https://social.coop/@zufrieden/456</link>
	  <description>x</description>
	</item></channel></rss>`

	posts, err := parseFeed([]byte(feed))
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 || posts[0].ID != "456" {
		t.Fatalf("got %+v, want id 456", posts)
	}
	if posts[0].URL != "https://social.coop/@zufrieden/456" {
		t.Errorf("URL = %q, want the link", posts[0].URL)
	}
}

func TestStatusID(t *testing.T) {
	cases := map[string]string{
		"https://social.coop/@zufrieden/117066375180847363":  "117066375180847363",
		"https://social.coop/@zufrieden/117066375180847363/": "117066375180847363",
		"":                                        "",
		"not-a-url":                               "",
		"https://social.coop/@zufrieden":          "",
		"https://social.coop/@zufrieden/":         "",
		"https://social.coop/@zufrieden/abc":      "",
		"https://social.coop/@zufrieden/12a34":    "",
		"https://social.coop/@zufrieden/../../up": "",
	}
	for in, want := range cases {
		if got := statusID(in); got != want {
			t.Errorf("statusID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParsePubDate(t *testing.T) {
	cases := map[string]string{
		"Sun, 09 Aug 2026 15:58:45 +0000": "2026-08-09T15:58:45Z",
		"Sun, 09 Aug 2026 15:58:45 GMT":   "2026-08-09T15:58:45Z",
		"2026-08-09T17:58:45+02:00":       "2026-08-09T15:58:45Z",
		"":                                "",
		"last Tuesday":                    "",
	}
	for in, want := range cases {
		got := parsePubDate(in)
		if want == "" {
			if !got.IsZero() {
				t.Errorf("parsePubDate(%q) = %s, want the zero time", in, got)
			}
			continue
		}
		if formatted := got.UTC().Format("2006-01-02T15:04:05Z"); formatted != want {
			t.Errorf("parsePubDate(%q) = %s, want %s", in, formatted, want)
		}
	}
}

func TestMediaFilename(t *testing.T) {
	cases := []struct {
		media Media
		want  string
	}{
		{Media{URL: "https://media.example.com/a/b/original/2f8b3a8e.jpeg", Type: "image/jpeg"}, "1.jpeg"},
		{Media{URL: "https://media.example.com/a/b/original/2f8b3a8e.PNG"}, "1.png"},
		{Media{URL: "https://media.example.com/a/b/original/clip.mp4", Medium: "video"}, "1.mp4"},
		// No extension in the URL: fall back to the MIME subtype.
		{Media{URL: "https://media.example.com/a/b/original/2f8b3a8e", Type: "image/jpeg"}, "1.jpeg"},
		// Nothing usable anywhere, and a subtype that is not a plausible
		// extension: save without one rather than invent a name.
		{Media{URL: "https://media.example.com/a/b/original/2f8b3a8e"}, "1"},
		{Media{URL: "https://media.example.com/x.tar.gz/../../etc/passwd", Type: "application/octet-stream"}, "1"},
		{Media{URL: "https://media.example.com/f.j%2Fpeg", Type: "image/svg+xml"}, "1"},
	}
	for _, tc := range cases {
		if got := tc.media.Filename(1); got != tc.want {
			t.Errorf("Filename for %q (type %q) = %q, want %q", tc.media.URL, tc.media.Type, got, tc.want)
		}
	}
	// The position is what keeps two attachments on one post apart.
	m := Media{URL: "https://media.example.com/a.jpeg"}
	if got := m.Filename(2); got != "2.jpeg" {
		t.Errorf("Filename(2) = %q, want 2.jpeg", got)
	}
}

func TestNewValidatesFeedURL(t *testing.T) {
	for _, bad := range []string{"", "   ", "ftp://social.coop/@zufrieden.rss", "not a url", "/relative.rss"} {
		if _, err := New(bad); err == nil {
			t.Errorf("expected an error for feed URL %q", bad)
		}
	}
	if _, err := New("https://social.coop/@zufrieden.rss"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPostsOverHTTP(t *testing.T) {
	var gotUA, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA, gotPath = r.Header.Get("User-Agent"), r.URL.Path
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		fmt.Fprint(w, sampleFeed)
	}))
	defer server.Close()

	client, err := New(server.URL+"/@zufrieden.rss", WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}

	posts, err := client.Posts(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 3 {
		t.Errorf("got %d posts, want 3", len(posts))
	}
	if gotPath != "/@zufrieden.rss" {
		t.Errorf("path = %q", gotPath)
	}
	if gotUA != DefaultUA {
		t.Errorf("User-Agent = %q, want %q", gotUA, DefaultUA)
	}

	// The feed is newest-first, so a limit keeps the newest posts.
	limited, err := client.Posts(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(limited) != 2 || limited[0].ID != "117066375180847363" {
		t.Errorf("limited to %d posts starting %q", len(limited), limited[0].ID)
	}
}

func TestPostsReportsHTTPErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer server.Close()

	client, _ := New(server.URL, WithHTTPClient(server.Client()))
	_, err := client.Posts(context.Background(), 0)
	if err == nil {
		t.Fatal("expected an error for HTTP 429")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error does not mention the status: %v", err)
	}
}

func TestPostsReportsBadXML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "<rss><channel><item>truncated")
	}))
	defer server.Close()

	client, _ := New(server.URL, WithHTTPClient(server.Client()))
	if _, err := client.Posts(context.Background(), 0); err == nil {
		t.Fatal("expected an error for malformed XML")
	}
}

func TestDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok.jpeg":
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write([]byte("jpeg-bytes"))
		case "/gone.jpeg":
			http.Error(w, "gone", http.StatusNotFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, _ := New(server.URL, WithHTTPClient(server.Client()))

	body, contentType, err := client.Download(context.Background(), server.URL+"/ok.jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "jpeg-bytes" {
		t.Errorf("body = %q", body)
	}
	if contentType != "image/jpeg" {
		t.Errorf("content type = %q", contentType)
	}

	if _, _, err := client.Download(context.Background(), server.URL+"/gone.jpeg"); err == nil {
		t.Error("expected an error for a missing attachment")
	}
	if _, _, err := client.Download(context.Background(), "ftp://example.com/x.jpeg"); err == nil {
		t.Error("expected an error for a non-HTTP attachment URL")
	}
}
