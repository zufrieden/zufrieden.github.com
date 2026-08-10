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
	if !s.Exists(relPath) {
		return relPath
	}
	ext := path.Ext(relPath)
	stem := strings.TrimSuffix(relPath, ext)
	for n := 2; n < 100; n++ {
		candidate := fmt.Sprintf("%s_%d%s", stem, n, ext)
		if !s.Exists(candidate) {
			return candidate
		}
	}
	return relPath
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
