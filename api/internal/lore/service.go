package lore

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/ai"
	"github.com/gewall/dnd-forge/api/internal/aiprovider"
	"github.com/gewall/dnd-forge/api/internal/prompt"
	"github.com/gewall/dnd-forge/api/internal/quota"
)

// Store reads and updates drafts outside of a transaction.
type Store interface {
	Subject(ctx context.Context, characterID uuid.UUID) (Subject, error) // ErrCharacterNotFound
	DailyLimit(ctx context.Context, userID uuid.UUID) (int, error)       // the user's own limit or the default
	CountSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error)
	LoreByID(ctx context.Context, id uuid.UUID) (Lore, error) // ErrNotFound
	LoresOf(ctx context.Context, characterID, userID uuid.UUID) ([]Lore, error)
	Complete(ctx context.Context, id uuid.UUID, content string) error
	Fail(ctx context.Context, id uuid.UUID, message string) error
	SetBackstory(ctx context.Context, characterID, userID uuid.UUID, text string) error // ErrCharacterNotFound
	InTx(ctx context.Context, fn func(TxStore) error) error
}

// TxStore runs inside the request transaction, so the quota check, the row
// and its job commit together.
type TxStore interface {
	LockUser(ctx context.Context, userID uuid.UUID) error
	CountSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error)
	CreateLore(ctx context.Context, l NewLore) (Lore, error)
	EnqueueGenerate(ctx context.Context, args GenerateArgs) error
}

// Providers selects and resolves AI text providers.
type Providers interface {
	Active(ctx context.Context, c ai.Capability) (aiprovider.Selected, error)
	ResolveText(ctx context.Context, id uuid.UUID) (ai.TextProvider, string, error)
}

// Templates returns the active prompt template of a kind.
type Templates interface {
	Active(ctx context.Context, k prompt.Kind) (prompt.Template, error)
}

// Moderator cleans user prompts.
type Moderator interface {
	CleanUpTo(ctx context.Context, text string, maxRunes int) (string, error)
}

// Deps groups the collaborators of Service.
type Deps struct {
	Store     Store
	Providers Providers
	Templates Templates
	Moderator Moderator
	Logger    *slog.Logger
}

// Service requests, writes and applies lore drafts.
type Service struct {
	Deps
	now func() time.Time
}

// NewService returns a Service.
func NewService(d Deps) *Service {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	return &Service{Deps: d, now: time.Now}
}

// Request validates and queues a draft for a character owned by userID.
// It returns the PENDING draft; clients poll until it completes.
func (s *Service) Request(ctx context.Context, userID, characterID uuid.UUID, r Request) (Lore, error) {
	if r.Length == "" {
		r.Length = DefaultLength
	}
	if r.Language == "" {
		r.Language = DefaultLanguage
	}
	tone, ok := Tones[r.Tone]
	if !ok {
		return Lore{}, ErrInvalidTone
	}
	size, ok := Lengths[r.Length]
	if !ok {
		return Lore{}, ErrInvalidLength
	}
	lang, ok := Languages[r.Language]
	if !ok {
		return Lore{}, ErrInvalidLanguage
	}

	subj, err := s.Store.Subject(ctx, characterID)
	if err != nil {
		return Lore{}, fmt.Errorf("lore: %w", err)
	}
	if subj.UserID != userID {
		return Lore{}, ErrCharacterNotFound
	}
	wishes, err := s.Moderator.CleanUpTo(ctx, r.UserPrompt, MaxPromptRunes)
	if err != nil {
		return Lore{}, err
	}
	provider, err := s.Providers.Active(ctx, ai.Text)
	if err != nil {
		return Lore{}, fmt.Errorf("lore: %w", err)
	}
	tpl, err := s.Templates.Active(ctx, prompt.KindLore)
	if err != nil {
		return Lore{}, fmt.Errorf("lore: %w", err)
	}
	limit, err := s.Store.DailyLimit(ctx, userID)
	if err != nil {
		return Lore{}, fmt.Errorf("lore: %w", err)
	}

	final := prompt.BuildLore(tpl.Template, prompt.LoreVars{
		Name: subj.Name, Race: subj.Race, Subrace: subj.Subrace, Class: subj.Class, Background: subj.Background,
		Gender: subj.Gender, Appearance: subj.Appearance, UserPrompt: wishes,
		Tone: tone.Prompt, Length: size.Prompt, Language: lang.Prompt,
	})
	if neg := strings.TrimSpace(tpl.NegativePrompt); neg != "" {
		final += "\nAvoid: " + neg + "."
	}
	since, _ := quota.Day(s.now())

	var l Lore
	err = s.Store.InTx(ctx, func(tx TxStore) error {
		if err := tx.LockUser(ctx, userID); err != nil {
			return err
		}
		used, err := tx.CountSince(ctx, userID, since)
		if err != nil {
			return err
		}
		if used >= limit {
			return ErrQuotaExceeded
		}
		l, err = tx.CreateLore(ctx, NewLore{
			CharacterID: characterID, UserID: userID, ProviderID: provider.ID, Model: provider.Model,
			UserPrompt: wishes, FinalPrompt: final, Tone: r.Tone, Length: r.Length, Language: r.Language,
		})
		if err != nil {
			return err
		}
		return tx.EnqueueGenerate(ctx, GenerateArgs{LoreID: l.ID})
	})
	if err != nil {
		return Lore{}, fmt.Errorf("lore: request: %w", err)
	}
	return l, nil
}

// errPermanent marks failures that retrying cannot fix.
var errPermanent = errors.New("lore: permanent failure")

// Generate runs one attempt of a queued draft. lastAttempt tells it to mark
// the draft FAILED instead of leaving it for a retry.
func (s *Service) Generate(ctx context.Context, args GenerateArgs, lastAttempt bool) error {
	l, err := s.Store.LoreByID(ctx, args.LoreID)
	if errors.Is(err, ErrNotFound) {
		return fmt.Errorf("%w: lore %s is gone", errPermanent, args.LoreID)
	}
	if err != nil {
		return err
	}
	if l.Status != StatusPending {
		return nil // already finished by an earlier attempt
	}

	err = s.write(ctx, l)
	if err == nil {
		return nil
	}
	log := s.Logger.With("lore_id", l.ID, "provider_id", l.ProviderID, "model", l.Model)
	switch {
	case errors.Is(err, ai.ErrRejected):
		log.WarnContext(ctx, "lore rejected by provider", "err", err)
		return s.fail(ctx, l.ID, msgRejected, err)
	case errors.Is(err, ai.ErrBadKey), errors.Is(err, ai.ErrBilling), errors.Is(err, ai.ErrUnknownProvider),
		errors.Is(err, ai.ErrUnsupported), errors.Is(err, aiprovider.ErrKeyUnreadable):
		log.ErrorContext(ctx, "lore provider misconfigured", "err", err)
		return s.fail(ctx, l.ID, msgConfig, err)
	case lastAttempt:
		log.ErrorContext(ctx, "lore failed after retries", "err", err)
		return s.fail(ctx, l.ID, msgGaveUp, err)
	}
	log.WarnContext(ctx, "lore attempt failed, will retry", "err", err)
	return err
}

func (s *Service) write(ctx context.Context, l Lore) error {
	impl, key, err := s.Providers.ResolveText(ctx, l.ProviderID)
	if err != nil {
		return err
	}
	text, err := impl.Complete(ctx, ai.TextInput{
		System: systemPrompt, Prompt: l.FinalPrompt, Model: l.Model, MaxTokens: Lengths[l.Length].maxTokens, APIKey: key,
	})
	if err != nil {
		return err
	}
	return s.Store.Complete(ctx, l.ID, truncate(strings.TrimSpace(text), maxContentRunes))
}

func (s *Service) fail(ctx context.Context, id uuid.UUID, msg string, cause error) error {
	if err := s.Store.Fail(ctx, id, msg); err != nil {
		return err
	}
	return fmt.Errorf("%w: %w", errPermanent, cause)
}

// Get returns a draft owned by userID.
func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (Lore, error) {
	l, err := s.Store.LoreByID(ctx, id)
	if err != nil {
		return Lore{}, fmt.Errorf("lore: %w", err)
	}
	if l.UserID != userID {
		return Lore{}, ErrNotFound
	}
	return l, nil
}

// ListForCharacter returns the drafts of a character owned by userID.
func (s *Service) ListForCharacter(ctx context.Context, userID, characterID uuid.UUID) ([]Lore, error) {
	return s.Store.LoresOf(ctx, characterID, userID)
}

// Apply copies a finished draft into its character's backstory.
func (s *Service) Apply(ctx context.Context, userID, id uuid.UUID) error {
	l, err := s.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	if l.Status != StatusSucceeded {
		return ErrNotReady
	}
	if l.CharacterID == nil {
		return ErrCharacterNotFound // the character was deleted
	}
	if err := s.Store.SetBackstory(ctx, *l.CharacterID, userID, l.Content); err != nil {
		return fmt.Errorf("lore: apply: %w", err)
	}
	return nil
}

// Quota reports the user's lore allowance for today (WIB).
func (s *Service) Quota(ctx context.Context, userID uuid.UUID) (quota.Usage, error) {
	limit, err := s.Store.DailyLimit(ctx, userID)
	if err != nil {
		return quota.Usage{}, fmt.Errorf("lore: quota: %w", err)
	}
	start, next := quota.Day(s.now())
	used, err := s.Store.CountSince(ctx, userID, start)
	if err != nil {
		return quota.Usage{}, fmt.Errorf("lore: quota: %w", err)
	}
	return quota.NewUsage(limit, used, next), nil
}

func truncate(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	return string([]rune(s)[:maxRunes])
}
