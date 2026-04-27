package scheduler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/barteq100/rccc-ingestion/internal/client"
	"github.com/barteq100/rccc-ingestion/internal/dedupe"
	"github.com/barteq100/rccc-ingestion/internal/normalize"
	"github.com/barteq100/rccc-ingestion/internal/sources"
)

func TestPipelineRunOnceUsesExpectedStageOrder(t *testing.T) {
	steps := make([]string, 0)
	adapter := &stubAdapter{steps: &steps}
	normalizer := &stubNormalizer{steps: &steps}
	deduper := &stubDeduper{steps: &steps}
	deliverer := &stubDeliverer{steps: &steps}

	pipeline := NewPipeline([]sources.Adapter{adapter}, normalizer, deduper, deliverer)
	result, err := pipeline.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	expectedSteps := []string{"fetch", "parse", "normalize", "dedupe", "deliver"}
	if !reflect.DeepEqual(steps, expectedSteps) {
		t.Fatalf("expected steps %v, got %v", expectedSteps, steps)
	}
	if result.FetchedItems != 1 || result.ParsedJobs != 1 || result.NormalizedJobs != 1 || result.DeliveredJobs != 1 {
		t.Fatalf("unexpected pipeline result: %#v", result)
	}
}

type stubAdapter struct {
	steps *[]string
}

func (a *stubAdapter) Source() sources.Name { return sources.SourceGreenhouse }

func (a *stubAdapter) Fetch(context.Context) ([]sources.FetchItem, error) {
	*a.steps = append(*a.steps, "fetch")
	return []sources.FetchItem{{
		RequestURL:  "https://example.com/jobs",
		Payload:     []byte(`{"jobs":[]}`),
		RetrievedAt: time.Date(2026, 4, 27, 9, 0, 0, 0, time.UTC),
	}}, nil
}

func (a *stubAdapter) Parse(context.Context, sources.FetchItem) ([]sources.ParsedJob, error) {
	*a.steps = append(*a.steps, "parse")
	return []sources.ParsedJob{{
		ProviderJobID: "job-1",
		Title:         "Senior Go Engineer",
		Company:       "Acme",
		Location:      "Remote",
		Remote:        true,
		Description:   "Build services.",
		SourceURL:     "https://example.com/jobs/1",
		PostedAt:      time.Date(2026, 4, 26, 9, 0, 0, 0, time.UTC),
	}}, nil
}

type stubNormalizer struct {
	steps *[]string
}

func (n *stubNormalizer) Job(input normalize.Input) (normalize.Job, error) {
	*n.steps = append(*n.steps, "normalize")
	return normalize.Job{
		ID:          "greenhouse-job-1",
		Title:       input.ParsedJob.Title,
		Company:     input.ParsedJob.Company,
		Location:    input.ParsedJob.Location,
		Remote:      input.ParsedJob.Remote,
		Description: input.ParsedJob.Description,
		Source:      string(input.Source),
		SourceURL:   input.ParsedJob.SourceURL,
		PostedAt:    input.ParsedJob.PostedAt,
		IngestedAt:  input.RetrievedAt,
	}, nil
}

type stubDeduper struct {
	steps *[]string
}

func (d *stubDeduper) Evaluate(jobs []normalize.Job) dedupe.Result {
	*d.steps = append(*d.steps, "dedupe")
	return dedupe.Result{UniqueJobs: jobs}
}

type stubDeliverer struct {
	steps *[]string
}

func (d *stubDeliverer) UpsertJobs(context.Context, []normalize.Job) (client.UpsertJobsResult, error) {
	*d.steps = append(*d.steps, "deliver")
	return client.UpsertJobsResult{Received: 1, Upserted: 1}, nil
}
