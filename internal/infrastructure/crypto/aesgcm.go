package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

const (
	// AESKeyLen — требуемая длина ключа (AES-256).
	AESKeyLen = 32
	// AESGCMNonceLen — длина nonce для GCM (стандарт NIST SP 800-38D).
	AESGCMNonceLen = 12
)

var (
	// ErrInvalidKeyLen — ключ не 32 байта.
	ErrInvalidKeyLen = errors.New("aesgcm: key must be 32 bytes")
	// ErrInvalidNonceLen — подан nonce не 12 байт.
	ErrInvalidNonceLen = errors.New("aesgcm: nonce must be 12 bytes")
)

// AESGCMEncryptor реализует domain.FieldEncryptor: AES-256-GCM c уникальным
// на каждую запись 12-байтным nonce из crypto/rand.
type AESGCMEncryptor struct {
	gcm cipher.AEAD
}

// NewAESGCMEncryptor принимает ключ как raw bytes.
func NewAESGCMEncryptor(key []byte) (*AESGCMEncryptor, error) {
	if len(key) != AESKeyLen {
		return nil, ErrInvalidKeyLen
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aesgcm: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aesgcm: new gcm: %w", err)
	}
	return &AESGCMEncryptor{gcm: gcm}, nil
}

// NewAESGCMEncryptorFromBase64 удобен для .env (PII_ENCRYPTION_KEY в base64).
// Поддерживаются и стандартный base64, и URL-safe.
func NewAESGCMEncryptorFromBase64(b64key string) (*AESGCMEncryptor, error) {
	key, err := decodeBase64(b64key)
	if err != nil {
		return nil, fmt.Errorf("aesgcm: decode key: %w", err)
	}
	return NewAESGCMEncryptor(key)
}

// Encrypt возвращает (ciphertext, nonce, err).
// Ciphertext включает GCM-tag на конце (authenticated encryption).
func (e *AESGCMEncryptor) Encrypt(plaintext []byte) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, AESGCMNonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("aesgcm: read nonce: %w", err)
	}
	ciphertext = e.gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Decrypt проверяет тег и возвращает plaintext.
func (e *AESGCMEncryptor) Decrypt(ciphertext, nonce []byte) ([]byte, error) {
	if len(nonce) != AESGCMNonceLen {
		return nil, ErrInvalidNonceLen
	}
	pt, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("aesgcm: open: %w", err)
	}
	return pt, nil
}

func decodeBase64(s string) ([]byte, error) {
	// Пробуем оба варианта: std и URL-safe. Пробелы по краям игнорируем.
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(s); err == nil {
			return b, nil
		}
	}
	return nil, errors.New("invalid base64")
}
