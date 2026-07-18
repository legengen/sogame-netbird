package crypto

import (
	"bytes"
	"testing"
)

func TestSealOpenRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{0x2a}, 32)
	plaintext := []byte("room setup key")

	ciphertext, err := Seal(key, plaintext)
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("Seal() returned plaintext")
	}
	opened, err := Open(key, ciphertext)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if !bytes.Equal(opened, plaintext) {
		t.Fatalf("Open() = %q, want %q", opened, plaintext)
	}
}

func TestOpenRejectsWrongKey(t *testing.T) {
	key := bytes.Repeat([]byte{0x2a}, 32)
	ciphertext, err := Seal(key, []byte("secret"))
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	if _, err := Open(bytes.Repeat([]byte{0x2b}, 32), ciphertext); err == nil {
		t.Fatal("Open() accepted a ciphertext encrypted with another key")
	}
}
