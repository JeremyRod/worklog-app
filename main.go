package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "github.com/mattn/go-sqlite3"
)

type model struct {
	inputs     []textinput.Model // items on the to-do list
	focusIndex int               // which to-do list item our cursor is pointing at
	cursorMode cursor.Mode       // which to-do items are selected

	list  list.Model
	state ViewState
	id    int // Last current query id
}

type ListModel struct {
	list list.Model
}

type AddModel struct {
}

var (
	focusedStyle             = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle             = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle              = focusedStyle
	noStyle                  = lipgloss.NewStyle()
	helpStyle                = blurredStyle
	cursorModeHelpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	docStyle                 = lipgloss.NewStyle().Margin(2, 2)
	focusedButton            = focusedStyle.Render("[ Submit ]")
	blurredButton            = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
	prevButtonFocus          = focusedStyle.Render("[ Prev ]")
	prevButtonBlur           = fmt.Sprintf("[ %s ]", blurredStyle.Render("Prev"))
	nextButtonFocus          = focusedStyle.Render("[ Next ]")
	nextButtonBlur           = fmt.Sprintf("[ %s ]", blurredStyle.Render("Next"))
	submitFailed        bool = false
)

func (e EntryRow) Title() string {
	return fmt.Sprintf("%d: %v: %s: %v", e.entryId, e.entry.date, e.entry.projCode, e.entry.hours)
}
func (e EntryRow) Description() string { return e.entry.desc }
func (e EntryRow) FilterValue() string { return e.entry.projCode }

const (
	date = iota
	code
	desc
	startTime
	endTime
	hours
	submit
)

type ViewState int

const (
	New ViewState = iota
	Get
	Modify
)

func initialModel() model {
	m := model{
		inputs: make([]textinput.Model, submit),
		list:   list.Model{},
		state:  New,
		id:     0,
	}

	var t textinput.Model
	tt := time.Now()

	items := []list.Item{}

	m.list = list.New(items, list.NewDefaultDelegate(), 0, 0)
	m.list.Title = "Worklog Entries"

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
			t.SetValue(fmt.Sprintf("%v", tt.Format("15:04")))

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
	var cmd tea.Cmd
	if m.state == Get {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+p":
				m.state = New
			}

		}
		if m.list.Cursor() == len(m.list.Items())-1 {
			e, err := db.QueryEntries(&m)
			if err != nil {
				return m, nil
			}
			for _, v := range e {
				m.list.InsertItem(99999, v)
			}
		}
		m.list, cmd = m.list.Update(msg)
	} else if m.state == New {
		switch msg := msg.(type) {

		case tea.WindowSizeMsg:
			h, v := docStyle.GetFrameSize()
			m.list.SetSize(msg.Width-h, msg.Height-v)

		case tea.KeyMsg:
			switch msg.String() {

			case "ctrl+u":
				e, err := db.QueryEntries(&m)
				if err != nil {
					return m, nil
				}
				for _, v := range e {
					m.list.InsertItem(99999, v)
				}
				m.state = Get

			case "ctrl+c", "esc":
				return m, tea.Quit

			// Set focus to next input
			case "tab", "shift+tab", "enter", "up", "down", "left", "right":
				s := msg.String()

				// Did the user press enter while the submit button was focused?
				// If so, exit.
				if s == "enter" && m.focusIndex == len(m.inputs) {
					entry := EntryRow{}
					//Testing with local copy incase pointer edits data.
					entry.FillData(m)
					if err := db.SaveEntry(&entry); err != nil {
						submitFailed = true
					} else {
						submitFailed = false
						m.resetState()
					}
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
		cmd = m.updateInputs(msg)
	}

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
	switch m.state {
	case New:
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

	case Get:
		_, err := b.WriteString(docStyle.Render(m.list.View()))
		if err != nil {
			b.WriteString(fmt.Sprintf("%v", err))
		}

		// button := &prevButtonBlur
		// if m.focusIndex == len(m.inputs) {
		// 	button = &prevButtonFocus
		// }
		// fmt.Fprintf(&b, "\n\n%s\n\n", *button)

		// button2 := &nextButtonBlur
		// if m.focusIndex == len(m.inputs) {
		// 	button = &nextButtonFocus
		// }
		// fmt.Fprintf(&b, "\n\n%s\n\n", *button2)
	}
	if submitFailed {
		b.WriteString(helpStyle.Render("Please fix input issues and try again"))
	} else {
		b.WriteString(helpStyle.Render("cursor mode is "))
	}
	b.WriteString(cursorModeHelpStyle.Render(m.cursorMode.String()))
	b.WriteString(helpStyle.Render(" (ctrl+r to change style)"))

	return docStyle.Render(b.String())
}

func (m *model) resetState() {
	//fmt.Println(m.inputs[hours].Value())
	t := time.Now()
	for v := range m.inputs {
		m.inputs[v].Reset()
	}
	m.inputs[date].SetValue(fmt.Sprintf("%v", t.Format("02/01/2006")))
	m.inputs[endTime].SetValue(fmt.Sprintf("%v", t.Format("15:04")))
}

var db Database = Database{db: nil}

func main() {

	// row2 := EntryRow{entry: Entry{hours: 3.1, projCode: "EOS", desc: "hih"}, entryId: 100}
	if err := db.OpenDatabase(); err != nil {
		fmt.Println(err)
		err = db.CreateDatabase()
		if err != nil {
			fmt.Printf("err: %v", err)
		}
	}
	// if err := db.SaveEntry(&row2); err != nil {
	// 	fmt.Println(err)
	// }
	// if err := db.DeleteEntry(&row); err != nil {
	// 	fmt.Println(err)
	// }

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	db.CloseDatabase()
}

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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}