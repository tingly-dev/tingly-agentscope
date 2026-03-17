package tools

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// FenceDiff represents a fence diff format edit
type FenceDiff struct {
	FilePath  string
	Search    string
	Replace   string
	LineStart int
	LineEnd   int
}

// String returns the fence diff as a formatted string
func (f *FenceDiff) String() string {
	lineInfo := ""
	if f.LineStart > 0 && f.LineEnd > 0 {
		lineInfo = fmt.Sprintf(" lines: %d-%d", f.LineStart, f.LineEnd)
	} else if f.LineStart > 0 {
		lineInfo = fmt.Sprintf(" lines: %d-", f.LineStart)
	} else if f.LineEnd > 0 {
		lineInfo = fmt.Sprintf(" lines: -%d", f.LineEnd)
	}
	return fmt.Sprintf("%s%s\n<<<<<<< SEARCH\n%s\n=======\n%s\n>>>>>>> REPLACE",
		f.FilePath, lineInfo, f.Search, f.Replace)
}

// ParseFenceDiff parses a fence diff format string
func ParseFenceDiff(input string) (*FenceDiff, error) {
	lines := strings.Split(input, "\n")
	if len(lines) < 4 {
		return nil, errors.New("fence diff too short")
	}

	// Parse first line for filepath and optional line range
	firstLine := strings.TrimSpace(lines[0])
	filePath, lineStart, lineEnd, err := parseFileLine(firstLine)
	if err != nil {
		return nil, err
	}

	// Find markers
	searchStart, separator, replaceEnd := -1, -1, -1
	for i, line := range lines {
		if searchStart == -1 && strings.Contains(line, "<<<<<<< SEARCH") {
			searchStart = i
		} else if separator == -1 && strings.TrimSpace(line) == "=======" {
			separator = i
		} else if replaceEnd == -1 && strings.Contains(line, ">>>>>>> REPLACE") {
			replaceEnd = i
			break
		}
	}

	if searchStart == -1 {
		return nil, errors.New("missing search marker <<<<<<< SEARCH")
	}
	if separator == -1 {
		return nil, errors.New("missing separator =======")
	}
	if replaceEnd == -1 {
		return nil, errors.New("missing replace marker >>>>>>> REPLACE")
	}

	// Validate marker order
	if !(searchStart < separator && separator < replaceEnd) {
		return nil, errors.New("invalid marker order: expected <<<<<<< SEARCH ... ======= ... >>>>>>> REPLACE")
	}

	search := strings.Join(lines[searchStart+1:separator], "\n")
	replace := strings.Join(lines[separator+1:replaceEnd], "\n")

	return &FenceDiff{
		FilePath:  filePath,
		Search:    strings.TrimSuffix(search, "\n"),
		Replace:   strings.TrimSuffix(replace, "\n"),
		LineStart: lineStart,
		LineEnd:   lineEnd,
	}, nil
}

// parseFileLine parses the first line to extract filepath and optional line range
// Supports formats:
//   - filepath
//   - filepath lines: start-end
//   - filepath lines: start-
//   - filepath lines: -end
func parseFileLine(line string) (string, int, int, error) {
	// Check for "lines:" suffix
	re := regexp.MustCompile(`^(.+?)\s+lines:\s*(\d*)-(\d*)$`)
	matches := re.FindStringSubmatch(line)
	if matches != nil {
		filePath := strings.TrimSpace(matches[1])
		var lineStart, lineEnd int
		var err error

		if matches[2] != "" {
			lineStart, err = strconv.Atoi(matches[2])
			if err != nil {
				return "", 0, 0, fmt.Errorf("invalid start line: %w", err)
			}
		}

		if matches[3] != "" {
			lineEnd, err = strconv.Atoi(matches[3])
			if err != nil {
				return "", 0, 0, fmt.Errorf("invalid end line: %w", err)
			}
		}

		return filePath, lineStart, lineEnd, nil
	}

	// No line range specified, just return the filepath
	return strings.TrimSpace(line), 0, 0, nil
}

// Validate validates the fence diff
func (f *FenceDiff) Validate() error {
	if f.FilePath == "" {
		return errors.New("file path is required")
	}

	// Line range validation
	if f.LineStart > 0 && f.LineEnd > 0 && f.LineStart > f.LineEnd {
		return fmt.Errorf("invalid line range: start (%d) > end (%d)", f.LineStart, f.LineEnd)
	}

	return nil
}

// Apply applies the fence diff to the given content
func (f *FenceDiff) Apply(content string) (string, error) {
	if err := f.Validate(); err != nil {
		return "", err
	}

	// If line range is specified, extract that portion
	if f.LineStart > 0 || f.LineEnd > 0 {
		return applyWithLineRange(content, f)
	}

	// Simple string replacement
	if !strings.Contains(content, f.Search) {
		return "", fmt.Errorf("search content not found in file")
	}

	return strings.Replace(content, f.Search, f.Replace, 1), nil
}

// applyWithLineRange applies the diff to a specific line range
func applyWithLineRange(content string, f *FenceDiff) (string, error) {
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// Adjust line numbers (1-based to 0-based)
	start := f.LineStart - 1
	if start < 0 {
		start = 0
	}

	end := f.LineEnd
	if end <= 0 || end > totalLines {
		end = totalLines
	}

	// Validate range
	if start >= totalLines {
		return "", fmt.Errorf("start line %d exceeds file length %d", f.LineStart, totalLines)
	}

	// Extract the section to be modified
	section := strings.Join(lines[start:end], "\n")

	// Verify search content matches
	if !strings.Contains(section, f.Search) {
		return "", fmt.Errorf("search content not found in specified line range")
	}

	// Apply replacement within the section
	newSection := strings.Replace(section, f.Search, f.Replace, 1)

	// Reconstruct the file
	var result []string
	result = append(result, lines[:start]...)
	result = append(result, strings.Split(newSection, "\n")...)
	result = append(result, lines[end:]...)

	return strings.Join(result, "\n"), nil
}

// ParseMultipleFenceDiffs parses multiple fence diffs from a single input
func ParseMultipleFenceDiffs(input string) ([]*FenceDiff, error) {
	var diffs []*FenceDiff
	var currentDiff strings.Builder
	inDiff := false

	lines := strings.Split(input, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect start of a diff (filepath line)
		if !inDiff && trimmed != "" && !strings.HasPrefix(trimmed, "<<<<<<<") &&
			!strings.HasPrefix(trimmed, "=======") && !strings.HasPrefix(trimmed, ">>>>>>>") {
			// Check if next line starts with <<<<<<<
			if i+1 < len(lines) && strings.Contains(lines[i+1], "<<<<<<< SEARCH") {
				inDiff = true
				currentDiff.WriteString(line)
				currentDiff.WriteString("\n")
			}
			continue
		}

		if inDiff {
			currentDiff.WriteString(line)
			currentDiff.WriteString("\n")

			// Check for end of diff
			if strings.Contains(line, ">>>>>>> REPLACE") {
				diff, err := ParseFenceDiff(strings.TrimSuffix(currentDiff.String(), "\n"))
				if err != nil {
					return nil, fmt.Errorf("failed to parse diff: %w", err)
				}
				diffs = append(diffs, diff)
				currentDiff.Reset()
				inDiff = false
			}
		}
	}

	return diffs, nil
}
