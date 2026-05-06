package hue

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/BattloXX/EchoLox/internal/device"
	"github.com/BattloXX/EchoLox/internal/loxone"
)

type API struct {
	mgr      *device.Manager
	lox      *loxone.Client
	verifier *loxone.Verifier
}

func NewAPI(mgr *device.Manager, lox *loxone.Client, verifier *loxone.Verifier) *API {
	return &API{mgr: mgr, lox: lox, verifier: verifier}
}

func (a *API) Register(mux *http.ServeMux) {
	// Hue pairing
	mux.HandleFunc("/api", a.handleRoot)

	// Config
	mux.HandleFunc("/api/", a.route)
}

func (a *API) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Alexa pairing - return success with any username
		var body struct {
			DeviceType string `json:"devicetype"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		username := fmt.Sprintf("EchoLox%d", time.Now().Unix()%100000)
		writeJSON(w, []map[string]interface{}{
			{"success": map[string]string{"username": username}},
		})
		return
	}
	writeJSON(w, map[string]interface{}{})
}

func (a *API) route(w http.ResponseWriter, r *http.Request) {
	// /api/{user}/...
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 {
		// /api/{user} → config
		a.serveConfig(w, r)
		return
	}

	resource := parts[1]
	rest := ""
	if len(parts) > 2 {
		rest = parts[2]
	}

	switch resource {
	case "lights":
		a.handleLights(w, r, rest)
	case "groups":
		a.handleGroups(w, r, rest)
	case "config":
		a.serveConfig(w, r)
	case "":
		a.serveFullConfig(w, r, parts[0])
	default:
		writeJSON(w, map[string]interface{}{})
	}
}

func (a *API) serveConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"name":             "EchoLox",
		"swversion":        "9999999999",
		"apiversion":       "1.47.0",
		"mac":              "00:00:00:00:00:00",
		"bridgeid":         "ECHOLOX000000",
		"factorynew":       false,
		"replacesbridgeid": nil,
		"modelid":          "BSB002",
	})
}

func (a *API) serveFullConfig(w http.ResponseWriter, r *http.Request, user string) {
	devices := a.mgr.All()
	lights := map[string]interface{}{}
	for _, d := range devices {
		lights[d.ID] = a.deviceToLight(d)
	}
	writeJSON(w, map[string]interface{}{
		"lights": lights,
		"groups": map[string]interface{}{},
		"config": map[string]interface{}{
			"name":       "EchoLox",
			"swversion":  "9999999999",
			"apiversion": "1.47.0",
			"bridgeid":   "ECHOLOX000000",
			"modelid":    "BSB002",
		},
	})
}

func (a *API) handleLights(w http.ResponseWriter, r *http.Request, rest string) {
	if rest == "" {
		// GET /api/{user}/lights
		devices := a.mgr.All()
		result := map[string]interface{}{}
		for _, d := range devices {
			result[d.ID] = a.deviceToLight(d)
		}
		writeJSON(w, result)
		return
	}

	// /api/{user}/lights/{id} or /api/{user}/lights/{id}/state
	idAndRest := strings.SplitN(rest, "/", 2)
	id := idAndRest[0]
	subpath := ""
	if len(idAndRest) > 1 {
		subpath = idAndRest[1]
	}

	d, ok := a.mgr.Get(id)
	if !ok {
		http.Error(w, "not found", 404)
		return
	}

	if subpath == "state" && r.Method == http.MethodPut {
		a.handleStateChange(w, r, d)
		return
	}

	writeJSON(w, a.deviceToLight(d))
}

type stateBody struct {
	On             *bool     `json:"on"`
	Bri            *int      `json:"bri"`
	Hue            *int      `json:"hue"`
	Sat            *int      `json:"sat"`
	CT             *int      `json:"ct"`
	XY             []float64 `json:"xy"`
	Effect         string    `json:"effect"`
	Transitiontime *int      `json:"transitiontime"`
}

func (a *API) handleStateChange(w http.ResponseWriter, r *http.Request, d *device.Device) {
	var body stateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	state := a.mgr.GetState(d.ID)
	responses := []interface{}{}

	if body.On != nil {
		state.On = *body.On
		val := "0"
		if *body.On {
			val = "1"
		}
		if vi, ok := d.VirtualInputs["on"]; ok {
			a.send(d.ID, vi, val)
		}
		if vi, ok := d.VirtualInputs["activate"]; ok && *body.On {
			a.send(d.ID, vi, "1")
		}
		responses = append(responses, map[string]interface{}{
			"success": map[string]interface{}{
				"/lights/" + d.ID + "/state/on": *body.On,
			},
		})
	}

	if body.Bri != nil {
		state.Brightness = *body.Bri
		pct := BriToPct(*body.Bri)
		if vi, ok := d.VirtualInputs["brightness"]; ok {
			a.send(d.ID, vi, strconv.Itoa(pct))
		}
		responses = append(responses, map[string]interface{}{
			"success": map[string]interface{}{
				"/lights/" + d.ID + "/state/bri": *body.Bri,
			},
		})
	}

	if body.Hue != nil {
		state.Hue = *body.Hue
		deg := HueTo360(*body.Hue)
		if vi, ok := d.VirtualInputs["hue"]; ok {
			a.send(d.ID, vi, strconv.Itoa(deg))
		}
		responses = append(responses, map[string]interface{}{
			"success": map[string]interface{}{
				"/lights/" + d.ID + "/state/hue": *body.Hue,
			},
		})
	}

	if body.Sat != nil {
		state.Saturation = *body.Sat
		pct := SatToPct(*body.Sat)
		if vi, ok := d.VirtualInputs["saturation"]; ok {
			a.send(d.ID, vi, strconv.Itoa(pct))
		}
		responses = append(responses, map[string]interface{}{
			"success": map[string]interface{}{
				"/lights/" + d.ID + "/state/sat": *body.Sat,
			},
		})
	}

	a.mgr.SetState(d.ID, state)
	writeJSON(w, responses)
}

func (a *API) send(deviceID, viName, value string) {
	if a.lox == nil {
		log.Printf("[dry-run] %s = %s", viName, value)
		return
	}
	if err := a.lox.Send(viName, value); err != nil {
		log.Printf("loxone send error %s=%s: %v", viName, value, err)
	}
	a.mgr.RecordSent(deviceID, viName, value)
}

func (a *API) handleGroups(w http.ResponseWriter, r *http.Request, rest string) {
	writeJSON(w, map[string]interface{}{})
}

func (a *API) deviceToLight(d *device.Device) map[string]interface{} {
	s := a.mgr.GetState(d.ID)
	colormode := ""
	if d.Type == device.TypeColor {
		colormode = "hs"
	}
	lightType := "Dimmable light"
	if d.Type == device.TypeColor {
		lightType = "Extended color light"
	} else if d.Type == device.TypeSwitch || d.Type == device.TypeScene {
		lightType = "On/Off plug-in unit"
	}
	result := map[string]interface{}{
		"state": map[string]interface{}{
			"on":        s.On,
			"bri":       s.Brightness,
			"hue":       s.Hue,
			"sat":       s.Saturation,
			"ct":        s.ColorTemp,
			"colormode": colormode,
			"reachable": true,
			"effect":    "none",
			"alert":     "none",
			"xy":        []float64{0.0, 0.0},
		},
		"type":             lightType,
		"name":             d.Name,
		"modelid":          "LCT001",
		"uniqueid":         hueUniqueID(d.ID),
		"swversion":        "9999999999",
		"manufacturername": "Philips",
	}
	return result
}

func hueUniqueID(id string) string {
	h := 0
	for _, c := range id {
		h = h*31 + int(c)
	}
	h = h & 0xFFFFFF
	return fmt.Sprintf("%02x:%02x:%02x:00:00:00:00:00-00",
		(h>>16)&0xFF, (h>>8)&0xFF, h&0xFF)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
