package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/BattloXX/EchoLox/internal/bridge"
)

var version = "dev"

func main() {
	defaultCfg := filepath.Join(lbPath("LBPDATADIR", "./data"), "EchoLox.cfg")
	cfgPath := flag.String("config", defaultCfg, "path to config file")
	importFrom := flag.String("import", "", "import old devices.db path (CLI mode)")
	importOut := flag.String("import-out", "", "output dir for imported devices")
	flag.Parse()

	cfg, err := bridge.LoadConfig(*cfgPath)
	if err != nil {
		log.Printf("config load error (%v), using defaults", err)
		cfg = bridge.DefaultConfig()
	}

	if *importFrom != "" {
		// CLI import mode
		bridge.RunCLIImport(*importFrom, *importOut, cfg)
		return
	}

	log.Printf("EchoLox v%s starting on port %d", version, cfg.Server.Port)
	if err := bridge.Run(cfg, *cfgPath, version); err != nil {
		log.Fatal(err)
	}
}

func lbPath(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}
