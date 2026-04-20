package audit_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	appaudit "attendance/internal/application/audit"
	"attendance/internal/domain/audit"
	"attendance/internal/platform/clock"
)

// ----- in-memory fake repo -----

type fakeRepo struct {
	entries []audit.Entry
	nextID  int64
}

func (r *fakeRepo) Append(_ context.Context, e audit.Entry) (audit.Entry, error) {
	r.nextID++
	e.ID = r.nextID
	r.entries = append(r.entries, e)
	return e, nil
}

func (r *fakeRepo) Last(_ context.Context) (audit.Entry, bool, error) {
	if len(r.entries) == 0 {
		return audit.Entry{}, false, nil
	}
	return r.entries[len(r.entries)-1], true, nil
}

func (r *fakeRepo) List(_ context.Context, _ audit.ListFilter) ([]audit.Entry, int, error) {
	cp := make([]audit.Entry, len(r.entries))
	copy(cp, r.entries)
	return cp, len(cp), nil
}

func (r *fakeRepo) Scan(_ context.Context, _ int, fn func(audit.Entry) error) error {
	for _, e := range r.entries {
		if err := fn(e); err != nil {
			return err
		}
	}
	return nil
}

// ----- tests -----

func newService() (*appaudit.Service, *fakeRepo) {
	repo := &fakeRepo{}
	svc := appaudit.NewService(appaudit.Deps{
		Repo:  repo,
		Clock: clock.Fixed(time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)),
	})
	return svc, repo
}

func TestAppend_BuildsValidChain(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, repo := newService()

	_, err := svc.Append(ctx, audit.Entry{Action: audit.ActionLoginSuccess, EntityType: "user", EntityID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Append(ctx, audit.Entry{Action: audit.ActionPolicyCreated, EntityType: "policy", EntityID: "p1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Append(ctx, audit.Entry{Action: audit.ActionLogout, EntityType: "user", EntityID: "u1"})
	if err != nil {
		t.Fatal(err)
	}

	if len(repo.entries) != 3 {
		t.Fatalf("len = %d, want 3", len(repo.entries))
	}
	// Первая запись — genesis prev.
	if !bytes.Equal(repo.entries[0].PrevHash, audit.GenesisPrevHash()) {
		t.Fatalf("first entry prev_hash != genesis")
	}
	// Каждая следующая запись должна ссылаться на record_hash предыдущей.
	for i := 1; i < len(repo.entries); i++ {
		if !bytes.Equal(repo.entries[i].PrevHash, repo.entries[i-1].RecordHash) {
			t.Fatalf("chain broken between %d and %d", i-1, i)
		}
	}
	// Все record_hash заполнены и уникальны (entropy через payload).
	seen := map[string]bool{}
	for _, e := range repo.entries {
		if len(e.RecordHash) != audit.HashLen {
			t.Fatalf("record_hash len = %d", len(e.RecordHash))
		}
		if seen[string(e.RecordHash)] {
			t.Fatalf("duplicate record_hash")
		}
		seen[string(e.RecordHash)] = true
	}
}

func TestVerify_ValidChain_OK(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, _ := newService()

	for i := 0; i < 5; i++ {
		_, err := svc.Append(ctx, audit.Entry{
			Action:     audit.ActionPolicyCreated,
			EntityType: "policy",
			EntityID:   "p",
			Payload:    map[string]any{"i": i},
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	res, err := svc.Verify(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("verify not ok: %+v", res)
	}
	if res.TotalEntries != 5 {
		t.Fatalf("total = %d, want 5", res.TotalEntries)
	}
	if res.FirstBrokenID != nil {
		t.Fatalf("first_broken_id = %v, want nil", res.FirstBrokenID)
	}
}

func TestVerify_DetectsPayloadTampering(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, repo := newService()

	for i := 0; i < 4; i++ {
		_, err := svc.Append(ctx, audit.Entry{
			Action:     audit.ActionPolicyCreated,
			EntityType: "policy",
			EntityID:   "p",
			Payload:    map[string]any{"i": i},
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	// Подменяем payload у записи #2 (id=2).
	repo.entries[1].Payload = map[string]any{"i": 999}

	res, err := svc.Verify(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Fatalf("verify should have detected tampering")
	}
	if res.FirstBrokenID == nil || *res.FirstBrokenID != 2 {
		t.Fatalf("first_broken_id = %v, want 2", res.FirstBrokenID)
	}
	if res.BrokenReason != "record_hash mismatch" {
		t.Fatalf("reason = %q", res.BrokenReason)
	}
}

func TestVerify_DetectsChainLinkTampering(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, repo := newService()

	for i := 0; i < 3; i++ {
		_, err := svc.Append(ctx, audit.Entry{
			Action: audit.ActionLoginSuccess,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	// Подменяем prev_hash у записи #2.
	bogus := make([]byte, audit.HashLen)
	for i := range bogus {
		bogus[i] = 0xff
	}
	repo.entries[1].PrevHash = bogus

	res, err := svc.Verify(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Fatalf("verify should have detected chain tampering")
	}
	if res.FirstBrokenID == nil || *res.FirstBrokenID != 2 {
		t.Fatalf("first_broken_id = %v, want 2", res.FirstBrokenID)
	}
	if res.BrokenReason != "prev_hash mismatch" {
		t.Fatalf("reason = %q", res.BrokenReason)
	}
}

func TestVerify_EmptyChain_OK(t *testing.T) {
	t.Parallel()
	svc, _ := newService()
	res, err := svc.Verify(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || res.TotalEntries != 0 {
		t.Fatalf("empty chain: got %+v", res)
	}
}
