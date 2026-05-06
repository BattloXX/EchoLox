package web

import (
	"io/fs"
	"net/http"

	"github.com/BattloXX/EchoLox/internal/device"
	"github.com/BattloXX/EchoLox/internal/loxone"
	"github.com/BattloXX/EchoLox/webembed"
)

// WebConfig holds the configuration values needed by the web handler.
type WebConfig struct {
	ServerPort int
	ServerIP   string
}

type Handler struct {
	mgr      *device.Manager
	verifier *loxone.Verifier
	lbs      map[string]loxone.LBMiniserver
	cfg      WebConfig
}

func NewHandler(mgr *device.Manager, verifier *loxone.Verifier, lbs map[string]loxone.LBMiniserver, cfg WebConfig) *Handler {
	return &Handler{mgr: mgr, verifier: verifier, lbs: lbs, cfg: cfg}
}

func (h *Handler) Register(mux *http.ServeMux) {
	sub, err := fs.Sub(webembed.Files, "web")
	if err != nil {
		panic("web static files not found: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))
	mux.Handle("/ui/", http.StripPrefix("/ui", fileServer))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/ui/index.html", http.StatusFound)
			return
		}
	})
}
