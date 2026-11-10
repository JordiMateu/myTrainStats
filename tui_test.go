package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestDashboardRefreshAndQuit(t *testing.T) {
	m := newDashboard(false)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd != nil {
		t.Fatal("No ha de duplicar una consulta en curs")
	}
	model, _ := m.Update(loadedMsg{report: "Temps real ld\nSants → Girona | AVE | 03163 | CANCEL·LAT"})
	m = model.(dashboard)
	if m.loading {
		t.Fatal("La consulta ha acabat")
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("r ha de consultar les fonts")
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q ha de sortir")
	}
}

func TestDashboardFitsTerminal(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 64, Height: 16}, {Width: 32, Height: 16}} {
		m := newDashboard(false)
		model, _ := m.Update(size)
		m = model.(dashboard)
		model, _ = m.Update(loadedMsg{report: strings.Repeat("- Una incidència amb un text llarg que ha de quedar ajustat a l'amplada del terminal.\n", 30)})
		m = model.(dashboard)
		view := m.View()
		for _, line := range strings.Split(view, "\n") {
			if ansi.StringWidth(line) > size.Width {
				t.Fatalf("Fila massa ampla (%d): %q", size.Width, line)
			}
		}
		if len(strings.Split(view, "\n")) > size.Height {
			t.Fatal("La vista excedeix l'alçada")
		}
	}
}
