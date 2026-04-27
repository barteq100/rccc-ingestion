package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/barteq100/rccc-ingestion/internal/normalize"
)

const jobsUpsertPath = "/internal/ingestion/jobs"

// UpsertJobsResult mirrors the ingestion-facing API response payload.
type UpsertJobsResult struct {
	Received int `json:"received"`
	Upserted int `json:"upserted"`
}

// ValidationIssue mirrors the structured validation errors returned by rccc-api.
type ValidationIssue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError is returned when rccc-api rejects a canonical batch payload.
type ValidationError struct {
	Details []ValidationIssue
}

func (e ValidationError) Error() string {
	return "rccc-api rejected canonical jobs payload"
}

// HTTPError is returned for non-validation API failures.
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("rccc-api returned status %d: %s", e.StatusCode, e.Message)
}

type upsertJobsRequest struct {
	Jobs []normalize.Job `json:"jobs"`
}

type errorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Details []ValidationIssue `json:"details,omitempty"`
}

// HTTPClient delivers canonical jobs to the API via REST.
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPClient constructs a REST client for rccc-api.
func NewHTTPClient(baseURL string, httpClient *http.Client) (*HTTPClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, errors.New("base URL must not be empty")
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	return &HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}, nil
}

// UpsertJobs sends the canonical job payload to the ingestion-facing jobs endpoint.
func (c *HTTPClient) UpsertJobs(ctx context.Context, jobs []normalize.Job) (UpsertJobsResult, error) {
	body, err := json.Marshal(upsertJobsRequest{Jobs: jobs})
	if err != nil {
		return UpsertJobsResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+jobsUpsertPath, bytes.NewReader(body))
	if err != nil {
		return UpsertJobsResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return UpsertJobsResult{}, err
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return UpsertJobsResult{}, err
	}

	if res.StatusCode == http.StatusOK {
		var result UpsertJobsResult
		if err := json.Unmarshal(responseBody, &result); err != nil {
			return UpsertJobsResult{}, err
		}
		return result, nil
	}

	var apiErr errorResponse
	if err := json.Unmarshal(responseBody, &apiErr); err != nil {
		return UpsertJobsResult{}, HTTPError{StatusCode: res.StatusCode, Message: strings.TrimSpace(string(responseBody))}
	}

	if res.StatusCode == http.StatusBadRequest && apiErr.Error == "validation_failed" {
		return UpsertJobsResult{}, ValidationError{Details: apiErr.Details}
	}

	return UpsertJobsResult{}, HTTPError{StatusCode: res.StatusCode, Message: apiErr.Message}
}
