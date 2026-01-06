package rmclient

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/platinummonkey/legible/internal/logger"
)

// TokenMonitor tracks token renewal events and provides statistics
type TokenMonitor struct {
	logger *logger.Logger
	mu     sync.Mutex

	// Statistics
	renewalCount   int
	lastRenewal    time.Time
	renewalHistory []time.Time

	// Output file for statistics (optional)
	statsFile string
}

// TokenRenewalEvent represents a single token renewal event
type TokenRenewalEvent struct {
	Timestamp     time.Time `json:"timestamp"`
	TokenType     string    `json:"token_type"` // "user" or "device"
	ValidFor      string    `json:"valid_for"`
	TimeSinceLast string    `json:"time_since_last,omitempty"`
}

// TokenStatistics holds aggregated token statistics
type TokenStatistics struct {
	StartTime       time.Time           `json:"start_time"`
	LastUpdate      time.Time           `json:"last_update"`
	RenewalCount    int                 `json:"renewal_count"`
	RenewalEvents   []TokenRenewalEvent `json:"renewal_events"`
	AverageInterval string              `json:"average_interval,omitempty"`
}

// NewTokenMonitor creates a new token monitor
func NewTokenMonitor(log *logger.Logger, statsFile string) *TokenMonitor {
	return &TokenMonitor{
		logger:         log,
		renewalHistory: make([]time.Time, 0),
		statsFile:      statsFile,
	}
}

// RecordRenewal records a token renewal event
func (tm *TokenMonitor) RecordRenewal(tokenType string, validFor time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()

	// Calculate time since last renewal
	var timeSinceLast time.Duration
	if !tm.lastRenewal.IsZero() {
		timeSinceLast = now.Sub(tm.lastRenewal)
	}

	// Update statistics
	tm.renewalCount++
	tm.lastRenewal = now
	tm.renewalHistory = append(tm.renewalHistory, now)

	// Log the event
	fields := map[string]interface{}{
		"token_type":    tokenType,
		"renewal_count": tm.renewalCount,
		"valid_for":     validFor,
	}

	if timeSinceLast > 0 {
		fields["time_since_last"] = timeSinceLast
	}

	tm.logger.WithFields(fields).Info("Token renewal tracked")

	// Save statistics if file is configured
	if tm.statsFile != "" {
		if err := tm.saveStatistics(); err != nil {
			tm.logger.WithError(err).Warn("Failed to save token statistics")
		}
	}
}

// GetStatistics returns current token statistics
func (tm *TokenMonitor) GetStatistics() TokenStatistics {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	stats := TokenStatistics{
		RenewalCount: tm.renewalCount,
		LastUpdate:   time.Now(),
	}

	// Set start time from first renewal
	if len(tm.renewalHistory) > 0 {
		stats.StartTime = tm.renewalHistory[0]
	}

	// Build renewal events
	stats.RenewalEvents = make([]TokenRenewalEvent, 0, len(tm.renewalHistory))
	for i, timestamp := range tm.renewalHistory {
		event := TokenRenewalEvent{
			Timestamp: timestamp,
			TokenType: "user", // Currently only tracking user token renewals
		}

		// Calculate time since previous renewal
		if i > 0 {
			timeSinceLast := timestamp.Sub(tm.renewalHistory[i-1])
			event.TimeSinceLast = formatDuration(timeSinceLast)
		}

		stats.RenewalEvents = append(stats.RenewalEvents, event)
	}

	// Calculate average interval
	if len(tm.renewalHistory) > 1 {
		totalDuration := tm.renewalHistory[len(tm.renewalHistory)-1].Sub(tm.renewalHistory[0])
		avgInterval := totalDuration / time.Duration(len(tm.renewalHistory)-1)
		stats.AverageInterval = formatDuration(avgInterval)
	}

	return stats
}

// saveStatistics saves current statistics to the configured file
func (tm *TokenMonitor) saveStatistics() error {
	if tm.statsFile == "" {
		return nil
	}

	stats := tm.GetStatistics()

	// Ensure directory exists
	dir := filepath.Dir(tm.statsFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create stats directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal statistics: %w", err)
	}

	// Write to file
	if err := os.WriteFile(tm.statsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write statistics file: %w", err)
	}

	return nil
}

// PrintSummary prints a human-readable summary of token statistics
func (tm *TokenMonitor) PrintSummary() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	fmt.Println()
	fmt.Println("=== Token Renewal Statistics ===")
	fmt.Println()

	if tm.renewalCount == 0 {
		fmt.Println("No token renewals recorded yet")
		return
	}

	fmt.Printf("Total renewals: %d\n", tm.renewalCount)

	if len(tm.renewalHistory) > 0 {
		fmt.Printf("First renewal: %s\n", tm.renewalHistory[0].Format(time.RFC3339))
		fmt.Printf("Last renewal: %s\n", tm.lastRenewal.Format(time.RFC3339))

		// Calculate and display monitoring duration
		duration := tm.lastRenewal.Sub(tm.renewalHistory[0])
		fmt.Printf("Monitoring duration: %s\n", formatDuration(duration))

		// Calculate average interval
		if len(tm.renewalHistory) > 1 {
			avgInterval := duration / time.Duration(len(tm.renewalHistory)-1)
			fmt.Printf("Average interval: %s\n", formatDuration(avgInterval))
		}
	}

	fmt.Println()

	// Show recent renewals (last 5)
	if len(tm.renewalHistory) > 0 {
		fmt.Println("Recent renewals:")
		start := 0
		if len(tm.renewalHistory) > 5 {
			start = len(tm.renewalHistory) - 5
		}

		for i := start; i < len(tm.renewalHistory); i++ {
			timestamp := tm.renewalHistory[i]
			fmt.Printf("  %d. %s", i+1, timestamp.Format(time.RFC3339))

			if i > 0 {
				interval := timestamp.Sub(tm.renewalHistory[i-1])
				fmt.Printf(" (after %s)", formatDuration(interval))
			}

			fmt.Println()
		}
	}

	fmt.Println()

	if tm.statsFile != "" {
		fmt.Printf("Statistics file: %s\n", tm.statsFile)
	}
}

// formatDuration formats a duration in human-readable format
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "0s"
	}

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	result := ""
	if days > 0 {
		result += fmt.Sprintf("%dd", days)
	}
	if hours > 0 {
		if result != "" {
			result += " "
		}
		result += fmt.Sprintf("%dh", hours)
	}
	if minutes > 0 {
		if result != "" {
			result += " "
		}
		result += fmt.Sprintf("%dm", minutes)
	}
	if seconds > 0 || result == "" {
		if result != "" {
			result += " "
		}
		result += fmt.Sprintf("%ds", seconds)
	}

	return result
}
