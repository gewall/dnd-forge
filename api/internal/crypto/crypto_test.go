package crypto

import (
	"bytes"
	"errors"
	"testing"
)

func TestSealOpen(t *testing.T) {
	s, err := NewSealer(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	secret := []byte("sk-test-1234567890")

	a, _ := s.Seal(secret)
	b, _ := s.Seal(secret)
	if bytes.Equal(a, b) {
		t.Error("sealing twice gave identical ciphertext; nonce reused")
	}
	if bytes.Contains(a, secret) {
		t.Error("ciphertext contains the plaintext")
	}
	got, err := s.Open(a)
	if err != nil || !bytes.Equal(got, secret) {
		t.Fatalf("Open = %q, %v", got, err)
	}

	tampered := append([]byte{}, a...)
	tampered[len(tampered)-1] ^= 1
	other, _ := NewSealer(bytes.Repeat([]byte{8}, 32))
	for name, c := range map[string][]byte{"tampered": tampered, "short": a[:5], "empty": nil} {
		if _, err := s.Open(c); !errors.Is(err, ErrDecrypt) {
			t.Errorf("%s: err = %v, want ErrDecrypt", name, err)
		}
	}
	if _, err := other.Open(a); !errors.Is(err, ErrDecrypt) {
		t.Errorf("wrong key: err = %v, want ErrDecrypt", err)
	}
	if _, err := NewSealer(make([]byte, 16)); err == nil {
		t.Error("16-byte key accepted")
	}
}
