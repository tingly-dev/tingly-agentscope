# LucyBot Skills

This directory contains bundled skills that are distributed with LucyBot.

## Installing Skills

To install these skills to your global configuration:

```bash
lucybot init-config
```

When prompted, choose "Yes" to install skills. They will be copied to:
- `~/.lucybot/skills/` (global)
- `.lucybot/skills/` (local with --local flag)

## Skill Structure

Each skill is a directory containing:
- `SKILL.md` - Main skill definition (YAML frontmatter + content)
- Additional files (optional) - Workflows, examples, templates

## Creating Custom Skills

You can create custom skills by:
1. Creating a new directory in `~/.lucybot/skills/`
2. Adding a `SKILL.md` file with YAML frontmatter
3. Defining the skill content

See existing skills for examples.
