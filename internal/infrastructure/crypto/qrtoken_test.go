package crypto_test

import (
	"crypto/rand"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"attendance/internal/domain"
	"attendance/internal/infrastructure/crypto"
)

func newSecret(t *testing.T) []byte {
	t.Helper()
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestQRToken_RoundTrip(t *testing.T) {
	t.Parallel()
	c := crypto.NewQRTokenCodec()
	secret := newSecret(t)

	src := domain.QRToken{
		SessionID: uuid.New(),
		Counter:   42,
		IssuedAt:  time.Date(2026, 4, 20, 12, 34, 56, 789, time.UTC),
	}
	token, err := c.Encode(secret, src)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("empty token")
	}
	if strings.ContainsAny(token, "+/=") {
		t.Fatalf("expected url-safe base64 without padding: %q", token)
	}

	got, err := c.Decode(secret, token)
	if err != nil {
		t.Fatal(err)
	}
	if got.SessionID != src.SessionID {
		t.Errorf("sid mismatch")
	}
	if got.Counter != src.Counter {
		t.Errorf("counter: got=%d want=%d", got.Counter, src.Counter)
	}
	if !got.IssuedAt.Equal(src.IssuedAt) {
		t.Errorf("issued_at: got=%s want=%s", got.IssuedAt, src.IssuedAt)
	}
}

func TestQRToken_WrongSecret_RejectsHMAC(t *testing.T) {
	t.Parallel()
	c := crypto.NewQRTokenCodec()
	secret1 := newSecret(t)
	secret2 := newSecret(t)

	token, _ := c.Encode(secret1, domain.QRToken{
		SessionID: uuid.New(), Counter: 1, IssuedAt: time.Now().UTC(),
	})
	if _, err := c.Decode(secret2, token); err == nil {
		t.Fatal("decode must fail with wrong secret")
	}
}

func TestQRToken_TamperedTokenRejected(t *testing.T) {
	t.Parallel()
	c := crypto.NewQRTokenCodec()
	secret := newSecret(t)

	token, _ := c.Encode(secret, domain.QRToken{
		SessionID: uuid.New(), Counter: 1, IssuedAt: time.Now().UTC(),
	})

	// Меняем 1 символ в середине base64 — это сломает либо payload, либо HMAC.
	mid := len(token) / 2
	mutated := token[:mid] + flipChar(token[mid]) + token[mid+1:]
	if mutated == token {
		t.Fatal("flipChar gave back same char")
	}
	if _, err := c.Decode(secret, mutated); err == nil {
		t.Fatal("decode must fail on tampered token")
	}
}

func TestQRToken_GarbageRejected(t *testing.T) {
	t.Parallel()
	c := crypto.NewQRTokenCodec()
	if _, err := c.Decode(newSecret(t), "not_base64_at_all!"); err == nil {
		t.Fatal("must reject garbage")
	}
	if _, err := c.Decode(newSecret(t), ""); err == nil {
		t.Fatal("must reject empty token")
	}
}

func TestQRToken_EncodeRequiresSecret(t *testing.T) {
	t.Parallel()
	c := crypto.NewQRTokenCodec()
	if _, err := c.Encode(nil, domain.QRToken{SessionID: uuid.New()}); err == nil {
		t.Fatal("must reject empty secret")
	}
}

func flipChar(b byte) string {
	if b == 'A' {
		return "B"
	}
	return "A"
}
