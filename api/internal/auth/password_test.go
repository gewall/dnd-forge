package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("rahasia-naga-123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("unexpected hash format: %s", hash)
	}

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{"correct", "rahasia-naga-123", true},
		{"wrong", "rahasia-naga-124", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := VerifyPassword(hash, tt.password)
			if err != nil {
				t.Fatalf("VerifyPassword: %v", err)
			}
			if got != tt.want {
				t.Errorf("VerifyPassword(%q) = %v, want %v", tt.password, got, tt.want)
			}
		})
	}
}

func TestHashPasswordUsesRandomSalt(t *testing.T) {
	a, _ := HashPassword("sama")
	b, _ := HashPassword("sama")
	if a == b {
		t.Fatal("two hashes of the same password are identical")
	}
}

func TestVerifyPasswordMalformed(t *testing.T) {
	for _, h := range []string{"", "plain", "$bcrypt$x$y$z$w", "$argon2id$v=19$m=1,t=1,p=1$!!$!!"} {
		if _, err := VerifyPassword(h, "x"); err == nil {
			t.Errorf("VerifyPassword(%q) returned no error", h)
		}
	}
}
