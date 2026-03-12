Generate a tasks.json file for the following feature:

Project: {{.Project}}
Feature: {{.Feature}}

Requirements:
1. Break down the feature into 3-7 small, manageable user stories
2. Each story must be completable in one iteration (one context window)
3. Order stories by dependency: database → backend → UI
4. Each acceptance criterion must be verifiable (not vague)
5. Always include "Typecheck passes" in acceptance criteria
6. For UI stories, include "Verify in browser" criterion

Output ONLY valid JSON in this exact format:
{
  "project": "{{.Project}}",
  "branchName": "feature/[kebab-case-feature-name]",
  "description": "[one-line description]",
  "userStories": [
    {
      "id": "US-001",
      "title": "[short title]",
      "description": "As a [user], I want [feature] so that [benefit]",
      "acceptanceCriteria": [
        "[specific criterion]",
        "Typecheck passes"
      ],
      "priority": 1,
      "passes": false,
      "notes": ""
    }
  ]
}

Do not include any text before or after the JSON.
