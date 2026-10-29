// Package slug validates URL slugs used by content and game data.
package slug

import "regexp"

// Pattern is the slug format, also used in OpenAPI `pattern` tags.
const Pattern = `^[a-z0-9]+(-[a-z0-9]+)*$`

const maxLen = 64

var re = regexp.MustCompile(Pattern)

// Valid reports whether s is a lowercase, hyphen-separated slug.
func Valid(s string) bool {
	return len(s) <= maxLen && re.MatchString(s)
}
