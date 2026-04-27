package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/barteq100/rccc-ingestion/internal/normalize"
)

func TestHTTPClientUpsertJobsUsesCanonicalEndpointAndPayload(t *testing.T) {
	var seenPath string
	var seenMethod string
	var request struct {
		Jobs []normalize.Job `json:"jobs"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenMethod = r.Method
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("Decode returned error: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(UpsertJobsResult{Received: 1, Upserted: 1})
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewHTTPClient returned error: %v", err)
	}

	result, err := client.UpsertJobs(context.Background(), []normalize.Job{sampleJob()})
	if err != nil {
		t.Fatalf("UpsertJobs returned error: %v", err)
	}

	if seenMethod != http.MethodPost || seenPath != jobsUpsertPath {
		t.Fatalf("expected POST %s, got %s %s", jobsUpsertPath, seenMethod, seenPath)
	}
	if len(request.Jobs) != 1 || request.Jobs[0].ID != "gh-acme-senior-go-001" {
		t.Fatalf("unexpected request payload: %#v", request)
	}
	if result.Received != 1 || result.Upserted != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestHTTPClientUpsertJobsReturnsValidationErrorDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":   "validation_failed",
			"message": "request validation failed",
			"details": []map[string]string{{
				"field":   "jobs[0].title",
				"message": "must not be empty",
			}},
		})
	}))
	defer server.Close()

	client, err := NewHTTPClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewHTTPClient returned error: %v", err)
	}

	_, err = client.UpsertJobs(context.Background(), []normalize.Job{sampleJob()})
	if err == nil {
		t.Fatal("expected validation error")
	}

	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if len(validationErr.Details) != 1 || validationErr.Details[0].Field != "jobs[0].title" {
		t.Fatalf("unexpected validation details: %#v", validationErr.Details)
	}
}

func sampleJob() normalize.Job {
	return normalize.Job{
		ID:          "gh-acme-senior-go-001",
		Title:       "Senior Go Engineer",
		Company:     "Acme",
		Location:    "Remote - Poland",
		Remote:      true,
		Description: "Build backend services for remote teams.",
		Source:      "greenhouse",
		SourceURL:   "https://boards.greenhouse.io/acme/jobs/1",
		PostedAt:    time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC),
		IngestedAt:  time.Date(2026, 3, 23, 8, 0, 0, 0, time.UTC),
	}
}
