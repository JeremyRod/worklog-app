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

	Tabs       []string
	TabContent []string
	activeTab  int

	list list.Model
}

var (
	focusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle         = focusedStyle
	noStyle             = lipgloss.NewStyle()
	helpStyle           = blurredStyle
	cursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	focusedButton        = focusedStyle.Render("[ Submit ]")
	blurredButton        = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
	prevButtonFocus      = focusedStyle.Render("[ Prev ]")
	prevButtonBlur       = fmt.Sprintf("[ %s ]", blurredStyle.Render("Prev"))
	nextButtonFocus      = focusedStyle.Render("[ Next ]")
	nextButtonBlur       = fmt.Sprintf("[ %s ]", blurredStyle.Render("Next"))
	submitFailed    bool = false
)

var (
	inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder   = tabBorderWithBottom("┘", " ", "└")
	docStyle          = lipgloss.NewStyle().Padding(1, 2, 1, 2)
	highlightColor    = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	inactiveTabStyle  = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(highlightColor).Padding(0, 1)
	activeTabStyle    = inactiveTabStyle.Border(activeTabBorder, true)
	windowStyle       = lipgloss.NewStyle().BorderForeground(highlightColor).Padding(2, 0).Align(lipgloss.Left).Border(lipgloss.NormalBorder()).UnsetBorderTop()
)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

const (
	date = iota
	code
	desc
	startTime
	endTime
	hours
	submit
)

const (
	New = iota
	Get
	Modify
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
		inputs:     make([]textinput.Model, submit),
		Tabs:       []string{"New Entry", "View Entries", "Modify Entry"},
		TabContent: []string{},
		list:       list.Model{},
	}

	var t textinput.Model
	tt := time.Now()

	items := []list.Item{
		item{title: "Raspberry Pi’s", desc: "I have ’em all over my house"},
		item{title: "Nutella", desc: "It's good on toast"},
		item{title: "Bitter melon", desc: "It cools you down"},
		item{title: "Nice socks", desc: "And by that I mean socks without holes"},
		item{title: "Eight hours of sleep", desc: "I had this once"},
		item{title: "Cats", desc: "Usually"},
		item{title: "Plantasia, the album", desc: "My plants love it too"},
		item{title: "Pour over coffee", desc: "It takes forever to make though"},
		item{title: "VR", desc: "Virtual reality...what is there to say?"},
		item{title: "Noguchi Lamps", desc: "Such pleasing organic forms"},
		item{title: "Linux", desc: "Pretty much the best OS"},
		item{title: "Business school", desc: "Just kidding"},
		item{title: "Pottery", desc: "Wet clay is a great feeling"},
		item{title: "Shampoo", desc: "Nothing like clean hair"},
		item{title: "Table tennis", desc: "It’s surprisingly exhausting"},
		item{title: "Milk crates", desc: "Great for packing in your extra stuff"},
		item{title: "Afternoon tea", desc: "Especially the tea sandwich part"},
		item{title: "Stickers", desc: "The thicker the vinyl the better"},
		item{title: "20° Weather", desc: "Celsius, not Fahrenheit"},
		item{title: "Warm light", desc: "Like around 2700 Kelvin"},
		item{title: "The vernal equinox", desc: "The autumnal equinox is pretty good too"},
		item{title: "Gaffer’s tape", desc: "Basically sticky fabric"},
		item{title: "Terrycloth", desc: "In other words, towel fabric"},
	}

	m.list = list.New(items, list.NewDefaultDelegate(), 0, 0)
	m.list.Title = "My Fave Things"

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
	switch msgg := msg.(type) {
	case tea.KeyMsg:
		switch msgg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		// Set focus to next input
		case "tab", "shift+tab", "enter", "up", "down", "left", "right":
			s := msgg.String()

			if m.activeTab == New {
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
				} else if s == "left" {
					m.activeTab = max(m.activeTab-1, 0)
				} else if s == "right" {
					m.activeTab = min(m.activeTab+1, len(m.Tabs)-1)
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
			} else if m.activeTab == Get {
				var cmd tea.Cmd

				switch masg := msg.(type) {
				case tea.KeyMsg:
					if s == "ctrl+c" {
						return m, tea.Quit
					} else if s == "left" {
						m.activeTab = max(m.activeTab-1, 0)
					} else if s == "right" {
						m.activeTab = min(m.activeTab+1, len(m.Tabs)-1)
					}
				case tea.WindowSizeMsg:
					h, v := docStyle.GetFrameSize()
					m.list.SetSize(masg.Width-h, masg.Height-v)

				}

				m.list, cmd = m.list.Update(msg)
				return m, cmd
			} else if m.activeTab == Modify {
				if s == "left" {
					m.activeTab = max(m.activeTab-1, 0)
				} else if s == "right" {
					m.activeTab = min(m.activeTab+1, len(m.Tabs)-1)
				}

			}
		}
	}

	// Handle character input and blinking
	cmd := m.updateInputs(msg)

	return m, cmd
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
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
	var renderedTabs []string

	for j := range m.Tabs {
		var s strings.Builder
		switch j {
		case New:
			for i := range m.inputs {
				s.WriteString(m.inputs[i].View())
				if i < len(m.inputs)-1 {
					s.WriteRune('\n')
				}
			}

			button := &blurredButton
			if m.focusIndex == len(m.inputs) {
				button = &focusedButton
			}
			fmt.Fprintf(&s, "\n\n%s\n\n", *button)
		case Get:
			docStyle.Render(m.list.View())

			// button := &prevButtonBlur
			// if m.focusIndex == len(m.inputs) {
			// 	button = &prevButtonFocus
			// }
			// fmt.Fprintf(&s, "\n\n%s\n\n", *button)

			// button2 := &nextButtonBlur
			// if m.focusIndex == len(m.inputs) {
			// 	button = &nextButtonFocus
			// }
			// fmt.Fprintf(&s, "\n\n%s\n\n", *button2)

		case Modify:

		}
		m.TabContent = append(m.TabContent, s.String())
	}

	for i, t := range m.Tabs {
		var style lipgloss.Style
		isFirst, isLast, isActive := i == 0, i == len(m.Tabs)-1, i == m.activeTab
		if isActive {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}
		border, _, _, _, _ := style.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst && !isActive {
			border.BottomLeft = "├"
		} else if isLast && isActive {
			border.BottomRight = "│"
		} else if isLast && !isActive {
			border.BottomRight = "┤"
		}
		style = style.Border(border)
		renderedTabs = append(renderedTabs, style.Render(t))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	b.WriteString(row)
	b.WriteString("\n")
	b.WriteString(windowStyle.Width((lipgloss.Width(row) - windowStyle.GetHorizontalFrameSize())).Height(10).Render(m.TabContent[m.activeTab]))
	b.WriteString("\n")

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

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	db.CloseDatabase()
}
