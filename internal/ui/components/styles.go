package components

import "tocli/internal/ui/theme"

// SetStyles swaps the palette at runtime (e.g. when the terminal background is detected).
func (m *TaskListModel) SetStyles(s theme.Styles)     { m.styles = s }
func (m *AgendaModel) SetStyles(s theme.Styles)       { m.styles = s }
func (m *ContributionModel) SetStyles(s theme.Styles) { m.styles = s }
func (m *ProgressBarModel) SetStyles(s theme.Styles)  { m.styles = s }
