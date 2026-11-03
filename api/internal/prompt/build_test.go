package prompt

import "testing"

const docTemplate = `Fantasy character portrait of a {race} {class} with a {background} background.
{raceVisual}. {classVisual}. {appearance}.
Additional details: {userPrompt}.
Style: {style}, detailed, dramatic lighting, upper body, no text, no watermark.`

func TestBuild(t *testing.T) {
	full := Vars{
		Race: "Elf", Class: "Wizard", Background: "Sage", Appearance: "silver hair",
		UserPrompt: "holding a crystal staff", Style: "classic oil painting",
		RaceVisual: "pointed ears, slender build", ClassVisual: "flowing robes",
	}
	tests := []struct {
		name string
		vars Vars
		want string
	}{
		{"all values", full, `Fantasy character portrait of a Elf Wizard with a Sage background.
pointed ears, slender build. flowing robes. silver hair.
Additional details: holding a crystal staff.
Style: classic oil painting, detailed, dramatic lighting, upper body, no text, no watermark.`},
		{"no user prompt or appearance", Vars{
			Race: "Elf", Class: "Wizard", Background: "Sage", Style: "anime illustration",
			RaceVisual: "pointed ears", ClassVisual: "flowing robes",
		}, `Fantasy character portrait of a Elf Wizard with a Sage background.
pointed ears. flowing robes.
Style: anime illustration, detailed, dramatic lighting, upper body, no text, no watermark.`},
		{"only appearance on the visual line", Vars{
			Race: "Human", Class: "Fighter", Background: "Soldier", Appearance: "scar on cheek", Style: "comic",
		}, `Fantasy character portrait of a Human Fighter with a Soldier background.
scar on cheek.
Style: comic, detailed, dramatic lighting, upper body, no text, no watermark.`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Build(docTemplate, tt.vars); got != tt.want {
				t.Errorf("Build =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

func TestUnknownPlaceholders(t *testing.T) {
	if bad := unknownPlaceholders(KindPortrait, docTemplate); len(bad) != 0 {
		t.Errorf("doc template has unknown placeholders %v", bad)
	}
	if bad := unknownPlaceholders(KindPortrait, "{race} {weapon} {Class}"); len(bad) != 2 {
		t.Errorf("unknown = %v, want {weapon} and {Class}", bad)
	}
}
