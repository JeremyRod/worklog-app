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
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "github.com/mattn/go-sqlite3"
)

var useHighPerformanceRenderer = false

type model struct {
	// New inputs
	inputs     []textinput.Model // items on the to-do list
	focusIndex int               // which to-do list item our cursor is pointing at
	inputsPos  []int             //array to track cursor pos for each input

	// Modify inputs
	modInputs     []textinput.Model // items for the modify list, same as the new list.
	modFocusIndex int               // Focus index for Modify List
	modRowID      int
	currentDate   time.Time // Date to get entries from

	// Summary View
	sumContent string
	viewport   viewport.Model
	ready      bool

	// List view
	list       list.Model
	id         int         // Last current query id
	maxId      int         // For offset tracking
	cursorMode cursor.Mode // which to-do items are selected

	// Track app state for view rendering
	state ViewState

	// Maintain current window size in model for list rerendering.
	winH int
	winW int

	// Track error messages in string builder and print in view
	errBuilder string
}

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()
)

func (m model) headerView() string {
	title := titleStyle.Render("Summary View")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m model) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

var (
	focusedStyle             = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle             = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle              = focusedStyle
	noStyle                  = lipgloss.NewStyle()
	helpStyle                = blurredStyle
	cursorModeHelpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	docStyle                 = lipgloss.NewStyle().Margin(2, 2, 0, 2)
	focusedButton            = focusedStyle.Render("[ Submit ]")
	blurredButton            = blurredStyle.Render("[ Submit ]")
	focusDelete              = focusedStyle.Render("[ Delete ]")
	blurDelete               = blurredStyle.Render("[ Delete ]")
	focusSave                = focusedStyle.Render("[ Save ]")
	blurSave                 = blurredStyle.Render("[ Save ]")
	focusImport              = focusedStyle.Render("[ Import ]")
	blurImport               = blurredStyle.Render("[ Import ]")
	submitFailed        bool = false
)

// FIXME: Fix the formatting here
func (e EntryRow) Title() string {
	date := e.entry.date.Format("02/01/2006")
	time := e.entry.hours.Hours()
	return fmt.Sprintf("%d: Date: %v Project: %s Hours: %.2f", e.entryId, date, e.entry.projCode, time)
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
	delete
	imp
)

type ViewState int

const (
	New ViewState = iota
	Get
	Modify
	Summary
)

func initialModel() model {
	m := model{
		inputs:      make([]textinput.Model, submit),
		modInputs:   make([]textinput.Model, submit),
		inputsPos:   make([]int, submit),
		list:        list.Model{},
		state:       New,
		id:          0,
		currentDate: time.Now(),
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
			t.Validate = durValidator
			t.CharLimit = 6
		}
		m.inputs[i] = t
	}

	for i := range m.modInputs {
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
			t.Placeholder = "Hours (opt) XXhXXm"
			t.Validate = durValidator
			t.CharLimit = 6
		}
		m.modInputs[i] = t
	}

	m.cursorMode = cursor.CursorStatic
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	if m.state == Get {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			h, v := docStyle.GetFrameSize()
			m.winH = msg.Height - v
			m.winW = msg.Width - h
			m.list.SetSize(m.winW, m.winH)
			//fmt.Println("resize")

		case tea.KeyMsg:
			switch msg.String() {

			case "ctrl+p":
				ents, err := db.QuerySummary(&m)
				//fmt.Println(ents)
				if err != nil {
					m.errBuilder = err.Error()
					return m, nil
				}
				for i := range ents {
					m.sumContent += fmt.Sprintf("%s\n%s\n", ents[i].Title(), ents[i].Description())
				}

				headerHeight := lipgloss.Height(m.headerView())
				footerHeight := lipgloss.Height(m.footerView())
				verticalMarginHeight := headerHeight + footerHeight

				m.viewport = viewport.New(m.winW, m.winH-verticalMarginHeight)
				m.viewport.YPosition = headerHeight
				m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
				m.viewport.SetContent(m.sumContent)
				m.ready = true
				m.viewport.YPosition = headerHeight + 1

				if useHighPerformanceRenderer {
					// Render (or re-render) the whole viewport. Necessary both to
					// initialize the viewport and when the window is resized.
					//
					// This is needed for high-performance rendering only.
					cmds = append(cmds, viewport.Sync(m.viewport))
				}
				m.state = Summary
				return m, tea.Batch(cmds...)

			case "delete":
				if items := m.list.Items(); len(items) != 0 {
					item := items[m.list.Index()].(EntryRow)
					if err := db.DeleteEntry(item.entryId); err != nil {
						fmt.Println(err)
					}
					m.modRowID = 0
					m.id = 0
					items = []list.Item{}
					m.list = list.New(items, list.NewDefaultDelegate(), 0, 0)
					m.list.Title = "Worklog Entries"
					m.ListUpdate()
					m.state = Get
				}

			case "enter":
				items := m.list.Items()
				item := items[m.list.Index()].(EntryRow)

				cmds := make([]tea.Cmd, len(m.modInputs))
				for i := 0; i <= len(m.modInputs)-1; i++ {
					if i == m.modFocusIndex {
						// Set focused state
						cmds[i] = m.modInputs[i].Focus()
						m.modInputs[i].PromptStyle = focusedStyle
						m.modInputs[i].TextStyle = focusedStyle
						continue
					}
					// Remove focused state
					m.modInputs[i].Blur()
					m.modInputs[i].PromptStyle = noStyle
					m.modInputs[i].TextStyle = noStyle
				}
				m.modInputs[date].SetValue(item.entry.date.Format("02/01/2006"))
				m.modInputs[code].SetValue(item.entry.projCode)
				m.modInputs[desc].SetValue(item.entry.desc)
				m.modInputs[startTime].SetValue(item.entry.startTime.String())
				m.modInputs[endTime].SetValue(item.entry.endTime.String())
				m.modInputs[hours].SetValue(item.entry.hours.String()[:len(item.entry.hours.String())-2])
				m.modRowID = item.entryId
				m.state = Modify
				return m, tea.Batch(cmds...)

				// db.ModifyEntry(item)
			case "tab":
				items := []list.Item{}
				m.list = list.New(items, list.NewDefaultDelegate(), 0, 0)
				m.list.Title = "Worklog Entries"
				m.id = 0
				m.state = New
			}
		}
		if m.list.Index() == len(m.list.Items())-1 && m.id != 1 {
			e, err := db.QueryEntries(&m)
			if err != nil {
				return m, nil
			}
			for _, v := range e {
				m.list.InsertItem(99999, v)
			}
		}
		m.list, cmd = m.list.Update(msg)
	} else if m.state == Summary {
		var (
			cmd  tea.Cmd
			cmds []tea.Cmd
		)
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q", "esc":
				return m, tea.Quit

			case "tab":
				m.sumContent = ""
				m.viewport.SetContent(m.sumContent)
				m.ListUpdate()
			}

		case tea.WindowSizeMsg:
			headerHeight := lipgloss.Height(m.headerView())
			footerHeight := lipgloss.Height(m.footerView())
			verticalMarginHeight := headerHeight + footerHeight
			useHighPerformanceRenderer = true

			// if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(m.winW, m.winH-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
			m.viewport.SetContent(m.sumContent)
			m.ready = true

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			m.viewport.YPosition = headerHeight + 1

			if useHighPerformanceRenderer {
				// Render (or re-render) the whole viewport. Necessary both to
				// initialize the viewport and when the window is resized.
				//
				// This is needed for high-performance rendering only.
				cmds = append(cmds, viewport.Sync(m.viewport))
			}
		}

		// Handle keyboard and mouse events in the viewport
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)

	} else if m.state == New {
		switch msg := msg.(type) {

		case tea.WindowSizeMsg:
			h, v := docStyle.GetFrameSize()
			m.winH = msg.Height - v
			m.winW = msg.Width - h
			m.list.SetSize(m.winW, m.winH)

		case tea.KeyMsg:
			switch msg.String() {

			case "tab":
				m.ListUpdate()

			case "ctrl+c", "esc":
				return m, tea.Quit

			// Set focus to next input
			case "enter", "up", "down", "left", "right":
				s := msg.String()

				// Did the user press enter while the submit button was focused?
				// If so, exit.
				if s == "enter" && m.focusIndex == len(m.inputs) {
					entry := EntryRow{}
					//Testing with local copy incase pointer edits data.
					if err := entry.FillData(m.inputs); err != nil {
						m.errBuilder = err.Error()
						submitFailed = true
						break
					}
					if err := db.SaveEntry(entry); err != nil {
						submitFailed = true
					} else {
						submitFailed = false
						m.resetState()
					}
				} else if s == "enter" && m.focusIndex == len(m.inputs)+1 {
					line, err := ImportWorklog()
					if err != nil {
						submitFailed = true
						m.errBuilder = err.Error() + " " + strconv.Itoa(line)
					}
				}

				// Cycle cursor position in input
				if m.focusIndex <= len(m.inputs)-1 && s == "right" {
					m.inputsPos[m.focusIndex] += 1
					if m.inputsPos[m.focusIndex] < 0 {
						m.inputsPos[m.focusIndex] = 0
					}
					m.inputs[m.focusIndex].SetCursor(m.inputsPos[m.focusIndex])
				} else if m.focusIndex <= len(m.inputs)-1 && s == "left" {
					m.inputsPos[m.focusIndex] -= 1
					if m.inputsPos[m.focusIndex] > len(m.inputs[m.focusIndex].Value()) {
						m.inputsPos[m.focusIndex] = len(m.inputs[m.focusIndex].Value()) - 1
					}
					m.inputs[m.focusIndex].SetCursor(m.inputsPos[m.focusIndex])
				}

				// Cycle indexes
				if s == "up" {
					m.focusIndex--
				} else if s == "down" {
					m.focusIndex++
				}

				if m.focusIndex > len(m.inputs)+1 {
					m.focusIndex = 0
				} else if m.focusIndex < 0 {
					m.focusIndex = len(m.inputs) + 1
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
	} else if m.state == Modify {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			h, v := docStyle.GetFrameSize()
			m.winH = msg.Height - v
			m.winW = msg.Width - h

		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+i": // Switch back to New entry screen.
				m.state = New

			case "tab":
				m.resetModState()
				m.ListUpdate()

			case "enter", "up", "down", "left", "right":
				s := msg.String()
				if s == "enter" && m.modFocusIndex == len(m.modInputs) {
					entry := EntryRow{}
					//Testing with local copy incase pointer edits data.
					if err := entry.FillData(m.modInputs); err != nil {
						m.errBuilder = err.Error()
						submitFailed = true
						break
					}
					entry.entryId = m.modRowID
					if err := db.ModifyEntry(entry); err != nil {
						submitFailed = true
					}
					m.modRowID = 0
					m.state = Get
				} else if s == "enter" && m.modFocusIndex == len(m.modInputs)+1 {
					if err := db.DeleteEntry(m.modRowID); err != nil {
						fmt.Println(err)
					}
					m.modRowID = 0
					m.state = Get
				}

				// Cycle indexes
				if s == "up" || s == "left" {
					m.modFocusIndex--
				} else {
					m.modFocusIndex++
				}

				if m.modFocusIndex > len(m.modInputs)+1 {
					m.modFocusIndex = 0
				} else if m.modFocusIndex < 0 {
					m.modFocusIndex = len(m.modInputs) + 1
				}
				cmds := make([]tea.Cmd, len(m.modInputs))
				for i := 0; i <= len(m.modInputs)-1; i++ {
					if i == m.modFocusIndex {
						// Set focused state
						cmds[i] = m.modInputs[i].Focus()
						m.modInputs[i].PromptStyle = focusedStyle
						m.modInputs[i].TextStyle = focusedStyle
						continue
					}
					// Remove focused state
					m.modInputs[i].Blur()
					m.modInputs[i].PromptStyle = noStyle
					m.modInputs[i].TextStyle = noStyle
				}
				return m, tea.Batch(cmds...)
			}
		}
		cmd = m.updateInputs(msg)
	}
	return m, cmd
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	if m.state == New {
		for i := range m.inputs {
			m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
		}
	} else if m.state == Modify {
		for i := range m.modInputs {
			m.modInputs[i], cmds[i] = m.modInputs[i].Update(msg)
		}
	}

	return tea.Batch(cmds...)
}

func (m model) View() string {
	var b strings.Builder
	switch m.state {
	case New:
		useHighPerformanceRenderer = false
		for i := range m.inputs {
			b.WriteString(m.inputs[i].View())
			if i < len(m.inputs)-1 {
				b.WriteRune('\n')
			}
		}

		button := blurImport
		if m.focusIndex == len(m.inputs)+1 {
			button = focusImport
		}
		button2 := blurredButton
		if m.focusIndex == len(m.inputs) {
			button2 = focusedButton
		}
		fmt.Fprintf(&b, "\n\n%s\t\t%s\n\n", button2, button)

	case Get:
		useHighPerformanceRenderer = false
		// if err := m.ListUpdate(); err != nil {
		// 	b.WriteString(fmt.Sprintf("%v", err))
		// }
		_, err := b.WriteString(docStyle.Render(m.list.View()))
		if err != nil {
			b.WriteString(fmt.Sprintf("%v", err))
		}

	case Modify:
		useHighPerformanceRenderer = false
		for i := range m.modInputs {
			b.WriteString(m.modInputs[i].View())
			if i < len(m.modInputs)-1 {
				b.WriteRune('\n')
			}
		}

		button := blurSave
		if m.modFocusIndex == len(m.modInputs) {
			button = focusSave
		}
		button2 := blurDelete
		if m.modFocusIndex == len(m.modInputs)+1 {
			button2 = focusDelete
		}
		fmt.Fprintf(&b, "\n\n%s\t\t%s\n\n", button, button2)

	case Summary:
		// if !m.ready {
		// 	b.WriteString("\n  Initializing...")
		// } else {
		fmt.Fprintf(&b, "\n%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView()) //, m.footerView())
		//}
	}
	if submitFailed {
		b.WriteString(helpStyle.Render(m.errBuilder))
	} else {
		b.WriteString(helpStyle.Render(fmt.Sprintf("\n list cur idx: %d list len: %d last id: %d focus idx: %d", m.list.Index(), len(m.list.Items()), m.id, m.focusIndex)))
	}
	submitFailed = false

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
	m.inputsPos[date] = len(m.inputs[date].Value())
	m.inputsPos[endTime] = len(m.inputs[endTime].Value())
}

func (m *model) resetModState() {
	//fmt.Println(m.inputs[hours].Value())
	for v := range m.inputs {
		m.modInputs[v].Reset()
	}
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
	//if err := db.SaveEntry(&row2); err != nil {
	//	fmt.Println(err)
	//}
	//if err := db.DeleteEntry(&row); err != nil {
	//	fmt.Println(err)
	//}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	//_, err := ImportWorklog()
	//fmt.Println(err)
	db.CloseDatabase()
}

func (m *model) ListUpdate() error {
	e, err := db.QueryEntries(m)
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	for _, v := range e {
		m.list.InsertItem(99999, v)
		//fmt.Println(v.entryId)
	}
	m.state = Get
	//	fmt.Println(m.winW, m.winH)
	m.list.SetSize(m.winW, m.winH)
	return nil
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

func durValidator(s string) error {
	if len(s) > 6 {
		return fmt.Errorf("dur length incorrect")
	}
	if len(s) == 3 {
		if strings.Index(s, "h") != 2 && strings.Index(s, "m") != 2 {
			return fmt.Errorf("dur invalid")
		}
		if _, err := strconv.Atoi(s[:2]); err != nil {
			return fmt.Errorf("hour format incorrect")
		}
	}
	if len(s) == 6 {
		if strings.Index(s, "m") != 5 {
			return fmt.Errorf("dur invalid")
		}
		if _, err := strconv.Atoi(s[3:5]); err != nil {
			return fmt.Errorf("min format incorrect")
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
