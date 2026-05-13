package security

import (
	"encoding/base64"
	"testing"
)

func TestKeyringEncryptUsesActiveKeyAndDecryptUsesStoredKeyID(t *testing.T) {
	key1 := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef"))
	key2 := base64.StdEncoding.EncodeToString([]byte("fedcba9876543210"))

	keyring, err := NewKeyring(KeyringConfig{
		EncodedKeys: key1ID + ":" + key1 + "," + key2ID + ":" + key2,
		ActiveKeyID: key2ID,
		LegacyKeyID: "legacy-v1",
	})
	if err != nil {
		t.Fatalf("new keyring: %v", err)
	}

	encrypted, err := keyring.Encrypt("smtp-secret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if encrypted.KeyID != key2ID {
		t.Fatalf("expected active key id %s, got %s", key2ID, encrypted.KeyID)
	}

	plaintext, err := keyring.Decrypt(encrypted.Ciphertext, encrypted.KeyID)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if plaintext != "smtp-secret" {
		t.Fatalf("expected plaintext smtp-secret, got %s", plaintext)
	}
}

func TestKeyringDecryptSupportsDifferentStoredKeyIDs(t *testing.T) {
	key1 := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef"))
	key2 := base64.StdEncoding.EncodeToString([]byte("fedcba9876543210"))

	keyring, err := NewKeyring(KeyringConfig{
		EncodedKeys: key1ID + ":" + key1 + "," + key2ID + ":" + key2,
		ActiveKeyID: key2ID,
		LegacyKeyID: "legacy-v1",
	})
	if err != nil {
		t.Fatalf("new keyring: %v", err)
	}

	oldCiphertext, err := encryptWithKey("old-secret", keyring.derivedKeys[key1ID])
	if err != nil {
		t.Fatalf("encrypt old secret: %v", err)
	}
	newCiphertext, err := encryptWithKey("new-secret", keyring.derivedKeys[key2ID])
	if err != nil {
		t.Fatalf("encrypt new secret: %v", err)
	}

	oldPlaintext, err := keyring.Decrypt(oldCiphertext, key1ID)
	if err != nil {
		t.Fatalf("decrypt old secret: %v", err)
	}
	if oldPlaintext != "old-secret" {
		t.Fatalf("expected old-secret, got %s", oldPlaintext)
	}

	newPlaintext, err := keyring.Decrypt(newCiphertext, key2ID)
	if err != nil {
		t.Fatalf("decrypt new secret: %v", err)
	}
	if newPlaintext != "new-secret" {
		t.Fatalf("expected new-secret, got %s", newPlaintext)
	}
}

func TestKeyringDecryptFallsBackToLegacyKeyIDWhenStoredKeyIDMissing(t *testing.T) {
	keyring, err := NewKeyring(KeyringConfig{
		LegacyKey:   "legacy-secret-material-that-is-long-enough",
		LegacyKeyID: "legacy-v1",
	})
	if err != nil {
		t.Fatalf("new keyring: %v", err)
	}

	ciphertext, err := encryptWithKey("legacy-secret", keyring.derivedKeys["legacy-v1"])
	if err != nil {
		t.Fatalf("encrypt legacy secret: %v", err)
	}

	plaintext, err := keyring.Decrypt(ciphertext, "")
	if err != nil {
		t.Fatalf("decrypt legacy secret: %v", err)
	}
	if plaintext != "legacy-secret" {
		t.Fatalf("expected legacy-secret, got %s", plaintext)
	}
}

const (
	key1ID = "key1"
	key2ID = "key2"
)
