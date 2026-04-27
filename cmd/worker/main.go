package main

import (
	"log"
	"os"
	"strings"

	"github.com/barteq100/rccc-ingestion/internal/scheduler"
	"github.com/barteq100/rccc-ingestion/internal/sources"
)

func main() {
	log.Printf(
		"rccc-ingestion worker contract baseline ready api=%s schedule=%s sources=%s stages=%s",
		envOrDefault("RCCC_API_BASE_URL", "http://api:8080"),
		envOrDefault("INGEST_SCHEDULE", "@every 15m"),
		strings.Join([]string{string(sources.SourceGreenhouse), string(sources.SourceLever)}, ","),
		strings.Join(scheduler.DefaultStageOrder, " -> "),
	)
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
