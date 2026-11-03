package moderation

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeStore struct{ words []string }

func (f *fakeStore) BannedWords(context.Context) ([]string, error) { return f.words, nil }
func (f *fakeStore) AddBannedWord(_ context.Context, w string) error {
	f.words = append(f.words, w)
	return nil
}
func (f *fakeStore) DeleteBannedWord(context.Context, string) error { return nil }

func TestSanitize(t *testing.T) {
	tests := map[string]string{
		"  rambut\n\nperak\t ":     "rambut perak",
		"zero\u200bwidth":          "zerowidth",
		"bell\x07 char":            "bell char",
		"teks biasa, dengan koma.": "teks biasa, dengan koma.",
	}
	for in, want := range tests {
		if got := Sanitize(in); got != want {
			t.Errorf("Sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClean(t *testing.T) {
	svc := NewService(&fakeStore{words: []string{"gore", "darah segar"}})
	ctx := context.Background()

	tests := []struct {
		in   string
		want error
	}{
		{"rambut perak, latar hutan", nil},
		{"", nil},
		{"penuh GORE", ErrBanned},
		{"pedang berlumur darah segar", ErrBanned},
		{"gorengan di meja", nil}, // whole words only
		{"darah", nil},
		{strings.Repeat("a", 301), ErrTooLong},
	}
	for _, tt := range tests {
		if _, err := svc.Clean(ctx, tt.in); !errors.Is(err, tt.want) {
			t.Errorf("Clean(%.20q) err = %v, want %v", tt.in, err, tt.want)
		}
	}
}

func TestAddNormalizes(t *testing.T) {
	svc := NewService(&fakeStore{})
	w, err := svc.Add(context.Background(), "  Kata  KASAR ")
	if err != nil || w != "kata kasar" {
		t.Errorf("Add = %q, %v", w, err)
	}
	if _, err := svc.Add(context.Background(), "x"); !errors.Is(err, ErrInvalidWord) {
		t.Errorf("short word err = %v", err)
	}
}
