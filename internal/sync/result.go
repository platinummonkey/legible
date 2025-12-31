// Package sync provides document synchronization and orchestration.
package sync

import (
	"fmt"
	"strings"
	"time"
)

// Result contains the results of a complete sync operation
type Result struct {
	TotalDocuments     int
	ProcessedDocuments int
	SuccessCount       int
	FailureCount       int
	Duration           time.Duration
	Successes          []DocumentResult
	Failures           []DocumentFailure
}

// DocumentResult contains the results of processing a single document
type DocumentResult struct {
	DocumentID string
	Title      string
	PageCount  int
	OutputPath string
	StartTime  time.Time
	Duration   time.Duration
}

// DocumentFailure contains information about a failed document
type DocumentFailure struct {
	DocumentID string
	Title      string
	Error      error
}

// NewResult creates a new sync result
func NewResult() *Result {
	return &Result{
		Successes: make([]DocumentResult, 0),
		Failures:  make([]DocumentFailure, 0),
	}
}

// AddSuccess adds a successful document result
func (sr *Result) AddSuccess(result *DocumentResult) {
	sr.Successes = append(sr.Successes, *result)
	sr.SuccessCount++
}

// AddError adds a failed document
func (sr *Result) AddError(docID, title string, err error) {
	sr.Failures = append(sr.Failures, DocumentFailure{
		DocumentID: docID,
		Title:      title,
		Error:      err,
	})
	sr.FailureCount++
}

// HasFailures returns true if there were any failures
func (sr *Result) HasFailures() bool {
	return sr.FailureCount > 0
}

// Summary returns a human-readable summary of the sync result
func (sr *Result) Summary() string {
	var sb strings.Builder

	sb.WriteString("Sync Summary:\n")
	sb.WriteString(fmt.Sprintf("  Total Documents: %d\n", sr.TotalDocuments))
	sb.WriteString(fmt.Sprintf("  Processed: %d\n", sr.ProcessedDocuments))
	sb.WriteString(fmt.Sprintf("  Successful: %d\n", sr.SuccessCount))
	sb.WriteString(fmt.Sprintf("  Failed: %d\n", sr.FailureCount))
	sb.WriteString(fmt.Sprintf("  Duration: %v\n", sr.Duration))

	if sr.HasFailures() {
		sb.WriteString("\nFailures:\n")
		for _, failure := range sr.Failures {
			sb.WriteString(fmt.Sprintf("  - %s (%s): %v\n",
				failure.Title, failure.DocumentID, failure.Error))
		}
	}

	return sb.String()
}

// String returns a string representation of the sync result
func (sr *Result) String() string {
	return sr.Summary()
}
