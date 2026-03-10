// Package plan provides task planning and decomposition functionality
package plan

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/tingly-dev/tingly-agentscope/pkg/types"
)

// Constants for markdown formatting
const (
	mdCodeBlock = "```"
)

// SubTaskState represents the state of a subtask
type SubTaskState string

const (
	SubTaskStateTodo       SubTaskState = "todo"
	SubTaskStateInProgress SubTaskState = "in_progress"
	SubTaskStateDone       SubTaskState = "done"
	SubTaskStateAbandoned  SubTaskState = "abandoned"
)

// SubTask represents a single subtask in a plan
type SubTask struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	ExpectedOutcome string         `json:"expected_outcome"`
	Outcome         string         `json:"outcome,omitempty"`
	State           SubTaskState   `json:"state"`
	CreatedAt       string         `json:"created_at"`
	FinishedAt      string         `json:"finished_at,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// NewSubTask creates a new subtask
func NewSubTask(name, description, expectedOutcome string) *SubTask {
	return &SubTask{
		ID:              types.GenerateID(),
		Name:            name,
		Description:     description,
		ExpectedOutcome: expectedOutcome,
		State:           SubTaskStateTodo,
		CreatedAt:       types.Timestamp(),
		Metadata:        make(map[string]any),
	}
}

// Finish marks the subtask as done with an outcome
func (st *SubTask) Finish(outcome string) {
	st.State = SubTaskStateDone
	st.Outcome = outcome
	st.FinishedAt = types.Timestamp()
}

// Abandon marks the subtask as abandoned
func (st *SubTask) Abandon() {
	st.State = SubTaskStateAbandoned
	st.FinishedAt = types.Timestamp()
}

// SetInProgress marks the subtask as in progress
func (st *SubTask) SetInProgress() {
	st.State = SubTaskStateInProgress
}

// ToMarkdown converts the subtask to markdown format
func (st *SubTask) ToMarkdown(detailed bool) string {
	statusMap := map[SubTaskState]string{
		SubTaskStateTodo:       "- [ ] ",
		SubTaskStateInProgress: "- [ ] [WIP]",
		SubTaskStateDone:       "- [x] ",
		SubTaskStateAbandoned:  "- [ ] [Abandoned]",
	}

	if detailed {
		var lines []string
		lines = append(lines, fmt.Sprintf("%s%s", statusMap[st.State], st.Name))
		lines = append(lines, fmt.Sprintf("\t- Created At: %s", st.CreatedAt))
		lines = append(lines, fmt.Sprintf("\t- Description: %s", st.Description))
		lines = append(lines, fmt.Sprintf("\t- Expected Outcome: %s", st.ExpectedOutcome))
		lines = append(lines, fmt.Sprintf("\t- State: %s", st.State))

		if st.State == SubTaskStateDone {
			lines = append(lines, fmt.Sprintf("\t- Finished At: %s", st.FinishedAt))
			lines = append(lines, fmt.Sprintf("\t- Actual Outcome: %s", st.Outcome))
		}

		return strings.Join(lines, "\n")
	}

	return fmt.Sprintf("%s%s", statusMap[st.State], st.Name)
}

// PlanState represents the state of a plan
type PlanState string

const (
	PlanStateTodo       PlanState = "todo"
	PlanStateInProgress PlanState = "in_progress"
	PlanStateDone       PlanState = "done"
	PlanStateAbandoned  PlanState = "abandoned"
)

// Plan represents a task decomposition plan
type Plan struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	ExpectedOutcome string         `json:"expected_outcome"`
	SubTasks        []*SubTask     `json:"subtasks"`
	CreatedAt       string         `json:"created_at"`
	State           PlanState      `json:"state"`
	FinishedAt      string         `json:"finished_at,omitempty"`
	Outcome         string         `json:"outcome,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// NewPlan creates a new plan
func NewPlan(name, description, expectedOutcome string) *Plan {
	return &Plan{
		ID:              types.GenerateID(),
		Name:            name,
		Description:     description,
		ExpectedOutcome: expectedOutcome,
		SubTasks:        make([]*SubTask, 0),
		State:           PlanStateTodo,
		CreatedAt:       types.Timestamp(),
		Metadata:        make(map[string]any),
	}
}

// AddSubTask adds a new subtask to the plan
func (p *Plan) AddSubTask(subtask *SubTask) {
	p.SubTasks = append(p.SubTasks, subtask)
}

// RefreshState refreshes the plan state based on subtask states
func (p *Plan) RefreshState() string {
	if p.State == PlanStateDone || p.State == PlanStateAbandoned {
		return ""
	}

	anyInProgress := false
	for _, st := range p.SubTasks {
		if st.State == SubTaskStateInProgress {
			anyInProgress = true
			break
		}
	}

	if anyInProgress && p.State == PlanStateTodo {
		p.State = PlanStateInProgress
		return "Plan state updated to 'in_progress'"
	}

	if !anyInProgress && p.State == PlanStateInProgress {
		p.State = PlanStateTodo
		return "Plan state updated to 'todo'"
	}

	return ""
}

// Finish marks the plan as finished
func (p *Plan) Finish(state PlanState, outcome string) {
	p.State = state
	p.Outcome = outcome
	p.FinishedAt = types.Timestamp()
}

// ToMarkdown converts the plan to markdown format
func (p *Plan) ToMarkdown(detailed bool) string {
	var subtaskMarkdowns []string
	for _, st := range p.SubTasks {
		subtaskMarkdowns = append(subtaskMarkdowns, st.ToMarkdown(detailed))
	}

	lines := []string{
		fmt.Sprintf("# %s", p.Name),
		fmt.Sprintf("**Description**: %s", p.Description),
		fmt.Sprintf("**Expected Outcome**: %s", p.ExpectedOutcome),
		fmt.Sprintf("**State**: %s", p.State),
		fmt.Sprintf("**Created At**: %s", p.CreatedAt),
		"## Subtasks",
		strings.Join(subtaskMarkdowns, "\n"),
	}

	return strings.Join(lines, "\n")
}

// GetSubTaskIndex returns the index of a subtask by ID
func (p *Plan) GetSubTaskIndex(id string) (int, error) {
	for i, st := range p.SubTasks {
		if st.ID == id {
			return i, nil
		}
	}
	return -1, fmt.Errorf("subtask with ID '%s' not found", id)
}

// GetInProgressSubTaskIndex returns the index of the in-progress subtask
func (p *Plan) GetInProgressSubTaskIndex() (int, error) {
	for i, st := range p.SubTasks {
		if st.State == SubTaskStateInProgress {
			return i, nil
		}
	}
	return -1, fmt.Errorf("no subtask in progress")
}

// PlanChangeHook is a function called when a plan changes
type PlanChangeHook func(ctx context.Context, plan *Plan)

// PlanNotebook manages task plans with persistence
type PlanNotebook struct {
	currentPlan *Plan
	storage     PlanStorage
	hooks       map[string]PlanChangeHook
	mu          sync.RWMutex
}

// PlanStorage defines the interface for plan persistence
type PlanStorage interface {
	Save(ctx context.Context, plan *Plan) error
	Load(ctx context.Context, id string) (*Plan, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*Plan, error)
}

// InMemoryPlanStorage provides in-memory plan storage
type InMemoryPlanStorage struct {
	plans map[string]*Plan
	mu    sync.RWMutex
}

// NewInMemoryPlanStorage creates a new in-memory plan storage
func NewInMemoryPlanStorage() *InMemoryPlanStorage {
	return &InMemoryPlanStorage{
		plans: make(map[string]*Plan),
	}
}

// Save saves a plan
func (s *InMemoryPlanStorage) Save(ctx context.Context, plan *Plan) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.plans[plan.ID] = plan
	return nil
}

// Load loads a plan by ID
func (s *InMemoryPlanStorage) Load(ctx context.Context, id string) (*Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plan, ok := s.plans[id]
	if !ok {
		return nil, fmt.Errorf("plan with ID '%s' not found", id)
	}

	return plan, nil
}

// Delete deletes a plan
func (s *InMemoryPlanStorage) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.plans, id)
	return nil
}

// List lists all plans
func (s *InMemoryPlanStorage) List(ctx context.Context) ([]*Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plans := make([]*Plan, 0, len(s.plans))
	for _, plan := range s.plans {
		plans = append(plans, plan)
	}

	return plans, nil
}

// NewPlanNotebook creates a new plan notebook
func NewPlanNotebook(storage PlanStorage) *PlanNotebook {
	if storage == nil {
		storage = NewInMemoryPlanStorage()
	}

	return &PlanNotebook{
		storage: storage,
		hooks:   make(map[string]PlanChangeHook),
	}
}

// CreatePlan creates a new plan and sets it as current
func (pn *PlanNotebook) CreatePlan(ctx context.Context, name, description, expectedOutcome string, subtasks []*SubTask) (*Plan, error) {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	plan := NewPlan(name, description, expectedOutcome)
	plan.SubTasks = subtasks

	pn.currentPlan = plan

	if err := pn.storage.Save(ctx, plan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	pn.notifyHooks(ctx, plan)

	return plan, nil
}

// GetCurrentPlan returns the current plan
func (pn *PlanNotebook) GetCurrentPlan() *Plan {
	pn.mu.RLock()
	defer pn.mu.RUnlock()

	return pn.currentPlan
}

// SetCurrentPlan sets the current plan
func (pn *PlanNotebook) SetCurrentPlan(ctx context.Context, plan *Plan) error {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	pn.currentPlan = plan

	if err := pn.storage.Save(ctx, plan); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	pn.notifyHooks(ctx, plan)

	return nil
}

// UpdateSubtaskState updates the state of a subtask
func (pn *PlanNotebook) UpdateSubtaskState(ctx context.Context, subtaskID string, state SubTaskState) error {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	if pn.currentPlan == nil {
		return fmt.Errorf("no current plan")
	}

	// Find subtask
	idx, err := pn.currentPlan.GetSubTaskIndex(subtaskID)
	if err != nil {
		return err
	}

	// Update state
	switch state {
	case SubTaskStateDone:
		pn.currentPlan.SubTasks[idx].Finish("")
	case SubTaskStateAbandoned:
		pn.currentPlan.SubTasks[idx].Abandon()
	case SubTaskStateInProgress:
		pn.currentPlan.SubTasks[idx].SetInProgress()
	default:
		pn.currentPlan.SubTasks[idx].State = state
	}

	// Refresh plan state
	pn.currentPlan.RefreshState()

	if err := pn.storage.Save(ctx, pn.currentPlan); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	pn.notifyHooks(ctx, pn.currentPlan)

	return nil
}

// FinishSubtask finishes a subtask with an outcome
func (pn *PlanNotebook) FinishSubtask(ctx context.Context, subtaskID, outcome string) error {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	if pn.currentPlan == nil {
		return fmt.Errorf("no current plan")
	}

	// Find subtask
	idx, err := pn.currentPlan.GetSubTaskIndex(subtaskID)
	if err != nil {
		return err
	}

	// Finish subtask
	pn.currentPlan.SubTasks[idx].Finish(outcome)

	// Refresh plan state
	pn.currentPlan.RefreshState()

	if err := pn.storage.Save(ctx, pn.currentPlan); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	pn.notifyHooks(ctx, pn.currentPlan)

	return nil
}

// RevisePlan revises the current plan with new subtasks
func (pn *PlanNotebook) RevisePlan(ctx context.Context, name, description, expectedOutcome string, subtasks []*SubTask) error {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	if pn.currentPlan == nil {
		return fmt.Errorf("no current plan")
	}

	// Update plan
	if name != "" {
		pn.currentPlan.Name = name
	}
	if description != "" {
		pn.currentPlan.Description = description
	}
	if expectedOutcome != "" {
		pn.currentPlan.ExpectedOutcome = expectedOutcome
	}
	if len(subtasks) > 0 {
		pn.currentPlan.SubTasks = subtasks
	}

	if err := pn.storage.Save(ctx, pn.currentPlan); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	pn.notifyHooks(ctx, pn.currentPlan)

	return nil
}

// FinishPlan finishes the current plan
func (pn *PlanNotebook) FinishPlan(ctx context.Context, state PlanState, outcome string) error {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	if pn.currentPlan == nil {
		return fmt.Errorf("no current plan")
	}

	pn.currentPlan.Finish(state, outcome)

	if err := pn.storage.Save(ctx, pn.currentPlan); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	pn.notifyHooks(ctx, pn.currentPlan)

	return nil
}

// GenerateHint generates a hint message for the agent based on current plan state
func (pn *PlanNotebook) GenerateHint() string {
	pn.mu.RLock()
	defer pn.mu.RUnlock()

	if pn.currentPlan == nil {
		return `<system-hint>If the user's query is complex (e.g. programming a website, game or app), or requires a long chain of steps to complete (e.g. conduct research on a certain topic from different sources), you NEED to create a plan first by creating and following a plan. Otherwise, you can directly execute the user's query without planning.</system-hint>`
	}

	// Count subtasks by state
	nTodo, nInProgress, nDone, nAbandoned := 0, 0, 0, 0
	inProgressIdx := -1

	for i, st := range pn.currentPlan.SubTasks {
		switch st.State {
		case SubTaskStateTodo:
			nTodo++
		case SubTaskStateInProgress:
			nInProgress++
			inProgressIdx = i
		case SubTaskStateDone:
			nDone++
		case SubTaskStateAbandoned:
			nAbandoned++
		}
	}

	var hint string

	if nInProgress == 0 && nDone == 0 {
		// All subtasks are todo
		hint = fmt.Sprintf("<system-hint>The current plan:\n%s\n%s\n%s\nYour options include:\n- Mark the first subtask as 'in_progress' and start executing it.\n- If the first subtask is not executable, analyze why and revise the plan.\n- If the user asks you to do something unrelated to the plan, prioritize the user's query first.\n</system-hint>",
			mdCodeBlock, pn.currentPlan.ToMarkdown(false), mdCodeBlock)

	} else if nInProgress > 0 && inProgressIdx >= 0 {
		// One subtask is in progress
		st := pn.currentPlan.SubTasks[inProgressIdx]
		hint = fmt.Sprintf("<system-hint>The current plan:\n%s\n%s\n%s\nNow the subtask at index %d, named '%s', is 'in_progress'. Its details:\n%s\n%s\n%s\nYour options include:\n- Continue executing the subtask.\n- Call finish_subtask if the subtask is finished.\n- Ask the user for more information if needed.\n- Revise the plan if necessary.\n</system-hint>",
			mdCodeBlock, pn.currentPlan.ToMarkdown(false), mdCodeBlock,
			inProgressIdx, st.Name, mdCodeBlock, st.ToMarkdown(true), mdCodeBlock)

	} else if nInProgress == 0 && nDone > 0 && (nDone+nAbandoned) < len(pn.currentPlan.SubTasks) {
		// No subtask in progress, some done
		hint = fmt.Sprintf("<system-hint>The current plan:\n%s\n%s\n%s\nThe first %d subtasks are done, and no subtask is 'in_progress'.\nYour options include:\n- Mark the next subtask as 'in_progress' and start executing it.\n- Ask the user for more information if needed.\n- Revise the plan if necessary.\n</system-hint>",
			mdCodeBlock, pn.currentPlan.ToMarkdown(false), mdCodeBlock, nDone)

	} else if (nDone + nAbandoned) == len(pn.currentPlan.SubTasks) {
		// All subtasks are done or abandoned
		hint = fmt.Sprintf("<system-hint>The current plan:\n%s\n%s\n%s\nAll subtasks are done. Your options:\n- Finish the plan with the outcome and summarize to the user.\n- Revise the plan if necessary.\n</system-hint>",
			mdCodeBlock, pn.currentPlan.ToMarkdown(false), mdCodeBlock)
	}

	return hint
}

// RegisterHook registers a plan change hook
func (pn *PlanNotebook) RegisterHook(name string, hook PlanChangeHook) {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	pn.hooks[name] = hook
}

// UnregisterHook unregisters a plan change hook
func (pn *PlanNotebook) UnregisterHook(name string) {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	delete(pn.hooks, name)
}

// notifyHooks notifies all registered hooks of a plan change
func (pn *PlanNotebook) notifyHooks(ctx context.Context, plan *Plan) {
	for _, hook := range pn.hooks {
		if hook != nil {
			go hook(ctx, plan)
		}
	}
}

// LoadPlan loads a plan from storage
func (pn *PlanNotebook) LoadPlan(ctx context.Context, id string) error {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	plan, err := pn.storage.Load(ctx, id)
	if err != nil {
		return err
	}

	pn.currentPlan = plan
	return nil
}

// ListPlans lists all plans
func (pn *PlanNotebook) ListPlans(ctx context.Context) ([]*Plan, error) {
	return pn.storage.List(ctx)
}

// DeletePlan deletes a plan
func (pn *PlanNotebook) DeletePlan(ctx context.Context, id string) error {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	if pn.currentPlan != nil && pn.currentPlan.ID == id {
		pn.currentPlan = nil
	}

	return pn.storage.Delete(ctx, id)
}

// StateDict returns the state for serialization
func (pn *PlanNotebook) StateDict() map[string]any {
	pn.mu.RLock()
	defer pn.mu.RUnlock()

	if pn.currentPlan == nil {
		return map[string]any{
			"has_current_plan": false,
		}
	}

	return map[string]any{
		"has_current_plan":  true,
		"current_plan_id":   pn.currentPlan.ID,
		"current_plan_name": pn.currentPlan.Name,
		"plan_state":        pn.currentPlan.State,
		"subtask_count":     len(pn.currentPlan.SubTasks),
	}
}
