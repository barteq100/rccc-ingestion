package sources

import (
	"context"
	"time"
)

// Name identifies a supported upstream job source.
type Name string

const (
	SourceGreenhouse Name = "greenhouse"
	SourceLever      Name = "lever"
)

// FetchItem is the raw payload fetched from a provider endpoint.
type FetchItem struct {
	RequestURL  string
	Payload     []byte
	RetrievedAt time.Time
}

// ParsedJob is the provider-neutral shape extracted from a source payload
// before canonical normalization.
type ParsedJob struct {
	ProviderJobID string
	Title         string
	Company       string
	Location      string
	Remote        bool
	Description   string
	SourceURL     string
	PostedAt      time.Time
}

// Adapter isolates provider-specific fetch and parse logic.
type Adapter interface {
	Source() Name
	Fetch(context.Context) ([]FetchItem, error)
	Parse(context.Context, FetchItem) ([]ParsedJob, error)
}
