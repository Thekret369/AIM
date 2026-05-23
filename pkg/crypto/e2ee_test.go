package crypto

import "testing"

func TestEncryptStringRoundTrip(t *testing.T) {
	encrypted, err := EncryptString("secret-key", "storage-secret")
	if err != nil {
		t.Fatalf("encrypt string: %v", err)
	}
	if encrypted == "secret-key" {
		t.Fatal("expected encrypted value to differ from plaintext")
	}
	if !IsEncryptedString(encrypted) {
		t.Fatalf("expected encrypted prefix, got %q", encrypted)
	}

	plaintext, err := DecryptString(encrypted, "storage-secret")
	if err != nil {
		t.Fatalf("decrypt string: %v", err)
	}
	if plaintext != "secret-key" {
		t.Fatalf("expected plaintext, got %q", plaintext)
	}
}

func TestDecryptStringKeepsPlaintextForLegacyData(t *testing.T) {
	plaintext, err := DecryptString("legacy-key", "storage-secret")
	if err != nil {
		t.Fatalf("decrypt legacy plaintext: %v", err)
	}
	if plaintext != "legacy-key" {
		t.Fatalf("expected legacy plaintext, got %q", plaintext)
	}
}

func TestEncryptStringRequiresSecret(t *testing.T) {
	if _, err := EncryptString("secret-key", ""); err == nil {
		t.Fatal("expected missing secret error")
	}
}
