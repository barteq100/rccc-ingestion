package scheduler

import (
	"context"

	"github.com/barteq100/rccc-ingestion/internal/client"
	"github.com/barteq100/rccc-ingestion/internal/dedupe"
	"github.com/barteq100/rccc-ingestion/internal/normalize"
	"github.com/barteq100/rccc-ingestion/internal/sources"
)

// DefaultStageOrder is the required pipeline sequence for every ingestion run.
var DefaultStageOrder = []string{"fetch", "parse", "normalize", "dedupe", "deliver"}

// Normalizer is the normalization boundary used by the pipeline.
type Normalizer interface {
	Job(normalize.Input) (normalize.Job, error)
}

// Deduper is the duplicate-filtering boundary used by the pipeline.
type Deduper interface {
	Evaluate([]normalize.Job) dedupe.Result
}

// Deliverer is the API client delivery boundary used by the pipeline.
type Deliverer interface {
	UpsertJobs(context.Context, []normalize.Job) (client.UpsertJobsResult, error)
}

// Pipeline runs the canonical ingestion stage order for a set of source adapters.
type Pipeline struct {
	adapters   []sources.Adapter
	normalizer Normalizer
	deduper    Deduper
	deliverer  Deliverer
}

// RunResult summarizes a single pipeline execution.
type RunResult struct {
	FetchedItems    int
	ParsedJobs      int
	NormalizedJobs  int
	DedupedJobs     int
	DroppedJobs     int
	DeliveredJobs   int
	DeliveredResult client.UpsertJobsResult
}

// NewPipeline constructs the ingestion pipeline.
func NewPipeline(adapters []sources.Adapter, normalizer Normalizer, deduper Deduper, deliverer Deliverer) *Pipeline {
	return &Pipeline{
		adapters:   adapters,
		normalizer: normalizer,
		deduper:    deduper,
		deliverer:  deliverer,
	}
}

// RunOnce executes one ingestion cycle using the fixed stage order.
func (p *Pipeline) RunOnce(ctx context.Context) (RunResult, error) {
	var result RunResult
	normalizedJobs := make([]normalize.Job, 0)

	for _, adapter := range p.adapters {
		items, err := adapter.Fetch(ctx)
		if err != nil {
			return RunResult{}, err
		}
		result.FetchedItems += len(items)

		for _, item := range items {
			parsedJobs, err := adapter.Parse(ctx, item)
			if err != nil {
				return RunResult{}, err
			}
			result.ParsedJobs += len(parsedJobs)

			for _, parsedJob := range parsedJobs {
				job, err := p.normalizer.Job(normalize.Input{
					Source:      adapter.Source(),
					RetrievedAt: item.RetrievedAt,
					ParsedJob:   parsedJob,
				})
				if err != nil {
					return RunResult{}, err
				}
				normalizedJobs = append(normalizedJobs, job)
				result.NormalizedJobs++
			}
		}
	}

	dedupeResult := p.deduper.Evaluate(normalizedJobs)
	result.DedupedJobs = len(dedupeResult.UniqueJobs)
	result.DroppedJobs = dedupeResult.DuplicateCount

	if len(dedupeResult.UniqueJobs) == 0 {
		return result, nil
	}

	deliveredResult, err := p.deliverer.UpsertJobs(ctx, dedupeResult.UniqueJobs)
	if err != nil {
		return RunResult{}, err
	}

	result.DeliveredJobs = len(dedupeResult.UniqueJobs)
	result.DeliveredResult = deliveredResult
	return result, nil
}
