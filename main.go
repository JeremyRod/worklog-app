package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	i "github.com/JeremyRod/worklog-app/v2/internal"
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

	// Timer for periodic timeReset
	resetTimer *time.Timer

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
	ents       []i.EntryRow

	// Entries List view
	list       list.Model
	id         int         // Last current query id
	maxId      int         // For offset tracking
	cursorMode cursor.Mode // which to-do items are selected

	// Retreived tasks list view
	listTask list.Model
	choice   []string
	index    int

	// Retreived act list view
	listAct list.Model
	//actChoice []string
	actIndex int
	// taskDone bool // Use this to make the check loop wait for the user to choose a task

	// Login view for uploads
	loginInputs     []textinput.Model
	loginFocusIndex int
	formLogged      bool

	// Confirmation screen
	confirmationIndex int

	//Date selector view linked to summary
	dateCursor  int
	selectStart bool
	startDate   time.Time
	endDate     time.Time
	// TODO: logger for error logging, do rotations and nice logging later

	// Track app state for view rendering
	state    ViewState
	substate SubState
	retState ViewState

	// Maintain current window size in model for list rerendering.
	winH int
	winW int

	// Track error messages in string builder and print in view
	errBuilder string
}

var logger *log.Logger

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 0)
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
	focusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	summaryTotalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#390099"))
	summaryDateStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#592e83"))
	summaryProjStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#9984d4"))
	cursorStyle         = focusedStyle
	noStyle             = lipgloss.NewStyle()
	helpStyle           = blurredStyle
	cursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	docStyle            = lipgloss.NewStyle().Margin(4, 2, 0, 2)
	focusedButton       = focusedStyle.Render("[ Submit ]")
	blurredButton       = blurredStyle.Render("[ Submit ]")
	focusDelete         = focusedStyle.Render("[ Delete ]")
	blurDelete          = blurredStyle.Render("[ Delete ]")
	focusCancel         = focusedStyle.Render("[ Cancel ]")
	blurCancel          = blurredStyle.Render("[ Cancel ]")
	focusSave           = focusedStyle.Render("[ Save ]")
	blurSave            = blurredStyle.Render("[ Save ]")
	focusUpload         = focusedStyle.Render("[ Upload ]")
	blurUpload          = blurredStyle.Render("[ Upload ]")
	focusImport         = focusedStyle.Render("[ Import ]")
	blurImport          = blurredStyle.Render("[ Import ]")
	focusExport         = focusedStyle.Render("[ Export ]")
	blurExport          = blurredStyle.Render("[ Export ]")
	focusUnlink         = focusedStyle.Render("[ Unlink ]")
	blurUnlink          = blurredStyle.Render("[ Unlink ]")
	focusConfirm        = focusedStyle.Render("[ Confirm ]")
	blurConfirm         = blurredStyle.Render("[ Confirm ]")

	submitFailed bool = false
)

const (
	Username = iota
	Password
	Cancel
	Submitted
)

type ViewState int

const (
	New ViewState = iota
	Get
	Modify
	Summary
	Task
	Login
	DateSelect
	Act
	Confirmation
)

type SubState int

const (
	ListView SubState = iota
	NotesView
)

type uploadMsg int

type errMsg struct{ err error }

func uploadCmd(m *model, ents ...i.EntryRow) tea.Cmd {
	return func() tea.Msg {
		//This should now go to confirmation state and perform the required task once accepted
		if err := i.DoTaskSubmit(ents...); err != nil {
			m.errBuilder += err.Error()
			submitFailed = true
			return errMsg{err: err}
		}
		// if check event codes needs some interaction, dont go to get state.
		m.resetUpload()
		return uploadMsg(1)
	}
}

func initialModel() model {
	m := model{
		inputs:       make([]textinput.Model, i.Submit),
		modInputs:    make([]textinput.Model, i.Submit),
		loginInputs:  make([]textinput.Model, Cancel), // Only up to cancel since only two are inputs, rest are buttons.
		inputsPos:    make([]int, i.Submit),
		modInputsPos: make([]int, i.Submit),
		list:         list.Model{},
		state:        New,
		substate:     ListView,
		id:           0,
		currentDate:  time.Now(),
		startDate:    time.Time{},
		endDate:      time.Time{},
		resetTimer:   nil,
	}

	var t textinput.Model
	tt := time.Now()

	items := []list.Item{}
	m.ents = nil

	m.list = list.New(items, list.NewDefaultDelegate(), 0, 0)
	m.list.Title = "Worklog Entries"

	ti := textarea.New()
	ti.Placeholder = "Add notes here...."
	ti.CharLimit = 2000
	m.textarea = ti
	m.modtextarea = ti

	for j := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 32

		switch j {
		case i.Date:
			t.Placeholder = fmt.Sprintf("%v", tt.Format("02/01/2006"))
			t.EchoMode = textinput.EchoNormal
			t.Validate = dateValidator
			t.Focus()
			t.SetValue(fmt.Sprintf("%v", tt.Format("02/01/2006")))

		case i.Code:
			t.Placeholder = "Proj Code"
			t.CharLimit = 10

		case i.Desc:
			t.Placeholder = "Entry Desc"
			t.CharLimit = 500
			t.Width = 50

		case i.StartTime:
			t.Placeholder = "Start time: HH:MM"
			t.Validate = timeValidator
			t.CharLimit = 5

		case i.EndTime:
			t.Placeholder = fmt.Sprintf("%v", tt.Format("15:04"))
			t.Validate = timeValidator
			t.CharLimit = 5
			t.SetValue(fmt.Sprintf("%v", tt.Format("15:04")))

		case i.Hours:
			t.Placeholder = "Hours (opt) HH:MM"
			t.Validate = durValidator
			t.CharLimit = 6
		}
		m.inputs[j] = t
	}

	for j := range m.modInputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 32

		switch j {
		case i.Date:
			t.Placeholder = fmt.Sprintf("%v", tt.Format("02/01/2006"))
			t.EchoMode = textinput.EchoNormal
			t.Validate = dateValidator
			t.Focus()
			t.SetValue(fmt.Sprintf("%v", tt.Format("02/01/2006")))

		case i.Code:
			t.Placeholder = "Proj Code"
			t.CharLimit = 10

		case i.Desc:
			t.Placeholder = "Entry Desc"
			t.CharLimit = 500
			t.Width = 50

		case i.StartTime:
			t.Placeholder = "Start time: HH:MM"
			t.Validate = timeValidator
			t.CharLimit = 5

		case i.EndTime:
			t.Placeholder = fmt.Sprintf("%v", tt.Format("15:04"))
			t.Validate = timeValidator
			t.CharLimit = 5

		case i.Hours:
			t.Placeholder = "Hours (opt) XXhXXm"
			t.Validate = durValidator
			t.CharLimit = 6
		}
		m.modInputs[j] = t
	}

	for i := range m.loginInputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 50

		switch i {
		case Username:
			t.Placeholder = "Username/Email here"
			t.EchoMode = textinput.EchoNormal
			t.Focus()

		case Password:
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
	switch m.state {
	case Get:
		switch msg := msg.(type) {
		case errMsg:
			logger.Println(msg.err.Error())
			m.resetUpload()
			m.ents = nil
		case uploadMsg:
			logger.Println("Summary uploaded")
			m.resetUpload()
			m.ents = nil
		case tea.WindowSizeMsg:
			h, v := docStyle.GetFrameSize()
			m.winH = msg.Height - v
			m.winW = msg.Width - h
			m.list.SetSize(m.winW, m.winH)
			//fmt.Println("resize")

		case tea.KeyMsg:
			switch msg.String() {

			case "ctrl+p":
				if m.startDate.IsZero() && m.endDate.IsZero() {
					m.retState = Get
					m.startDate = m.currentDate
					m.endDate = m.currentDate
					m.state = DateSelect
					break
				}
				// log.Println(m.startDate.String(), m.endDate.String())
				m.currentDate = time.Now()
				ents, err := db.QuerySummary(&m.startDate, &m.endDate)
				//log.Println(ents)
				if err != nil {
					m.errBuilder = err.Error()
					m.startDate = time.Time{}
					m.endDate = time.Time{}
					return m, nil
				}
				if len(ents) == 0 {
					m.errBuilder = "No Entries this week"
					m.startDate = time.Time{}
					m.endDate = time.Time{}
					submitFailed = true
					m.state = Get
					return m, nil
				}
				date := ents[0].Entry.Date
				var dayTotal time.Duration
				duration := make(map[string]time.Duration)
				desc := make(map[string]string)
				first := true
				for i := 0; i < len(ents); i++ {
					if first {
						m.sumContent += summaryDateStyle.Render(ents[0].Entry.Date.Format("02/01/2006"))
						m.sumContent += "\n\n"
						first = false
					}
					if date != ents[i].Entry.Date {
						date = ents[i].Entry.Date
						for k, v := range duration {
							m.sumContent += summaryProjStyle.Render(fmt.Sprintf("Project: %s Hours: %02d:%02d\n", k, int(v.Hours()), int(v.Minutes())%60))
							m.sumContent += "\n"
							m.sumContent += desc[k] + "\n"
						}
						m.sumContent += summaryTotalStyle.Render(fmt.Sprintf("Total Hours in the Day: %02d:%02d\n", int(dayTotal.Hours()), int(dayTotal.Minutes())%60))
						m.sumContent += "\n"
						clear(desc)
						clear(duration)
						dayTotal, _ = time.ParseDuration("0s")
						m.sumContent += summaryDateStyle.Render(ents[i].Entry.Date.Format("02/01/2006"))
						m.sumContent += "\n\n"
					}
					duration[ents[i].Entry.ProjCode] += ents[i].Entry.Hours
					dayTotal += ents[i].Entry.Hours
					desc[ents[i].Entry.ProjCode] += ents[i].Entry.Desc + "\n"
				}
				// Flush last date data since loop will prematurely end
				for k, v := range duration {
					m.sumContent += summaryProjStyle.Render(fmt.Sprintf("Project: %s Hours: %02d:%02d\n", k, int(v.Hours()), int(v.Minutes())%60))
					m.sumContent += "\n"
					m.sumContent += desc[k] + "\n"
				}
				m.sumContent += "\n"
				m.sumContent += summaryTotalStyle.Render(fmt.Sprintf("Total Hours in the Day: %02d:%02d\n", int(dayTotal.Hours()), int(dayTotal.Minutes())%60))
				clear(desc)
				clear(duration)
				dayTotal, _ = time.ParseDuration("0s") // probs dont need this

				headerHeight := lipgloss.Height(m.headerView())
				footerHeight := lipgloss.Height(m.footerView())
				verticalMarginHeight := headerHeight + footerHeight + 4

				m.viewport = viewport.New(m.winW-3, m.winH-verticalMarginHeight)
				m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
				m.viewport.SetContent(m.sumContent)
				m.ready = true
				m.viewport.YPosition = headerHeight - 2

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
					item := items[m.list.Index()].(i.EntryRow)
					if err := db.DeleteEntry(item.EntryId); err != nil {
						logger.Println(err)
					}
					m.modRowID = 0
					m.id -= 1
					m.list.RemoveItem(m.list.Index())
					m.state = Get
				}

			case "enter":
				items := m.list.Items()
				item := items[m.list.Index()].(i.EntryRow)

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
				m.modInputs[i.Date].SetValue(item.Entry.Date.Format("02/01/2006"))
				m.modInputs[i.Code].SetValue(item.Entry.ProjCode)
				m.modInputs[i.Desc].SetValue(item.Entry.Desc)
				if !item.Entry.StartTime.IsZero() {
					m.modInputs[i.StartTime].SetValue(item.Entry.StartTime.Format("15:04"))
				}
				m.modInputs[i.EndTime].SetValue(item.Entry.EndTime.Format("15:04"))
				m.modInputs[i.Hours].SetValue(fmt.Sprintf("%02dh%02dm", int(item.Entry.Hours.Hours()), int(item.Entry.Hours.Minutes())%60))
				m.modRowID = item.EntryId
				m.modtextarea.SetValue(item.Entry.Notes)
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

			m.ListUpdate()
		}
		m.list, cmd = m.list.Update(msg)
	case Summary:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit

			case "tab":
				m.resetUpload()

			case "enter":
				var err error
				m.ents, err = db.QuerySummary(&m.startDate, &m.endDate)
				if err != nil {
					logger.Println(err)
					m.state = Get
					submitFailed = true
					m.errBuilder = "Submit Summary Failed"
					break
				}
				check := i.LoginGetTasks(&m.formLogged)
				if check {
					m.state = Login
					m.retState = Summary
					logger.Println("Login failed/need creds")
					break
				}
				m.retState = Summary
				ok, err := CheckEventCodeMap(&m, m.ents...)
				if err != nil {
					m.errBuilder += err.Error()
					submitFailed = true
					break
				}
				if ok {
					//cmd = confirmUpload
					m.state = Confirmation
				}
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

	case New:
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
						entry := i.EntryRow{}
						//Testing with local copy incase pointer edits data.
						if err := entry.FillData(m.inputs, &m.textarea); err != nil {
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
							m.list.Title = "worklog entries"
							submitFailed = false
							m.resetState()
							m.id = 0
							m.ListUpdate()
						}

					} else if s == "enter" && m.focusIndex == len(m.inputs)+1 {
						line, err := i.ImportWorklog(&db)
						if err != nil {
							submitFailed = true
							m.errBuilder = err.Error() + " " + strconv.Itoa(line)
						}
					} else if s == "enter" && m.focusIndex == len(m.inputs)+2 {
						err := db.QueryAndExport()
						if err != nil {
							logger.Println(err)
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
	case Modify:
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
						entry := i.EntryRow{}
						//Testing with local copy incase pointer edits data.
						if err := entry.ModFillData(m.modInputs, &m.modtextarea); err != nil {
							m.errBuilder = err.Error()
							submitFailed = true
							break
						}
						entry.EntryId = m.modRowID
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
						item := items[m.list.Index()].(i.EntryRow)
						if err := db.DeleteEntry(item.EntryId); err != nil {
							logger.Println(err)
						}
						m.modRowID = 0
						m.id -= 1
						m.resetModState()
						m.list.RemoveItem(m.list.Index())
						m.state = Get

					} else if s == "enter" && m.modFocusIndex == len(m.modInputs)+2 {
						// scoro upload
						entry := i.EntryRow{}
						if err := entry.ModFillData(m.modInputs, &m.modtextarea); err != nil {
							m.errBuilder = err.Error()
							submitFailed = true
							break
						}
						entry.EntryId = m.modRowID
						// Get user token
						check := i.LoginGetTasks(&m.formLogged)
						if check {
							m.state = Login
							m.retState = Modify
							logger.Println("Login failed/need creds")
							break
						}
						m.retState = Modify
						ok, err := CheckEventCodeMap(&m, entry)
						if err != nil {
							m.errBuilder += err.Error()
							submitFailed = true
							break
						}
						if ok {
							// Get user token
							if err := i.DoTaskSubmit(entry); err != nil {
								m.errBuilder += err.Error()
							}
							// if check event codes needs some interaction, dont go to get state.
							m.modRowID = 0
							m.state = Get
							m.retState = Get
							m.resetModState()
						}
					} else if s == "enter" && m.modFocusIndex == len(m.modInputs)+3 {
						entry := i.EntryRow{}
						if err := entry.ModFillData(m.modInputs, &m.modtextarea); err != nil {
							m.errBuilder = err.Error()
							submitFailed = true
							break
						}
						err := db.DeleteLink(entry.Entry.ProjCode)
						if err != nil {
							m.errBuilder = err.Error()
							submitFailed = true
							break
						}
						delete(i.ProjCodeToTask, entry.Entry.ProjCode)
						delete(i.ProjCodeToAct, entry.Entry.ProjCode)
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

	case DateSelect:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				m.state = m.retState
			case "ctrl+c":
				return m, tea.Quit
			case "tab": // Switch between start and end date
				m.selectStart = !m.selectStart
			case "left":
				if m.dateCursor > 0 {
					m.dateCursor--
				}
			case "right":
				if m.dateCursor < 2 {
					m.dateCursor++
				}
			case "up":
				if m.selectStart {
					m.startDate = adjustDate(m.startDate, m.dateCursor, 1)
				} else {
					m.endDate = adjustDate(m.endDate, m.dateCursor, 1)
				}
			case "down":
				if m.selectStart {
					m.startDate = adjustDate(m.startDate, m.dateCursor, -1)
				} else {
					m.endDate = adjustDate(m.endDate, m.dateCursor, -1)
				}
			}
		}
		return m, nil
	case Task:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.listTask.SetWidth(msg.Width)
			return m, nil

		case tea.KeyMsg:
			switch keypress := msg.String(); keypress {
			case "ctrl+c":
				return m, tea.Quit

			case "tab":
				m.resetUpload()
			case "enter":
				// The pick from the task will then go straight to the act choice
				item := m.listTask.SelectedItem()
				//logger.Println(item)
				db.AddToTaskMap(m.choice[m.index], item)
				m.index++
				if m.index < len(m.choice) {
					items := i.TaskList.ConstructTaskList()
					m.listTask = list.New(items, list.NewDefaultDelegate(), 0, 0)
					m.listTask.Title = fmt.Sprintf("Choose a task for %s", m.choice[m.index])
					m.listTask.SetSize(m.winW, m.winH)
					break
				}
				m.index = 0
				// Go to activity choice now.
				m.state = Act
			}
		}
		m.listTask, cmd = m.listTask.Update(msg)
	case Act:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.listAct.SetWidth(msg.Width)
			return m, nil

		case tea.KeyMsg:
			switch keypress := msg.String(); keypress {
			case "ctrl+c":
				return m, tea.Quit

			case "tab":
				m.resetUpload()

			case "enter":
				// If from modify go back to modify
				// If from summary go back to summary
				// Loop through every task that needs linking before returning.
				item := m.listAct.SelectedItem()
				db.AddToActMap(m.choice[m.actIndex], item)
				if m.actIndex != len(m.choice)-1 {
					items := i.ActResp.ConstructActList()
					m.listAct = list.New(items, list.NewDefaultDelegate(), 0, 0)
					m.listAct.Title = fmt.Sprintf("Choose an activity for %s", m.choice[m.actIndex])
					m.listAct.SetSize(m.winW, m.winH)
					m.actIndex++
					break
				}
				m.actIndex = 0
				if len(m.choice) > 1 {
					m.state = Summary
					m.choice = nil
				} else {
					m.state = Modify
					m.choice = nil
				}
			}
		}
		m.listAct, cmd = m.listAct.Update(msg)
	case Confirmation:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			h, v := docStyle.GetFrameSize()
			m.winH = msg.Height - v
			m.winW = msg.Width - h
			return m, nil

		case tea.KeyMsg:
			switch key := msg.String(); key {
			case "ctrl+c":
				return m, tea.Quit

			case "up", "down", "left", "right", "enter":
				if key == "enter" {
					if m.confirmationIndex == 0 {
						m.state = Get
						cmd = uploadCmd(&m, m.ents...)
						m.resetUpload()
					} else {
						m.resetUpload()
					}
				}
				// Cycle indexes
				if key == "up" || key == "left" {
					m.confirmationIndex--
				} else {
					m.confirmationIndex++
				}

				if m.confirmationIndex > 1 { // only two options, will hardcode
					m.confirmationIndex = 0
				} else if m.confirmationIndex < 0 {
					m.confirmationIndex = 1
				}
			}
		}

	case Login:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			h, v := docStyle.GetFrameSize()
			m.winH = msg.Height - v
			m.winW = msg.Width - h
			return m, nil

		case tea.KeyMsg:
			switch keypress := msg.String(); keypress {
			case "ctrl+c":
				return m, tea.Quit

			case "enter", "up", "down", "left", "right": // Once a task is selected go back to modify view
				if keypress == "enter" && m.loginFocusIndex == len(m.loginInputs) {
					if err := i.LoginGetTaskForm(&m.formLogged, m.loginInputs[Username].Value(), m.loginInputs[Password].Value()); err != nil {
						m.errBuilder = "Login Failed try again"
						submitFailed = true
						break
					}
					// Start periodic timeReset after successful login
					if m.resetTimer != nil {
						m.resetTimer.Stop()
					}
					m.resetTimer = time.NewTimer(12 * time.Hour) // Reset every 12 hours
					go func() {
						for range m.resetTimer.C {
							m.timeReset()
							m.resetTimer.Reset(12 * time.Hour)
						}
					}()
					m.state = m.retState
				} else if keypress == "enter" && m.loginFocusIndex == len(m.loginInputs)+1 {
					m.resetLoginState()
					m.state = m.retState
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
	case Act:
		_, err := b.WriteString(docStyle.Render(m.listAct.View()))
		if err != nil {
			b.WriteString(fmt.Sprintf("%v", err))
		}

	case Task:
		_, err := b.WriteString(docStyle.Render(m.listTask.View()))
		if err != nil {
			b.WriteString(fmt.Sprintf("%v", err))
		}
	case DateSelect:
		startView := fmt.Sprintf("Start Date: %s", highlightField(m.startDate, m.dateCursor, m.selectStart))
		endView := fmt.Sprintf("End Date:   %s", highlightField(m.endDate, m.dateCursor, !m.selectStart))

		instructions := "Use arrow keys to adjust, Tab to switch between start/end dates, ctrl+c to quit."

		b.WriteString(fmt.Sprintf("%s\n%s\n\n%s", startView, endView, instructions))
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
	case Confirmation:
		button1 := blurConfirm
		if m.confirmationIndex == 0 {
			button1 = focusConfirm
		}
		button2 := blurCancel
		if m.confirmationIndex == 1 {
			button2 = focusCancel
		}
		b.WriteString(helpStyle.Render(fmt.Sprintf("\n\t\t\tDo you want to continue?\n\n\t\t%s\t\t  %s\n", button1, button2)))
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
	m.inputs[i.Date].SetValue(fmt.Sprintf("%v", t.Format("02/01/2006")))
	m.inputs[i.EndTime].SetValue(fmt.Sprintf("%v", t.Format("15:04")))
	m.inputsPos[i.Date] = len(m.inputs[i.Date].Value())
	m.inputsPos[i.EndTime] = len(m.inputs[i.EndTime].Value())
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

var db i.Database = i.Database{Db: nil}

func main() {
	// Logger for dev
	f, err := os.OpenFile("testlogfile.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	logger = &log.Logger{}
	if err != nil {
		logger.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	logger.SetFlags(log.LstdFlags | log.Lshortfile)
	logger.SetOutput(f)
	i.SetLogger(logger)

	if err := db.OpenDatabase(nil); err != nil {
		logger.Println(err)
	}

	// Get the saved projevent links, errs will return empty map, system can still run.
	i.ProjCodeToTask, i.ProjCodeToAct, _, err = db.QueryLinks()
	if err != nil {
		logger.Println(err)
	}
	err = godotenv.Load("user.env")
	if err != nil {
		logger.Println("Error loading user.env file")
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		logger.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	db.CloseDatabase()
}

// I think this will only work for one entry, rethink logic for multi submission.
func CheckEventCodeMap(m *model, entries ...i.EntryRow) (bool, error) {
	// Check to see if we have all proj codes mapped to an event_id
	check := true
	added := make(map[string]bool, 5)
	for j := 0; j < len(entries); j++ {
		_, ok := i.ProjCodeToTask[entries[j].Entry.ProjCode]
		//_, ok2 := i.ProjCodeToAct[entries[j].Entry.ProjCode]
		// Add all entries that need linking
		// the second ok check was making it so that submits failed unless all tasks had an activity.
		if !ok && !added[entries[j].Entry.ProjCode] {
			added[entries[j].Entry.ProjCode] = true
			check = false
			items := i.TaskList.ConstructTaskList()
			if len(items) == 0 {
				m.state = Login
				return false, fmt.Errorf("bad login")
			}
			m.choice = append(m.choice, entries[j].Entry.ProjCode)
			m.listTask = list.New(items, list.NewDefaultDelegate(), 0, 0)
			m.listTask.Title = fmt.Sprintf("Choose a task for %s", m.choice[0])
			m.state = Task
			m.listTask.SetSize(m.winW, m.winH)

			actitems := i.ActResp.ConstructActList()
			m.listAct = list.New(actitems, list.NewDefaultDelegate(), 0, 0)
			m.listAct.Title = fmt.Sprintf("Choose an activity for %s", entries[j].Entry.ProjCode)
			m.listAct.SetSize(m.winW, m.winH)
		}
	}
	return check, nil
}

func (m *model) ListUpdate() error {
	// Stop resetting the list. Append and keep index tracked.
	e, err := db.QueryEntries(&m.id, &m.maxId)
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

func highlightField(date time.Time, cursor int, selected bool) string {
	year := fmt.Sprintf("%04d", date.Year())
	month := fmt.Sprintf("%02d", date.Month())
	day := fmt.Sprintf("%02d", date.Day())

	switch cursor {
	case 0: // Highlight year
		if selected {
			return focusedStyle.Render(fmt.Sprintf("[%s]-%s-%s", year, month, day))
		}
	case 1: // Highlight month
		if selected {
			return focusedStyle.Render(fmt.Sprintf("%s-[%s]-%s", year, month, day))
		}
	case 2: // Highlight day
		if selected {
			return focusedStyle.Render(fmt.Sprintf("%s-%s-[%s]", year, month, day))
		}
	}
	return fmt.Sprintf("%s-%s-%s", year, month, day)
}

func adjustDate(date time.Time, cursor int, delta int) time.Time {
	switch cursor {
	case 0: // Year
		return date.AddDate(delta, 0, 0)
	case 1: // Month
		return date.AddDate(0, delta, 0)
	case 2: // Day
		return date.AddDate(0, 0, delta)
	}
	return date
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

func (m *model) resetUpload() { // used to reset all variable that are used when uploading from summary view.
	m.state = Get
	m.retState = Get
	m.sumContent = ""
	m.viewport.SetContent(m.sumContent)
	m.startDate = time.Time{}
	m.endDate = time.Time{}
	// this will reset the list of selected upload items.
	m.choice = nil
}

// Adding this functionality to fix issues with new buckets at the end/beginning of a new month.
// Does running this in a seperate goroutine line up issues for race conditions? Probably
// should this just be called any time the user logs in? since that would be a fresh fetch?
// What if the user has had the program open for a while.
// Add a bool to determine if it should.
func (m *model) timeReset() {
	t := time.Now()
	year, month, _ := t.Date()
	lastDay := time.Date(year, month, 0, 0, 0, 0, 0, t.Location())
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
	// if the day matches the first day of the current month
	taskmap, _, updateFlag, err := db.QueryLinks()
	update := false
	// check we haven't already done this today. can use any since they all match.
	// TODO: rewrite if we can have some rows as true and false at the same time.
	for _, val := range updateFlag {
		update = val
		break
	}
	if t.Day() == firstDay.Day() && update {
		// reset task and act list and regrab.
		// if form logged is set this means we have a user_token from scoro, dont know how long this lasts, assume we are good.
		cont := i.LoginGetTasks(&m.formLogged)
		//TODO: review this: if the return is true this means we aren't logged, and we have no creds saved.
		// we would need to prompt the user for these which wont work if this happens when they are away.
		// at that point we ignore and use what list we have.
		if !cont {
			i.RefetchLists(&m.formLogged)
			i.ActResp = i.ActivityResp{}
			i.TaskList = i.TaskListResp{}
		}

		// check if any of the tasks that are linked are no longer in the task list
		// if yes, delete the links.
		//taskmap, _, _, err := db.QueryLinks()
		if err != nil {
			logger.Println("query link fail")
		}
		// Keep track of project codes that should have update flag set to false
		keptProjCodes := make([]string, 0)
		for k, v := range taskmap {
			contained := false
			for _, vv := range i.TaskList.Data {
				if v == vv.EventID {
					// This would mean that the task is in this month task list for the user
					// Probably means that the scoro bucket hasnt changed, this should not be unlinked so that code isnt auto uploaded to the wrong one.
					contained = true
					keptProjCodes = append(keptProjCodes, k)
					break
				}
			}
			if !contained {
				db.DeleteLink(k)
			}
		}
		// Set update flag to false for all kept links
		if err := db.SetUpdateFlagFalse(keptProjCodes); err != nil {
			logger.Println("Failed to set update flag to false:", err)
		}
		// if the day matches the last day of this month.
	} else if t.Day() == lastDay.Day() {
		// Set update flag to true for all links on last day of month
		if err := db.SetUpdateFlag(); err != nil {
			logger.Println("Failed to set update flag:", err)
		}
	}
}
