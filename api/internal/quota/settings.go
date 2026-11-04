package quota

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/apperr"
	"github.com/gewall/dnd-forge/api/internal/audit"
)

// ErrInvalidLimit rejects a limit outside 0..MaxLimit.
var ErrInvalidLimit = apperr.New(apperr.Invalid, fmt.Sprintf("kuota harian harus 0 sampai %d", MaxLimit))

const entitySetting = "app_setting"

// Defaults are the daily limits of users without their own limit.
type Defaults struct {
	Portrait int
	Lore     int
}

// SettingsStore persists the default limits.
type SettingsStore interface {
	SettingReader
	SetSettingValue(ctx context.Context, key string, value []byte) error
	RecordAudit(ctx context.Context, e audit.Entry) error
	InTx(ctx context.Context, fn func(SettingsStore) error) error
}

// Settings lets admins read and change the default limits.
type Settings struct {
	store SettingsStore
}

// NewSettings returns a Settings.
func NewSettings(store SettingsStore) *Settings { return &Settings{store: store} }

// Defaults returns the current default limits.
func (s *Settings) Defaults(ctx context.Context) (Defaults, error) {
	return readDefaults(ctx, s.store)
}

// SetDefaults stores d and records the change in the audit log.
func (s *Settings) SetDefaults(ctx context.Context, actor uuid.UUID, d Defaults) (Defaults, error) {
	if !ValidLimit(d.Portrait) || !ValidLimit(d.Lore) {
		return Defaults{}, ErrInvalidLimit
	}
	err := s.store.InTx(ctx, func(tx SettingsStore) error {
		old, err := readDefaults(ctx, tx)
		if err != nil {
			return err
		}
		for k, n := range map[Kind]int{Portrait: d.Portrait, Lore: d.Lore} {
			if err := tx.SetSettingValue(ctx, defaults[k].key, []byte(fmt.Sprint(n))); err != nil {
				return fmt.Errorf("set %s: %w", defaults[k].key, err)
			}
		}
		return tx.RecordAudit(ctx, audit.Entry{
			ActorID: actor, Action: audit.ActionUpdate, Entity: entitySetting, EntityID: "quota_defaults",
			Diff: map[string][2]int{"portrait": {old.Portrait, d.Portrait}, "lore": {old.Lore, d.Lore}},
		})
	})
	if err != nil {
		return Defaults{}, fmt.Errorf("quota: set defaults: %w", err)
	}
	return d, nil
}

// ValidLimit reports whether n may be stored as a daily limit.
func ValidLimit(n int) bool { return n >= 0 && n <= MaxLimit }

func readDefaults(ctx context.Context, r SettingReader) (Defaults, error) {
	p, err := DefaultLimit(ctx, r, Portrait)
	if err != nil {
		return Defaults{}, err
	}
	l, err := DefaultLimit(ctx, r, Lore)
	if err != nil {
		return Defaults{}, err
	}
	return Defaults{Portrait: p, Lore: l}, nil
}
