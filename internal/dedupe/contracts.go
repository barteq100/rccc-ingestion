package dedupe

import (
	"fmt"
	"strings"
	"time"

	"github.com/barteq100/rccc-ingestion/internal/normalize"
)

// Candidate is the duplicate-comparison boundary for canonical jobs.
type Candidate struct {
	Job           normalize.Job
	DuplicateKey  string
	PreferenceKey string
}

// Result reports which jobs were delivered and which were dropped as duplicates.
type Result struct {
	UniqueJobs     []normalize.Job
	DuplicateJobs  []normalize.Job
	DuplicateCount int
}

// Service applies deterministic duplicate heuristics to canonical jobs.
type Service struct{}

// New constructs a duplicate-filtering service.
func New() *Service {
	return &Service{}
}

// Evaluate drops obvious duplicates and keeps the preferred representative.
func (s *Service) Evaluate(jobs []normalize.Job) Result {
	candidates := make([]Candidate, 0, len(jobs))
	for _, job := range jobs {
		candidates = append(candidates, NewCandidate(job))
	}

	uniqueByKey := make(map[string]Candidate, len(candidates))
	duplicates := make([]normalize.Job, 0)
	for _, candidate := range candidates {
		existing, found := uniqueByKey[candidate.DuplicateKey]
		if !found {
			uniqueByKey[candidate.DuplicateKey] = candidate
			continue
		}

		if prefer(candidate, existing) {
			duplicates = append(duplicates, existing.Job)
			uniqueByKey[candidate.DuplicateKey] = candidate
			continue
		}

		duplicates = append(duplicates, candidate.Job)
	}

	unique := make([]normalize.Job, 0, len(uniqueByKey))
	for _, candidate := range candidates {
		selected, ok := uniqueByKey[candidate.DuplicateKey]
		if !ok || selected.Job.ID != candidate.Job.ID {
			continue
		}
		unique = append(unique, candidate.Job)
		delete(uniqueByKey, candidate.DuplicateKey)
	}

	return Result{
		UniqueJobs:     unique,
		DuplicateJobs:  duplicates,
		DuplicateCount: len(duplicates),
	}
}

// NewCandidate converts a canonical job into the dedupe comparison shape.
func NewCandidate(job normalize.Job) Candidate {
	return Candidate{
		Job:           job,
		DuplicateKey:  buildDuplicateKey(job),
		PreferenceKey: buildPreferenceKey(job),
	}
}

func buildDuplicateKey(job normalize.Job) string {
	postedDay := job.PostedAt.UTC().Format(time.DateOnly)
	return strings.Join([]string{
		canonicalText(job.Company),
		canonicalText(job.Title),
		canonicalText(job.Location),
		fmt.Sprintf("%t", job.Remote),
		postedDay,
	}, "|")
}

func buildPreferenceKey(job normalize.Job) string {
	return fmt.Sprintf("%020d|%s|%s", -job.PostedAt.UTC().Unix(), canonicalText(job.Source), job.ID)
}

func prefer(candidate Candidate, existing Candidate) bool {
	if !candidate.Job.PostedAt.Equal(existing.Job.PostedAt) {
		return candidate.Job.PostedAt.After(existing.Job.PostedAt)
	}

	return candidate.PreferenceKey < existing.PreferenceKey
}

func canonicalText(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}
