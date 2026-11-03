// Package moderation cleans user-written prompt text and rejects banned
// words before anything reaches an AI provider.
package moderation

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gewall/dnd-forge/api/internal/apperr"
)

// MaxPromptRunes bounds the extra prompt a user may write.
const MaxPromptRunes = 300

var (
	ErrBanned       = apperr.New(apperr.Invalid, "prompt mengandung kata yang tidak diizinkan; ubah prompt kamu")
	ErrTooLong      = apperr.New(apperr.Invalid, "prompt terlalu panjang")
	ErrInvalidWord  = apperr.New(apperr.Invalid, "kata terlarang harus 2–60 karakter")
	ErrWordNotFound = apperr.New(apperr.NotFound, "kata tidak ada di daftar")
)

// Store persists the banned word list (lowercase).
type Store interface {
	BannedWords(ctx context.Context) ([]string, error)
	AddBannedWord(ctx context.Context, word string) error
	DeleteBannedWord(ctx context.Context, word string) error // ErrWordNotFound
}

// Service sanitises prompts and manages the banned word list.
type Service struct {
	store Store
}

// NewService returns a Service.
func NewService(store Store) *Service { return &Service{store: store} }

var spaces = regexp.MustCompile(`\s+`)

// Sanitize removes control and formatting characters (which can smuggle
// instructions or break layouts) and collapses whitespace.
func Sanitize(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\t' || r == '\r':
			return ' '
		case unicode.IsControl(r), unicode.Is(unicode.Cf, r):
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(spaces.ReplaceAllString(s, " "))
}

// Clean sanitises a user prompt of at most MaxPromptRunes and checks it
// against the banned words. The returned text is what goes into the
// {userPrompt} slot.
func (s *Service) Clean(ctx context.Context, prompt string) (string, error) {
	return s.CleanUpTo(ctx, prompt, MaxPromptRunes)
}

// CleanUpTo is Clean with a different length limit.
func (s *Service) CleanUpTo(ctx context.Context, prompt string, maxRunes int) (string, error) {
	prompt = Sanitize(prompt)
	if utf8.RuneCountInString(prompt) > maxRunes {
		return "", ErrTooLong
	}
	if prompt == "" {
		return "", nil
	}
	words, err := s.store.BannedWords(ctx)
	if err != nil {
		return "", fmt.Errorf("moderation: %w", err)
	}
	if containsAny(strings.ToLower(prompt), words) {
		return "", ErrBanned
	}
	return prompt, nil
}

// containsAny matches whole words (or whole phrases), so "ass" does not
// block "class".
func containsAny(text string, words []string) bool {
	for _, w := range words {
		re, err := regexp.Compile(`(^|[^\p{L}\p{N}])` + regexp.QuoteMeta(w) + `($|[^\p{L}\p{N}])`)
		if err == nil && re.MatchString(text) {
			return true
		}
	}
	return false
}

// Words lists the banned words.
func (s *Service) Words(ctx context.Context) ([]string, error) { return s.store.BannedWords(ctx) }

// Add bans a word or phrase.
func (s *Service) Add(ctx context.Context, word string) (string, error) {
	word = strings.ToLower(Sanitize(word))
	if n := utf8.RuneCountInString(word); n < 2 || n > 60 {
		return "", ErrInvalidWord
	}
	return word, s.store.AddBannedWord(ctx, word)
}

// Remove unbans a word.
func (s *Service) Remove(ctx context.Context, word string) error {
	return s.store.DeleteBannedWord(ctx, strings.ToLower(Sanitize(word)))
}
