package quota

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/gewall/dnd-forge/api/internal/audit"
)

type memSettings struct {
	settings
	audits []audit.Entry
}

func (m *memSettings) SetSettingValue(_ context.Context, key string, value []byte) error {
	m.settings[key] = string(value)
	return nil
}

func (m *memSettings) RecordAudit(_ context.Context, e audit.Entry) error {
	m.audits = append(m.audits, e)
	return nil
}

func (m *memSettings) InTx(_ context.Context, fn func(SettingsStore) error) error { return fn(m) }

func TestSetDefaults(t *testing.T) {
	ctx := context.Background()
	store := &memSettings{settings: settings{}}
	svc := NewSettings(store)
	actor := uuid.New()

	if d, err := svc.Defaults(ctx); err != nil || d != (Defaults{Portrait: 5, Lore: 10}) {
		t.Errorf("fallback defaults = %+v, %v", d, err)
	}
	for _, bad := range []Defaults{{Portrait: -1, Lore: 1}, {Portrait: 1, Lore: MaxLimit + 1}} {
		if _, err := svc.SetDefaults(ctx, actor, bad); !errors.Is(err, ErrInvalidLimit) {
			t.Errorf("SetDefaults(%+v) err = %v", bad, err)
		}
	}
	if _, err := svc.SetDefaults(ctx, actor, Defaults{Portrait: 3, Lore: 0}); err != nil {
		t.Fatal(err)
	}
	if d, _ := svc.Defaults(ctx); d != (Defaults{Portrait: 3, Lore: 0}) {
		t.Errorf("stored defaults = %+v", d)
	}
	if len(store.audits) != 1 || store.audits[0].ActorID != actor {
		t.Fatalf("audits = %+v", store.audits)
	}
	if diff := store.audits[0].Diff.(map[string][2]int); diff["portrait"] != [2]int{5, 3} || diff["lore"] != [2]int{10, 0} {
		t.Errorf("diff = %v", diff)
	}
}
