package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Popup represents a floating popup menu
type Popup struct {
	Title    string
	Items    []PopupItem
	Selected int
	Visible  bool
	width    int
	height   int
	allItems []PopupItem // Stores original unfiltered items
}

// PopupItem represents an item in the popup
type PopupItem struct {
	Title       string
	Description string
	Icon        string
	Value       string
}

// NewPopup creates a new popup
func NewPopup(title string, height int) *Popup {
	return &Popup{
		Title:    title,
		Items:    []PopupItem{},
		Selected: 0,
		Visible:  false,
		width:    40,
		height:   height,
	}
}

// SetItems updates the popup items
func (p *Popup) SetItems(items []PopupItem) {
	p.allItems = items
	p.Items = items
	if p.Selected >= len(p.Items) {
		p.Selected = 0
	}
}

// Show makes the popup visible
func (p *Popup) Show() {
	p.Visible = true
	p.Selected = 0
}

// Hide hides the popup
func (p *Popup) Hide() {
	p.Visible = false
}

// Toggle toggles popup visibility
func (p *Popup) Toggle() {
	p.Visible = !p.Visible
}

// Next selects the next item
func (p *Popup) Next() {
	if len(p.Items) == 0 {
		return
	}
	p.Selected = (p.Selected + 1) % len(p.Items)
}

// Prev selects the previous item
func (p *Popup) Prev() {
	if len(p.Items) == 0 {
		return
	}
	p.Selected = (p.Selected - 1 + len(p.Items)) % len(p.Items)
}

// GetSelected returns the currently selected item
func (p *Popup) GetSelected() (PopupItem, bool) {
	if len(p.Items) == 0 || p.Selected >= len(p.Items) {
		return PopupItem{}, false
	}
	return p.Items[p.Selected], true
}

// SetWidth updates the popup width
func (p *Popup) SetWidth(width int) {
	p.width = width
}

// View renders the popup
func (p *Popup) View() string {
	if !p.Visible || len(p.Items) == 0 {
		return ""
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#565f89")).
		Padding(0, 1).
		Width(p.width - 2)

	itemStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(p.width - 2)

	selectedStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Background(lipgloss.Color("#3d59a1")).
		Foreground(lipgloss.Color("#ffffff")).
		Width(p.width - 2)

	descriptionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#565f89")).
		Italic(true)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#565f89")).
		Background(lipgloss.Color("#1a1b26"))

	// Build content
	var content strings.Builder

	// Title
	content.WriteString(titleStyle.Render(p.Title))
	content.WriteString("\n")

	// Items
	for i, item := range p.Items {
		var line string
		icon := item.Icon
		if icon == "" {
			icon = "  "
		}

		titleText := truncate(item.Title, p.width-10)
		if item.Description != "" {
			descText := truncate(item.Description, p.width-15)
			line = fmt.Sprintf("%s %-20s %s", icon, titleText, descriptionStyle.Render(descText))
		} else {
			line = fmt.Sprintf("%s %s", icon, titleText)
		}

		if i == p.Selected {
			content.WriteString(selectedStyle.Render(line))
		} else {
			content.WriteString(itemStyle.Render(line))
		}

		if i < len(p.Items)-1 {
			content.WriteString("\n")
		}
	}

	return borderStyle.Render(content.String())
}

// Height returns the rendered height
func (p *Popup) Height() int {
	if !p.Visible {
		return 0
	}
	return len(p.Items) + 3 // items + title + borders
}

// Width returns the popup width
func (p *Popup) Width() int {
	return p.width
}

// CommandPopup creates a popup for slash commands
func CommandPopup() *Popup {
	return NewPopup("Commands", 8)
}

// SetCommandItems sets the items for command popup
func (p *Popup) SetCommandItems() {
	p.SetItems([]PopupItem{
		{Title: "/help", Description: "Show help", Icon: "❓", Value: "help"},
		{Title: "/clear", Description: "Clear screen", Icon: "🧹", Value: "clear"},
		{Title: "/tools", Description: "List tools", Icon: "🔧", Value: "tools"},
		{Title: "/model", Description: "Show model info", Icon: "🧠", Value: "model"},
		{Title: "/quit", Description: "Exit", Icon: "👋", Value: "quit"},
	})
}

// AgentPopup creates a popup for agent mentions
func AgentPopup() *Popup {
	return NewPopup("Agents", 6)
}

// Filter filters items based on prefix (matching against Title without the command/mention prefix)
func (p *Popup) Filter(prefix string) {
	if prefix == "" {
		// Restore all items when prefix is empty
		p.Items = p.allItems
		p.Selected = 0
		return
	}

	prefix = strings.ToLower(prefix)
	var filtered []PopupItem
	for _, item := range p.allItems {
		// Strip the prefix character (/ or @) from the title for matching
		title := item.Title
		if len(title) > 0 && (title[0] == '/' || title[0] == '@') {
			title = title[1:]
		}
		if strings.HasPrefix(strings.ToLower(title), prefix) {
			filtered = append(filtered, item)
		}
	}
	p.Items = filtered
	p.Selected = 0
}
