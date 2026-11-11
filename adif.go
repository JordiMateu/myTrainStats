package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const adifLayer = "https://services7.arcgis.com/XTupIrLX53AjaJqO/arcgis/rest/services/LTV_2/FeatureServer/0"

type restriction struct {
	Line   string  `json:"CODLINEA"`
	From   string  `json:"DESCPSINI"`
	To     string  `json:"DESCPSFIN"`
	Speed  *int    `json:"RESTRICCIONVELOCIDAD"`
	Start  float64 `json:"PKINI"`
	End    float64 `json:"PKFIN"`
	Reason string  `json:"MOTIVO"`
	Notes  string  `json:"OBSERVACIONES"`
}

func collectAdif(w *bytes.Buffer) error {
	fmt.Fprintln(w, "\nLimitacions Adif · àrea Barcelona–Girona")
	fmt.Fprintln(w, "© Administrador de infraestructuras ferroviarias")
	q := url.Values{"f": {"json"}, "where": {"CODLINEA IN ('050','270')"}, "geometry": {"2.05,41.35,2.9,42.05"}, "geometryType": {"esriGeometryEnvelope"}, "inSR": {"4326"}, "spatialRel": {"esriSpatialRelIntersects"}, "outFields": {"CODLINEA,DESCPSINI,DESCPSFIN,RESTRICCIONVELOCIDAD,PKINI,PKFIN,MOTIVO,OBSERVACIONES"}, "returnGeometry": {"false"}, "orderByFields": {"OBJECTID"}, "resultRecordCount": {"1000"}}
	count := 0
	for offset := 0; ; offset += 1000 {
		q.Set("resultOffset", fmt.Sprint(offset))
		b, err := download(adifLayer + "/query?" + q.Encode())
		if err != nil {
			return err
		}
		var result struct {
			Features []struct {
				Attributes restriction `json:"attributes"`
			} `json:"features"`
			More  bool `json:"exceededTransferLimit"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err = json.Unmarshal(b, &result); err != nil {
			return err
		}
		if result.Error != nil {
			return fmt.Errorf("error d'Adif: %s", result.Error.Message)
		}
		for _, f := range result.Features {
			a := f.Attributes
			speed := "velocitat no publicada"
			if a.Speed != nil {
				speed = fmt.Sprintf("%d km/h", *a.Speed)
			}
			fmt.Fprintf(w, "- Línia %s · %s → %s · PK %.3f–%.3f · %s\n", a.Line, a.From, a.To, a.Start, a.End, speed)
			if detail := strings.TrimSpace(a.Reason + " " + a.Notes); detail != "" {
				fmt.Fprintln(w, "  "+detail)
			}
			count++
		}
		if !result.More {
			break
		}
		if len(result.Features) == 0 {
			return fmt.Errorf("paginació incompleta d'Adif")
		}
	}
	if count == 0 {
		fmt.Fprintln(w, "Cap limitació geolocalitzada coincident publicada; no implica absència d'incidències.")
	}
	fmt.Fprintln(w, "Selecció aproximada per àrea i línies 050/270: pot incloure trams fora del trajecte. No inclou limitacions sense localització ni totes les incidències. No s'avaluen calendaris o horaris particulars. Les restriccions no equivalen a retards mesurats.")
	return nil
}
