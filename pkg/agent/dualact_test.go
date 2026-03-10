package agent

import (
	"context"
	"testing"

	"github.com/tingly-dev/tingly-agentscope/pkg/memory"
	"github.com/tingly-dev/tingly-agentscope/pkg/message"
	"github.com/tingly-dev/tingly-agentscope/pkg/model"
	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// createTestAgents creates test human and reactive agents for testing
func createTestAgents(responses []*model.ChatResponse) (*ReActAgent, *ReActAgent) {
	human := NewReActAgent(&ReActAgentConfig{
		Name:         "test-human",
		SystemPrompt: "Test",
		Model:        newMockModel(responses, false),
		Memory:       memory.NewHistory(10),
	})
	reactive := NewReActAgent(&ReActAgentConfig{
		Name:         "test-reactive",
		SystemPrompt: "Test",
		Model:        newMockModel(responses, false),
		Memory:       memory.NewHistory(10),
	})
	return human, reactive
}

// TestConclusion tests Conclusion methods
func TestConclusion(t *testing.T) {
	t.Run("NewConclusion creates empty conclusion", func(t *testing.T) {
		c := NewConclusion()
		if c == nil {
			t.Fatal("NewConclusion returned nil")
		}
		if len(c.Steps) != 0 {
			t.Errorf("expected empty steps, got %d", len(c.Steps))
		}
		if c.Confidence != 0.0 {
			t.Errorf("expected 0.0 confidence, got %f", c.Confidence)
		}
	})

	t.Run("AddStep adds steps", func(t *testing.T) {
		c := NewConclusion()
		c.AddStep("Step 1")
		c.AddStep("Step 2")

		if len(c.Steps) != 2 {
			t.Errorf("expected 2 steps, got %d", len(c.Steps))
		}
		if c.Steps[0] != "Step 1" {
			t.Errorf("expected 'Step 1', got '%s'", c.Steps[0])
		}
	})

	t.Run("AddArtifact adds artifacts", func(t *testing.T) {
		c := NewConclusion()
		c.AddArtifact("file", "test.go")
		c.AddArtifact("lines", 100)

		if len(c.Artifacts) != 2 {
			t.Errorf("expected 2 artifacts, got %d", len(c.Artifacts))
		}
		if c.Artifacts["file"] != "test.go" {
			t.Errorf("expected 'test.go', got '%v'", c.Artifacts["file"])
		}
	})

	t.Run("IsComplete returns true when confidence >= 0.8", func(t *testing.T) {
		c := NewConclusion()
		c.Confidence = 0.7
		if c.IsComplete() {
			t.Error("expected incomplete for 0.7 confidence")
		}

		c.Confidence = 0.8
		if !c.IsComplete() {
			t.Error("expected complete for 0.8 confidence")
		}

		c.Confidence = 0.9
		if !c.IsComplete() {
			t.Error("expected complete for 0.9 confidence")
		}
	})
}

// TestHumanDecision tests HumanDecision methods
func TestHumanDecision(t *testing.T) {
	t.Run("NewHumanDecision creates decision", func(t *testing.T) {
		d := NewHumanDecision(DecisionActionContinue)
		if d.Action != DecisionActionContinue {
			t.Errorf("expected Continue, got %v", d.Action)
		}
	})

	t.Run("ShouldContinue returns true for Continue", func(t *testing.T) {
		d := NewHumanDecision(DecisionActionContinue)
		if !d.ShouldContinue() {
			t.Error("expected ShouldContinue to return true")
		}
	})

	t.Run("ShouldContinue returns true for Redirect", func(t *testing.T) {
		d := NewHumanDecision(DecisionActionRedirect)
		if !d.ShouldContinue() {
			t.Error("expected ShouldContinue to return true for Redirect")
		}
	})

	t.Run("ShouldContinue returns false for Terminate", func(t *testing.T) {
		d := NewHumanDecision(DecisionActionTerminate)
		if d.ShouldContinue() {
			t.Error("expected ShouldContinue to return false for Terminate")
		}
	})

	t.Run("ShouldTerminate returns true only for Terminate", func(t *testing.T) {
		tests := []struct {
			action    DecisionAction
			terminate bool
		}{
			{DecisionActionContinue, false},
			{DecisionActionRedirect, false},
			{DecisionActionTerminate, true},
		}

		for _, tt := range tests {
			d := NewHumanDecision(tt.action)
			if d.ShouldTerminate() != tt.terminate {
				t.Errorf("action %v: expected terminate=%v", tt.action, tt.terminate)
			}
		}
	})

	t.Run("IsRedirect returns true only for Redirect", func(t *testing.T) {
		tests := []struct {
			action   DecisionAction
			redirect bool
		}{
			{DecisionActionContinue, false},
			{DecisionActionRedirect, true},
			{DecisionActionTerminate, false},
		}

		for _, tt := range tests {
			d := NewHumanDecision(tt.action)
			if d.IsRedirect() != tt.redirect {
				t.Errorf("action %v: expected redirect=%v", tt.action, tt.redirect)
			}
		}
	})
}

// TestDecisionActionString tests DecisionAction String method
func TestDecisionActionString(t *testing.T) {
	tests := []struct {
		action DecisionAction
		want   string
	}{
		{DecisionActionContinue, "CONTINUE"},
		{DecisionActionTerminate, "TERMINATE"},
		{DecisionActionRedirect, "REDIRECT"},
		{DecisionAction(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.action.String(); got != tt.want {
			t.Errorf("action %d: expected '%s', got '%s'", tt.action, tt.want, got)
		}
	}
}

// TestDualActConfig tests DualActConfig validation
func TestDualActConfig(t *testing.T) {
	responses := []*model.ChatResponse{
		model.NewChatResponse([]message.ContentBlock{message.Text("OK")}),
	}

	t.Run("Validate requires Human agent", func(t *testing.T) {
		config := &DualActConfig{
			Human:      nil,
			Reactive:   NewReActAgent(&ReActAgentConfig{Name: "r", SystemPrompt: "T", Model: newMockModel(responses, false), Memory: memory.NewHistory(10)}),
			MaxHRLoops: 3,
		}
		err := config.Validate()
		if err == nil {
			t.Error("expected error for nil Human agent")
		}
	})

	t.Run("Validate requires Reactive agent", func(t *testing.T) {
		config := &DualActConfig{
			Human:      NewReActAgent(&ReActAgentConfig{Name: "h", SystemPrompt: "T", Model: newMockModel(responses, false), Memory: memory.NewHistory(10)}),
			Reactive:   nil,
			MaxHRLoops: 3,
		}
		err := config.Validate()
		if err == nil {
			t.Error("expected error for nil Reactive agent")
		}
	})

	t.Run("Validate requires positive MaxHRLoops", func(t *testing.T) {
		h, r := createTestAgents(responses)
		config := &DualActConfig{
			Human:      h,
			Reactive:   r,
			MaxHRLoops: 0,
		}
		err := config.Validate()
		if err == nil {
			t.Error("expected error for zero MaxHRLoops")
		}
	})

	t.Run("Validate passes with valid config", func(t *testing.T) {
		h, r := createTestAgents(responses)
		config := &DualActConfig{
			Human:      h,
			Reactive:   r,
			MaxHRLoops: 3,
		}
		err := config.Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("DefaultDualActConfig creates valid config", func(t *testing.T) {
		h, r := createTestAgents(responses)
		config := DefaultDualActConfig(h, r)
		if config.MaxHRLoops != 3 {
			t.Errorf("expected default MaxHRLoops=3, got %d", config.MaxHRLoops)
		}
		if err := config.Validate(); err != nil {
			t.Errorf("default config failed validation: %v", err)
		}
	})

	t.Run("ApplyOptions applies options correctly", func(t *testing.T) {
		h, r := createTestAgents(responses)
		config := DefaultDualActConfig(h, r)
		config.ApplyOptions([]DualActOption{
			WithMaxHRLoops(10),
			WithVerboseLogging(),
		})

		if config.MaxHRLoops != 10 {
			t.Errorf("expected MaxHRLoops=10, got %d", config.MaxHRLoops)
		}
		if !config.EnableVerboseLogging {
			t.Error("expected EnableVerboseLogging=true")
		}
	})
}

// TestDualActAgent tests DualActAgent creation and basic operations
func TestDualActAgent(t *testing.T) {
	responses := []*model.ChatResponse{
		model.NewChatResponse([]message.ContentBlock{message.Text("OK")}),
	}

	t.Run("NewDualActAgent creates agent", func(t *testing.T) {
		h, r := createTestAgents(responses)
		config := &DualActConfig{
			Human:      h,
			Reactive:   r,
			MaxHRLoops: 3,
		}

		da := NewDualActAgent(config)
		if da == nil {
			t.Fatal("NewDualActAgent returned nil")
		}
		if da.Name() != "dualact" {
			t.Errorf("expected name 'dualact', got '%s'", da.Name())
		}
	})

	t.Run("NewDualActAgent panics with invalid config", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for invalid config")
			}
		}()

		config := &DualActConfig{
			Human:      nil,
			Reactive:   nil,
			MaxHRLoops: 3,
		}
		NewDualActAgent(config)
	})

	t.Run("NewDualActAgentWithOptions creates agent", func(t *testing.T) {
		h, r := createTestAgents(responses)

		da := NewDualActAgentWithOptions(h, r,
			WithMaxHRLoops(5),
			WithVerboseLogging(),
		)

		if da == nil {
			t.Fatal("NewDualActAgentWithOptions returned nil")
		}
		if da.GetConfig().MaxHRLoops != 5 {
			t.Errorf("expected MaxHRLoops=5, got %d", da.GetConfig().MaxHRLoops)
		}
	})

	t.Run("GetHumanAgent returns human agent", func(t *testing.T) {
		h, r := createTestAgents(responses)
		da := NewDualActAgentWithOptions(h, r)

		if da.GetHumanAgent() != h {
			t.Error("GetHumanAgent returned different agent")
		}
	})

	t.Run("GetReactiveAgent returns reactive agent", func(t *testing.T) {
		h, r := createTestAgents(responses)
		da := NewDualActAgentWithOptions(h, r)

		if da.GetReactiveAgent() != r {
			t.Error("GetReactiveAgent returned different agent")
		}
	})

	t.Run("GetConfig returns config", func(t *testing.T) {
		h, r := createTestAgents(responses)
		da := NewDualActAgentWithOptions(h, r, WithMaxHRLoops(7))

		config := da.GetConfig()
		if config.MaxHRLoops != 7 {
			t.Errorf("expected MaxHRLoops=7, got %d", config.MaxHRLoops)
		}
	})
}

// TestExtractConclusion tests conclusion extraction from responses
func TestExtractConclusion(t *testing.T) {
	responses := []*model.ChatResponse{
		model.NewChatResponse([]message.ContentBlock{message.Text("OK")}),
	}
	h, r := createTestAgents(responses)
	da := NewDualActAgentWithOptions(h, r)

	t.Run("ExtractConclusion from completion text", func(t *testing.T) {
		msg := message.NewMsg("test", "Task is done and complete", types.RoleAssistant)
		conclusion := da.extractConclusion(msg)

		if conclusion.Confidence < 0.8 {
			t.Errorf("expected high confidence for 'done', got %f", conclusion.Confidence)
		}
		if conclusion.Summary != "Task is done and complete" {
			t.Errorf("unexpected summary: %s", conclusion.Summary)
		}
	})

	t.Run("ExtractConclusion from error text", func(t *testing.T) {
		msg := message.NewMsg("test", "Execution failed with error", types.RoleAssistant)
		conclusion := da.extractConclusion(msg)

		if conclusion.Confidence > 0.5 {
			t.Errorf("expected low confidence for 'failed', got %f", conclusion.Confidence)
		}
	})

	t.Run("ExtractConclusion extracts steps from list", func(t *testing.T) {
		msg := message.NewMsg("test", "- Step one\n- Step two\n- Step three", types.RoleAssistant)
		conclusion := da.extractConclusion(msg)

		if len(conclusion.Steps) != 3 {
			t.Errorf("expected 3 steps, got %d", len(conclusion.Steps))
		}
	})
}

// TestParseDecision tests decision parsing from text
func TestParseDecision(t *testing.T) {
	responses := []*model.ChatResponse{
		model.NewChatResponse([]message.ContentBlock{message.Text("OK")}),
	}
	h, r := createTestAgents(responses)
	da := NewDualActAgentWithOptions(h, r)

	t.Run("Parse TERMINATE decision", func(t *testing.T) {
		text := "The task is complete and done."
		decision := da.parseDecision(text)

		if decision.Action != DecisionActionTerminate {
			t.Errorf("expected Terminate, got %v", decision.Action)
		}
	})

	t.Run("Parse CONTINUE decision", func(t *testing.T) {
		text := "Need to continue with next steps."
		decision := da.parseDecision(text)

		if decision.Action != DecisionActionContinue {
			t.Errorf("expected Continue, got %v", decision.Action)
		}
	})

	t.Run("Parse REDIRECT decision", func(t *testing.T) {
		text := "We should redirect and change approach."
		decision := da.parseDecision(text)

		if decision.Action != DecisionActionRedirect {
			t.Errorf("expected Redirect, got %v", decision.Action)
		}
	})
}

// TestObserve tests Observe method
func TestObserve(t *testing.T) {
	responses := []*model.ChatResponse{
		model.NewChatResponse([]message.ContentBlock{message.Text("OK")}),
	}
	h, r := createTestAgents(responses)
	da := NewDualActAgentWithOptions(h, r)

	msg := message.NewMsg("test", "Observation test", types.RoleUser)
	err := da.Observe(context.Background(), msg)
	if err != nil {
		t.Errorf("Observe failed: %v", err)
	}
}
