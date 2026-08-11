package hugosite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const fmDelimiter = "---"

// Field is a single front matter key/value. Value is already rendered YAML.
type Field struct {
	Key   string
	Value string
}

// YAMLString renders s as a double-quoted YAML scalar. JSON string escaping is
// a valid subset of YAML's double-quoted style, so the JSON encoder does the
// job — with HTML escaping off, so `&` and `<` stay readable in the source.
func YAMLString(s string) string {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(s); err != nil { // impossible for a string
		return `""`
	}
	return strings.TrimRight(buf.String(), "\n")
}

// YAMLRaw passes a value through untouched (dates, booleans, numbers).
func YAMLRaw(s string) string { return s }

// YAMLStringSlice renders a flow-style array, e.g. ["keep","js"], matching the
// style of the existing linking pages. An empty slice renders as [].
func YAMLStringSlice(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, YAMLString(v))
	}
	return "[" + strings.Join(quoted, ",") + "]"
}

// patchFrontMatter takes a page produced from an archetype, overrides the given
// front matter fields (appending any that the archetype didn't define), and
// replaces the body.
//
// Keys are matched leniently because the archetypes in this repo write
// `description :` with a space before the colon.
func patchFrontMatter(source string, fields []Field, body string) (string, error) {
	normalized := strings.ReplaceAll(source, "\r\n", "\n")

	rest, ok := strings.CutPrefix(normalized, fmDelimiter+"\n")
	if !ok {
		return "", fmt.Errorf("page does not start with a %q front matter delimiter", fmDelimiter)
	}
	frontMatter, _, ok := strings.Cut(rest, "\n"+fmDelimiter)
	if !ok {
		return "", fmt.Errorf("unterminated front matter block")
	}

	lines := strings.Split(frontMatter, "\n")
	for _, field := range fields {
		lines = setField(lines, field)
	}

	var out strings.Builder
	out.WriteString(fmDelimiter + "\n")
	out.WriteString(strings.Join(lines, "\n"))
	out.WriteString("\n" + fmDelimiter + "\n")
	if body = strings.TrimSpace(body); body != "" {
		out.WriteString(body + "\n")
	}
	return out.String(), nil
}

// setField overwrites the first line declaring field.Key, appending it if the
// archetype does not declare it at all.
//
// Any *further* line declaring the same key is dropped: an archetype that
// accidentally lists a key twice would otherwise produce front matter with a
// duplicate key, where which value wins is anyone's guess.
func setField(lines []string, field Field) []string {
	rendered := field.Key + ": " + field.Value
	pattern := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(field.Key) + `\s*:`)

	out := lines[:0:0] // fresh backing array; never alias the caller's slice
	replaced := false
	for _, line := range lines {
		if pattern.MatchString(line) {
			if replaced {
				continue
			}
			replaced = true
			out = append(out, rendered)
			continue
		}
		out = append(out, line)
	}
	if replaced {
		return out
	}

	// Drop trailing blank lines so the appended key stays inside the block.
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	return append(out, rendered)
}
