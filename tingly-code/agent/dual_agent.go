package agent

import (
	"fmt"

	"example/tingly-code/config"

	"github.com/tingly-dev/tingly-agentscope/pkg/agent"
	"github.com/tingly-dev/tingly-agentscope/pkg/formatter"
)

// CreateDualTinglyAgent creates a DualActAgent for tingly-code
// H (Human): Planner that evaluates progress and makes decisions
// R (Reactive): TinglyAgent that executes with tools
func CreateDualTinglyAgent(cfg *config.Config, workDir string) (*agent.DualActAgent, error) {
	// Use dual config if enabled and provided
	dualCfg := cfg.Dual

	// Determine reactive agent config (use main config or dual's reactive config)
	reactiveCfg := &cfg.Agent
	if dualCfg.Enabled && dualCfg.Human != nil {
		// In dual mode with separate human config, use main agent as reactive
		reactiveCfg = &cfg.Agent
	}

	// Create Reactive Agent (R) - The Executor (using existing TinglyAgent)
	reactiveAgent, err := CreateTinglyAgent(reactiveCfg, &cfg.Tools, &cfg.TaskInjector, workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create reactive agent: %w", err)
	}

	// Determine human agent config
	humanCfg := &cfg.Agent
	if dualCfg.Enabled && dualCfg.Human != nil {
		humanCfg = dualCfg.Human
	}

	// Create Human Agent (H) - The Planner
	humanSystemPrompt := `You are a technical project planner reviewing code development work.

Your responsibilities:
1. Review what has been accomplished
2. Check if the implementation correctly addresses the user's requirements
3. Identify any issues, missing features, or quality concerns
4. Decide whether to:
   - TERMINATE: Task is complete and working correctly
   - CONTINUE: More work needed (provide specific next steps)
   - REDIRECT: Approach is wrong (explain new direction)

Be thorough - don't terminate until you're satisfied with the quality!

When responding, be concise and clearly indicate your decision with this format:
**Decision:** TERMINATE/CONTINUE/REDIRECT

**Reasoning:**
Your detailed reasoning here.

Always respond in English.`

	// Create human agent as a ReActAgent with planning focus
	factory := NewModelFactory()
	chatModel, err := factory.CreateModel(&humanCfg.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to create model for human agent: %w", err)
	}

	memory := agent.NewSimpleMemory(50)

	humanAgent := agent.NewReActAgent(&agent.ReActAgentConfig{
		Name:          "planner",
		SystemPrompt:  humanSystemPrompt,
		Model:         chatModel,
		Memory:        memory,
		MaxIterations: 3,
		Temperature:   &humanCfg.Model.Temperature,
		MaxTokens:     &humanCfg.Model.MaxTokens,
	})
	humanAgent.SetFormatter(formatter.NewTeaFormatter())

	// Determine dual act options
	var maxHRLoops int
	var verboseLogging bool

	if dualCfg.Enabled {
		maxHRLoops = dualCfg.MaxHRLoops
		if maxHRLoops <= 0 {
			maxHRLoops = 5 // default
		}
		verboseLogging = dualCfg.VerboseLogging
	} else {
		maxHRLoops = 5
		verboseLogging = false
	}

	// Create DualActAgent
	dualAct := agent.NewDualActAgentWithOptions(
		humanAgent,
		reactiveAgent,
		agent.WithMaxHRLoops(maxHRLoops),
	)
	if verboseLogging {
		agent.WithVerboseLogging()(dualAct.GetConfig())
	}
	dualAct.SetFormatter(formatter.NewTeaFormatter())

	return dualAct, nil
}

// IsDualModeEnabled checks if dual mode is enabled in the config
func IsDualModeEnabled(cfg *config.Config) bool {
	return cfg.Dual.Enabled
}
