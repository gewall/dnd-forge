package portrait

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/ai"
	"github.com/gewall/dnd-forge/api/internal/aiprovider"
	"github.com/gewall/dnd-forge/api/internal/imaging"
	"github.com/gewall/dnd-forge/api/internal/prompt"
	"github.com/gewall/dnd-forge/api/internal/quota"
)

// Store reads and updates portraits outside of a transaction.
type Store interface {
	Subject(ctx context.Context, characterID uuid.UUID) (Subject, error) // ErrCharacterNotFound
	DailyLimit(ctx context.Context, userID uuid.UUID) (int, error)
	CountSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error)
	PortraitByID(ctx context.Context, id uuid.UUID) (Portrait, error) // ErrNotFound
	PortraitsOf(ctx context.Context, characterID, userID uuid.UUID) ([]Portrait, error)
	Gallery(ctx context.Context, before *time.Time, limit int) ([]GalleryItem, error)
	Complete(ctx context.Context, id uuid.UUID, storageKey, thumbKey string) error
	Fail(ctx context.Context, id uuid.UUID, message string) error
	InTx(ctx context.Context, fn func(TxStore) error) error
}

// TxStore runs inside the request transaction, so the quota check, the row
// and its job commit together: no job without a row, no row without a job.
type TxStore interface {
	LockUser(ctx context.Context, userID uuid.UUID) error
	CountSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error)
	CreatePortrait(ctx context.Context, p NewPortrait) (Portrait, error)
	EnqueueGenerate(ctx context.Context, args GenerateArgs) error
}

// Providers selects and resolves AI providers.
type Providers interface {
	Active(ctx context.Context, c ai.Capability) (aiprovider.Selected, error)
	ResolveImage(ctx context.Context, id uuid.UUID) (ai.ImageProvider, string, error)
}

// Templates returns the active prompt template of a kind.
type Templates interface {
	Active(ctx context.Context, k prompt.Kind) (prompt.Template, error)
}

// Moderator cleans user prompts.
type Moderator interface {
	Clean(ctx context.Context, text string) (string, error)
}

// ImageEncoder converts provider output to stored images.
type ImageEncoder interface {
	Encode(ctx context.Context, data []byte, sizes ...imaging.Size) ([]imaging.Result, error)
}

// ObjectStore saves images and presigns downloads.
type ObjectStore interface {
	Put(ctx context.Context, key string, data []byte, contentType string) error
	PresignGet(ctx context.Context, key string) (string, time.Duration, error)
}

// Deps groups the collaborators of Service.
type Deps struct {
	Store     Store
	Providers Providers
	Templates Templates
	Moderator Moderator
	Encoder   ImageEncoder
	Objects   ObjectStore
	Logger    *slog.Logger
}

// Service requests and generates portraits.
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

var (
	fullSize  = imaging.Size{MaxWidth: 1536, MaxHeight: 1536}
	thumbSize = imaging.Size{MaxWidth: 384, MaxHeight: 384}
)

// Request validates and queues a portrait of a character owned by userID.
// It returns the PENDING portrait; clients poll until it completes.
func (s *Service) Request(ctx context.Context, userID, characterID uuid.UUID, r Request) (Portrait, error) {
	style, ok := prompt.Styles[r.Style]
	if !ok {
		return Portrait{}, ErrInvalidStyle
	}
	if r.AspectRatio == "" {
		r.AspectRatio = ai.Square
	}
	switch r.AspectRatio {
	case ai.Square, ai.Portrait, ai.Landscape:
	default:
		return Portrait{}, ErrInvalidAspect
	}

	subj, err := s.Store.Subject(ctx, characterID)
	if err != nil {
		return Portrait{}, fmt.Errorf("portrait: %w", err)
	}
	if subj.UserID != userID {
		return Portrait{}, ErrCharacterNotFound
	}
	userPrompt, err := s.Moderator.Clean(ctx, r.UserPrompt)
	if err != nil {
		return Portrait{}, err
	}
	provider, err := s.Providers.Active(ctx, ai.Image)
	if err != nil {
		return Portrait{}, fmt.Errorf("portrait: %w", err)
	}
	tpl, err := s.Templates.Active(ctx, prompt.KindPortrait)
	if err != nil {
		return Portrait{}, fmt.Errorf("portrait: %w", err)
	}
	limit, err := s.Store.DailyLimit(ctx, userID)
	if err != nil {
		return Portrait{}, fmt.Errorf("portrait: %w", err)
	}

	final := prompt.Build(tpl.Template, prompt.Vars{
		Race: subj.Race, Class: subj.Class, Background: subj.Background,
		Gender: subj.Gender, Appearance: subj.Appearance, UserPrompt: userPrompt, Style: style,
		RaceVisual:       joinNonEmpty(", ", subj.RaceVisual, subj.SubraceVisual),
		ClassVisual:      subj.ClassVisual,
		BackgroundVisual: subj.BackgroundVisual,
	})
	since, _ := quota.Day(s.now())

	var p Portrait
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
		p, err = tx.CreatePortrait(ctx, NewPortrait{
			CharacterID: characterID, UserID: userID, ProviderID: provider.ID, Model: provider.Model,
			UserPrompt: userPrompt, FinalPrompt: final, Style: r.Style, AspectRatio: r.AspectRatio,
		})
		if err != nil {
			return err
		}
		return tx.EnqueueGenerate(ctx, GenerateArgs{PortraitID: p.ID, NegativePrompt: tpl.NegativePrompt})
	})
	if err != nil {
		return Portrait{}, fmt.Errorf("portrait: request: %w", err)
	}
	return p, nil
}

// errPermanent marks failures that retrying cannot fix.
var errPermanent = errors.New("portrait: permanent failure")

// Generate runs one attempt of a queued portrait. lastAttempt tells it to
// mark the portrait FAILED instead of leaving it for a retry.
func (s *Service) Generate(ctx context.Context, args GenerateArgs, lastAttempt bool) error {
	p, err := s.Store.PortraitByID(ctx, args.PortraitID)
	if errors.Is(err, ErrNotFound) {
		return fmt.Errorf("%w: portrait %s is gone", errPermanent, args.PortraitID)
	}
	if err != nil {
		return err
	}
	if p.Status != StatusPending {
		return nil // already finished by an earlier attempt
	}

	err = s.generate(ctx, p, args.NegativePrompt)
	if err == nil {
		return nil
	}
	log := s.Logger.With("portrait_id", p.ID, "provider_id", p.ProviderID, "model", p.Model)
	switch {
	case errors.Is(err, ai.ErrRejected):
		log.WarnContext(ctx, "portrait rejected by provider", "err", err)
		return s.fail(ctx, p.ID, msgRejected, err)
	case errors.Is(err, ai.ErrBadKey), errors.Is(err, ai.ErrBilling), errors.Is(err, ai.ErrUnknownProvider), errors.Is(err, ai.ErrUnsupported),
		errors.Is(err, aiprovider.ErrKeyUnreadable):
		log.ErrorContext(ctx, "portrait provider misconfigured", "err", err)
		return s.fail(ctx, p.ID, msgConfig, err)
	case lastAttempt:
		log.ErrorContext(ctx, "portrait failed after retries", "err", err)
		return s.fail(ctx, p.ID, msgGaveUp, err)
	}
	log.WarnContext(ctx, "portrait attempt failed, will retry", "err", err)
	return err
}

func (s *Service) generate(ctx context.Context, p Portrait, negative string) error {
	impl, key, err := s.Providers.ResolveImage(ctx, p.ProviderID)
	if err != nil {
		return err
	}
	out, err := impl.Generate(ctx, ai.GenerateInput{
		Prompt: p.FinalPrompt, NegativePrompt: negative, Model: p.Model, AspectRatio: p.AspectRatio, APIKey: key,
	})
	if err != nil {
		return err
	}
	imgs, err := s.Encoder.Encode(ctx, out.Data, fullSize, thumbSize)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	base := fmt.Sprintf("portraits/%s/%s", p.UserID, p.ID)
	full, thumb := base+".webp", base+"_thumb.webp"
	if err := s.Objects.Put(ctx, full, imgs[0].Data, imgs[0].ContentType); err != nil {
		return err
	}
	if err := s.Objects.Put(ctx, thumb, imgs[1].Data, imgs[1].ContentType); err != nil {
		return err
	}
	return s.Store.Complete(ctx, p.ID, full, thumb)
}

func (s *Service) fail(ctx context.Context, id uuid.UUID, msg string, cause error) error {
	if err := s.Store.Fail(ctx, id, msg); err != nil {
		return err
	}
	return fmt.Errorf("%w: %w", errPermanent, cause)
}

// Get returns a portrait owned by userID.
func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (Portrait, error) {
	p, err := s.Store.PortraitByID(ctx, id)
	if err != nil {
		return Portrait{}, fmt.Errorf("portrait: %w", err)
	}
	if p.UserID != userID {
		return Portrait{}, ErrNotFound
	}
	return p, nil
}

// ListForCharacter returns the portraits of a character owned by userID.
func (s *Service) ListForCharacter(ctx context.Context, userID, characterID uuid.UUID) ([]Portrait, error) {
	return s.Store.PortraitsOf(ctx, characterID, userID)
}

// Quota reports the user's portrait allowance for today (WIB).
func (s *Service) Quota(ctx context.Context, userID uuid.UUID) (quota.Usage, error) {
	limit, err := s.Store.DailyLimit(ctx, userID)
	if err != nil {
		return quota.Usage{}, fmt.Errorf("portrait: quota: %w", err)
	}
	start, next := quota.Day(s.now())
	used, err := s.Store.CountSince(ctx, userID, start)
	if err != nil {
		return quota.Usage{}, fmt.Errorf("portrait: quota: %w", err)
	}
	return quota.NewUsage(limit, used, next), nil
}

// Gallery lists public characters with their newest portrait.
func (s *Service) Gallery(ctx context.Context, before *time.Time, limit int) ([]GalleryItem, error) {
	return s.Store.Gallery(ctx, before, min(max(limit, 1), 60))
}

// ImageURL presigns the stored image of a viewable portrait. A portrait is
// viewable once it succeeded and unless an admin hid it; its random id acts
// as the capability, since <img> tags cannot send credentials.
func (s *Service) ImageURL(ctx context.Context, id uuid.UUID, thumb bool) (string, time.Duration, error) {
	p, err := s.Store.PortraitByID(ctx, id)
	if err != nil {
		return "", 0, err
	}
	if p.Status != StatusSucceeded || p.IsHidden {
		return "", 0, ErrNotFound
	}
	key := p.StorageKey
	if thumb {
		key = p.ThumbKey
	}
	return s.Objects.PresignGet(ctx, key)
}

func joinNonEmpty(sep string, parts ...string) string {
	var out []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, sep)
}
