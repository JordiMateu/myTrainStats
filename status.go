package main

import (
	"fmt"
	"strconv"
	"strings"
)

func departureLabel(t *Trip) string {
	origin := "71801"
	if t.Stops["71801"] > t.Stops["79300"] {
		origin = "79300"
	}

	raw := t.Departures[origin]

	parts := strings.Split(raw, ":")
	if len(parts) != 3 {
		return "desconeguda"
	}

	h, e := strconv.Atoi(parts[0])
	m, e2 := strconv.Atoi(parts[1])

	sec, e3 := strconv.Atoi(parts[2])
	if e != nil || e2 != nil || e3 != nil || h < 0 || m < 0 || m > 59 || sec < 0 || sec > 59 {
		return "desconeguda"
	}

	label := fmt.Sprintf("%02d:%02d", h%24, m)
	if h >= 24 {
		label += fmt.Sprintf(" (+%d dia)", h/24)
	}

	return label
}

type lineState struct {
	available, stale, alertsUnavailable          bool
	timestamp                                    string
	trains, canceled, delayed, measured, unknown int
	maxDelay                                     int
	alerts                                       []string
}

// Counts the whole R11 feed, including trips outside the Sants–Girona segment.
func (s *lineState) measure(schedule Schedule, feed Feed) {
	seen := map[string]bool{}

	for _, e := range feed.Entities {
		u := e.Update
		if u == nil || seen[u.Trip.ID] {
			continue
		}

		t := schedule.Trips[u.Trip.ID]
		if t == nil || schedule.Routes[t.Route] != "R11" {
			continue
		}

		seen[u.Trip.ID] = true

		s.trains++
		if u.Trip.Relationship == "CANCELED" {
			s.canceled++
			continue
		}

		d := u.Delay
		// Without a trip-level delay, summarize the largest published stop delay.
		if d == nil {
			for _, stop := range u.Stops {
				for _, value := range []*int{stop.Arrival.Delay, stop.Departure.Delay} {
					if value != nil && (d == nil || *value > *d) {
						d = value
					}
				}
			}
		}

		if d == nil {
			s.unknown++
			continue
		}

		s.measured++
		if *d > 0 {
			s.delayed++
		}

		if *d > s.maxDelay {
			s.maxDelay = *d
		}
	}
}

func (s lineState) summary() string {
	state := "SENSE DADES SUFICIENTS"

	switch {
	case !s.available:
		state = "TEMPS REAL NO DISPONIBLE"
	case s.stale:
		state = "DADES ANTIGUES · estat actual desconegut"
	case s.canceled > 0 || s.delayed > 0 || len(s.alerts) > 0:
		state = "AMB AFECTACIONS PUBLICADES"
	case s.measured > 0:
		state = "SENSE RETARDS EN LES DADES REBUDES"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "R11 · ESTAT DE LA LÍNIA\n%s\n", state)

	if s.available {
		fmt.Fprintf(&b, "Font: %s · %d trajectes informats · %d cancel·lats · %d amb retard · %d sense retard mesurable\n", s.timestamp, s.trains, s.canceled, s.delayed, s.unknown)

		if s.measured > 0 {
			fmt.Fprintf(&b, "Retard màxim publicat: %.1f min\n", float64(s.maxDelay)/60)
		}
	}

	for _, a := range s.alerts {
		fmt.Fprintln(&b, "- Avís R11: "+a)
	}

	if s.alertsUnavailable {
		fmt.Fprintln(&b, "Avisos R11: NO DISPONIBLES")
	} else if !s.available {
		fmt.Fprintln(&b, "No es poden relacionar tots els avisos amb l'R11 sense els horaris.")
	}

	fmt.Fprintln(&b, "Resum calculat de tota l'R11, no un estat oficial ni només del tram Sants–Girona. Cancel·lacions i retards del feed poden incloure serveis d'altres hores. Cobertura parcial; absència de dades no implica normalitat.")

	return b.String()
}
