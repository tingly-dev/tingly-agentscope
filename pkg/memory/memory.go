package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/module"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// Memory is the interface for agent memory implementations
type Memory interface {
	// Add adds a message to memory
	Add(ctx context.Context, msg *message.Msg) error

	// GetMessages returns all messages in memory
	GetMessages() []*message.Msg

	// GetLastN returns the last n messages
	GetLastN(n int) []*message.Msg

	// Clear clears all messages from memory
	Clear()

	// Size returns the number of messages in memory
	Size() int
}

// MemoryBase extends the Memory interface with additional features
type MemoryBase interface {
	Memory

	// AddWithMark adds a message with associated marks
	AddWithMark(ctx context.Context, msg *message.Msg, marks []string) error

	// Delete removes messages by their IDs
	Delete(ctx context.Context, msgIds []string) (int, error)

	// DeleteByMark removes messages by their marks
	DeleteByMark(ctx context.Context, marks []string) (int, error)

	// GetMemory returns messages filtered by mark
	GetMemory(ctx context.Context, mark string, excludeMark string, prependSummary bool) ([]*message.Msg, error)

	// UpdateMessagesMark updates marks on messages
	UpdateMessagesMark(ctx context.Context, newMark *string, oldMark *string, msgIds []string) (int, error)

	// UpdateCompressedSummary updates the compressed summary
	UpdateCompressedSummary(ctx context.Context, summary string) error

	// GetCompressedSummary returns the compressed summary
	GetCompressedSummary() string
}

// Config holds the configuration for memory
type Config struct {
	MaxSize int `json:"max_size"`
}

// DefaultConfig returns the default memory configuration
func DefaultConfig() *Config {
	return &Config{
		MaxSize: 1000,
	}
}

// History implements an in-memory message store with full feature support
type History struct {
	*module.StateModuleBase
	mu                sync.RWMutex
	messages          []*memoryEntry
	maxSize           int
	compressedSummary string
	allowDuplicates   bool
}

// memoryEntry stores a message with its associated marks
type memoryEntry struct {
	Message *message.Msg
	Marks   []string
}

// NewHistory creates a new history memory
func NewHistory(maxSize int) *History {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &History{
		StateModuleBase:   module.NewStateModuleBase(),
		messages:          make([]*memoryEntry, 0, maxSize),
		maxSize:           maxSize,
		compressedSummary: "",
		allowDuplicates:   false,
	}
}

// NewHistoryWithDuplicates creates a history that allows duplicate messages
func NewHistoryWithDuplicates(maxSize int) *History {
	h := NewHistory(maxSize)
	h.allowDuplicates = true
	return h
}

// Add adds a message to memory
func (h *History) Add(ctx context.Context, msg *message.Msg) error {
	return h.AddWithMark(ctx, msg, nil)
}

// AddWithMark adds a message with associated marks
func (h *History) AddWithMark(ctx context.Context, msg *message.Msg, marks []string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check for duplicates if not allowed
	if !h.allowDuplicates {
		for _, entry := range h.messages {
			if entry.Message.ID == msg.ID {
				return nil // Skip duplicate
			}
		}
	}

	entry := &memoryEntry{
		Message: msg,
		Marks:   marks,
	}

	h.messages = append(h.messages, entry)

	// Trim if over max size
	if h.maxSize > 0 && len(h.messages) > h.maxSize {
		h.messages = h.messages[len(h.messages)-h.maxSize:]
	}

	return nil
}

// GetMessages returns all messages in memory
func (h *History) GetMessages() []*message.Msg {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]*message.Msg, len(h.messages))
	for i, entry := range h.messages {
		result[i] = entry.Message
	}
	return result
}

// GetLastN returns the last n messages
func (h *History) GetLastN(n int) []*message.Msg {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if n <= 0 {
		return []*message.Msg{}
	}

	start := len(h.messages) - n
	if start < 0 {
		start = 0
	}

	result := make([]*message.Msg, len(h.messages)-start)
	for i := start; i < len(h.messages); i++ {
		result[i-start] = h.messages[i].Message
	}
	return result
}

// GetMemory returns messages filtered by mark
func (h *History) GetMemory(ctx context.Context, mark string, excludeMark string, prependSummary bool) ([]*message.Msg, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var filtered []*memoryEntry

	// Filter by mark
	for _, entry := range h.messages {
		// Check if entry should be included
		include := true
		if mark != "" {
			include = hasMark(entry, mark)
		}
		if excludeMark != "" && hasMark(entry, excludeMark) {
			include = false
		}

		if include {
			filtered = append(filtered, entry)
		}
	}

	var result []*message.Msg

	// Prepend summary if requested
	if prependSummary && h.compressedSummary != "" {
		summaryMsg := message.NewMsg(
			"system",
			h.compressedSummary,
			types.RoleSystem,
		)
		result = append(result, summaryMsg)
	}

	for _, entry := range filtered {
		result = append(result, entry.Message)
	}

	return result, nil
}

// hasMark checks if an entry has a specific mark
func hasMark(entry *memoryEntry, mark string) bool {
	for _, m := range entry.Marks {
		if m == mark {
			return true
		}
	}
	return false
}

// Delete removes messages by their IDs
func (h *History) Delete(ctx context.Context, msgIds []string) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	initialSize := len(h.messages)
	var filtered []*memoryEntry

	for _, entry := range h.messages {
		found := false
		for _, id := range msgIds {
			if entry.Message.ID == id {
				found = true
				break
			}
		}
		if !found {
			filtered = append(filtered, entry)
		}
	}

	h.messages = filtered
	return initialSize - len(h.messages), nil
}

// DeleteByMark removes messages by their marks
func (h *History) DeleteByMark(ctx context.Context, marks []string) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	initialSize := len(h.messages)
	var filtered []*memoryEntry

	for _, entry := range h.messages {
		hasTargetMark := false
		for _, mark := range marks {
			if hasMark(entry, mark) {
				hasTargetMark = true
				break
			}
		}
		if !hasTargetMark {
			filtered = append(filtered, entry)
		}
	}

	h.messages = filtered
	return initialSize - len(h.messages), nil
}

// UpdateMessagesMark updates marks on messages
func (h *History) UpdateMessagesMark(ctx context.Context, newMark *string, oldMark *string, msgIds []string) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	updatedCount := 0

	for idx, entry := range h.messages {
		// Check if message ID matches
		if msgIds != nil && len(msgIds) > 0 {
			idMatched := false
			for _, id := range msgIds {
				if entry.Message.ID == id {
					idMatched = true
					break
				}
			}
			if !idMatched {
				continue
			}
		}

		// Check if old mark matches
		if oldMark != nil && !hasMark(entry, *oldMark) {
			continue
		}

		// Update marks
		var newMarks []string
		if newMark == nil {
			// Remove old mark
			for _, m := range entry.Marks {
				if oldMark == nil || m != *oldMark {
					newMarks = append(newMarks, m)
				}
			}
		} else {
			// Add or replace mark
			marksMap := make(map[string]bool)
			for _, m := range entry.Marks {
				if oldMark == nil || m != *oldMark {
					marksMap[m] = true
				}
			}
			marksMap[*newMark] = true

			for m := range marksMap {
				newMarks = append(newMarks, m)
			}
		}

		h.messages[idx].Marks = newMarks
		updatedCount++
	}

	return updatedCount, nil
}

// Clear clears all messages from memory
func (h *History) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.messages = make([]*memoryEntry, 0, h.maxSize)
}

// Size returns the number of messages in memory
func (h *History) Size() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.messages)
}

// UpdateCompressedSummary updates the compressed summary
func (h *History) UpdateCompressedSummary(ctx context.Context, summary string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.compressedSummary = summary
	return nil
}

// GetCompressedSummary returns the compressed summary
func (h *History) GetCompressedSummary() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.compressedSummary
}

// GetMessagesByRole returns messages filtered by role
func (h *History) GetMessagesByRole(role types.Role) []*message.Msg {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []*message.Msg
	for _, entry := range h.messages {
		if entry.Message.Role == role {
			result = append(result, entry.Message)
		}
	}
	return result
}

// GetMessagesAfter returns messages after the given timestamp
func (h *History) GetMessagesAfter(timestamp string) []*message.Msg {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []*message.Msg
	for _, entry := range h.messages {
		if entry.Message.Timestamp > timestamp {
			result = append(result, entry.Message)
		}
	}
	return result
}

// GetMessagesBetween returns messages between two timestamps
func (h *History) GetMessagesBetween(start, end string) []*message.Msg {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []*message.Msg
	for _, entry := range h.messages {
		if entry.Message.Timestamp >= start && entry.Message.Timestamp <= end {
			result = append(result, entry.Message)
		}
	}
	return result
}

// StateDict returns the state for serialization
func (h *History) StateDict() map[string]any {
	h.mu.RLock()
	defer h.mu.RUnlock()

	entries := make([]map[string]any, len(h.messages))
	for i, entry := range h.messages {
		entries[i] = map[string]any{
			"message": entry.Message.ToDict(),
			"marks":   entry.Marks,
		}
	}

	return map[string]any{
		"messages":           entries,
		"compressed_summary": h.compressedSummary,
		"max_size":           h.maxSize,
		"allow_duplicates":   h.allowDuplicates,
	}
}

// LoadStateDict loads state from serialized data
func (h *History) LoadStateDict(ctx context.Context, state map[string]any) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	messages, ok := state["messages"].([]any)
	if !ok {
		return fmt.Errorf("invalid state: missing messages")
	}

	h.messages = make([]*memoryEntry, 0, len(messages))
	for _, m := range messages {
		msgMap, ok := m.(map[string]any)
		if !ok {
			continue
		}

		msgData, ok := msgMap["message"].(map[string]any)
		if !ok {
			continue
		}

		msg, err := message.FromDict(msgData)
		if err != nil {
			return fmt.Errorf("failed to load message: %w", err)
		}

		var marks []string
		if marksData, ok := msgMap["marks"].([]any); ok {
			for _, mark := range marksData {
				if markStr, ok := mark.(string); ok {
					marks = append(marks, markStr)
				}
			}
		}

		h.messages = append(h.messages, &memoryEntry{
			Message: msg,
			Marks:   marks,
		})
	}

	if summary, ok := state["compressed_summary"].(string); ok {
		h.compressedSummary = summary
	}

	if maxSize, ok := state["max_size"].(float64); ok {
		h.maxSize = int(maxSize)
	}

	if allowDup, ok := state["allow_duplicates"].(bool); ok {
		h.allowDuplicates = allowDup
	}

	return nil
}

// MemoryWithEmbedding represents memory that supports embedding-based retrieval
type MemoryWithEmbedding interface {
	Memory

	// AddEmbedding adds a message with its embedding to memory
	AddEmbedding(ctx context.Context, msg *message.Msg, embedding []float32) error

	// Search searches for similar messages based on embedding
	Search(ctx context.Context, queryEmbedding []float32, topK int) []*message.Msg
}

// MessageWithMeta extends a message with metadata for memory
type MessageWithMeta struct {
	Message   *message.Msg `json:"message"`
	Embedding []float32    `json:"embedding,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
}

// EmbeddingModel is the interface for embedding models
type EmbeddingModel interface {
	// Encode generates an embedding for the given text
	Encode(ctx context.Context, text string) ([]float32, error)

	// BatchEncode generates embeddings for multiple texts
	BatchEncode(ctx context.Context, texts []string) ([][]float32, error)
}

// VectorMemory implements memory with embedding-based search
type VectorMemory struct {
	*module.StateModuleBase
	mu        sync.RWMutex
	messages  []*MessageWithMeta
	maxSize   int
	embedding EmbeddingModel
}

// NewVectorMemory creates a new vector memory
func NewVectorMemory(maxSize int, embeddingModel EmbeddingModel) *VectorMemory {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &VectorMemory{
		StateModuleBase: module.NewStateModuleBase(),
		messages:        make([]*MessageWithMeta, 0, maxSize),
		maxSize:         maxSize,
		embedding:       embeddingModel,
	}
}

// Add adds a message to memory
func (v *VectorMemory) Add(ctx context.Context, msg *message.Msg) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Generate embedding
	embedding, err := v.embedding.Encode(ctx, msg.GetTextContent())
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	meta := &MessageWithMeta{
		Message:   msg,
		Embedding: embedding,
		Timestamp: time.Now(),
	}

	v.messages = append(v.messages, meta)

	// Trim if over max size
	if v.maxSize > 0 && len(v.messages) > v.maxSize {
		v.messages = v.messages[len(v.messages)-v.maxSize:]
	}

	return nil
}

// AddEmbedding adds a message with a pre-computed embedding
func (v *VectorMemory) AddEmbedding(ctx context.Context, msg *message.Msg, embedding []float32) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	meta := &MessageWithMeta{
		Message:   msg,
		Embedding: embedding,
		Timestamp: time.Now(),
	}

	v.messages = append(v.messages, meta)

	// Trim if over max size
	if v.maxSize > 0 && len(v.messages) > v.maxSize {
		v.messages = v.messages[len(v.messages)-v.maxSize:]
	}

	return nil
}

// GetMessages returns all messages in memory
func (v *VectorMemory) GetMessages() []*message.Msg {
	v.mu.RLock()
	defer v.mu.RUnlock()

	result := make([]*message.Msg, len(v.messages))
	for i, meta := range v.messages {
		result[i] = meta.Message
	}
	return result
}

// GetLastN returns the last n messages
func (v *VectorMemory) GetLastN(n int) []*message.Msg {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if n <= 0 {
		return []*message.Msg{}
	}

	start := len(v.messages) - n
	if start < 0 {
		start = 0
	}

	result := make([]*message.Msg, len(v.messages)-start)
	for i := start; i < len(v.messages); i++ {
		result[i-start] = v.messages[i].Message
	}
	return result
}

// Clear clears all messages from memory
func (v *VectorMemory) Clear() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.messages = make([]*MessageWithMeta, 0, v.maxSize)
}

// Size returns the number of messages in memory
func (v *VectorMemory) Size() int {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return len(v.messages)
}

// Search searches for similar messages based on embedding
func (v *VectorMemory) Search(ctx context.Context, queryEmbedding []float32, topK int) []*message.Msg {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if topK <= 0 || len(v.messages) == 0 {
		return []*message.Msg{}
	}

	// Calculate cosine similarity
	type scoreResult struct {
		message *message.Msg
		score   float32
	}

	results := make([]scoreResult, 0, len(v.messages))

	for _, meta := range v.messages {
		if len(meta.Embedding) != len(queryEmbedding) {
			continue
		}

		similarity := cosineSimilarity(queryEmbedding, meta.Embedding)
		results = append(results, scoreResult{
			message: meta.Message,
			score:   similarity,
		})
	}

	// Sort by similarity (descending)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Return top K
	if topK > len(results) {
		topK = len(results)
	}

	output := make([]*message.Msg, topK)
	for i := 0; i < topK; i++ {
		output[i] = results[i].message
	}

	return output
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt32(normA) * sqrt32(normB))
}

// sqrt32 calculates square root for float32
func sqrt32(x float32) float32 {
	return float32(sqrt(float64(x)))
}

// sqrt is a simple square root implementation
func sqrt(x float64) float64 {
	// Newton's method
	z := 1.0
	for i := 0; i < 20; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}

// TemporaryBuffer is a temporary memory that clears after a certain number of operations
type TemporaryBuffer struct {
	mu       sync.RWMutex
	messages []*message.Msg
	maxOps   int
	opCount  int
}

// NewTemporaryBuffer creates a new temporary buffer
func NewTemporaryBuffer(maxOps int) *TemporaryBuffer {
	return &TemporaryBuffer{
		messages: make([]*message.Msg, 0),
		maxOps:   maxOps,
	}
}

// Add adds a message to the temporary buffer
func (t *TemporaryBuffer) Add(ctx context.Context, msg *message.Msg) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.messages = append(t.messages, msg)
	t.opCount++

	// Clear after max operations
	if t.opCount >= t.maxOps {
		t.messages = make([]*message.Msg, 0)
		t.opCount = 0
	}

	return nil
}

// GetMessages returns all messages in the buffer
func (t *TemporaryBuffer) GetMessages() []*message.Msg {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]*message.Msg, len(t.messages))
	copy(result, t.messages)
	return result
}

// GetLastN returns the last n messages
func (t *TemporaryBuffer) GetLastN(n int) []*message.Msg {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if n <= 0 {
		return []*message.Msg{}
	}

	start := len(t.messages) - n
	if start < 0 {
		start = 0
	}

	result := make([]*message.Msg, len(t.messages)-start)
	copy(result, t.messages[start:])
	return result
}

// Clear clears the buffer
func (t *TemporaryBuffer) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.messages = make([]*message.Msg, 0)
	t.opCount = 0
}

// Size returns the number of messages in the buffer
func (t *TemporaryBuffer) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return len(t.messages)
}

// Helper function to encode state to JSON
func EncodeState(state map[string]any) ([]byte, error) {
	return json.Marshal(state)
}

// Helper function to decode state from JSON
func DecodeState(data []byte) (map[string]any, error) {
	var state map[string]any
	err := json.Unmarshal(data, &state)
	return state, err
}
