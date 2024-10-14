package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

var useHighPerformanceRenderer = false

var (
	version   = "dev"  // default value, overridden at build time
	gitCommit = "none" // default value, overridden at build time
)

type model struct {
	// New inputs
	inputs     []textinput.Model // items on the to-do list
	focusIndex int               // which to-do list item our cursor is pointing at
	inputsPos  []int             //array to track cursor pos for each input
	// New Notes text area
	textarea textarea.Model

	// Modify inputs
	modInputs     []textinput.Model // items for the modify list, same as the new list.
	modFocusIndex int               // Focus index for Modify List
	modRowID      int
	modInputsPos  []int     //array to track cursor pos for each input
	currentDate   time.Time // Date to get entries from
	// Modify Notes text area
	modtextarea textarea.Model

	// Summary View
	sumContent string
	viewport   viewport.Model
	ready      bool

	// Entries List view
	list       list.Model
	id         int         // Last current query id
	maxId      int         // For offset tracking
	cursorMode cursor.Mode // which to-do items are selected

	// Retreived tasks list view
	listTask list.Model
	choice   string
	// taskDone bool // Use this to make the check loop wait for the user to choose a task

	// Login view for uploads
	loginInputs     []textinput.Model
	loginFocusIndex int
	formLogged      bool

	// TODO: logger for error logging, do rotations and nice logging later

	// Track app state for view rendering
	state    ViewState
	substate SubState

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
	modelStyle = lipgloss.NewStyle().
			Width(50).
			Height(10).
			Align(lipgloss.Left).
			BorderStyle(lipgloss.HiddenBorder())
	focusedModelStyle = lipgloss.NewStyle().
				Width(50).
				Height(10).
				Align(lipgloss.Left).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("69"))
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
	focusCancel              = focusedStyle.Render("[ Cancel ]")
	blurCancel               = blurredStyle.Render("[ Cancel ]")
	focusSave                = focusedStyle.Render("[ Save ]")
	blurSave                 = blurredStyle.Render("[ Save ]")
	focusUpload              = focusedStyle.Render("[ Upload ]")
	blurUpload               = blurredStyle.Render("[ Upload ]")
	focusImport              = focusedStyle.Render("[ Import ]")
	blurImport               = blurredStyle.Render("[ Import ]")
	focusExport              = focusedStyle.Render("[ Export ]")
	blurExport               = blurredStyle.Render("[ Export ]")
	focusUnlink              = focusedStyle.Render("[ Unlink ]")
	blurUnlink               = blurredStyle.Render("[ Unlink ]")
	submitFailed        bool = false
)

// FIXME: Fix the formatting here
func (e EntryRow) Title() string {
	date := e.entry.date.Format("02/01/2006")
	time := fmt.Sprintf("%d:%02d", int(e.entry.hours.Hours()), int(e.entry.hours.Minutes())%60)
	return fmt.Sprintf("Date: %v Project: %s Hours: %s", date, e.entry.projCode, time)
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
	deleted
	imp
	unlink
)

const (
	username = iota
	password
	cancel
	submitted
)

type ViewState int

const (
	New ViewState = iota
	Get
	Modify
	Summary
	Task
	Login
)

type SubState int

const (
	ListView SubState = iota
	NotesView
)

func initialModel() model {
	m := model{
		inputs:       make([]textinput.Model, submit),
		modInputs:    make([]textinput.Model, submit),
		loginInputs:  make([]textinput.Model, cancel), // Only up to cancel since only two are inputs, rest are buttons.
		inputsPos:    make([]int, submit),
		modInputsPos: make([]int, submit),
		list:         list.Model{},
		state:        New,
		substate:     ListView,
		id:           0,
		currentDate:  time.Now(),
	}

	var t textinput.Model
	tt := time.Now()

	items := []list.Item{}

	m.list = list.New(items, list.NewDefaultDelegate(), 0, 0)
	m.list.Title = "Worklog Entries"

	ti := textarea.New()
	ti.Placeholder = "Add notes here...."
	ti.CharLimit = 2000
	m.textarea = ti
	m.modtextarea = ti

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

	for i := range m.loginInputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 50

		switch i {
		case username:
			t.Placeholder = "Username/Email here"
			t.EchoMode = textinput.EchoNormal
			t.Focus()

		case password:
			t.Placeholder = "Password"
			t.EchoMode = textinput.EchoPassword
		}
		m.loginInputs[i] = t
	}
	m.ListUpdate()
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
				date := ents[0].entry.date
				duration := make(map[string]time.Duration)
				desc := make(map[string]string)
				for i := range ents {
					if date != ents[i].entry.date {
						date = ents[i].entry.date
						m.sumContent += ents[i].entry.date.Format("02/01/2006")
						m.sumContent += "\n\n"
						for k, v := range duration {
							m.sumContent += fmt.Sprintf("Project: %s Hours:%02d:%02d\n", k, int(v.Hours()), int(v.Minutes())%60)
							m.sumContent += desc[k] + "\n"
						}
						clear(desc)
						clear(duration)
					}
					duration[ents[i].entry.projCode] += ents[i].entry.hours
					desc[ents[i].entry.projCode] += ents[i].entry.desc + ". "

					//m.sumContent += fmt.Sprintf("%s\n%s\n", ents[i].Title(), ents[i].Description())
					//m.sumContent += "\n"
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
						log.Println(err)
					}
					m.modRowID = 0
					m.id -= 1
					m.list.RemoveItem(m.list.Index())
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
				m.modtextarea.SetValue(item.entry.notes)
				m.state = Modify
				return m, tea.Batch(cmds...)

				// db.ModifyEntry(item)
			case "tab":
				// items := []list.Item{}
				// m.list = list.New(items, list.NewDefaultDelegate(), 0, 0)
				// m.list.Title = "Worklog Entries"
				//m.id = 0
				m.state = New
			}
		}
		if m.list.Index() == len(m.list.Items())-1 && m.id != 1 {
			// e, err := db.QueryEntries(&m)
			// if err != nil {
			// 	return m, nil
			// }
			// for _, v := range e {
			// 	m.list.InsertItem(99999, v)
			// }
			m.ListUpdate()
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
			case "ctrl+c":
				return m, tea.Quit

			case "tab":
				m.sumContent = ""
				m.viewport.SetContent(m.sumContent)
				m.state = Get
			}

		case tea.WindowSizeMsg:
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
		}

		// // Handle keyboard and mouse events in the viewport
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
			if m.substate == ListView {
				switch msg.String() {

				case "tab":
					m.state = Get

				case "ctrl+shift+left", "ctrl+shift+right":
					m.substate = NotesView

				case "ctrl+c":
					return m, tea.Quit

				// Set focus to next input
				case "enter", "up", "down", "left", "right":
					s := msg.String()

					// Did the user press enter while the submit button was focused?
					// If so, exit.
					if s == "enter" && m.focusIndex == len(m.inputs) {
						entry := EntryRow{}
						//Testing with local copy incase pointer edits data.
						if err := entry.FillData(&m); err != nil {
							m.errBuilder = err.Error()
							submitFailed = true
							break
						}
						if err := db.SaveEntry(entry); err != nil {
							submitFailed = true
						} else {
							// Here is probably the only place we want to reset the list since we need the new id from the database
							// We also probably want to show the newest list at this point.
							items := []list.Item{}
							m.list = list.New(items, list.NewDefaultDelegate(), 0, 0)
							submitFailed = false
							m.resetState()
							m.id = 0
							m.ListUpdate()
						}

					} else if s == "enter" && m.focusIndex == len(m.inputs)+1 {
						line, err := ImportWorklog()
						if err != nil {
							submitFailed = true
							m.errBuilder = err.Error() + " " + strconv.Itoa(line)
						}
					} else if s == "enter" && m.focusIndex == len(m.inputs)+2 {
						err := db.QueryAndExport()
						if err != nil {
							log.Println(err)
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

					if m.focusIndex > len(m.inputs)+2 {
						m.focusIndex = 0
					} else if m.focusIndex < 0 {
						m.focusIndex = len(m.inputs) + 2
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
			} else {
				m.textarea.Focus()
				switch msg.String() {
				case "ctrl+shift+left", "ctrl+shift+right":
					m.substate = ListView

				case "ctrl+c":
					return m, tea.Quit

				case "tab":
					m.state = Get
				}
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
			if m.substate == ListView {
				switch msg.String() {
				case "ctrl+i": // Switch back to New entry screen.
					m.state = New

				case "ctrl+c":
					return m, tea.Quit

				case "tab":
					m.resetModState()
					m.state = Get

				case "ctrl+shift+left", "ctrl+shift+right":
					m.substate = NotesView

				case "enter", "up", "down", "left", "right":
					s := msg.String()
					if s == "enter" && m.modFocusIndex == len(m.modInputs) {
						entry := EntryRow{}
						//Testing with local copy incase pointer edits data.
						if err := entry.ModFillData(&m); err != nil {
							m.errBuilder = err.Error()
							submitFailed = true
							break
						}
						entry.entryId = m.modRowID
						if err := db.ModifyEntry(entry); err != nil {
							submitFailed = true
							break
						}
						m.modRowID = 0
						m.resetModState()
						// ent, err := db.QueryEntry(entry)
						// if err != nil {
						// 	log.Println(err)
						// 	break
						// }
						m.list.SetItem(m.list.Index(), entry)
						m.state = Get

					} else if s == "enter" && m.modFocusIndex == len(m.modInputs)+1 {
						items := m.list.Items()
						item := items[m.list.Index()].(EntryRow)
						if err := db.DeleteEntry(item.entryId); err != nil {
							log.Println(err)
						}
						m.modRowID = 0
						m.id -= 1
						m.resetModState()
						m.list.RemoveItem(m.list.Index())
						m.state = Get

					} else if s == "enter" && m.modFocusIndex == len(m.modInputs)+2 {
						// scoro upload
						entry := EntryRow{}
						if err := entry.ModFillData(&m); err != nil {
							m.errBuilder = err.Error()
							submitFailed = true
							break
						}
						entry.entryId = m.modRowID
						// Get user token
						check := LoginGetTasks(&m)
						if check {
							m.state = Login
							log.Println("Login failed/need creds")
							break
						}
						ok, err := CheckEventCodeMap(&m, entry)
						if err != nil {
							m.errBuilder += err.Error()
							break
						}
						if ok {
							// Get user token
							if err := DoTaskSubmit(entry); err != nil {
								m.errBuilder += err.Error()
							}
							// if check event codes needs some interaction, dont go to get state.
							m.modRowID = 0
							m.state = Get
							m.resetModState()
						}
					} else if s == "enter" && m.modFocusIndex == len(m.modInputs)+3 {
						entry := EntryRow{}
						if err := entry.ModFillData(&m); err != nil {
							m.errBuilder = err.Error()
							submitFailed = true
							break
						}
						err := db.DeleteLink(entry.entry.projCode)
						if err != nil {
							m.errBuilder = err.Error()
							submitFailed = true
							break
						}
						delete(ProjCodeToTask, entry.entry.projCode)
					}

					// Cycle cursor position in input
					if m.modFocusIndex <= len(m.modInputs)-1 && s == "right" {
						m.modInputsPos[m.modFocusIndex] += 1
						if m.modInputsPos[m.modFocusIndex] < 0 {
							m.modInputsPos[m.modFocusIndex] = 0
						}
						m.modInputs[m.modFocusIndex].SetCursor(m.modInputsPos[m.modFocusIndex])
					} else if m.modFocusIndex <= len(m.modInputs)-1 && s == "left" {
						m.modInputsPos[m.modFocusIndex] -= 1
						if m.modInputsPos[m.modFocusIndex] > len(m.modInputs[m.modFocusIndex].Value()) {
							m.modInputsPos[m.modFocusIndex] = len(m.modInputs[m.modFocusIndex].Value()) - 1
						}
						m.modInputs[m.modFocusIndex].SetCursor(m.modInputsPos[m.modFocusIndex])
					}
					// Cycle indexes
					if s == "up" {
						m.modFocusIndex--
					} else if s == "down" {
						m.modFocusIndex++
					}

					if m.modFocusIndex > len(m.modInputs)+3 {
						m.modFocusIndex = 0
					} else if m.modFocusIndex < 0 {
						m.modFocusIndex = len(m.modInputs) + 3
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
			} else {
				m.modtextarea.Focus()
				switch msg.String() {
				case "ctrl+shift+left", "ctrl+shift+right":
					m.substate = ListView

				case "ctrl+c":
					return m, tea.Quit

				case "tab":
					m.state = Get
				}
			}
		}
		cmd = m.updateInputs(msg)
	}
	if m.state == Task {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.listTask.SetWidth(msg.Width)
			return m, nil

		case tea.KeyMsg:
			switch keypress := msg.String(); keypress {
			case "ctrl+c":
				return m, tea.Quit

			case "enter": // Once a task is selected go back to modify view
				i := m.listTask.SelectedItem()
				AddToTaskMap(m.choice, i.(Data).EventName)
				m.state = Modify
			}
		}
		m.listTask, cmd = m.listTask.Update(msg)
	}
	if m.state == Login {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.listTask.SetWidth(msg.Width)
			return m, nil

		case tea.KeyMsg:
			switch keypress := msg.String(); keypress {
			case "ctrl+c":
				return m, tea.Quit

			case "enter", "up", "down", "left", "right": // Once a task is selected go back to modify view
				if keypress == "enter" && m.loginFocusIndex == len(m.loginInputs) {
					m.state = Modify
					LoginGetTaskForm(&m, m.loginInputs[username].Value(), m.loginInputs[password].Value())
				} else if keypress == "enter" && m.loginFocusIndex == len(m.loginInputs)+1 {
					m.resetLoginState()
					m.state = Modify
				}
				// Cycle indexes
				if keypress == "up" || keypress == "left" {
					m.loginFocusIndex--
				} else {
					m.loginFocusIndex++
				}

				if m.loginFocusIndex > len(m.loginInputs)+1 {
					m.loginFocusIndex = 0
				} else if m.loginFocusIndex < 0 {
					m.loginFocusIndex = len(m.loginInputs) + 1
				}
				cmds := make([]tea.Cmd, len(m.loginInputs))
				for i := 0; i <= len(m.loginInputs)-1; i++ {
					if i == m.loginFocusIndex {
						// Set focused state
						cmds[i] = m.loginInputs[i].Focus()
						m.loginInputs[i].PromptStyle = focusedStyle
						m.loginInputs[i].TextStyle = focusedStyle
						continue
					}
					// Remove focused state
					m.loginInputs[i].Blur()
					m.loginInputs[i].PromptStyle = noStyle
					m.loginInputs[i].TextStyle = noStyle
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
	loginCmds := make([]tea.Cmd, len(m.loginInputs))
	var textareaCmd tea.Cmd
	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	if m.state == New {
		if m.substate == ListView {
			for i := range m.inputs {
				m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
			}
		} else {
			m.textarea, textareaCmd = m.textarea.Update(msg)
			cmds = append(cmds, textareaCmd)
		}
	} else if m.state == Modify {
		if m.substate == ListView {
			for i := range m.inputs {
				m.modInputs[i], cmds[i] = m.modInputs[i].Update(msg)
			}
		} else {
			m.modtextarea, textareaCmd = m.modtextarea.Update(msg)
			cmds = append(cmds, textareaCmd)
		}
	} else if m.state == Login {
		for i := range m.loginInputs {
			m.loginInputs[i], loginCmds[i] = m.loginInputs[i].Update(msg)
		}
	}
	return tea.Batch(cmds...)
}

func (m model) View() string {
	var b strings.Builder
	switch m.state {
	case Task:
		_, err := b.WriteString(docStyle.Render(m.listTask.View()))
		if err != nil {
			b.WriteString(fmt.Sprintf("%v", err))
		}
	case New:
		var s string
		var n string
		for i := range m.inputs {
			s += m.inputs[i].View()
			if i < len(m.inputs)-1 {
				s += "\n" //b.WriteRune('\n')
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
		button3 := blurExport
		if m.focusIndex == len(m.inputs)+2 {
			button3 = focusExport
		}
		s += fmt.Sprintf("\n\n%s\t\t%s\t\t%s\n\n", button2, button, button3)
		n += fmt.Sprintf(
			"Notes for current entry.\n\n%s",
			m.textarea.View(),
		) + "\n\n"
		if m.substate == ListView {
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, focusedModelStyle.Render(s), modelStyle.Render(n)))
		} else {
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, modelStyle.Render(s), focusedModelStyle.Render(n)))
		}

	case Get:
		_, err := b.WriteString(docStyle.Render(m.list.View()))
		if err != nil {
			b.WriteString(fmt.Sprintf("%v", err))
		}

	case Modify:
		var (
			n string
			s string
		)
		for i := range m.modInputs {
			s += m.modInputs[i].View()
			if i < len(m.modInputs)-1 {
				s += "\n"
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
		button3 := blurUpload
		if m.modFocusIndex == len(m.modInputs)+2 {
			button3 = focusUpload
		}
		button4 := blurUnlink
		if m.modFocusIndex == len(m.modInputs)+3 {
			button4 = focusUnlink
		}
		s += fmt.Sprintf("\n\n%s\t%s\t%s\t%s\n\n", button, button2, button3, button4)
		n += fmt.Sprintf(
			"Notes for current entry.\n\n%s",
			m.modtextarea.View(),
		) + "\n\n"
		if m.substate == ListView {
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, focusedModelStyle.Render(s), modelStyle.Render(n)))
		} else {
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, modelStyle.Render(s), focusedModelStyle.Render(n)))
		}
	case Summary:
		fmt.Fprintf(&b, "\n%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView()) //, m.footerView())
		//}
	case Login:
		for i := range m.loginInputs {
			b.WriteString(m.loginInputs[i].View())
			if i < len(m.loginInputs)-1 {
				b.WriteRune('\n')
			}
		}
		button := blurSave
		if m.loginFocusIndex == len(m.loginInputs) {
			button = focusSave
		}
		button2 := blurCancel
		if m.loginFocusIndex == len(m.loginInputs)+1 {
			button2 = focusCancel
		}
		fmt.Fprintf(&b, "\n\n%s\t%s\n\n", button, button2)
	}
	b.WriteString(helpStyle.Render(fmt.Sprintf("\nVersion: %s\t rev: %s\n", version, gitCommit)))
	if submitFailed {
		b.WriteString(helpStyle.Render(m.errBuilder))
	} else {
		b.WriteString(helpStyle.Render(fmt.Sprintf("\n substate: %d list idx: %d list len %d, last id: %d focus idx: %d", m.substate, m.list.Index(), len(m.list.Items()), m.id, m.focusIndex)))
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
	m.textarea.Reset()
}

func (m *model) resetModState() {
	//fmt.Println(m.inputs[hours].Value())
	for v := range m.inputs {
		m.modInputs[v].Reset()
	}
	m.modtextarea.Reset()
}

func (m *model) resetLoginState() {
	//fmt.Println(m.inputs[hours].Value())
	for v := range m.loginInputs {
		m.loginInputs[v].Reset()
	}
}

var db Database = Database{db: nil}

func main() {
	// Logger for dev
	f, err := os.OpenFile("testlogfile.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(f)

	if err := db.OpenDatabase(); err != nil {
		log.Println(err)
	}

	// Get the saved projevent links, errs will return empty map, system can still run.
	ProjCodeToTask, err = db.QueryLinks()
	if err != nil {
		log.Println(err)
	}
	err = godotenv.Load("user.env")
	if err != nil {
		log.Println("Error loading user.env file")
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	db.CloseDatabase()
}

// I think this will only work for one entry, rethink logic for multi submission.
func CheckEventCodeMap(m *model, entries ...EntryRow) (bool, error) {
	// Check to see if we have all proj codes mapped to an event_id
	check := true
	for i := 0; i < len(entries); i++ {
		_, ok := ProjCodeToTask[entries[i].entry.projCode]
		if !ok {
			check = false
			items := TaskList.constructTaskList()
			m.choice = entries[i].entry.projCode
			m.listTask = list.New(items, list.NewDefaultDelegate(), 0, 0)
			m.listTask.Title = "Choose a task"
			m.state = Task
			m.listTask.SetSize(m.winW, m.winH)
		}
	}
	return check, nil
}

func (d *TaskListResp) constructTaskList() []list.Item {
	list := []list.Item{}
	for _, v := range TaskList.Data {
		list = append(list, v)
	}
	return list
}

func (m *model) ListUpdate() error {
	// Stop resetting the list. Append and keep index tracked.

	// items := []list.Item{}
	// m.list = list.New(items, list.NewDefaultDelegate(), 0, 0)
	// m.list.Title = "Worklog Entries"
	e, err := db.QueryEntries(m)
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	for _, v := range e {
		m.list.InsertItem(99999, v)
		//fmt.Println(v.entryId)
	}
	// m.state = Get
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
