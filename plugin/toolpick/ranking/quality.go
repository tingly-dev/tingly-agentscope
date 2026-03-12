// Package ranking provides quality-aware tool ranking.
package ranking

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// QualityRecord tracks performance metrics for a tool.
type QualityRecord struct {
	Name               string
	CallCount          int64
	SuccessCount       int64
	SuccessRate        float64
	DescriptionQuality float64
	LastUpdated        time.Time
}

// ComputeSuccessRate updates the success rate based on current metrics.
func (q *QualityRecord) ComputeSuccessRate() {
	if q.CallCount > 0 {
		q.SuccessRate = float64(q.SuccessCount) / float64(q.CallCount)
	}
}

// QualityScore computes the overall quality score.
func (q *QualityRecord) QualityScore() float64 {
	// Formula: 0.6 * success_rate + 0.3 * description_quality + 0.1 * log_factor
	successScore := 0.6 * q.SuccessRate
	descScore := 0.3 * q.DescriptionQuality

	// Log factor: log10(call_count + 1) / 10 for normalization
	logFactor := 0.1
	if q.CallCount > 0 {
		logFactor = 0.1 * (math.Log10(float64(q.CallCount+1)) / 10.0)
	}

	return successScore + descScore + logFactor
}

// GetName returns the tool name.
func (q *QualityRecord) GetNameValue() string {
	return q.Name
}

// GetCallCount returns the call count.
func (q *QualityRecord) GetCallCount() int64 {
	return q.CallCount
}

// GetSuccessCount returns the success count.
func (q *QualityRecord) GetSuccessCount() int64 {
	return q.SuccessCount
}

// GetSuccessRate returns the success rate.
func (q *QualityRecord) GetSuccessRateValue() float64 {
	return q.SuccessRate
}

// GetDescriptionQuality returns the description quality score.
func (q *QualityRecord) GetDescriptionQualityValue() float64 {
	return q.DescriptionQuality
}

// ScoredTool represents a tool with its relevance score.
type ScoredTool struct {
	Tool   ToolDefinition
	Score  float64
	Reason string
}

// ToolDefinition is a minimal interface for tool definitions.
type ToolDefinition interface {
	GetName() string
}

// QualityManager tracks tool performance and adjusts ranking.
type QualityManager struct {
	mu       sync.RWMutex
	records  map[string]*QualityRecord
	persist  bool
	cacheDir string
	dirty    bool
}

// NewQualityManager creates a new quality manager.
func NewQualityManager(cacheDir string, enablePersistence bool) *QualityManager {
	qm := &QualityManager{
		records:  make(map[string]*QualityRecord),
		persist:  enablePersistence,
		cacheDir: cacheDir,
	}

	if enablePersistence {
		qm.load()
	}

	return qm
}

// RecordExecution records a tool execution for quality tracking.
func (qm *QualityManager) RecordExecution(toolName string, success bool, duration time.Duration) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	record, ok := qm.records[toolName]
	if !ok {
		record = &QualityRecord{
			Name:               toolName,
			DescriptionQuality: 0.5, // Default until evaluated
			LastUpdated:        time.Now(),
		}
		qm.records[toolName] = record
	}

	record.CallCount++
	if success {
		record.SuccessCount++
	}
	record.ComputeSuccessRate()
	record.LastUpdated = time.Now()
	qm.dirty = true
}

// AdjustRanking adjusts tool scores based on quality metrics.
func (qm *QualityManager) AdjustRanking(tools []ScoredTool, qualityWeight float64) []ScoredTool {
	if qualityWeight <= 0 {
		return tools
	}

	qm.mu.RLock()
	defer qm.mu.RUnlock()

	adjusted := make([]ScoredTool, len(tools))
	for i, st := range tools {
		adjusted[i] = st

		record, ok := qm.records[st.Tool.GetName()]
		if !ok {
			continue
		}

		// Get quality score
		qualityScore := record.QualityScore()

		// Adjust final score
		// final_score = semantic_score * (1 - quality_weight) + quality_score * quality_weight
		adjusted[i].Score = st.Score*(1-qualityWeight) + qualityScore*qualityWeight
		if adjusted[i].Score > 1.0 {
			adjusted[i].Score = 1.0
		}

		adjusted[i].Reason = st.Reason + " (quality adjusted)"
	}

	// Re-sort by adjusted score
	for i := 0; i < len(adjusted)-1; i++ {
		for j := i + 1; j < len(adjusted); j++ {
			if adjusted[j].Score > adjusted[i].Score {
				adjusted[i], adjusted[j] = adjusted[j], adjusted[i]
			}
		}
	}

	return adjusted
}

// GetReport returns a quality report for all tracked tools.
func (qm *QualityManager) GetReport() map[string]*QualityRecord {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	// Return a copy to avoid concurrent modification
	report := make(map[string]*QualityRecord, len(qm.records))
	for k, v := range qm.records {
		recordCopy := *v
		report[k] = &recordCopy
	}

	return report
}

// Save persists quality data to disk.
func (qm *QualityManager) Save() error {
	if !qm.persist || !qm.dirty {
		return nil
	}

	qm.mu.Lock()
	defer qm.mu.Unlock()

	if err := os.MkdirAll(qm.cacheDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(qm.cacheDir, "quality.json")
	data, err := json.MarshalIndent(qm.records, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file first
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	// Atomic rename
	if err := os.Rename(tmpPath, filePath); err != nil {
		return err
	}

	qm.dirty = false
	return nil
}

// load loads quality data from disk.
func (qm *QualityManager) load() error {
	filePath := filepath.Join(qm.cacheDir, "quality.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No quality data yet
		}
		return err
	}

	return json.Unmarshal(data, &qm.records)
}

// UpdateDescriptionQuality updates the description quality for a tool.
func (qm *QualityManager) UpdateDescriptionQuality(toolName string, quality float64) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	record, ok := qm.records[toolName]
	if !ok {
		record = &QualityRecord{
			Name:         toolName,
			CallCount:    0,
			SuccessCount: 0,
			SuccessRate:  0,
			LastUpdated:  time.Now(),
		}
		qm.records[toolName] = record
	}

	record.DescriptionQuality = quality
	record.LastUpdated = time.Now()
	qm.dirty = true
}

// GetToolStats returns statistics for a specific tool.
func (qm *QualityManager) GetToolStats(toolName string) (*QualityRecord, bool) {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	record, ok := qm.records[toolName]
	if !ok {
		return nil, false
	}

	// Return a copy
	recordCopy := *record
	return &recordCopy, true
}
