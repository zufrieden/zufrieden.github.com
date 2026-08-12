// Package hugosite writes content pages into a Hugo site, going through
// `hugo new` so the section archetype stays the single source of truth for
// front matter.
package hugosite

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// bundleIndex is the filename that turns a directory into a Hugo leaf bundle,
// where the page and the files it carries live together.
const bundleIndex = "index.md"

// BundlePath turns a plain page path into the leaf bundle rendering at the same
// URL: "content/sharing/20260810_thing.md" becomes
// "content/sharing/20260810_thing/index.md". A page needs to be a bundle for
// the files beside it to be page resources, which is what Hugo's image
// processing works on — anything under static/ it can only copy.
func BundlePath(pagePath string) string {
	return strings.TrimSuffix(pagePath, path.Ext(pagePath)) + "/" + bundleIndex
}

var configFilenames = []string{
	"hugo.toml", "hugo.yaml", "hugo.yml", "hugo.json",
	"config.toml", "config.yaml", "config.yml", "config.json",
}

type Site struct {
	// Root is the absolute path of the Hugo project.
	Root string
	// HugoBin is the `hugo` executable to shell out to. When it can't be
	// found, pages are rendered straight from the archetype instead.
	HugoBin string
}

// Discover walks up from start looking for a Hugo config file.
func Discover(start, hugoBin string) (*Site, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	for {
		for _, name := range configFilenames {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				return &Site{Root: dir, HugoBin: hugoBin}, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("no Hugo config found in %s or any parent directory", start)
		}
		dir = parent
	}
}

// Exists reports whether a content page already lives at relPath
// (e.g. "content/sharing/20260810_something.md").
func (s *Site) Exists(relPath string) bool {
	_, err := os.Stat(filepath.Join(s.Root, filepath.FromSlash(relPath)))
	return err == nil
}

// AvailablePath returns relPath, or the first free `_2`, `_3`, ... variant of
// it. Two items shared on the same day can slugify to the same name.
func (s *Site) AvailablePath(relPath string) string {
	return s.AvailablePathExcluding(relPath, nil)
}

// AvailablePathExcluding is AvailablePath, additionally treating every path in
// taken as occupied. It exists for a dry run, where nothing is written and so
// two same-day, same-slug items would otherwise both be reported at the very
// same path — making the preview disagree with what a real run would do.
//
// A page and the leaf bundle of the same name occupy one slot, not two: Hugo
// renders `sharing/thing.md` and `sharing/thing/index.md` at the same URL and
// refuses to build a site containing both. So the numbering is applied to the
// stem the two forms share, and both forms are checked before a name counts as
// free.
func (s *Site) AvailablePathExcluding(relPath string, taken map[string]bool) string {
	stem, suffix := pageSlot(relPath)

	free := func(stem string) bool {
		for _, form := range slotForms(stem, suffix) {
			if taken[form] || s.Exists(form) {
				return false
			}
		}
		return true
	}

	if free(stem) {
		return stem + suffix
	}
	for n := 2; n < 100; n++ {
		candidate := fmt.Sprintf("%s_%d", stem, n)
		if free(candidate) {
			return candidate + suffix
		}
	}
	return relPath
}

// pageSlot splits a path into the part a numbered variant is built from and the
// part that identifies its form — ".md" for a plain page, "/index.md" for a
// leaf bundle. A bundle is numbered on its directory, so the suffix stays
// "index.md" rather than becoming "index_2.md", which Hugo would not recognise
// as a bundle at all.
func pageSlot(relPath string) (stem, suffix string) {
	if path.Base(relPath) == bundleIndex {
		return path.Dir(relPath), "/" + bundleIndex
	}
	ext := path.Ext(relPath)
	return strings.TrimSuffix(relPath, ext), ext
}

// slotForms lists everything that would occupy the slot named by stem. For a
// page that is the flat file and the bundle; for a bundle it is the whole
// directory, which also rejects one left half-written by an interrupted run.
func slotForms(stem, suffix string) []string {
	switch suffix {
	case "/" + bundleIndex:
		return []string{stem, stem + ".md"}
	case ".md":
		return []string{stem + ".md", stem + "/" + bundleIndex}
	default:
		return []string{stem + suffix}
	}
}

// CreatePage scaffolds relPath from its section archetype, then overrides the
// given front matter fields and writes body. It refuses to touch an existing
// file. The returned cleanup function removes the page again — used to roll
// back when the follow-up tagging call fails.
func (s *Site) CreatePage(relPath string, fields []Field, body string) (cleanup func(), err error) {
	if s.Exists(relPath) {
		return nil, fmt.Errorf("%s already exists", relPath)
	}

	absPath := filepath.Join(s.Root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, err
	}

	scaffolded, err := s.scaffold(relPath)
	if err != nil {
		return nil, err
	}

	patched, err := patchFrontMatter(scaffolded, fields, body)
	if err != nil {
		os.Remove(absPath)
		return nil, fmt.Errorf("%s: %w", relPath, err)
	}
	if err := os.WriteFile(absPath, []byte(patched), 0o644); err != nil {
		os.Remove(absPath)
		return nil, err
	}

	return func() { os.Remove(absPath) }, nil
}

// WriteFile writes data to relPath inside the site, creating parent
// directories. Like CreatePage it refuses to overwrite, and returns a cleanup
// function that removes the file again — used to undo a downloaded asset when
// the page it belongs to cannot be written.
func (s *Site) WriteFile(relPath string, data []byte) (cleanup func(), err error) {
	if s.Exists(relPath) {
		return nil, fmt.Errorf("%s already exists", relPath)
	}

	absPath := filepath.Join(s.Root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(absPath, data, 0o644); err != nil {
		os.Remove(absPath)
		return nil, err
	}
	return func() { os.Remove(absPath) }, nil
}

// RemoveAll deletes relPath and anything under it, and is a no-op when it does
// not exist. It is for clearing assets left behind by a run that was killed
// between downloading them and writing the page they belong to.
func (s *Site) RemoveAll(relPath string) error {
	return os.RemoveAll(filepath.Join(s.Root, filepath.FromSlash(relPath)))
}

// scaffold produces the archetype-expanded page content, preferring `hugo new`
// and falling back to reading the archetype directly.
func (s *Site) scaffold(relPath string) (string, error) {
	absPath := filepath.Join(s.Root, filepath.FromSlash(relPath))

	if err := s.runHugoNew(relPath); err != nil {
		return renderArchetype(s.Root, relPath)
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("reading page created by hugo: %w", err)
	}
	return string(content), nil
}

func (s *Site) runHugoNew(relPath string) error {
	bin := s.HugoBin
	if bin == "" {
		bin = "hugo"
	}
	if _, err := exec.LookPath(bin); err != nil {
		return err
	}
	// Hugo >= 0.123 wants `hugo new content <path>`; older releases take the
	// path directly. Try the modern form first.
	var lastErr error
	for _, args := range [][]string{
		{"new", "content", relPath},
		{"new", relPath},
	} {
		cmd := exec.Command(bin, args...)
		cmd.Dir = s.Root
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			return nil
		} else {
			lastErr = fmt.Errorf("%s %s: %w: %s", bin, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
		}
	}
	return lastErr
}

// renderArchetype is the no-Hugo fallback. It keeps the archetype's static
// lines and drops templated ones — every templated key in this site's
// archetypes (title, date) is overridden by the caller anyway.
func renderArchetype(root, relPath string) (string, error) {
	section := sectionOf(relPath)
	candidates := []string{
		filepath.Join(root, "archetypes", section+".md"),
		filepath.Join(root, "archetypes", "default.md"),
	}

	var raw []byte
	var err error
	for _, candidate := range candidates {
		if raw, err = os.ReadFile(candidate); err == nil {
			break
		}
	}
	if raw == nil {
		return "", fmt.Errorf("hugo is not available and no archetype found for section %q", section)
	}

	var kept []string
	for _, line := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		if strings.Contains(line, "{{") {
			// Keep the key with an empty value so patchFrontMatter can find
			// and fill it; drop it entirely if it isn't a key/value line.
			key, _, isKV := strings.Cut(line, ":")
			if !isKV || strings.Contains(key, "{{") {
				continue
			}
			kept = append(kept, key+":")
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n"), nil
}

func sectionOf(relPath string) string {
	parts := strings.Split(path.Clean(relPath), "/")
	if len(parts) >= 2 && parts[0] == "content" {
		return parts[1]
	}
	if len(parts) >= 2 {
		return parts[0]
	}
	return ""
}
