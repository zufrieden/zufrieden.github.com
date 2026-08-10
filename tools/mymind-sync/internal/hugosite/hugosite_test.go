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
