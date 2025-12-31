package sync

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewResult(t *testing.T) {
	result := NewResult()

	if result == nil {
		t.Fatal("NewResult() returned nil")
	}

	if result.Successes == nil {
		t.Error("Successes slice should be initialized")
	}

	if result.Failures == nil {
		t.Error("Failures slice should be initialized")
	}

	if len(result.Successes) != 0 {
		t.Error("Successes should be empty initially")
	}

	if len(result.Failures) != 0 {
		t.Error("Failures should be empty initially")
	}
}

func TestResult_AddSuccess(t *testing.T) {
	result := NewResult()

	docResult := &DocumentResult{
		DocumentID: "test-123",
		Title:      "Test Doc",
		PageCount:  5,
		OutputPath: "/tmp/test.pdf",
		Duration:   time.Second * 10,
	}

	result.AddSuccess(docResult)

	if result.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", result.SuccessCount)
	}

	if len(result.Successes) != 1 {
		t.Errorf("len(Successes) = %d, want 1", len(result.Successes))
	}

	if result.Successes[0].DocumentID != "test-123" {
		t.Error("Added success document ID doesn't match")
	}
}

func TestResult_AddError(t *testing.T) {
	result := NewResult()

	err := fmt.Errorf("test error")
	result.AddError("test-456", "Failed Doc", err)

	if result.FailureCount != 1 {
		t.Errorf("FailureCount = %d, want 1", result.FailureCount)
	}

	if len(result.Failures) != 1 {
		t.Errorf("len(Failures) = %d, want 1", len(result.Failures))
	}

	failure := result.Failures[0]
	if failure.DocumentID != "test-456" {
		t.Error("Added failure document ID doesn't match")
	}

	if failure.Title != "Failed Doc" {
		t.Error("Added failure title doesn't match")
	}

	if failure.Error == nil {
		t.Error("Added failure error should not be nil")
	}
}

func TestResult_HasFailures(t *testing.T) {
	result := NewResult()

	if result.HasFailures() {
		t.Error("HasFailures() should be false initially")
	}

	result.AddError("test-1", "Doc 1", fmt.Errorf("error 1"))

	if !result.HasFailures() {
		t.Error("HasFailures() should be true after adding error")
	}
}

func TestResult_Summary(t *testing.T) {
	result := NewResult()
	result.TotalDocuments = 10
	result.ProcessedDocuments = 8
	result.Duration = time.Second * 30

	// Add successes
	result.AddSuccess(&DocumentResult{
		DocumentID: "doc-1",
		Title:      "Success 1",
	})
	result.AddSuccess(&DocumentResult{
		DocumentID: "doc-2",
		Title:      "Success 2",
	})

	// Add failures
	result.AddError("doc-3", "Failed Doc", fmt.Errorf("test error"))

	summary := result.Summary()

	// Check that summary contains expected information
	expectedStrings := []string{
		"Sync Summary",
		"Total Documents: 10",
		"Processed: 8",
		"Successful: 2",
		"Failed: 1",
		"Duration: 30s",
		"Failures:",
		"Failed Doc",
		"doc-3",
		"test error",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(summary, expected) {
			t.Errorf("Summary should contain %q, got:\n%s", expected, summary)
		}
	}
}

func TestResult_Summary_NoFailures(t *testing.T) {
	result := NewResult()
	result.TotalDocuments = 5
	result.ProcessedDocuments = 5
	result.Duration = time.Second * 15

	result.AddSuccess(&DocumentResult{
		DocumentID: "doc-1",
		Title:      "Success 1",
	})

	summary := result.Summary()

	// Should not contain "Failures:" section when there are no failures
	if strings.Contains(summary, "Failures:") {
		t.Error("Summary should not contain Failures section when there are no failures")
	}

	// Should contain success information
	if !strings.Contains(summary, "Successful: 1") {
		t.Error("Summary should contain success count")
	}
}

func TestResult_String(t *testing.T) {
	result := NewResult()
	result.TotalDocuments = 5
	result.AddSuccess(&DocumentResult{DocumentID: "doc-1"})

	str := result.String()
	summary := result.Summary()

	if str != summary {
		t.Error("String() should return the same as Summary()")
	}
}

func TestDocumentResult(t *testing.T) {
	result := &DocumentResult{
		DocumentID: "test-doc-id",
		Title:      "Test Document",
		PageCount:  10,
		OutputPath: "/output/test.pdf",
		StartTime:  time.Now().Add(-time.Minute),
		Duration:   time.Minute,
	}

	if result.DocumentID != "test-doc-id" {
		t.Error("DocumentID doesn't match")
	}

	if result.PageCount != 10 {
		t.Error("PageCount doesn't match")
	}

	if result.Duration != time.Minute {
		t.Error("Duration doesn't match")
	}
}

func TestDocumentFailure(t *testing.T) {
	err := fmt.Errorf("test failure")
	failure := DocumentFailure{
		DocumentID: "failed-doc",
		Title:      "Failed Document",
		Error:      err,
	}

	if failure.DocumentID != "failed-doc" {
		t.Error("DocumentID doesn't match")
	}

	if failure.Title != "Failed Document" {
		t.Error("Title doesn't match")
	}

	if failure.Error != err {
		t.Error("Error doesn't match")
	}
}
