package audit_test

import (
	"bytes"
	"testing"
	"time"

	"attendance/internal/domain/audit"
)

func TestCanonicalize_Deterministic_DifferentKeyOrders(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)

	e1 := audit.CanonicalEvent{
		OccurredAt: ts,
		Action:     "x",
		EntityType: "t",
		EntityID:   "1",
		Payload: map[string]any{
			"b": 2,
			"a": 1,
			"nested": map[string]any{
				"z": "last",
				"a": "first",
			},
		},
	}
	// Изменим порядок вставки ключей — json.Marshal сортирует map'ы,
	// но наш wrapper должен гарантировать и для вложенных.
	e2 := audit.CanonicalEvent{
		OccurredAt: ts,
		Action:     "x",
		EntityType: "t",
		EntityID:   "1",
		Payload: map[string]any{
			"nested": map[string]any{
				"a": "first",
				"z": "last",
			},
			"a": 1,
			"b": 2,
		},
	}

	b1, err := audit.Canonicalize(e1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := audit.Canonicalize(e2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatalf("canonical not deterministic:\n  b1=%s\n  b2=%s", b1, b2)
	}
}

func TestCanonicalize_StableTopLevelOrder(t *testing.T) {
	t.Parallel()

	e := audit.CanonicalEvent{
		OccurredAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		ActorID:    "user-1",
		Action:     "login",
		EntityType: "user",
		EntityID:   "u1",
		Payload:    map[string]any{"ok": true},
	}
	got, err := audit.Canonicalize(e)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"action":"login","actor_id":"user-1","actor_role":"","entity_id":"u1","entity_type":"user","ip_address":"","occurred_at":"2026-04-20T00:00:00Z","payload":{"ok":true},"user_agent":""}`
	if string(got) != want {
		t.Fatalf("canonical mismatch:\n got=%s\nwant=%s", got, want)
	}
}

func TestCanonicalize_NilPayload(t *testing.T) {
	t.Parallel()
	e := audit.CanonicalEvent{
		OccurredAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
	}
	got, err := audit.Canonicalize(e)
	if err != nil {
		t.Fatal(err)
	}
	// payload:null — важно, чтобы не получить "payload":{} после roundtrip.
	want := `{"action":"","actor_id":"","actor_role":"","entity_id":"","entity_type":"","ip_address":"","occurred_at":"2026-04-20T00:00:00Z","payload":null,"user_agent":""}`
	if string(got) != want {
		t.Fatalf("got=%s\nwant=%s", got, want)
	}
}

func TestCanonicalize_TimeInPayload_RFC3339NanoUTC(t *testing.T) {
	t.Parallel()

	// Московское время — должно схлопнуться в UTC.
	moscow, _ := time.LoadLocation("Europe/Moscow")
	ts := time.Date(2026, 4, 20, 15, 30, 0, 123_456_789, moscow)

	e := audit.CanonicalEvent{
		OccurredAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		Action:     "x",
		Payload:    map[string]any{"when": ts},
	}
	got, err := audit.Canonicalize(e)
	if err != nil {
		t.Fatal(err)
	}
	// 15:30 Moscow (UTC+3) = 12:30 UTC
	want := `{"action":"x","actor_id":"","actor_role":"","entity_id":"","entity_type":"","ip_address":"","occurred_at":"2026-04-20T00:00:00Z","payload":{"when":"2026-04-20T12:30:00.123456789Z"},"user_agent":""}`
	if string(got) != want {
		t.Fatalf("got=%s\nwant=%s", got, want)
	}
}

func TestCanonicalize_NestedArrays(t *testing.T) {
	t.Parallel()
	e := audit.CanonicalEvent{
		OccurredAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		Action:     "x",
		Payload: map[string]any{
			"items": []any{
				map[string]any{"z": 1, "a": 2},
				map[string]any{"a": 3, "z": 4},
			},
		},
	}
	got, err := audit.Canonicalize(e)
	if err != nil {
		t.Fatal(err)
	}
	// Вложенные map'ы в массивах тоже нормализуются.
	want := `{"action":"x","actor_id":"","actor_role":"","entity_id":"","entity_type":"","ip_address":"","occurred_at":"2026-04-20T00:00:00Z","payload":{"items":[{"a":2,"z":1},{"a":3,"z":4}]},"user_agent":""}`
	if string(got) != want {
		t.Fatalf("got=%s\nwant=%s", got, want)
	}
}
