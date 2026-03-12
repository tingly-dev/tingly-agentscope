Convert the spec document to tasks.json format.

Spec file: {{.SpecPath}}

Instructions:
1. Read the spec file at the path above
2. Extract the following from the spec:
   - Project name (from title or context)
   - Branch name (use format: feat/kebab-case-feature-name)
   - Description (one-line summary)
3. Convert the Tasks section in the spec to userStories array
4. Each story should have:
   - id: US-001, US-002, etc.
   - title: short descriptive title
   - description: full story description
   - acceptanceCriteria: specific, verifiable criteria
   - priority: execution order (1 = highest)
   - passes: false
   - notes: empty string

Requirements:
- Each story must be completable in one iteration
- Order stories by dependency
- Each acceptance criterion must be verifiable
- Always include "Typecheck passes" in acceptance criteria

Output ONLY valid JSON in this exact format:
{
  "project": "...",
  "branchName": "feat/...",
  "description": "...",
  "userStories": [
    {
      "id": "US-001",
      "title": "...",
      "description": "...",
      "acceptanceCriteria": ["...", "Typecheck passes"],
      "priority": 1,
      "passes": false,
      "notes": ""
    }
  ]
}

Do not include any text before or after the JSON.
