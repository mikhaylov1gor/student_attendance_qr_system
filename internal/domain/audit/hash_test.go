package audit_test

import (
	"bytes"
	"crypto/sha256"
	"testing"
	"time"

	"attendance/internal/domain/audit"
)

func TestComputeRecordHash_MatchesManualSHA256(t *testing.T) {
	t.Parallel()

	prev := audit.GenesisPrevHash()
	entry := audit.Entry{
		OccurredAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		Action:     "login_success",
		EntityType: "user",
		EntityID:   "u1",
		Payload:    map[string]any{"email": "a@b"},
	}

	got, err := audit.ComputeRecordHash(prev, entry)
	if err != nil {
		t.Fatal(err)
	}

	// Пересчитаем вручную по той же формуле — должны совпасть.
	canon, _ := audit.Canonicalize(entry.ToCanonical())
	h := sha256.New()
	h.Write(prev)
	h.Write(canon)
	want := h.Sum(nil)

	if !bytes.Equal(got, want) {
		t.Fatalf("hash mismatch:\n got=%x\nwant=%x", got, want)
	}
	if len(got) != audit.HashLen {
		t.Fatalf("HashLen = %d, want %d", len(got), audit.HashLen)
	}
}

func TestComputeRecordHash_PayloadMutationChangesHash(t *testing.T) {
	t.Parallel()

	prev := audit.GenesisPrevHash()
	base := audit.Entry{
		OccurredAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		Action:     "policy_created",
		EntityType: "security_policy",
		EntityID:   "p1",
		Payload:    map[string]any{"name": "strict"},
	}
	h1, err := audit.ComputeRecordHash(prev, base)
	if err != nil {
		t.Fatal(err)
	}

	mut := base
	mut.Payload = map[string]any{"name": "relaxed"} // подделка — изменили 1 байт в payload
	h2, err := audit.ComputeRecordHash(prev, mut)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(h1, h2) {
		t.Fatalf("hash unchanged after payload mutation — tamper-evidence broken")
	}
}

func TestComputeRecordHash_PrevHashChangesHash(t *testing.T) {
	t.Parallel()

	entry := audit.Entry{
		OccurredAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		Action:     "x",
	}
	h1, _ := audit.ComputeRecordHash(audit.GenesisPrevHash(), entry)

	differentPrev := make([]byte, audit.HashLen)
	differentPrev[0] = 1
	h2, _ := audit.ComputeRecordHash(differentPrev, entry)

	if bytes.Equal(h1, h2) {
		t.Fatalf("hash unchanged after prev_hash mutation — chain linking broken")
	}
}

func TestComputeRecordHash_InvalidPrevHashLen(t *testing.T) {
	t.Parallel()
	_, err := audit.ComputeRecordHash([]byte{1, 2, 3}, audit.Entry{})
	if err == nil {
		t.Fatal("expected error for bad prev_hash len")
	}
}

func TestGenesisPrevHash_AllZero(t *testing.T) {
	t.Parallel()
	g := audit.GenesisPrevHash()
	if len(g) != audit.HashLen {
		t.Fatalf("len = %d, want %d", len(g), audit.HashLen)
	}
	for _, b := range g {
		if b != 0 {
			t.Fatalf("genesis contains non-zero byte: %x", g)
		}
	}
}
