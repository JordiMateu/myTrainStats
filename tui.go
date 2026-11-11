package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("221"))
	dangerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
)

var version = "dev"

type loadedMsg struct {
	report string
	err    error
}

type mascotTickMsg struct{}

func mascotTick() tea.Cmd {
	return tea.Tick(450*time.Millisecond, func(time.Time) tea.Msg { return mascotTickMsg{} })
}

type dashboard struct {
	viewport      viewport.Model
	spinner       spinner.Model
	loading       bool
	refresh       bool
	report        string
	err           error
	width, height int
	frame         int
	animate       bool
}

func loadReport(refresh bool) tea.Cmd {
	return func() tea.Msg {
		var b bytes.Buffer
		err := collect(&b, refresh)
		return loadedMsg{b.String(), err}
	}
}

func newDashboard(refresh bool) dashboard {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = titleStyle
	return dashboard{viewport: viewport.New(80, 16), spinner: s, loading: true, refresh: refresh, width: 84, height: 24, animate: true}
}

func (m dashboard) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadReport(m.refresh), mascotTick())
}

func (m dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case mascotTickMsg:
		if m.animate {
			m.frame++
		}
		return m, mascotTick()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = max(1, m.width-4)
		m.viewport.Height = max(1, m.height-9)
		m.viewport.SetContent(renderReport(m.report, m.viewport.Width))
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "a":
			m.animate = !m.animate
			return m, nil
		case "r":
			if !m.loading {
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, loadReport(false))
			}
		}
	case loadedMsg:
		m.loading = false
		m.report = msg.report
		m.err = msg.err
		m.viewport.SetContent(renderReport(m.report, m.viewport.Width))
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.loading {
			return m, cmd
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func mascotArt(frame int) string {
	cat := []string{" .------. /\\_/\\", "~|::::::|( o.o)", " '------'  \" \" "}
	if frame%2 != 0 {
		cat[1] = "-|::::::|( o.o)"
		cat[2] = " '------' \"   \""
	}
	colors := []string{"203", "221", "81"}
	trailFrame := []string{"====", "~===", "=~==", "==~="}[frame%4]
	rows := make([]string, 3)
	for i := range rows {
		trail := lipgloss.NewStyle().Foreground(lipgloss.Color(colors[i])).Render(trailFrame)
		rows[i] = trail + lipgloss.NewStyle().Foreground(lipgloss.Color("218")).Render(cat[i])
	}
	return strings.Join(rows, "\n")
}

func renderReport(report string, width int) string {
	var b strings.Builder
	for _, line := range strings.Split(report, "\n") {
		style := lipgloss.NewStyle()
		switch {
		case strings.Contains(line, " | "):
			fields := strings.SplitN(line, " | ", 4)
			if len(fields) == 4 {
				badge := sectionStyle.Render(fields[1])
				statusStyle := warningStyle
				if strings.Contains(fields[3], "CANCEL·LAT") {
					statusStyle = dangerStyle
				}
				line = fields[0] + "  ·  " + badge + "  ·  " + fields[2] + "\n  " + statusStyle.Render(fields[3])
			}
		case strings.HasPrefix(line, "Temps real"), strings.HasPrefix(line, "Avisos oficials"), strings.HasPrefix(line, "Limitacions Adif"), strings.HasPrefix(line, "R11 ·"):
			style = sectionStyle
		case strings.HasPrefix(line, "AMB AFECTACIONS"), strings.HasPrefix(line, "SENSE DADES"), strings.HasPrefix(line, "TEMPS REAL NO DISPONIBLE"):
			style = warningStyle.Bold(true)
		case strings.Contains(line, "NO DISPONIBLE"), strings.Contains(line, "DADES ANTIGUES"):
			style = dangerStyle
		case strings.HasPrefix(line, "- "):
			style = warningStyle
		default:
			style = mutedStyle
		}
		b.WriteString(style.Width(max(1, width)).Render(line))
		b.WriteByte('\n')
	}
	return b.String()
}

func (m dashboard) View() string {
	if m.width < 24 || m.height < 12 {
		return "Amplia el terminal (mínim 24 × 12).\nq · sortir"
	}
	status := "Dades consultades · r per actualitzar"
	if m.loading {
		status = m.spinner.View() + " Consultant Renfe i Adif…"
	} else if m.err != nil {
		status = dangerStyle.Render("Consulta parcial · alguna font no està disponible")
	}
	header := titleStyle.Render("RAIL / GIRONA") + "\n" + lipgloss.NewStyle().Bold(true).Render("Barcelona Sants ↔ Girona") + "\n" + ansi.Truncate(status, m.width-4, "…")
	if m.width >= 64 {
		leftWidth := m.width - 4 - 25
		left := titleStyle.Render("RAIL / GIRONA") + "\n" + lipgloss.NewStyle().Bold(true).Render("Barcelona Sants ↔ Girona") + "\n" + ansi.Truncate(status, leftWidth, "…")
		header = lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(leftWidth).Render(left), "  ", mascotArt(m.frame))
	}
	help := "↑↓ moure · r dades · q sortir"
	if m.width >= 72 {
		help = fmt.Sprintf("↑↓ moure · r dades · a animació · q sortir · %.0f%%", m.viewport.ScrollPercent()*100)
	}
	footer := mutedStyle.Render(ansi.Truncate(help, m.width-4, "…"))
	return lipgloss.NewStyle().Padding(1, 2).Render(header + "\n\n" + m.viewport.View() + "\n\n" + footer)
}

func run() error {
	refresh := flag.Bool("refresh", false, "Torna a descarregar els horaris")
	plain := flag.Bool("plain", false, "Sortida de text sense interfície interactiva")
	showVersion := flag.Bool("version", false, "Mostra la versió")
	flag.Parse()
	if *showVersion {
		fmt.Println("myTrainStats " + version)
		return nil
	}
	if *plain || !term.IsTerminal(os.Stdout.Fd()) || !term.IsTerminal(os.Stdin.Fd()) {
		return collect(os.Stdout, *refresh)
	}
	_, err := tea.NewProgram(newDashboard(*refresh), tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
