package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type offlineTransport struct{}

func (offlineTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("font no disponible durant la prova")
}

func TestCollectOfflineDoesNotPanic(t *testing.T) {
	previous := client
	client = &http.Client{Transport: offlineTransport{}}
	t.Cleanup(func() { client = previous })
	var report bytes.Buffer
	if err := collect(&report, true); err == nil {
		t.Fatal("Cal indicar que les fonts no estan disponibles")
	}
	if !strings.Contains(report.String(), "Barcelona Sants ↔ Girona") {
		t.Fatal("Cal mostrar el resultat parcial sense fallar amb un pànic")
	}
}

func TestCorridorRequiresBothStops(t *testing.T) {
	for _, tc := range []struct {
		stops map[string]int
		want  bool
	}{
		{map[string]int{"71801": 1, "79300": 3}, true},
		{map[string]int{"71801": 3, "79300": 1}, true},
		{map[string]int{"71801": 1}, false},
	} {
		if got := corridor(&Trip{Stops: tc.stops}); got != tc.want {
			t.Errorf("corridor(%v)=%v", tc.stops, got)
		}
	}
}

func TestMissingDestinationDelayIsUnknown(t *testing.T) {
	zero := 0
	u := Update{Delay: &zero}
	if delay(&u, "79300") != nil {
		t.Fatal("No s'ha d'inventar un retard a destinació")
	}
}

func TestAlertPeriods(t *testing.T) {
	a := Alert{}
	if !active(&a, 100) {
		t.Fatal("Sense períodes, l'avís és actiu")
	}
	a.Active = append(a.Active, struct {
		Start string `json:"start"`
		End   string `json:"end"`
	}{"100", "200"})
	if active(&a, 99) || !active(&a, 100) || active(&a, 200) {
		t.Fatal("Límits de vigència incorrectes")
	}
}
