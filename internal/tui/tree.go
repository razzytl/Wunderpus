package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type TreeNode struct {
	ID       string
	Label    string
	Children []*TreeNode
	Expanded bool
	Color    lipgloss.Color
}

func NewTreeNode(id, label string) *TreeNode {
	return &TreeNode{
		ID:       id,
		Label:    label,
		Children: make([]*TreeNode, 0),
		Expanded: true,
		Color:    textColor,
	}
}

func (n *TreeNode) AddChild(child *TreeNode) {
	n.Children = append(n.Children, child)
}

func (n *TreeNode) Toggle() {
	n.Expanded = !n.Expanded
}

func (n *TreeNode) SetExpanded(expanded bool) {
	n.Expanded = expanded
}

type TreeModel struct {
	Root       *TreeNode
	SelectedID string
	Indent     int
}

func NewTree(root *TreeNode) TreeModel {
	return TreeModel{
		Root:       root,
		SelectedID: "",
		Indent:     2,
	}
}

func (t *TreeModel) View() string {
	var lines []string
	t.renderNode(t.Root, "", true, &lines)
	return fmt.Sprintf("%s", lines)
}

func (t *TreeModel) renderNode(node *TreeNode, prefix string, isLast bool, lines *[]string) {
	var connector string
	if isLast {
		connector = "└── "
	} else {
		connector = "├── "
	}

	style := lipgloss.NewStyle().Foreground(node.Color)
	if t.SelectedID == node.ID {
		style = style.Bold(true).Foreground(accentColor)
	}

	var line string
	if len(node.Children) > 0 {
		if node.Expanded {
			line = fmt.Sprintf("%s%s%s", prefix+connector, style.Render("▼ "), style.Render(node.Label))
		} else {
			line = fmt.Sprintf("%s%s%s", prefix+connector, style.Render("▶ "), style.Render(node.Label))
		}
	} else {
		line = fmt.Sprintf("%s%s%s", prefix+connector, style.Render("● "), style.Render(node.Label))
	}
	*lines = append(*lines, line)

	newPrefix := prefix
	if isLast {
		newPrefix += "    "
	} else {
		newPrefix += "│   "
	}

	for i, child := range node.Children {
		t.renderNode(child, newPrefix, i == len(node.Children)-1, lines)
	}
}

func (t *TreeModel) Select(id string) {
	t.SelectedID = id
}

func RenderTreeBox(title string, tree TreeModel) string {
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(0, 1)

	titleStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true)

	content := titleStyle.Render(" "+title+" ") + "\n\n" + tree.View()

	return box.Render(content)
}
