package main

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"
)

type Event struct {
	Delay *int `json:"delay"`
}

type Update struct {
	Trip struct {
		ID           string `json:"tripId"`
		Relationship string `json:"scheduleRelationship"`
	} `json:"trip"`
	Delay     *int   `json:"delay"`
	Timestamp string `json:"timestamp"`
	Stops     []struct {
		ID        string `json:"stopId"`
		Arrival   Event  `json:"arrival"`
		Departure Event  `json:"departure"`
	} `json:"stopTimeUpdate"`
}

type Text struct {
	Translation []struct {
		Text     string `json:"text"`
		Language string `json:"language"`
	} `json:"translation"`
}

type Alert struct {
	Active []struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"activePeriod"`
	Informed []struct {
		Route string `json:"routeId"`
		Stop  string `json:"stopId"`
	} `json:"informedEntity"`
	Header      Text `json:"headerText"`
	Description Text `json:"descriptionText"`
}

type Feed struct {
	Header struct {
		Timestamp string `json:"timestamp"`
	} `json:"header"`
	Entities []struct {
		Update *Update `json:"tripUpdate"`
		Alert  *Alert  `json:"alert"`
	} `json:"entity"`
}

type Trip struct {
	Route, Number string
	Stops         map[string]int
	Departures    map[string]string
}

type Schedule struct {
	Trips  map[string]*Trip
	Routes map[string]string
}

var client = &http.Client{Timeout: 45 * time.Second}

func download(url string) ([]byte, error) {
	r, e := client.Get(url)
	if e != nil {
		return nil, e
	}

	defer func() { _ = r.Body.Close() }()

	if r.StatusCode != 200 {
		return nil, fmt.Errorf("%s: HTTP %d", url, r.StatusCode)
	}

	b, e := io.ReadAll(io.LimitReader(r.Body, 100<<20))

	return b, e
}

func schedule(kind, url string, refresh bool) (Schedule, error) {
	s := Schedule{
		Trips:  map[string]*Trip{},
		Routes: map[string]string{},
	}

	path := filepath.Join(".cache", kind+".zip")

	info, e := os.Stat(path)
	if refresh || e != nil || time.Since(info.ModTime()) > 24*time.Hour {
		b, e := download(url)
		if e != nil {
			return s, e
		}

		if _, e = zip.NewReader(bytes.NewReader(b), int64(len(b))); e != nil {
			return s, e
		}

		if e = os.MkdirAll(".cache", 0o755); e != nil {
			return s, e
		}

		if e = os.WriteFile(path, b, 0o644); e != nil {
			return s, e
		}
	}

	z, e := zip.OpenReader(path)
	if e != nil {
		return s, e
	}

	defer func() { _ = z.Close() }()

	for _, name := range []string{"routes.txt", "trips.txt", "stop_times.txt"} {
		found := false

		for _, f := range z.File {
			if filepath.Base(f.Name) != name {
				continue
			}

			found = true

			r, e := f.Open()
			if e != nil {
				return s, e
			}

			c := csv.NewReader(r)
			c.FieldsPerRecord = -1

			h, e := c.Read()
			if e != nil {
				_ = r.Close()
				return s, e
			}

			idx := map[string]int{}
			for i, k := range h {
				idx[strings.TrimSpace(strings.TrimPrefix(k, "\ufeff"))] = i
			}

			for {
				row, e := c.Read()
				if e == io.EOF {
					break
				}

				if e != nil {
					_ = r.Close()
					return s, e
				}

				get := func(k string) string {
					i, ok := idx[k]
					if !ok || i >= len(row) {
						return ""
					}

					return strings.TrimSpace(row[i])
				}

				switch name {
				case "routes.txt":
					s.Routes[get("route_id")] = get("route_short_name")
				case "trips.txt":
					s.Trips[get("trip_id")] = &Trip{Route: get("route_id"), Number: get("trip_short_name"), Stops: map[string]int{}, Departures: map[string]string{}}
				case "stop_times.txt":
					if t := s.Trips[get("trip_id")]; t != nil {
						var seq int

						_, _ = fmt.Sscan(get("stop_sequence"), &seq)
						t.Stops[get("stop_id")] = seq
						t.Departures[get("stop_id")] = get("departure_time")
					}
				}
			}

			_ = r.Close()
		}

		if !found {
			return s, fmt.Errorf("falta %s al GTFS", name)
		}
	}

	return s, nil
}

func corridor(t *Trip) bool {
	a, ok := t.Stops["71801"]
	b, ok2 := t.Stops["79300"]

	return ok && ok2 && a != b
}

func delay(u *Update, stop string) *int {
	for _, s := range u.Stops {
		if s.ID == stop {
			if s.Arrival.Delay != nil {
				return s.Arrival.Delay
			}

			if s.Departure.Delay != nil {
				return s.Departure.Delay
			}
		}
	}

	return nil
}

func text(t Text) string {
	for _, x := range t.Translation {
		if x.Language == "ca" {
			return x.Text
		}
	}

	if len(t.Translation) > 0 {
		return t.Translation[0].Text
	}

	return ""
}

func active(a *Alert, now int64) bool {
	if len(a.Active) == 0 {
		return true
	}

	for _, p := range a.Active {
		var start, end int64

		_, _ = fmt.Sscan(p.Start, &start)

		_, _ = fmt.Sscan(p.End, &end)
		if (start == 0 || now >= start) && (end == 0 || now < end) {
			return true
		}
	}

	return false
}

func feed(url string) (Feed, error) {
	var f Feed

	b, e := download(url)
	if e != nil {
		return f, e
	}

	e = json.Unmarshal(b, &f)

	return f, e
}

func collect(output io.Writer, refresh bool) error {
	var body bytes.Buffer

	w := &body
	r11 := lineState{}
	r11Routes := map[string]bool{}

	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		return fmt.Errorf("no es pot carregar la zona horària Europe/Madrid: %w", err)
	}

	now := time.Now().In(loc)
	fmt.Fprintf(w, "Barcelona Sants ↔ Girona · %s\nOrigen de les dades: Renfe Operadora (CC BY 4.0)\n", now.Format("02/01/2006 15:04 MST"))

	failed := false
	relevant := map[string]bool{}

	sources := []struct{ kind, url, rt string }{
		{
			kind: "ld",
			url:  "https://ssl.renfe.com/gtransit/Fichero_AV_LD/google_transit.zip",
			rt:   "trip_updates_LD.json",
		},
		{
			kind: "cer",
			url:  "https://ssl.renfe.com/ftransit/Fichero_CER_FOMENTO/fomento_transit.zip",
			rt:   "trip_updates.json",
		},
	}
	for _, src := range sources {
		s, e := schedule(src.kind, src.url, refresh)
		if e != nil {
			fmt.Fprintf(w, "Horaris %s: NO DISPONIBLES: %v\n", src.kind, e)

			failed = true

			continue
		}

		for _, t := range s.Trips {
			if s.Routes[t.Route] == "R11" {
				r11Routes[t.Route] = true
			}

			if corridor(t) || s.Routes[t.Route] == "R11" {
				relevant[t.Route] = true
			}
		}

		f, e := feed("https://gtfsrt.renfe.com/" + src.rt)
		if e != nil {
			fmt.Fprintf(w, "Temps real %s: NO DISPONIBLE: %v\n", src.kind, e)

			failed = true

			continue
		}

		var stamp int64

		_, _ = fmt.Sscan(f.Header.Timestamp, &stamp)

		if src.kind == "cer" {
			r11.available = true
			r11.stale = stamp == 0 || now.Sub(time.Unix(stamp, 0)) > 15*time.Minute
			r11.timestamp = time.Unix(stamp, 0).In(loc).Format("02/01 15:04:05")
			r11.measure(s, f)
		}

		fmt.Fprintf(w, "\nTemps real %s · font: %s", src.kind, time.Unix(stamp, 0).In(loc).Format("02/01 15:04:05"))

		if stamp == 0 || now.Sub(time.Unix(stamp, 0)) > 15*time.Minute {
			fmt.Fprint(w, " · DADES ANTIGUES O SENSE DATA")
		}

		fmt.Fprintln(w)

		rows := []string{}

		for _, entity := range f.Entities {
			u := entity.Update
			if u == nil {
				continue
			}

			t := s.Trips[u.Trip.ID]
			if t == nil || !corridor(t) {
				continue
			}

			dest := "79300"
			direction := "Sants → Girona"

			if t.Stops["71801"] > t.Stops["79300"] {
				dest = "71801"
				direction = "Girona → Sants"
			}

			status := "sense retard publicat a destinació"
			if d := delay(u, dest); d != nil {
				status = fmt.Sprintf("retard a destinació: %+.1f min", float64(*d)/60)
			} else if u.Delay != nil {
				status = fmt.Sprintf("retard general del tren: %+.1f min (sense dada a destinació)", float64(*u.Delay)/60)
			}

			if u.Trip.Relationship == "CANCELED" {
				status = "CANCEL·LAT"
			}

			identifier := t.Number
			if identifier == "" {
				identifier = "trajecte " + u.Trip.ID
			} else {
				identifier += " [" + u.Trip.ID + "]"
			}

			rows = append(rows, fmt.Sprintf("%s | %s | %s · sortida programada %s | %s", direction, s.Routes[t.Route], identifier, departureLabel(t), status))
		}

		sort.Strings(rows)

		for _, r := range rows {
			fmt.Fprintln(w, r)
		}

		if len(rows) == 0 {
			fmt.Fprintln(w, "Cap tren del trajecte identificat al feed. Això no implica que circulin puntuals.")
		}
	}

	fmt.Fprintln(w, "\nAvisos oficials R11 i rutes del trajecte:")

	f, e := feed("https://gtfsrt.renfe.com/alerts.json")
	if e != nil {
		r11.alertsUnavailable = true

		fmt.Fprintf(w, "NO DISPONIBLES: %v\n", e)

		failed = true
	} else {
		count := 0

		for _, entity := range f.Entities {
			a := entity.Alert
			if a == nil || !active(a, now.Unix()) {
				continue
			}

			match := false

			for _, i := range a.Informed {
				if r11Routes[i.Route] {
					r11.alerts = append(r11.alerts, strings.TrimSpace(text(a.Header)+" "+text(a.Description)))
					break
				}
			}

			for _, i := range a.Informed {
				if relevant[i.Route] || i.Stop == "71801" || i.Stop == "79300" {
					match = true
				}
			}

			if match {
				fmt.Fprintln(w, "- "+strings.TrimSpace(text(a.Header)+" "+text(a.Description)))

				count++
			}
		}

		if count == 0 {
			fmt.Fprintln(w, "Cap avís coincident publicat en aquest feed; cobertura no garantida.")
		}
	}

	if err := collectAdif(w); err != nil {
		fmt.Fprintf(w, "Adif: NO DISPONIBLE: %v\n", err)

		failed = true
	}

	fmt.Fprintln(w, "Només es mostren trens amb actualització en temps real i correspondència exacta amb els horaris. Absència de dades ≠ puntualitat.")

	if _, err := fmt.Fprintln(output, r11.summary()); err != nil {
		return err
	}

	if _, err := io.Copy(output, &body); err != nil {
		return err
	}

	if failed {
		return fmt.Errorf("consulta parcial: alguna font no està disponible")
	}

	return nil
}
