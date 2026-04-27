package normalize

import (
	"strings"
	"testing"
	"time"

	"github.com/barteq100/rccc-ingestion/internal/sources"
)

func TestNormalizerBuildsCanonicalJobFromParsedRecord(t *testing.T) {
	now := time.Date(2026, 4, 27, 9, 0, 0, 0, time.UTC)
	normalizer := New(func() time.Time { return now })

	job, err := normalizer.Job(Input{
		Source:      sources.SourceGreenhouse,
		RetrievedAt: now.Add(-time.Minute),
		ParsedJob: sources.ParsedJob{
			ProviderJobID: " Senior Go 001 ",
			Title:         " Senior Go Engineer ",
			Company:       " Acme ",
			Location:      " Remote - Poland ",
			Remote:        true,
			Description:   " Build backend services. ",
			SourceURL:     " https://boards.greenhouse.io/acme/jobs/1 ",
			PostedAt:      time.Date(2026, 4, 26, 14, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("Job returned error: %v", err)
	}

	if job.ID != "greenhouse-senior-go-001" {
		t.Fatalf("expected canonical id from provider id, got %q", job.ID)
	}
	if job.Title != "Senior Go Engineer" || job.Company != "Acme" || job.Location != "Remote - Poland" {
		t.Fatalf("expected trimmed canonical fields, got %#v", job)
	}
	if !job.IngestedAt.Equal(now.Add(-time.Minute)) {
		t.Fatalf("expected retrieved_at to become ingested_at, got %s", job.IngestedAt)
	}
}

func TestNormalizerFallsBackToStableHashWhenProviderIDMissing(t *testing.T) {
	normalizer := New(func() time.Time { return time.Date(2026, 4, 27, 9, 0, 0, 0, time.UTC) })

	job, err := normalizer.Job(Input{
		Source: sources.SourceLever,
		ParsedJob: sources.ParsedJob{
			Title:       "Platform Engineer",
			Company:     "Orbit",
			Location:    "Remote",
			Description: "Own platform delivery.",
			SourceURL:   "https://jobs.lever.co/orbit/abc",
			PostedAt:    time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("Job returned error: %v", err)
	}

	if !strings.HasPrefix(job.ID, "lever-") {
		t.Fatalf("expected lever-prefixed id, got %q", job.ID)
	}
	if len(job.ID) != len("lever-")+16 {
		t.Fatalf("expected 16-hex hash suffix, got %q", job.ID)
	}
}

func TestNormalizerRejectsMissingRequiredFields(t *testing.T) {
	normalizer := New(nil)

	_, err := normalizer.Job(Input{
		Source: sources.SourceGreenhouse,
		ParsedJob: sources.ParsedJob{
			Company:   "Acme",
			Location:  "Remote",
			SourceURL: "https://example.com/jobs/1",
			PostedAt:  time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC),
		},
	})
	if err == nil {
		t.Fatal("expected missing title error")
	}
	if !strings.Contains(err.Error(), "title") {
		t.Fatalf("expected title error, got %v", err)
	}
}
