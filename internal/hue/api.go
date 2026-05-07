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
	"github.com/BattloXX/EchoLox/internal/identity"
	"github.com/BattloXX/EchoLox/internal/loxone"
)

type API struct {
	mgr      *device.Manager
	lox      *loxone.Client
	verifier *loxone.Verifier
	info     identity.BridgeInfo
}

func NewAPI(mgr *device.Manager, lox *loxone.Client, verifier *loxone.Verifier, info identity.BridgeInfo) *API {
	return &API{mgr: mgr, lox: lox, verifier: verifier, info: info}
}

func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api", a.handlePairing)
	mux.HandleFunc("/api/", a.route)
}

// handlePairing handles POST /api — Alexa pairs by posting {"devicetype":"..."}.
// Returns [{"success":{"username":"<32-hex-token>"}}].
// Always succeeds — no link-button press required for emulated bridges.
func (a *API) handlePairing(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method == http.MethodPost {
		var body struct {
			DeviceType string `json:"devicetype"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		username := fmt.Sprintf("%032x", hashStr(body.DeviceType+a.info.Suffix))
		writeJSON(w, []map[string]interface{}{
			{"success": map[string]string{"username": username}},
		})
		return
	}
	writeJSON(w, []interface{}{})
}

func (a *API) route(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	// /api/{user}[/{resource}[/{id}[/{sub}]]]
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	parts := strings.SplitN(path, "/", 4)
	// parts[0] = username (accepted without validation)
	if len(parts) < 2 || parts[1] == "" {
		a.serveDatastore(w, r)
		return
	}
	resource := parts[1]
	id := ""
	if len(parts) > 2 {
		id = parts[2]
	}
	sub := ""
	if len(parts) > 3 {
		sub = parts[3]
	}
	switch resource {
	case "lights":
		a.handleLights(w, r, id, sub)
	case "groups":
		a.handleGroups(w, r, id)
	case "config":
		a.serveConfig(w)
	case "scenes", "schedules", "sensors", "rules", "resourcelinks":
		writeJSON(w, map[string]interface{}{})
	default:
		writeJSON(w, map[string]interface{}{})
	}
}

// serveDatastore serves GET /api/{user} — Alexa fetches this during discovery.
func (a *API) serveDatastore(w http.ResponseWriter, r *http.Request) {
	lights := a.buildLightsMap()
	writeJSON(w, map[string]interface{}{
		"lights":        lights,
		"groups":        a.buildGroupsMap(lights),
		"config":        a.buildConfig(),
		"schedules":     map[string]interface{}{},
		"scenes":        map[string]interface{}{},
		"rules":         map[string]interface{}{},
		"sensors":       map[string]interface{}{},
		"resourcelinks": map[string]interface{}{},
	})
}

func (a *API) serveConfig(w http.ResponseWriter) {
	writeJSON(w, a.buildConfig())
}

func (a *API) buildConfig() map[string]interface{} {
	now := time.Now()
	return map[string]interface{}{
		"name":             "EchoLox",
		"zigbeechannel":    11,
		"bridgeid":         a.info.BridgeID,
		"mac":              a.info.MAC,
		"ipaddress":        a.info.IP,
		"netmask":          "255.255.255.0",
		"gateway":          "0.0.0.0",
		"dhcp":             true,
		"proxyaddress":     "none",
		"proxyport":        0,
		"UTC":              now.UTC().Format("2006-01-02T15:04:05"),
		"localtime":        now.Format("2006-01-02T15:04:05"),
		"timezone":         "Europe/Vienna",
		"modelid":          "BSB002",
		"datastoreversion": "93",
		"swversion":        "01061424042",
		"apiversion":       "1.47.0",
		"linkbutton":       false,
		"portalservices":   false,
		"portalconnection": "disconnected",
		"portalstate": map[string]interface{}{
			"signedon":      false,
			"incoming":      false,
			"outgoing":      false,
			"communication": "disconnected",
		},
		"factorynew":       false,
		"replacesbridgeid": nil,
		"backup": map[string]interface{}{
			"status":    "idle",
			"errorcode": 0,
		},
		"whitelist": map[string]interface{}{},
	}
}

func (a *API) handleLights(w http.ResponseWriter, r *http.Request, id, sub string) {
	if id == "" {
		writeJSON(w, a.buildLightsMap())
		return
	}
	d, ok := a.mgr.GetByHueID(id)
	if !ok {
		writeHueError(w, 3, "/lights/"+id, "resource, "+id+", not available")
		return
	}
	if sub == "state" && r.Method == http.MethodPut {
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
	prefix := "/lights/" + d.HueID + "/state/"

	if body.On != nil {
		state.On = *body.On
		if d.SwitchMode == device.SwitchModeImpulse {
			// Impuls: _on-VI bei Ein, _off-VI bei Aus
			if *body.On {
				if vi, ok := d.VirtualInputs["on"]; ok {
					a.send(d.ID, vi, "Impuls")
				}
			} else {
				if vi, ok := d.VirtualInputs["off"]; ok {
					a.send(d.ID, vi, "Impuls")
				}
			}
		} else {
			// Ein/Aus: ein einziger VI, 1 = ein, 0 = aus
			if vi, ok := d.VirtualInputs["on"]; ok {
				if *body.On {
					a.send(d.ID, vi, "1")
				} else {
					a.send(d.ID, vi, "0")
				}
			}
			if vi, ok := d.VirtualInputs["activate"]; ok {
				a.send(d.ID, vi, "1")
			}
		}
		responses = append(responses, success(prefix+"on", *body.On))
	}
	if body.Bri != nil {
		state.Brightness = *body.Bri
		if vi, ok := d.VirtualInputs["brightness"]; ok {
			a.send(d.ID, vi, strconv.Itoa(BriToPct(*body.Bri)))
		}
		responses = append(responses, success(prefix+"bri", *body.Bri))
	}
	if body.Hue != nil {
		state.Hue = *body.Hue
		state.ColorMode = "hs"
		if vi, ok := d.VirtualInputs["hue"]; ok {
			a.send(d.ID, vi, strconv.Itoa(HueTo360(*body.Hue)))
		}
		responses = append(responses, success(prefix+"hue", *body.Hue))
	}
	if body.Sat != nil {
		state.Saturation = *body.Sat
		state.ColorMode = "hs"
		if vi, ok := d.VirtualInputs["saturation"]; ok {
			a.send(d.ID, vi, strconv.Itoa(SatToPct(*body.Sat)))
		}
		responses = append(responses, success(prefix+"sat", *body.Sat))
	}
	if body.CT != nil {
		state.ColorTemp = *body.CT
		state.ColorMode = "ct"
		responses = append(responses, success(prefix+"ct", *body.CT))
	}
	if len(body.XY) == 2 {
		state.ColorMode = "xy"
		responses = append(responses, success(prefix+"xy", body.XY))
	}
	a.mgr.SetState(d.ID, state)
	writeJSON(w, responses)
}

func (a *API) handleGroups(w http.ResponseWriter, r *http.Request, id string) {
	lights := a.buildLightsMap()
	groups := a.buildGroupsMap(lights)
	if id == "" {
		writeJSON(w, groups)
		return
	}
	if g, ok := groups[id]; ok {
		writeJSON(w, g)
		return
	}
	writeJSON(w, map[string]interface{}{})
}

func (a *API) buildLightsMap() map[string]interface{} {
	result := map[string]interface{}{}
	for _, d := range a.mgr.All() {
		result[d.HueID] = a.deviceToLight(d)
	}
	return result
}

func (a *API) buildGroupsMap(lights map[string]interface{}) map[string]interface{} {
	ids := make([]string, 0, len(lights))
	for id := range lights {
		ids = append(ids, id)
	}
	group0 := map[string]interface{}{
		"name":   "Lightset 0",
		"lights": ids,
		"type":   "LightGroup",
		"action": map[string]interface{}{
			"on":        false,
			"bri":       254,
			"hue":       0,
			"sat":       0,
			"xy":        []float64{0.3127, 0.3290},
			"ct":        370,
			"effect":    "none",
			"colormode": "ct",
		},
		"state": map[string]interface{}{
			"all_on": false,
			"any_on": false,
		},
	}
	return map[string]interface{}{"0": group0}
}

// deviceToLight builds the full Hue light object for a device.
func (a *API) deviceToLight(d *device.Device) map[string]interface{} {
	s := a.mgr.GetState(d.ID)
	lightType, modelid, productname := lightMeta(d.Type)

	colormode := s.ColorMode
	if colormode == "" && d.Type == device.TypeColor {
		colormode = "ct"
	}

	stateMap := map[string]interface{}{
		"on":        s.On,
		"bri":       s.Brightness,
		"alert":     "none",
		"effect":    "none",
		"reachable": true,
	}
	if d.Type == device.TypeColor || d.Type == device.TypeDimmer {
		stateMap["hue"] = s.Hue
		stateMap["sat"] = s.Saturation
		stateMap["xy"] = []float64{0.3127, 0.3290}
		stateMap["ct"] = s.ColorTemp
		stateMap["colormode"] = colormode
	}

	return map[string]interface{}{
		"state":            stateMap,
		"type":             lightType,
		"name":             d.Name,
		"modelid":          modelid,
		"manufacturername": "Signify Netherlands B.V.",
		"productname":      productname,
		"uniqueid":         hueUniqueID(d.HueID),
		"swversion":        "67.91.213",
		"capabilities":     buildCapabilities(d.Type),
	}
}

func lightMeta(t device.DeviceType) (lightType, modelid, productname string) {
	switch t {
	case device.TypeColor:
		return "Extended color light", "LCT015", "Hue color lamp"
	case device.TypeDimmer:
		return "Dimmable light", "LWB006", "Hue White lamp"
	case device.TypeScene, device.TypeSwitch:
		return "On/Off plug-in unit", "LOM001", "Hue Smart plug"
	default:
		return "On/Off plug-in unit", "LOM001", "Hue Smart plug"
	}
}

func buildCapabilities(t device.DeviceType) map[string]interface{} {
	cap := map[string]interface{}{
		"certified": true,
		"streaming": map[string]interface{}{"renderer": false, "proxy": false},
	}
	switch t {
	case device.TypeColor:
		cap["control"] = map[string]interface{}{
			"mindimlevel":    200,
			"maxlumen":       800,
			"colorgamuttype": "C",
			"colorgamut":     [][]float64{{0.6915, 0.3083}, {0.17, 0.7}, {0.1532, 0.0475}},
			"ct":             map[string]int{"min": 153, "max": 500},
		}
	case device.TypeDimmer:
		cap["control"] = map[string]interface{}{
			"mindimlevel": 200,
			"maxlumen":    800,
		}
	default:
		cap["control"] = map[string]interface{}{}
	}
	return cap
}

// hueUniqueID returns a MAC-format uniqueid for a Hue light.
// Format: 00:17:88:01:00:{id-bytes}-{id-byte}
func hueUniqueID(hueID string) string {
	n, _ := strconv.Atoi(hueID)
	return fmt.Sprintf("00:17:88:01:00:%02x:%02x:%02x-%02x",
		(n>>16)&0xFF, (n>>8)&0xFF, n&0xFF, n&0xFF)
}

func (a *API) send(deviceID, viName, value string) {
	if a.lox == nil {
		log.Printf("[dry-run] %s = %s", viName, value)
		return
	}
	if err := a.lox.Send(viName, value); err != nil {
		log.Printf("loxone send %s=%s: %v", viName, value, err)
	}
	a.mgr.RecordSent(deviceID, viName, value)
}

func success(path string, val interface{}) map[string]interface{} {
	return map[string]interface{}{"success": map[string]interface{}{path: val}}
}

func writeHueError(w http.ResponseWriter, errType int, address, desc string) {
	writeJSON(w, []map[string]interface{}{
		{"error": map[string]interface{}{
			"type":        errType,
			"address":     address,
			"description": desc,
		}},
	})
}

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func hashStr(s string) uint64 {
	var h uint64 = 14695981039346656037
	for _, c := range []byte(s) {
		h ^= uint64(c)
		h *= 1099511628211
	}
	return h
}
