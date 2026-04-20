package policy_test

import (
	"context"
	"errors"
	"testing"

	"attendance/internal/domain/policy"
)

// --- stub checks ---

type stubCheck struct {
	name   string
	result policy.CheckResult
	err    error
	calls  int
}

func (s *stubCheck) Name() string { return s.name }
func (s *stubCheck) Check(_ context.Context, _ policy.MechanismsConfig, _ policy.CheckInput) (policy.CheckResult, error) {
	s.calls++
	return s.result, s.err
}

func newStub(name string, status policy.CheckStatus) *stubCheck {
	return &stubCheck{name: name, result: policy.CheckResult{Mechanism: name, Status: status}}
}

// --- tests ---

func TestEngine_EvaluatePreservesOrder(t *testing.T) {
	t.Parallel()

	a := newStub("a", policy.StatusPassed)
	b := newStub("b", policy.StatusFailed)
	c := newStub("c", policy.StatusSkipped)

	eng := policy.NewEngine(a, b, c)
	got, err := eng.Evaluate(context.Background(), policy.MechanismsConfig{}, policy.CheckInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(got))
	}
	wantNames := []string{"a", "b", "c"}
	for i, r := range got {
		if r.Mechanism != wantNames[i] {
			t.Errorf("results[%d].Mechanism = %s, want %s", i, r.Mechanism, wantNames[i])
		}
	}
}

func TestEngine_EvaluateEmptyReturnsEmpty(t *testing.T) {
	t.Parallel()

	eng := policy.NewEngine()
	got, err := eng.Evaluate(context.Background(), policy.MechanismsConfig{}, policy.CheckInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %d", len(got))
	}
}

func TestEngine_EvaluatePropagatesCheckError(t *testing.T) {
	t.Parallel()

	want := errors.New("boom")
	a := newStub("a", policy.StatusPassed)
	b := &stubCheck{name: "b", err: want}
	c := newStub("c", policy.StatusPassed)

	eng := policy.NewEngine(a, b, c)
	_, err := eng.Evaluate(context.Background(), policy.MechanismsConfig{}, policy.CheckInput{})
	if !errors.Is(err, want) {
		t.Fatalf("errors.Is(%v, %v) = false", err, want)
	}
	if a.calls != 1 || b.calls != 1 {
		t.Errorf("expected a=1 b=1 calls, got a=%d b=%d", a.calls, b.calls)
	}
	if c.calls != 0 {
		t.Errorf("check after failing one must not be called, got c.calls=%d", c.calls)
	}
}

func TestEngine_EvaluateRespectsContextCancel(t *testing.T) {
	t.Parallel()

	a := newStub("a", policy.StatusPassed)
	b := newStub("b", policy.StatusPassed)
	eng := policy.NewEngine(a, b)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := eng.Evaluate(ctx, policy.MechanismsConfig{}, policy.CheckInput{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestEngine_ChecksReturnsCopy(t *testing.T) {
	t.Parallel()

	a := newStub("a", policy.StatusPassed)
	eng := policy.NewEngine(a)
	got := eng.Checks()
	if len(got) != 1 {
		t.Fatalf("Checks() len = %d, want 1", len(got))
	}
	got[0] = nil
	// внутренний слайс не должен был поменяться
	got2 := eng.Checks()
	if got2[0] == nil {
		t.Fatalf("Checks() leaked internal slice")
	}
}
