package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type TableColumn struct {
	Title string
	Width int
	Align lipgloss.Position
	Color lipgloss.Color
}

type TableRow []string

type TableModel struct {
	Columns []TableColumn
	Rows    []TableRow
	Width   int
}

func NewTable(columns []TableColumn) TableModel {
	totalWidth := 0
	for _, col := range columns {
		totalWidth += col.Width
	}

	return TableModel{
		Columns: columns,
		Rows:    make([]TableRow, 0),
		Width:   totalWidth + len(columns)*3 + 1,
	}
}

func (t *TableModel) AddRow(row TableRow) {
	if len(row) == len(t.Columns) {
		t.Rows = append(t.Rows, row)
	}
}

func (t *TableModel) View() string {
	var lines []string

	lines = append(lines, t.renderHeader())
	lines = append(lines, t.renderSeparator())

	for _, row := range t.Rows {
		lines = append(lines, t.renderRow(row))
	}

	return strings.Join(lines, "\n")
}

func (t *TableModel) renderHeader() string {
	cells := make([]string, 0, len(t.Columns))

	for _, col := range t.Columns {
		style := lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true).
			Width(col.Width).
			Align(col.Align)

		cells = append(cells, style.Render(col.Title))
	}

	header := strings.Join(cells, " │ ")
	return "│ " + header + " │"
}

func (t *TableModel) renderSeparator() string {
	parts := make([]string, 0, len(t.Columns))

	for _, col := range t.Columns {
		part := strings.Repeat("─", col.Width)
		parts = append(parts, part)
	}

	sep := strings.Join(parts, "─┼─")
	return "├─" + sep + "─┤"
}

func (t *TableModel) renderRow(row TableRow) string {
	cells := make([]string, 0, len(t.Columns))

	for i, col := range t.Columns {
		content := row[i]
		if len(content) > col.Width {
			content = content[:col.Width-3] + "..."
		}

		style := lipgloss.NewStyle().
			Foreground(col.Color).
			Width(col.Width).
			Align(col.Align)

		cells = append(cells, style.Render(content))
	}

	rowStr := strings.Join(cells, " │ ")
	return "│ " + rowStr + " │"
}

func RenderBorderedTable(title string, table TableModel) string {
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(0, 1)

	titleStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true)

	content := titleStyle.Render(" "+title+" ") + "\n\n" + table.View()

	return box.Render(content)
}

func SimpleTable(headers []string, rows [][]string) string {
	columns := make([]TableColumn, len(headers))
	for i, h := range headers {
		columns[i] = TableColumn{
			Title: h,
			Width: 15,
			Align: lipgloss.Left,
			Color: textColor,
		}
	}

	table := NewTable(columns)
	for _, row := range rows {
		table.AddRow(row)
	}

	return table.View()
}

func StatusTable(data map[string]string) string {
	rows := make([][]string, 0, len(data))
	for k, v := range data {
		rows = append(rows, []string{k, v})
	}

	return SimpleTable([]string{"Status", "Value"}, rows)
}
