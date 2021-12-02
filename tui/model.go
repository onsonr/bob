package tui

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Benchkram/bob/pkg/ctl"
	"github.com/Benchkram/errz"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/logrusorgru/aurora"
	"github.com/xlab/treeprint"
)

func init() {
	treeprint.EdgeTypeLink = "│"
	treeprint.EdgeTypeMid = "├"
	treeprint.EdgeTypeEnd = "└"
	treeprint.IndentSize = 2
}

type keyMap struct {
	NextTab      key.Binding
	FollowOutput key.Binding
	Restart      key.Binding
	Quit         key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Restart, k.NextTab, k.FollowOutput, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Restart, k.NextTab, k.FollowOutput, k.Quit},
	}
}

var keys = keyMap{
	NextTab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("[TAB]", "next tab"),
	),
	FollowOutput: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("[ESC]", "follow output"),
	),
	Restart: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("[^R]", "restart task"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("[^C]", "quit"),
	),
}

type model struct {
	stdout     *os.File
	keys       keyMap
	events     chan interface{}
	cmder      ctl.Commander
	tabs       []tab
	currentTab int
	starting   bool
	restarting bool
	stopping   bool
	width      int
	height     int
	content    viewport.Model
	header     viewport.Model
	footer     help.Model
}

type tab struct {
	name   string
	output *bytes.Buffer
}

func newModel(cmder ctl.Commander, evts chan interface{}, output *os.File, actualOutput *os.File) *model {
	tabs := []tab{}

	buf := new(bytes.Buffer)

	tabs = append(tabs, tab{
		name:   "status",
		output: buf,
	})

	sglobal := bufio.NewScanner(output)
	sglobal.Split(bufio.ScanRunes)

	go func() {
		for sglobal.Scan() {
			err := sglobal.Err()
			if err != nil {
				return
			}

			_, err = buf.Write(sglobal.Bytes())
			if err != nil {
				// TODO: error handling
				fmt.Println(err)
			}

			evts <- Update{}
		}
	}()

	for _, cmd := range cmder.Subcommands() {
		buf := new(bytes.Buffer)

		tabs = append(tabs, tab{
			name:   cmd.Name(),
			output: buf,
		})

		sout := bufio.NewScanner(cmd.Stdout())
		sout.Split(bufio.ScanRunes)

		serr := bufio.NewScanner(cmd.Stderr())
		serr.Split(bufio.ScanRunes)

		go func() {
			for sout.Scan() {
				err := sglobal.Err()
				if err != nil {
					return
				}

				_, err = buf.Write(sout.Bytes())
				if err != nil {
					// TODO: error handling
					fmt.Println(err)
				}
				evts <- Update{}
			}
		}()

		go func() {
			for serr.Scan() {
				err := sglobal.Err()
				if err != nil {
					return
				}

				_, err = buf.Write(serr.Bytes())
				if err != nil {
					// TODO: error handling
					fmt.Println(err)
				}
				evts <- Update{}
			}
		}()
	}

	return &model{
		stdout:     actualOutput,
		cmder:      cmder,
		currentTab: 0,
		tabs:       tabs,
		events:     evts,
		keys:       keys,
		footer: help.Model{
			ShowAll:        false,
			ShortSeparator: " · ",
			FullSeparator:  "",
			Ellipsis:       "...",
			Styles: help.Styles{
				ShortKey:  lipgloss.NewStyle().Foreground(lipgloss.Color("#bbb")),
				ShortDesc: lipgloss.NewStyle().Foreground(lipgloss.Color("#999")),
			},
		},
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		tick(), // enable for more consistent rendering
		start(m),
		nextEvent(m.events),
	)
}

// TODO: becomes EXTREMELY slow real quick. Solution: refactor TUI's buffer so that it only wraps visible viewport,
// a.k.a. use bubbletea viewport's high performance renderer.
//func (m *model) softWrappedOutput() string {
//	output := m.tabs[m.currentTab].output.String()
//
//	wrapped := new(bytes.Buffer)
//
//	for _, s := range strings.Split(output, "\n") {
//		ws := wordwrap.WrapString(s, uint(m.width))
//
//		wrapped.WriteString(ws)
//		wrapped.WriteString("\n")
//	}
//
//	return wrapped.String()
//}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// first detect if the user has reached the end of the output, if they did then resume scrolling
	// if we query this after content is updated it's buggy
	follow := m.content.AtBottom()

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.NextTab):
			m.currentTab = (m.currentTab + 1) % len(m.tabs)

			m.content.SetContent(m.tabs[m.currentTab].output.String())
			m.content.GotoBottom()

		case key.Matches(msg, m.keys.FollowOutput):
			m.content.GotoBottom()

		case key.Matches(msg, m.keys.Restart):
			m.content.GotoBottom()

			cmds = append(cmds, restart(m))

		case key.Matches(msg, m.keys.Quit):
			m.content.GotoBottom()

			cmds = append(cmds, stop(m))
		}

	case time.Time:
		cmds = append(cmds, tick())

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.updateViewports()

		m.content.SetContent(m.tabs[m.currentTab].output.String())

	case Quit:
		cmds = append(cmds, tea.Quit)

	case Started:
		m.starting = false

	case Restarted:
		m.restarting = false

	case Update:
		// trigger an update of the viewport from whatever tab buffer is currently active
		m.content.SetContent(m.tabs[m.currentTab].output.String())
		// listen for the next update event
		cmds = append(cmds, nextEvent(m.events))
	}

	var status string
	if m.starting {
		status = fmt.Sprintf("%-*s", 10, "starting")
		status = aurora.Colorize(status, aurora.BlueFg|aurora.BoldFm).String()
	} else if m.restarting {
		status = fmt.Sprintf("%-*s", 10, "restarting")
		status = aurora.Colorize(status, aurora.CyanFg|aurora.BoldFm).String()
	} else if m.stopping {
		status = fmt.Sprintf("%-*s", 10, "stopping")
		status = aurora.Colorize(status, aurora.RedFg|aurora.BoldFm).String()
	} else {
		status = fmt.Sprintf("%-*s", 10, "running")
		status = aurora.Colorize(status, aurora.GreenFg|aurora.BoldFm).String()
	}

	tabs := make([]string, len(m.tabs))

	for i, tab := range m.tabs {
		var name string
		if i == m.currentTab {
			name = aurora.Colorize(tab.name, aurora.BoldFm).String()
		} else {
			name = aurora.Colorize(tab.name, aurora.WhiteFg).String()
		}

		tabs[i] = fmt.Sprintf("[%s]", name)
	}

	tabsView := strings.Join(tabs, " ")

	m.header.SetContent(fmt.Sprintf("%s  %s", status, tabsView))

	if follow {
		m.content.GotoBottom()
	}

	m.header, _ = m.header.Update(msg)
	m.content, _ = m.content.Update(msg)

	return m, tea.Batch(cmds...)
}

func (m *model) updateViewports() {
	if m.width == 0 || m.height == 0 {
		return
	}

	m.header.Width = m.width
	m.content.Width = m.width
	m.footer.Width = m.width

	m.header.Height = 2
	m.content.Height = m.height - 4
}

func (m *model) View() string {
	var view strings.Builder

	view.WriteString(m.header.View())
	view.WriteString("\n")
	view.WriteString(m.content.View())
	view.WriteString("\n\n")
	view.WriteString(m.footer.View(m.keys))

	return view.String()
}

func tick() tea.Cmd {
	return tea.Tick(
		100*time.Millisecond, func(t time.Time) tea.Msg {
			return t
		},
	)
}

func start(m *model) tea.Cmd {
	m.starting = true

	return func() tea.Msg {
		err := m.cmder.Start()
		// TODO: error handling
		errz.Log(err)

		return Started{}
	}
}

func restart(m *model) tea.Cmd {
	m.restarting = true

	return func() tea.Msg {
		err := m.cmder.Restart()
		if err != nil {
			// TODO: error handling
			fmt.Println(err)
		}

		return Restarted{}
	}
}

func stop(m *model) tea.Cmd {
	m.stopping = true

	return func() tea.Msg {
		err := m.cmder.Stop()
		if err != nil {
			// TODO: error handling
			fmt.Println(err)
		}

		return Quit{}
	}
}

func nextEvent(evts chan interface{}) tea.Cmd {
	return func() tea.Msg {
		return <-evts
	}
}

//func fmtCmds(cmds []ctl.Command) string {
//	out := ""
//	for _, cmd := range cmds {
//		status := fmtCmd(cmd)
//		out += fmt.Sprintf("- %s\n", status)
//	}
//	return out
//}

//func fmtCmdTree(tree ctl.CommandTree) string {
//	root := treeprint.New()
//
//	status := fmtCmd(tree)
//	root.SetValue(status)
//
//	for _, cmd := range tree.Subcommands() {
//		status := fmtCmd(cmd)
//		root.AddNode(status)
//	}
//
//	return root.String()
//}

//func fmtCmd(cmd ctl.Command) string {
//	running := aurora.Colorize("stopped", aurora.RedFg).String()
//	if cmd.Running() {
//		running = aurora.Colorize("running", aurora.GreenFg).String()
//	}
//
//	return fmt.Sprintf("%s (%s)", cmd.Name(), running)
//}