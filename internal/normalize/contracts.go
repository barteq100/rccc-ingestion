package normalize

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/barteq100/rccc-ingestion/internal/sources"
)

var errMissingRequiredField = errors.New("missing required field")

// Input is the normalization boundary between provider parsing and canonical jobs.
type Input struct {
	Source      sources.Name
	RetrievedAt time.Time
	ParsedJob   sources.ParsedJob
}

// Job is the canonical payload delivered to rccc-api.
type Job struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Company     string    `json:"company"`
	Location    string    `json:"location"`
	Remote      bool      `json:"remote"`
	Description string    `json:"description"`
	Source      string    `json:"source"`
	SourceURL   string    `json:"source_url"`
	PostedAt    time.Time `json:"posted_at"`
	IngestedAt  time.Time `json:"ingested_at"`
}

// Normalizer constructs canonical jobs from provider-neutral parsed records.
type Normalizer struct {
	clock func() time.Time
}

// New constructs a Normalizer with an optional clock.
func New(clock func() time.Time) *Normalizer {
	if clock == nil {
		clock = time.Now
	}

	return &Normalizer{clock: clock}
}

// Job converts a parsed source record into the shared canonical job payload.
func (n *Normalizer) Job(input Input) (Job, error) {
	source := strings.TrimSpace(string(input.Source))
	parsed := trimParsedJob(input.ParsedJob)
	if source == "" {
		return Job{}, fmt.Errorf("source: %w", errMissingRequiredField)
	}

	switch {
	case parsed.Title == "":
		return Job{}, fmt.Errorf("title: %w", errMissingRequiredField)
	case parsed.Company == "":
		return Job{}, fmt.Errorf("company: %w", errMissingRequiredField)
	case parsed.Location == "":
		return Job{}, fmt.Errorf("location: %w", errMissingRequiredField)
	case parsed.Description == "":
		return Job{}, fmt.Errorf("description: %w", errMissingRequiredField)
	case parsed.SourceURL == "":
		return Job{}, fmt.Errorf("source_url: %w", errMissingRequiredField)
	case parsed.PostedAt.IsZero():
		return Job{}, errors.New("posted_at: missing required field")
	}

	ingestedAt := input.RetrievedAt.UTC()
	if ingestedAt.IsZero() {
		ingestedAt = n.clock().UTC()
	}

	return Job{
		ID:          buildCanonicalID(input.Source, parsed),
		Title:       parsed.Title,
		Company:     parsed.Company,
		Location:    parsed.Location,
		Remote:      parsed.Remote,
		Description: parsed.Description,
		Source:      source,
		SourceURL:   parsed.SourceURL,
		PostedAt:    parsed.PostedAt.UTC(),
		IngestedAt:  ingestedAt,
	}, nil
}

func trimParsedJob(job sources.ParsedJob) sources.ParsedJob {
	job.ProviderJobID = strings.TrimSpace(job.ProviderJobID)
	job.Title = strings.TrimSpace(job.Title)
	job.Company = strings.TrimSpace(job.Company)
	job.Location = strings.TrimSpace(job.Location)
	job.Description = strings.TrimSpace(job.Description)
	job.SourceURL = strings.TrimSpace(job.SourceURL)
	return job
}

func buildCanonicalID(source sources.Name, job sources.ParsedJob) string {
	if slug := slugify(job.ProviderJobID); slug != "" {
		return fmt.Sprintf("%s-%s", source, slug)
	}

	sum := sha1.Sum([]byte(strings.ToLower(strings.TrimSpace(string(source) + "|" + job.SourceURL))))
	return fmt.Sprintf("%s-%s", source, hex.EncodeToString(sum[:8]))
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	var builder strings.Builder
	lastHyphen := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastHyphen = false
		case !lastHyphen:
			builder.WriteByte('-')
			lastHyphen = true
		}
	}

	return strings.Trim(builder.String(), "-")
}
