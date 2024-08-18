package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "github.com/mattn/go-sqlite3"
)

type model struct {
	inputs     []textinput.Model // items on the to-do list
	focusIndex int               // which to-do list item our cursor is pointing at
	cursorMode cursor.Mode       // which to-do items are selected

	Tabs       []string
	TabContent []string
	activeTab  int
}

var (
	focusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle         = focusedStyle
	noStyle             = lipgloss.NewStyle()
	helpStyle           = blurredStyle
	cursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	focusedButton      = focusedStyle.Render("[ Submit ]")
	blurredButton      = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
	submitFailed  bool = false
)

const (
	date = iota
	code
	desc
	startTime
	endTime
	hours
	submit
)

func dateValidator(s string) error {
	// Date in format is DD/MM/YYYY = 10c or 8 without slash
	if len(s) > 10 {
		return fmt.Errorf("date length incorrect")
	}
	if len(s) == 0 || (s[len(s)-1] < '0' || s[len(s)-1] > '9') && s[len(s)-1] != '/' {
		return fmt.Errorf("date is invalid")
	}
	if len(s) == 3 {
		if strings.Index(s, "/") != 2 {
			return fmt.Errorf("date invalid")
		}
		if dd, err := strconv.Atoi(s[:2]); err != nil || dd < 0 || dd > 31 {
			return fmt.Errorf("hour format incorrect")
		}
	}
	if len(s) == 6 {
		if strings.LastIndex(s, "/") != 5 {
			return fmt.Errorf("date invalid")
		}
		if mm, err := strconv.Atoi(s[3:5]); err != nil || mm < 0 || mm > 12 {
			return fmt.Errorf("min format incorrect")
		}
	}
	if len(s) == 10 {
		if yr, err := strconv.Atoi(s[6 : len(s)-1]); err != nil || yr < 0 {
			return fmt.Errorf("year format incorrect")
		}
	}
	return nil
}

func timeValidator(s string) error {
	if len(s) > 5 {
		return fmt.Errorf("date length incorrect")
	}
	if len(s) == 3 {
		if strings.Index(s, ":") != 2 {
			return fmt.Errorf("date invalid")
		}
		if hh, err := strconv.Atoi(s[:2]); err != nil || hh < 0 || hh > 23 {
			return fmt.Errorf("hour format incorrect")
		}
	}
	if len(s) == 5 {
		if mm, err := strconv.Atoi(s[3:5]); err != nil || mm < 0 || mm > 59 {
			return fmt.Errorf("hour format incorrect")
		}
	}
	return nil
}

func initialModel() model {
	m := model{
		inputs: make([]textinput.Model, submit),
	}

	var t textinput.Model
	tt := time.Now()

	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 32

		switch i {
		case date:
			t.Placeholder = fmt.Sprintf("%v", tt.Format("02/01/2006"))
			t.EchoMode = textinput.EchoNormal
			t.Validate = dateValidator
			t.Focus()
			t.SetValue(fmt.Sprintf("%v", tt.Format("02/01/2006")))

		case code:
			t.Placeholder = "Proj Code"
			t.CharLimit = 10

		case desc:
			t.Placeholder = "Entry Desc"
			t.CharLimit = 500
			t.Width = 50

		case startTime:
			t.Placeholder = "Start time: HH:MM"
			t.Validate = timeValidator
			t.CharLimit = 5

		case endTime:
			t.Placeholder = fmt.Sprintf("%v", tt.Format("15:04"))
			t.Validate = timeValidator
			t.CharLimit = 5

		case hours:
			t.Placeholder = "Hours (opt) HH:MM"
			t.Validate = timeValidator
			t.CharLimit = 5
		}
		m.inputs[i] = t
	}
	m.cursorMode = cursor.CursorStatic
	return m
}

func (m model) Init() tea.Cmd {

	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		// Set focus to next input
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Did the user press enter while the submit button was focused?
			// If so, exit.
			if s == "enter" && m.focusIndex == len(m.inputs) {
				m.resetState()
			}

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					// Set focused state
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				// Remove focused state
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}
	}

	// Handle character input and blinking
	cmd := m.updateInputs(msg)

	return m, cmd
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m model) View() string {
	var b strings.Builder

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	if submitFailed {
		b.WriteString(helpStyle.Render("Please fix input issues and try again"))
	} else {
		b.WriteString(helpStyle.Render("cursor mode is "))
	}
	//b.WriteString(cursorModeHelpStyle.Render(m.cursorMode.String()))
	//b.WriteString(helpStyle.Render(" (ctrl+r to change style)"))

	return b.String()
}

func (m *model) resetState() {
	//fmt.Println(m.inputs[hours].Value())
	for v := range m.inputs {
		m.inputs[v].Reset()
	}
}

var db Database = Database{db: nil}

func main() {

	// ent := Entry{hours: 4.2, projCode: "RPT", desc: "Hello this is my life now"}
	// row := EntryRow{entry: ent, entryId: 99}
	// row2 := EntryRow{entry: Entry{hours: 3.1, projCode: "EOS", desc: "hih"}, entryId: 100}
	// if err := db.OpenDatabase(); err != nil {
	// 	db.CreateDatabase()
	// }

	// if err := db.SaveEntry(&row); err != nil {
	// 	fmt.Println(err)
	// }
	// if err := db.SaveEntry(&row2); err != nil {
	// 	fmt.Println(err)
	// }
	// if err := db.DeleteEntry(&row); err != nil {
	// 	fmt.Println(err)
	// }
	// db.CloseDatabase()

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
