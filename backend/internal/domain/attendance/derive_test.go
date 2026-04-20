package attendance_test

import (
	"testing"

	"attendance/internal/domain/attendance"
	"attendance/internal/domain/policy"
)

func TestDerivePreliminaryStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		results []policy.CheckResult
		want    attendance.Status
	}{
		{
			name:    "пусто → accepted",
			results: nil,
			want:    attendance.StatusAccepted,
		},
		{
			name: "все passed → accepted",
			results: []policy.CheckResult{
				{Status: policy.StatusPassed},
				{Status: policy.StatusPassed},
			},
			want: attendance.StatusAccepted,
		},
		{
			name: "все skipped → accepted",
			results: []policy.CheckResult{
				{Status: policy.StatusSkipped},
				{Status: policy.StatusSkipped},
			},
			want: attendance.StatusAccepted,
		},
		{
			name: "passed + skipped → accepted",
			results: []policy.CheckResult{
				{Status: policy.StatusPassed},
				{Status: policy.StatusSkipped},
			},
			want: attendance.StatusAccepted,
		},
		{
			name: "один failed среди passed → needs_review",
			results: []policy.CheckResult{
				{Status: policy.StatusPassed},
				{Status: policy.StatusFailed},
			},
			want: attendance.StatusNeedsReview,
		},
		{
			name: "failed среди skipped → needs_review",
			results: []policy.CheckResult{
				{Status: policy.StatusSkipped},
				{Status: policy.StatusFailed},
			},
			want: attendance.StatusNeedsReview,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := attendance.DerivePreliminaryStatus(tt.results)
			if got != tt.want {
				t.Fatalf("DerivePreliminaryStatus: got=%s want=%s", got, tt.want)
			}
		})
	}
}
