package hugosite

import (
	"bufio"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// frontMatterScanLimit caps how much of a page is read while looking for the
// front matter block. Front matter is at the top of the file; the imported
// tweets in content/sharing carry a large JSON blob in the body that there is
// no reason to pull into memory.
const frontMatterScanLimit = 64 * 1024

// FrontMatterValues scans a content section and returns every value found for
// the given front matter key, mapped to the page carrying it (e.g.
// "117066375180847363" -> "content/sharing/20260809_sunset.md").
//
// This is how a tool syncing from a read-only source stays idempotent: the
// published pages *are* the record of what has been published, so there is no
// separate state file to drift out of step with the repo. A missing section is
// not an error — it just means nothing has been published there yet.
func (s *Site) FrontMatterValues(section, key string) (map[string]string, error) {
	dir := filepath.Join(s.Root, "content", filepath.FromSlash(section))
	found := map[string]string{}

	pattern := frontMatterKeyPattern(key)
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}

		value, err := frontMatterValue(path, pattern)
		if err != nil {
			return err
		}
		if value == "" {
			return nil
		}
		relPath, err := filepath.Rel(s.Root, path)
		if err != nil {
			return err
		}
		// First page wins, so the reported path is stable whatever order the
		// walk happens to visit duplicates in.
		if _, seen := found[value]; !seen {
			found[value] = filepath.ToSlash(relPath)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return found, nil
}

// frontMatterKeyPattern matches a front matter line declaring key. It is as
// lenient as setField about spacing, so `key :` is recognised too.
func frontMatterKeyPattern(key string) *regexp.Regexp {
	return regexp.MustCompile(`^\s*` + regexp.QuoteMeta(key) + `\s*:\s*(.*)$`)
}

// frontMatterValue returns the value declared for pattern in the page's front
// matter, or "" when the page has no front matter or does not declare the key.
func frontMatterValue(path string, pattern *regexp.Regexp) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(io.LimitReader(file, frontMatterScanLimit))
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != fmDelimiter {
		return "", scanner.Err() // not a front matter page; nothing to report
	}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == fmDelimiter {
			return "", scanner.Err()
		}
		if m := pattern.FindStringSubmatch(line); m != nil {
			return unquoteYAML(m[1]), scanner.Err()
		}
	}
	return "", scanner.Err()
}

// unquoteYAML strips the quoting that YAMLString adds, so the scanned value
// compares equal to the one the tool would write.
func unquoteYAML(value string) string {
	value = strings.TrimSpace(value)
	if len(value) < 2 {
		return value
	}
	first, last := value[0], value[len(value)-1]
	if first == '"' && last == '"' {
		if unquoted, err := strconv.Unquote(value); err == nil {
			return unquoted
		}
		return value[1 : len(value)-1]
	}
	if first == '\'' && last == '\'' {
		return value[1 : len(value)-1]
	}
	return value
}
