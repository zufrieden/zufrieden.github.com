package hugosite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Creative is 10%. Structure and systems are the rest.": "creative_is_10_structure_and_systems_are_the_rest",
		"Training AI with childhood journal entry":             "training_ai_with_childhood_journal_entry",
		"Good Design":              "good_design",
		"  leading and trailing  ": "leading_and_trailing",
		"Café & Crème":             "cafe_and_creme",
		"Über/Straße":              "uber_strasse",
		"!!!":                      "untitled",
		"":                         "untitled",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

// URLize must reproduce the paths Hugo generates under /tags/, or the links in
// a description would 404.
func TestURLize(t *testing.T) {
	cases := map[string]string{
		"Artificial intelligence": "artificial-intelligence",
		"Artificial Intelligence": "artificial-intelligence",
		"UX design":               "ux-design",
		"ai":                      "ai",
		"Business strategy":       "business-strategy",
		"  padded  words  ":       "padded-words",
		"already-hyphenated":      "already-hyphenated",
		"punctuation! removed?":   "punctuation-removed",
		"keeps_underscores":       "keeps_underscores",
		"":                        "",
		"!!!":                     "",
		// Non-ASCII: Hugo percent-encodes these, so refuse rather than guess.
		"Café":      "",
		"日本語":       "",
		"Zürich AG": "",
	}
	for in, want := range cases {
		if got := URLize(in); got != want {
			t.Errorf("URLize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSlugifyTruncatesOnWordBoundary(t *testing.T) {
	got := Slugify(strings.Repeat("alpha beta ", 20))
	if len(got) > maxSlugLen {
		t.Fatalf("slug too long: %d chars (%q)", len(got), got)
	}
	if strings.HasSuffix(got, "_") || strings.HasPrefix(got, "_") {
		t.Fatalf("slug has dangling underscore: %q", got)
	}
}

const archetypeScaffold = `---
title: " Tmp Probe Test"
date: 2026-08-10T12:21:26+02:00
showDate: true
draft: false
description :
---
`

func TestYAMLString(t *testing.T) {
	cases := map[string]string{
		`Structure & systems`: `"Structure & systems"`, // no & HTML escaping
		`He said "hi"`:        `"He said \"hi\""`,
		"two\nlines":          `"two\nlines"`,
		`a < b > c`:           `"a < b > c"`,
		"Café":                `"Café"`,
		"":                    `""`,
	}
	for in, want := range cases {
		if got := YAMLString(in); got != want {
			t.Errorf("YAMLString(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestPatchFrontMatter(t *testing.T) {
	got, err := patchFrontMatter(archetypeScaffold, []Field{
		{Key: "title", Value: YAMLString(`Creative is 10%. Structure: the rest.`)},
		{Key: "date", Value: YAMLRaw("2026-08-10T09:00:00+02:00")},
		{Key: "description", Value: YAMLString("A note\nover two lines")},
		{Key: "link", Value: YAMLString("https://example.com")},
	}, "[https://example.com](https://example.com)")
	if err != nil {
		t.Fatal(err)
	}

	want := `---
title: "Creative is 10%. Structure: the rest."
date: 2026-08-10T09:00:00+02:00
showDate: true
draft: false
description: "A note\nover two lines"
link: "https://example.com"
---
[https://example.com](https://example.com)
`
	if got != want {
		t.Errorf("patchFrontMatter mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// An archetype that lists the same key twice must not yield front matter with
// a duplicate key — the second one has to go.
func TestPatchFrontMatterCollapsesDuplicateKeys(t *testing.T) {
	scaffold := `---
title: "x"
showDate: false
tags: []
link: https://
tags: ["keep"]
description :
---
`
	got, err := patchFrontMatter(scaffold, []Field{
		{Key: "tags", Value: YAMLStringSlice([]string{"keep", "js"})},
		{Key: "link", Value: YAMLString("https://example.com")},
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	if n := strings.Count(got, "tags:"); n != 1 {
		t.Errorf("got %d tags keys, want 1:\n%s", n, got)
	}
	if !strings.Contains(got, `tags: ["keep","js"]`) {
		t.Errorf("the surviving tags line is wrong:\n%s", got)
	}
	// The first occurrence keeps its position in the block.
	if strings.Index(got, "tags:") > strings.Index(got, "link:") {
		t.Errorf("tags should stay at the first declaration's position:\n%s", got)
	}
}

func TestYAMLStringSlice(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{nil, "[]"},
		{[]string{}, "[]"},
		{[]string{"keep"}, `["keep"]`},
		{[]string{"keep", "js"}, `["keep","js"]`},
		{[]string{`quote"inside`}, `["quote\"inside"]`},
	}
	for _, tc := range cases {
		if got := YAMLStringSlice(tc.in); got != tc.want {
			t.Errorf("YAMLStringSlice(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestPatchFrontMatterRejectsMissingDelimiter(t *testing.T) {
	if _, err := patchFrontMatter("no front matter here\n", nil, "body"); err == nil {
		t.Fatal("expected an error for a page without front matter")
	}
}

func TestRenderArchetypeFallback(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "archetypes", "sharing.md"), `---
title: "{{ replace .Name "_" " " | title }}"
date: {{ .Date }}
showDate: true
draft: false
description :
---
`)

	scaffold, err := renderArchetype(root, "content/sharing/20260810_thing.md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(scaffold, "{{") {
		t.Fatalf("template actions survived: %q", scaffold)
	}

	got, err := patchFrontMatter(scaffold, []Field{
		{Key: "title", Value: YAMLString("Thing")},
		{Key: "date", Value: YAMLRaw("2026-08-10T09:00:00+02:00")},
		{Key: "description", Value: YAMLString("")},
	}, "[https://example.com](https://example.com)")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`title: "Thing"`, "date: 2026-08-10T09:00:00+02:00", "showDate: true", `description: ""`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestCreatePageAndAvailablePath(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "config.toml"), "baseURL = \"https://example.com/\"\n")
	mustWrite(t, filepath.Join(root, "archetypes", "sharing.md"), `---
title: "{{ .Name }}"
date: {{ .Date }}
showDate: true
draft: false
description :
---
`)

	// Force the archetype fallback so the test does not depend on a hugo binary.
	site := &Site{Root: root, HugoBin: "hugo-does-not-exist"}

	relPath := "content/sharing/20260810_thing.md"
	if got := site.AvailablePath(relPath); got != relPath {
		t.Fatalf("AvailablePath = %q, want %q", got, relPath)
	}

	cleanup, err := site.CreatePage(relPath, []Field{
		{Key: "title", Value: YAMLString("Thing")},
		{Key: "date", Value: YAMLRaw("2026-08-10T09:00:00+02:00")},
		{Key: "description", Value: YAMLString("desc")},
	}, "[https://example.com](https://example.com)")
	if err != nil {
		t.Fatal(err)
	}
	if !site.Exists(relPath) {
		t.Fatal("page was not written")
	}

	if got, want := site.AvailablePath(relPath), "content/sharing/20260810_thing_2.md"; got != want {
		t.Errorf("AvailablePath on collision = %q, want %q", got, want)
	}
	if _, err := site.CreatePage(relPath, nil, ""); err == nil {
		t.Error("expected CreatePage to refuse an existing file")
	}

	cleanup()
	if site.Exists(relPath) {
		t.Error("cleanup did not remove the page")
	}
}

func TestBundlePath(t *testing.T) {
	cases := map[string]string{
		"content/sharing/20260810_thing.md": "content/sharing/20260810_thing/index.md",
		"content/linking/thing.md":          "content/linking/thing/index.md",
	}
	for in, want := range cases {
		if got := BundlePath(in); got != want {
			t.Errorf("BundlePath(%q) = %q, want %q", in, got, want)
		}
	}
}

// Hugo renders `thing.md` and `thing/index.md` at the same URL and refuses to
// build a site holding both, so the two forms have to compete for one name —
// whichever is asked for.
func TestAvailablePathTreatsAPageAndItsBundleAsOneSlot(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "config.toml"), "baseURL = \"https://example.com/\"\n")
	site := &Site{Root: root}

	// A bundle is numbered on its directory: "thing_2/index.md", never
	// "thing/index_2.md", which Hugo would not read as a bundle at all.
	bundle := "content/sharing/20260810_thing/index.md"
	if got := site.AvailablePath(bundle); got != bundle {
		t.Fatalf("AvailablePath = %q, want %q", got, bundle)
	}

	mustWrite(t, filepath.Join(root, "content", "sharing", "20260810_thing.md"), "---\n---\n")
	if got, want := site.AvailablePath(bundle), "content/sharing/20260810_thing_2/index.md"; got != want {
		t.Errorf("bundle competing with a page = %q, want %q", got, want)
	}

	// And the other way round: a plain page must step aside for an existing
	// bundle of the same name.
	mustWrite(t, filepath.Join(root, "content", "sharing", "20260811_other", "index.md"), "---\n---\n")
	page := "content/sharing/20260811_other.md"
	if got, want := site.AvailablePath(page), "content/sharing/20260811_other_2.md"; got != want {
		t.Errorf("page competing with a bundle = %q, want %q", got, want)
	}

	// A directory left half-written by an interrupted run still holds the slot:
	// the next attempt writes a fresh bundle rather than reusing the debris.
	mustWrite(t, filepath.Join(root, "content", "sharing", "20260812_orphan", "1.jpg"), "not-a-page")
	if got, want := site.AvailablePath("content/sharing/20260812_orphan/index.md"), "content/sharing/20260812_orphan_2/index.md"; got != want {
		t.Errorf("bundle competing with an orphan directory = %q, want %q", got, want)
	}

	// The dry run's "already handed out" set works on both forms.
	taken := map[string]bool{"content/sharing/20260813_x.md": true}
	if got, want := site.AvailablePathExcluding("content/sharing/20260813_x/index.md", taken), "content/sharing/20260813_x_2/index.md"; got != want {
		t.Errorf("AvailablePathExcluding = %q, want %q", got, want)
	}
}

func TestFrontMatterValues(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "config.toml"), "baseURL = \"https://example.com/\"\n")

	// A page written by this tool.
	mustWrite(t, filepath.Join(root, "content", "sharing", "20260809_sunset.md"), `---
title: "Sunset in the Baltic Sea"
mastodon_id: "117066375180847363"
---
Sunset in the Baltic Sea
`)
	// A page bundle, the shape the imported tweets use.
	mustWrite(t, filepath.Join(root, "content", "sharing", "20090112_1114006814", "index.md"), `---
title: "Registering Twitter"
mastodon_id : 116380224548967393
---
`)
	// Pages that must not contribute: no such key, no front matter at all, and
	// a key that only appears in the body.
	mustWrite(t, filepath.Join(root, "content", "sharing", "20260810_mymind.md"), `---
title: "From mymind"
description: "x"
---
`)
	mustWrite(t, filepath.Join(root, "content", "sharing", "notes.txt"), "mastodon_id: \"999\"\n")
	mustWrite(t, filepath.Join(root, "content", "sharing", "20260811_body.md"), `---
title: "Body mention"
---
mastodon_id: "888"
`)

	site := &Site{Root: root}
	got, err := site.FrontMatterValues("sharing", "mastodon_id")
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"117066375180847363": "content/sharing/20260809_sunset.md",
		"116380224548967393": "content/sharing/20090112_1114006814/index.md",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d values, want %d: %v", len(got), len(want), got)
	}
	for id, path := range want {
		if got[id] != path {
			t.Errorf("%s -> %q, want %q", id, got[id], path)
		}
	}
}

// A section that has never been published to is normal on a first run, not an
// error.
func TestFrontMatterValuesOnMissingSection(t *testing.T) {
	site := &Site{Root: t.TempDir()}
	got, err := site.FrontMatterValues("sharing", "mastodon_id")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestUnquoteYAML(t *testing.T) {
	cases := map[string]string{
		`"117066375180847363"`: "117066375180847363",
		`117066375180847363`:   "117066375180847363",
		`'single'`:             "single",
		`"with \"quotes\""`:    `with "quotes"`,
		`""`:                   "",
		``:                     "",
	}
	for in, want := range cases {
		if got := unquoteYAML(in); got != want {
			t.Errorf("unquoteYAML(%s) = %q, want %q", in, got, want)
		}
	}
}

func TestWriteFile(t *testing.T) {
	site := &Site{Root: t.TempDir()}
	relPath := "static/images/mastodon/117066375180847363/1.jpeg"

	cleanup, err := site.WriteFile(relPath, []byte("jpeg-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if !site.Exists(relPath) {
		t.Fatal("file was not written")
	}
	if _, err := site.WriteFile(relPath, []byte("other")); err == nil {
		t.Error("expected WriteFile to refuse an existing file")
	}

	cleanup()
	if site.Exists(relPath) {
		t.Error("cleanup did not remove the file")
	}
}

func TestDiscover(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "config.toml"), "baseURL = \"https://example.com/\"\n")
	nested := filepath.Join(root, "tools", "mymind-sync")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	site, err := Discover(nested, "hugo")
	if err != nil {
		t.Fatal(err)
	}
	// t.TempDir can hand back a symlinked path on macOS; compare resolved paths.
	gotRoot, _ := filepath.EvalSymlinks(site.Root)
	wantRoot, _ := filepath.EvalSymlinks(root)
	if gotRoot != wantRoot {
		t.Errorf("Discover root = %q, want %q", gotRoot, wantRoot)
	}

	if _, err := Discover(t.TempDir(), "hugo"); err == nil {
		t.Error("expected Discover to fail outside a Hugo site")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
