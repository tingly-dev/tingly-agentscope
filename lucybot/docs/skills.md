# LucyBot Skills System

## Overview

LucyBot includes a skills system that allows you to define reusable capabilities that can be invoked via slash commands.

## Bundled Skills

LucyBot comes with three pre-installed skills:

### Code Analysis (`/code-analysis`)
Systematic code analysis for understanding patterns, relationships, and behaviors.
- Use when asking about architecture, patterns, dependencies
- Provides structured analysis with file locations and evidence

### Specification Generation (`/specification-generation`)
Generates formal specifications from code.
- Use when documenting code or extracting formal descriptions
- Creates detailed specs with parameters, preconditions, postconditions

### Verification (`/verification`)
Verifies code correctness and checks properties.
- Use when checking if code meets requirements
- Provides verification reports with findings and recommendations

## Installing Skills

Skills are installed automatically when you run `lucybot init-config` and choose "Yes" to install skills.

### Manual Installation

To install skills manually:

1. Copy skill directories to `~/.lucybot/skills/` (global) or `.lucybot/skills/` (project-specific)
2. Each skill directory must contain a `SKILL.md` file with YAML frontmatter

## Using Skills

Once installed, skills can be invoked via slash commands:

```
You: /code-analysis
How does the compression system work?

LucyBot: [Analyzes code and provides detailed analysis]
```

## Creating Custom Skills

Create a new skill by:

1. Creating a directory in `~/.lucybot/skills/`
2. Adding a `SKILL.md` file with the following format:

```markdown
---
name: your-skill-name
description: When to use this skill
version: 1.0.0
---

# Skill Name

## Overview
Description of what this skill does

## When to Use
When the user asks for X, Y, or Z

## Workflow
Step-by-step instructions
```

3. The skill will be automatically discovered and available as `/your-skill-name`

## Skill Configuration

Skills are configured in the `[skills]` section of `config.toml`:

```toml
[skills]
enabled = true
paths = ["~/.lucybot/skills"]
auto_reload = false
```

- `enabled`: Enable or disable the skills system
- `paths`: Directories to search for skills
- `auto_reload`: Automatically reload skills when files change (future feature)
