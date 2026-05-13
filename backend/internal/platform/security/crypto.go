package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

var (
	ErrMissingActiveEncryptionKey = errors.New("active encryption key is not configured")
	ErrUnknownEncryptionKeyID     = errors.New("unknown encryption key id")
)

type KeyringConfig struct {
	LegacyKey   string
	LegacyKeyID string
	EncodedKeys string
	ActiveKeyID string
}

type Keyring struct {
	activeKeyID string
	legacyKeyID string
	derivedKeys map[string][]byte
}

type EncryptedSecret struct {
	KeyID      string
	Ciphertext string
}

func NewKeyring(config KeyringConfig) (*Keyring, error) {
	keyring := &Keyring{
		activeKeyID: strings.TrimSpace(config.ActiveKeyID),
		legacyKeyID: strings.TrimSpace(config.LegacyKeyID),
		derivedKeys: make(map[string][]byte),
	}

	if strings.TrimSpace(config.EncodedKeys) != "" {
		if err := keyring.addEncodedKeys(config.EncodedKeys); err != nil {
			return nil, err
		}
	}

	if strings.TrimSpace(config.LegacyKey) != "" {
		if keyring.legacyKeyID == "" {
			return nil, errors.New("legacy encryption key id is required when ENCRYPTION_KEY is configured")
		}
		if _, exists := keyring.derivedKeys[keyring.legacyKeyID]; exists {
			return nil, errors.New("duplicate encryption key id: " + keyring.legacyKeyID)
		}
		keyring.derivedKeys[keyring.legacyKeyID] = deriveKey([]byte(config.LegacyKey))
	}

	if len(keyring.derivedKeys) == 0 {
		return nil, errors.New("at least one encryption key must be configured")
	}

	if keyring.activeKeyID == "" {
		keyring.activeKeyID = keyring.legacyKeyID
	}
	if keyring.activeKeyID == "" {
		return nil, ErrMissingActiveEncryptionKey
	}
	if _, ok := keyring.derivedKeys[keyring.activeKeyID]; !ok {
		return nil, ErrMissingActiveEncryptionKey
	}

	return keyring, nil
}

func (k *Keyring) Encrypt(secret string) (*EncryptedSecret, error) {
	if k == nil {
		return nil, ErrMissingActiveEncryptionKey
	}

	key, ok := k.derivedKeys[k.activeKeyID]
	if !ok {
		return nil, ErrMissingActiveEncryptionKey
	}

	ciphertext, err := encryptWithKey(secret, key)
	if err != nil {
		return nil, err
	}

	return &EncryptedSecret{
		KeyID:      k.activeKeyID,
		Ciphertext: ciphertext,
	}, nil
}

func (k *Keyring) Decrypt(ciphertext string, keyID string) (string, error) {
	if k == nil {
		return "", ErrMissingActiveEncryptionKey
	}

	resolvedKeyID := strings.TrimSpace(keyID)
	if resolvedKeyID == "" {
		resolvedKeyID = k.legacyKeyID
	}
	if resolvedKeyID == "" {
		resolvedKeyID = k.activeKeyID
	}

	key, ok := k.derivedKeys[resolvedKeyID]
	if !ok {
		return "", ErrUnknownEncryptionKeyID
	}

	return decryptWithKey(ciphertext, key)
}

func (k *Keyring) ActiveKeyID() string {
	if k == nil {
		return ""
	}
	return k.activeKeyID
}

func (k *Keyring) addEncodedKeys(raw string) error {
	entries := strings.Split(raw, ",")
	for _, entry := range entries {
		pair := strings.TrimSpace(entry)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			return errors.New("invalid ENCRYPTION_KEYS entry: " + pair)
		}

		keyID := strings.TrimSpace(parts[0])
		encodedMaterial := strings.TrimSpace(parts[1])
		if keyID == "" || encodedMaterial == "" {
			return errors.New("invalid ENCRYPTION_KEYS entry: " + pair)
		}
		if _, exists := k.derivedKeys[keyID]; exists {
			return errors.New("duplicate encryption key id: " + keyID)
		}

		material, err := decodeBase64Material(encodedMaterial)
		if err != nil {
			return err
		}
		if len(material) < 16 {
			return errors.New("encryption key material must decode to at least 16 bytes for key id: " + keyID)
		}

		k.derivedKeys[keyID] = deriveKey(material)
	}

	return nil
}

func decodeBase64Material(value string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err == nil {
		return decoded, nil
	}
	return base64.RawStdEncoding.DecodeString(value)
}

func encryptWithKey(secret string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(secret), nil)
	payload := append(nonce, ciphertext...)
	return base64.RawStdEncoding.EncodeToString(payload), nil
}

func decryptWithKey(ciphertext string, key []byte) (string, error) {
	payload, err := base64.RawStdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(payload) < gcm.NonceSize() {
		return "", errors.New("invalid encrypted payload")
	}

	nonce := payload[:gcm.NonceSize()]
	data := payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func deriveKey(material []byte) []byte {
	sum := sha256.Sum256(material)
	return sum[:]
}
