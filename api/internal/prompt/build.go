package prompt

import (
	"regexp"
	"sort"
	"strings"
)

// Vars are the values a template can reference as {name}.
type Vars struct {
	Race             string
	Class            string
	Background       string
	Gender           string
	Appearance       string
	UserPrompt       string
	Style            string // English description, see Styles
	RaceVisual       string // race + subrace visual prompts
	ClassVisual      string
	BackgroundVisual string
}

func (v Vars) values() map[string]string {
	return map[string]string{
		"race":             v.Race,
		"class":            v.Class,
		"background":       v.Background,
		"gender":           v.Gender,
		"appearance":       v.Appearance,
		"userPrompt":       v.UserPrompt,
		"style":            v.Style,
		"raceVisual":       v.RaceVisual,
		"classVisual":      v.ClassVisual,
		"backgroundVisual": v.BackgroundVisual,
	}
}

// fields returns the placeholder names templates of kind k may use.
func fields(k Kind) (map[string]string, bool) {
	switch k {
	case KindPortrait:
		return (Vars{}).values(), true
	}
	return nil, false
}

// Placeholders lists the names a template of kind k may use, sorted.
func Placeholders(k Kind) []string {
	known, _ := fields(k)
	names := make([]string, 0, len(known))
	for name := range known {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Styles maps the style ids a user can pick to the words sent to the model.
var Styles = map[string]string{
	"oil-painting": "classic oil painting",
	"watercolor":   "soft watercolor illustration",
	"anime":        "anime illustration",
	"realistic":    "photorealistic digital art",
	"comic":        "bold comic book art",
	"ink-sketch":   "detailed ink sketch",
}

var (
	placeholderRE = regexp.MustCompile(`\{(\w+)\}`)
	emptySegments = regexp.MustCompile(`(^|[.,:;]\s*)(?:[.,;]\s*)+`)
	spaces        = regexp.MustCompile(`[ \t]+`)
)

// Build fills tpl with v. A line whose placeholders are all empty is
// dropped, and punctuation left behind by empty values is tidied, so the
// template's optional parts disappear cleanly.
func Build(tpl string, v Vars) string {
	vals := v.values()
	var out []string
	for _, line := range strings.Split(tpl, "\n") {
		matches := placeholderRE.FindAllStringSubmatch(line, -1)
		filled := 0
		line = placeholderRE.ReplaceAllStringFunc(line, func(m string) string {
			val := strings.TrimSpace(vals[m[1:len(m)-1]])
			if val != "" {
				filled++
			}
			return val
		})
		if len(matches) > 0 && filled == 0 {
			continue
		}
		line = spaces.ReplaceAllString(line, " ")
		line = emptySegments.ReplaceAllString(line, "$1")
		line = strings.TrimLeft(strings.TrimSpace(line), ".,;: ")
		if line != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// unknownPlaceholders returns the {names} in tpl that templates of kind k
// cannot use.
func unknownPlaceholders(k Kind, tpl string) []string {
	known, _ := fields(k)
	var bad []string
	for _, m := range placeholderRE.FindAllStringSubmatch(tpl, -1) {
		if _, ok := known[m[1]]; !ok {
			bad = append(bad, m[0])
		}
	}
	return bad
}
