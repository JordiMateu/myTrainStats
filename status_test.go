package main

import (
	"strings"
	"testing"
)

func TestDepartureUsesJourneyOrigin(t *testing.T) {
	trip := &Trip{Stops: map[string]int{"71801": 1, "79300": 5}, Departures: map[string]string{"71801": "8:05:00", "79300": "25:12:00"}}
	if got := departureLabel(trip); got != "08:05" {
		t.Fatal(got)
	}

	trip.Stops["71801"] = 6
	if got := departureLabel(trip); got != "01:12 (+1 dia)" {
		t.Fatal(got)
	}

	delete(trip.Departures, "79300")

	if got := departureLabel(trip); got != "desconeguda" {
		t.Fatal(got)
	}
}

func TestR11CountsWholeLineAndDeduplicates(t *testing.T) {
	schedule := Schedule{Routes: map[string]string{"r": "R11", "other": "R1"}, Trips: map[string]*Trip{"a": {Route: "r"}, "b": {Route: "r"}, "c": {Route: "r"}, "d": {Route: "other"}}}
	f := Feed{}

	for _, id := range []string{"a", "a", "b", "c", "d"} {
		u := &Update{}

		u.Trip.ID = id
		if id == "a" {
			d := 360
			u.Delay = &d
		}

		if id == "b" {
			u.Trip.Relationship = "CANCELED"
		}

		f.Entities = append(f.Entities, struct {
			Update *Update `json:"tripUpdate"`
			Alert  *Alert  `json:"alert"`
		}{Update: u})
	}

	s := lineState{available: true}
	s.measure(schedule, f)

	if s.trains != 3 || s.delayed != 1 || s.canceled != 1 || s.unknown != 1 || s.maxDelay != 360 {
		t.Fatalf("%+v", s)
	}

	if !strings.Contains(s.summary(), "AMB AFECTACIONS") {
		t.Fatal(s.summary())
	}

	s.stale = true
	if !strings.Contains(s.summary(), "estat actual desconegut") {
		t.Fatal(s.summary())
	}
}

func TestEmptyR11IsNotNormal(t *testing.T) {
	s := lineState{available: true}
	if strings.Contains(s.summary(), "SENSE RETARDS") {
		t.Fatal(s.summary())
	}
}
