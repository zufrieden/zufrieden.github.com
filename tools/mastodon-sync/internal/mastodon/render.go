package mastodon

import (
	"html"
	"strings"
)

// Text renders the post as plain text: paragraphs separated by a blank line,
// <br> as a newline, and every link reduced to its own text. Mastodon writes a
// URL as three <span>s (scheme, visible middle, truncated tail) so that its own
// CSS can shorten it; joining their text gives the full URL back.
func (p Post) Text() string { return render(p.HTML, modeText) }

// Markdown renders the post as markdown, keeping the real link targets rather
// than Mastodon's display text.
//
// The text itself is passed through unescaped. People write markdown-ish
// punctuation in posts on purpose — a leading "> " for a quotation, most
// commonly — and escaping it would put backslashes on the page instead.
func (p Post) Markdown() string { return render(p.HTML, modeMarkdown) }

type renderMode int

const (
	modeText renderMode = iota
	modeMarkdown
)

// render walks the post's HTML once, in either mode. The two modes differ only
// in how a link and a line break come out, so they share the state machine.
func render(raw string, mode renderMode) string {
	var out strings.Builder

	// A link's text arrives as several tokens, so it is buffered until the
	// closing </a> tells us the label is complete.
	inAnchor := false
	anchorHref := ""
	var anchorLabel strings.Builder

	write := func(s string) {
		if inAnchor {
			anchorLabel.WriteString(s)
			return
		}
		out.WriteString(s)
	}

	for _, token := range tokenize(raw) {
		switch {
		case token.tag == "":
			write(token.text)

		case token.tag == "a" && !token.closing:
			inAnchor, anchorHref = true, token.attrs["href"]
			anchorLabel.Reset()

		case token.tag == "a" && token.closing:
			inAnchor = false
			out.WriteString(formatLink(strings.TrimSpace(anchorLabel.String()), anchorHref, mode))

		case token.tag == "br":
			if inAnchor {
				anchorLabel.WriteString(" ") // a label spanning lines is still one label
				continue
			}
			out.WriteString(lineBreak(mode))

		case token.tag == "p" && token.closing:
			write("\n\n")
		}
	}

	// An unclosed <a> would otherwise swallow the rest of the post.
	if inAnchor {
		out.WriteString(formatLink(strings.TrimSpace(anchorLabel.String()), anchorHref, mode))
	}
	return tidyLines(out.String())
}

// lineBreak is a plain newline in text, and an explicit hard break in markdown.
// Goldmark is not configured with hardWraps, so a bare newline would collapse
// into a space and the post's own line breaks would be lost. A trailing
// backslash is the CommonMark hard break that survives whitespace trimming,
// which the two-trailing-spaces form does not.
func lineBreak(mode renderMode) string {
	if mode == modeMarkdown {
		return "\\\n"
	}
	return "\n"
}

func formatLink(label, href string, mode renderMode) string {
	switch {
	case label == "" && href == "":
		return ""
	case mode == modeText:
		if label == "" {
			return href
		}
		return label
	case href == "":
		return label
	case label == "":
		return "<" + href + ">"
	}
	return "[" + escapeLinkLabel(label) + "](" + markdownDestination(href) + ")"
}

// escapeLinkLabel keeps brackets in the link text from closing the label early.
func escapeLinkLabel(label string) string {
	return strings.NewReplacer(`[`, `\[`, `]`, `\]`).Replace(label)
}

// markdownDestination wraps a URL in angle brackets when it contains characters
// that would otherwise end the destination early.
func markdownDestination(href string) string {
	if strings.ContainsAny(href, " ()<>") {
		return "<" + strings.NewReplacer("<", "%3C", ">", "%3E", " ", "%20").Replace(href) + ">"
	}
	return href
}

// tidyLines trims each line and collapses runs of blank lines, so the result is
// neither indented (four leading spaces would become a markdown code block) nor
// padded with the whitespace Mastodon leaves between elements.
func tidyLines(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")

	tidied := make([]string, 0, len(lines))
	blank := 0
	for _, line := range lines {
		// The hard-break backslash has to survive the trim on its own line.
		hardBreak := strings.HasSuffix(line, `\`)
		line = strings.TrimSpace(strings.TrimSuffix(line, `\`))
		if line == "" {
			blank++
			continue
		}
		if blank > 0 && len(tidied) > 0 {
			tidied = append(tidied, "")
		}
		blank = 0
		if hardBreak {
			line += `\`
		}
		tidied = append(tidied, line)
	}
	return strings.Join(tidied, "\n")
}

// token is either a run of text or a tag.
type token struct {
	text    string            // set when tag == ""
	tag     string            // lowercased element name
	closing bool              // </tag>
	attrs   map[string]string // lowercased attribute names, unescaped values
}

// tokenize splits Mastodon's status HTML into text and tag tokens.
//
// Mastodon renders statuses into a small, machine-generated subset of HTML —
// <p>, <br>, <a> and <span> — so this does not need to be, and deliberately is
// not, a general HTML parser: pulling in golang.org/x/net/html would make these
// tools the only thing in the repo with a dependency to vendor and update.
// Anything unexpected degrades quietly: an unknown tag is dropped and the text
// inside it kept.
func tokenize(raw string) []token {
	var tokens []token
	for i := 0; i < len(raw); {
		next := strings.IndexByte(raw[i:], '<')
		if next < 0 {
			tokens = appendText(tokens, raw[i:])
			break
		}
		tokens = appendText(tokens, raw[i:i+next])
		i += next

		end := strings.IndexByte(raw[i:], '>')
		if end < 0 {
			// A stray "<" that never closes is text, not a tag.
			tokens = appendText(tokens, raw[i:])
			break
		}

		inner := raw[i+1 : i+end]
		switch tag, ok := parseTag(inner); {
		case ok:
			tokens = append(tokens, tag)
		case isMarkupDeclaration(inner):
			// A comment, doctype or processing instruction: drop it, so its
			// contents cannot reach the page.
		default:
			// Not a tag at all — a bare "<" in the prose, most likely. Keep it
			// as the text it is rather than eating everything up to the ">".
			tokens = appendText(tokens, raw[i:i+end+1])
		}
		i += end + 1
	}
	return tokens
}

func isMarkupDeclaration(inner string) bool {
	trimmed := strings.TrimSpace(inner)
	return strings.HasPrefix(trimmed, "!") || strings.HasPrefix(trimmed, "?")
}

func appendText(tokens []token, raw string) []token {
	if raw == "" {
		return tokens
	}
	return append(tokens, token{text: html.UnescapeString(raw)})
}

// parseTag reads the inside of a tag, e.g. `a href="https://example.com"` or
// `/p` or `br /`. It reports false for anything that does not open with a
// plausible element name, which is what tells "3 < 4 and 5 > 4" from markup.
func parseTag(inner string) (token, bool) {
	inner = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(inner), "/"))
	if inner == "" {
		return token{}, false
	}

	tag := token{}
	if rest, ok := strings.CutPrefix(inner, "/"); ok {
		tag.closing = true
		inner = strings.TrimSpace(rest)
	}

	name, attrs, _ := strings.Cut(inner, " ")
	if !isElementName(name) {
		return token{}, false
	}
	tag.tag = strings.ToLower(name)
	tag.attrs = parseAttrs(attrs)
	return tag, true
}

// isElementName accepts the HTML element names that can actually occur here —
// letters optionally followed by digits, as in "p", "br", "a" or "h2".
func isElementName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}

// parseAttrs reads `name="value"` pairs, tolerating single quotes, unquoted
// values and valueless attributes.
func parseAttrs(raw string) map[string]string {
	attrs := map[string]string{}
	for i := 0; i < len(raw); {
		// Skip separators.
		for i < len(raw) && (raw[i] == ' ' || raw[i] == '\t' || raw[i] == '\n' || raw[i] == '\r') {
			i++
		}
		if i >= len(raw) {
			break
		}

		start := i
		for i < len(raw) && raw[i] != '=' && raw[i] != ' ' {
			i++
		}
		name := strings.ToLower(raw[start:i])
		if i >= len(raw) || raw[i] != '=' {
			attrs[name] = ""
			continue
		}
		i++ // past '='

		if i < len(raw) && (raw[i] == '"' || raw[i] == '\'') {
			quote := raw[i]
			i++
			valueStart := i
			for i < len(raw) && raw[i] != quote {
				i++
			}
			attrs[name] = html.UnescapeString(raw[valueStart:i])
			if i < len(raw) {
				i++ // past the closing quote
			}
			continue
		}

		valueStart := i
		for i < len(raw) && raw[i] != ' ' {
			i++
		}
		attrs[name] = html.UnescapeString(raw[valueStart:i])
	}
	return attrs
}
