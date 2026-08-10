package hugosite

import (
	"strings"
	"unicode"
)

const maxSlugLen = 60

// transliterations covers the non-ASCII characters that realistically show up
// in article titles. Anything else outside ASCII is dropped.
var transliterations = map[rune]string{
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'ā': "a",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e", 'ē': "e",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ī': "i",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'ō': "o",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ū': "u",
	'ç': "c", 'ñ': "n", 'ý': "y", 'ÿ': "y",
	'ß': "ss", 'æ': "ae", 'œ': "oe", 'ð': "d", 'þ': "th",
	'&': " and ",
}

// Slugify turns a title into the underscore-separated form used by the
// existing content files, e.g. "Training AI with childhood journal entry"
// becomes "training_ai_with_childhood_journal_entry".
func Slugify(title string) string {
	slug := collapseUnderscores(asciiOnly(transliterate(strings.ToLower(title))))

	if len(slug) > maxSlugLen {
		slug = slug[:maxSlugLen]
		// Prefer cutting on a word boundary rather than mid-word.
		if idx := strings.LastIndexByte(slug, '_'); idx > maxSlugLen/2 {
			slug = slug[:idx]
		}
		slug = strings.Trim(slug, "_")
	}
	if slug == "" {
		slug = "untitled"
	}
	return slug
}

func transliterate(s string) string {
	var b strings.Builder
	for _, r := range s {
		if replacement, ok := transliterations[r]; ok {
			b.WriteString(replacement)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// asciiOnly keeps [a-z0-9], turns separators into underscores, and discards
// letters from scripts that have no sensible ASCII form.
func asciiOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			continue
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func collapseUnderscores(s string) string {
	var b strings.Builder
	previousWasUnderscore := true // leading underscores get swallowed
	for i := 0; i < len(s); i++ {
		if s[i] == '_' {
			if previousWasUnderscore {
				continue
			}
			previousWasUnderscore = true
			b.WriteByte('_')
			continue
		}
		previousWasUnderscore = false
		b.WriteByte(s[i])
	}
	return strings.TrimRight(b.String(), "_")
}
