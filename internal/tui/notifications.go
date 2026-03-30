package tui

//go:generate stringer -type=NotificationType -linecomment

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type NotificationType int

const (
	NotificationInfo NotificationType = iota
	NotificationSuccess
	NotificationWarning
	NotificationError
)

type Notification struct {
	ID        string
	Type      NotificationType
	Title     string
	Message   string
	Timestamp time.Time
	Duration  time.Duration
	Dismissed bool
}

type NotificationManager struct {
	Notifications []Notification
	MaxVisible    int
}

func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		Notifications: make([]Notification, 0),
		MaxVisible:    3,
	}
}

func (nm *NotificationManager) Add(notifType NotificationType, title, message string) Notification {
	notif := Notification{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:      notifType,
		Title:     title,
		Message:   message,
		Timestamp: time.Now(),
		Duration:  5 * time.Second,
		Dismissed: false,
	}

	nm.Notifications = append(nm.Notifications, notif)

	if len(nm.Notifications) > nm.MaxVisible {
		nm.Notifications = nm.Notifications[1:]
	}

	return notif
}

func (nm *NotificationManager) Info(title, message string) Notification {
	return nm.Add(NotificationInfo, title, message)
}

func (nm *NotificationManager) Success(title, message string) Notification {
	return nm.Add(NotificationSuccess, title, message)
}

func (nm *NotificationManager) Warning(title, message string) Notification {
	return nm.Add(NotificationWarning, title, message)
}

func (nm *NotificationManager) Error(title, message string) Notification {
	return nm.Add(NotificationError, title, message)
}

func (nm *NotificationManager) Dismiss(id string) {
	for i := range nm.Notifications {
		if nm.Notifications[i].ID == id {
			nm.Notifications[i].Dismissed = true
			break
		}
	}
}

func (nm *NotificationManager) Clear() {
	nm.Notifications = make([]Notification, 0)
}

func (nm *NotificationManager) View() string {
	var lines []string

	visible := 0
	for _, notif := range nm.Notifications {
		if notif.Dismissed {
			continue
		}
		if visible >= nm.MaxVisible {
			break
		}

		lines = append(lines, nm.formatNotification(notif))
		visible++
	}

	if len(lines) == 0 {
		return ""
	}

	return lipgloss.JoinVertical(lipgloss.Right, lines...)
}

func (nm *NotificationManager) formatNotification(notif Notification) string {
	var icon string
	var borderColor lipgloss.Color
	var titleColor lipgloss.Color

	switch notif.Type {
	case NotificationInfo:
		icon = "ℹ"
		borderColor = accentColor
		titleColor = accentColor
	case NotificationSuccess:
		icon = "✓"
		borderColor = agentColor
		titleColor = agentColor
	case NotificationWarning:
		icon = "⚠"
		borderColor = systemColor
		titleColor = systemColor
	case NotificationError:
		icon = "✗"
		borderColor = errorColor
		titleColor = errorColor
	}

	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(40)

	titleStyle := lipgloss.NewStyle().
		Foreground(titleColor).
		Bold(true)

	msgStyle := lipgloss.NewStyle().
		Foreground(textColor)

	content := titleStyle.Render(icon+" "+notif.Title) + "\n" + msgStyle.Render(notif.Message)

	return box.Render(content)
}

func RenderToast(notifType NotificationType, title, message string) string {
	nm := NewNotificationManager()
	notif := nm.Add(notifType, title, message)
	return nm.formatNotification(notif)
}
