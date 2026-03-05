package tui

import (
	"github.com/charmbracelet/lipgloss"
)

type FormField struct {
	Name        string
	Label       string
	Value       string
	Placeholder string
	Required    bool
	Password    bool
}

type FormModel struct {
	Title  string
	Fields []FormField
	Active int
}

func NewForm(title string) *FormModel {
	return &FormModel{
		Title:  title,
		Fields: make([]FormField, 0),
		Active: 0,
	}
}

func (f *FormModel) AddField(name, label, placeholder string, required bool) {
	f.Fields = append(f.Fields, FormField{
		Name:        name,
		Label:       label,
		Value:       "",
		Placeholder: placeholder,
		Required:    required,
		Password:    false,
	})
}

func (f *FormModel) AddPasswordField(name, label, placeholder string, required bool) {
	f.Fields = append(f.Fields, FormField{
		Name:        name,
		Label:       label,
		Value:       "",
		Placeholder: placeholder,
		Required:    required,
		Password:    true,
	})
}

func (f *FormModel) SetValue(name, value string) {
	for i := range f.Fields {
		if f.Fields[i].Name == name {
			f.Fields[i].Value = value
			break
		}
	}
}

func (f *FormModel) GetValue(name string) string {
	for _, field := range f.Fields {
		if field.Name == name {
			return field.Value
		}
	}
	return ""
}

func (f *FormModel) GetValues() map[string]string {
	result := make(map[string]string)
	for _, field := range f.Fields {
		result[field.Name] = field.Value
	}
	return result
}

func (f *FormModel) NextField() {
	if f.Active < len(f.Fields)-1 {
		f.Active++
	}
}

func (f *FormModel) PrevField() {
	if f.Active > 0 {
		f.Active--
	}
}

func (f *FormModel) ActiveField() *FormField {
	if f.Active >= 0 && f.Active < len(f.Fields) {
		return &f.Fields[f.Active]
	}
	return nil
}

func (f FormModel) View() string {
	var lines []string

	titleStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Align(lipgloss.Center)

	lines = append(lines, titleStyle.Render(f.Title))
	lines = append(lines, "")

	for i, field := range f.Fields {
		lines = append(lines, f.renderField(i, field))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (f FormModel) renderField(index int, field FormField) string {
	var labelStyle lipgloss.Style
	var valueStyle lipgloss.Style

	if index == f.Active {
		labelStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)
		valueStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(statusBgColor)
	} else {
		labelStyle = lipgloss.NewStyle().
			Foreground(textColor)
		valueStyle = lipgloss.NewStyle().
			Foreground(dimColor)
	}

	label := labelStyle.Render(field.Label)
	if field.Required {
		label += lipgloss.NewStyle().Foreground(errorColor).Render(" *")
	}

	value := field.Value
	if value == "" {
		value = field.Placeholder
		valueStyle = valueStyle.Italic(true)
	}

	if field.Password {
		value = "••••••••"
	}

	return label + "\n" + valueStyle.Render("  "+value) + "\n"
}

func (f FormModel) Validate() (bool, string) {
	for _, field := range f.Fields {
		if field.Required && field.Value == "" {
			return false, field.Label + " is required"
		}
	}
	return true, ""
}

func RenderFormBox(title string, form *FormModel) string {
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(1, 2)

	return box.Width(50).Render(form.View())
}
