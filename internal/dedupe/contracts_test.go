package dedupe

import (
	"testing"
	"time"

	"github.com/barteq100/rccc-ingestion/internal/normalize"
)

func TestNewCandidateBuildsStableDuplicateKey(t *testing.T) {
	job := normalize.Job{
		ID:       "greenhouse-acme-1",
		Title:    "Senior Go Engineer",
		Company:  " Acme ",
		Location: " Remote - Poland ",
		Remote:   true,
		PostedAt: time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
		Source:   "greenhouse",
	}

	candidate := NewCandidate(job)

	if candidate.DuplicateKey != "acme|senior go engineer|remote - poland|true|2026-04-26" {
		t.Fatalf("unexpected duplicate key: %q", candidate.DuplicateKey)
	}
}

func TestServiceEvaluateKeepsPreferredRecordAndDropsDuplicates(t *testing.T) {
	service := New()
	older := sampleJob("greenhouse-acme-1", "greenhouse", time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC))
	newer := sampleJob("lever-acme-1", "lever", time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC))

	result := service.Evaluate([]normalize.Job{older, newer})

	if len(result.UniqueJobs) != 1 || result.UniqueJobs[0].ID != newer.ID {
		t.Fatalf("expected newer job to win dedupe, got %#v", result.UniqueJobs)
	}
	if result.DuplicateCount != 1 || len(result.DuplicateJobs) != 1 || result.DuplicateJobs[0].ID != older.ID {
		t.Fatalf("expected older job to be dropped, got %#v", result)
	}
}

func sampleJob(id string, source string, postedAt time.Time) normalize.Job {
	return normalize.Job{
		ID:          id,
		Title:       "Senior Go Engineer",
		Company:     "Acme",
		Location:    "Remote - Poland",
		Remote:      true,
		Description: "Build services.",
		Source:      source,
		SourceURL:   "https://example.com/jobs/1",
		PostedAt:    postedAt,
		IngestedAt:  postedAt.Add(time.Hour),
	}
}
