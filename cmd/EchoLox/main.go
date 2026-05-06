package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/BattloXX/EchoLox/internal/bridge"
)

func main() {
	defaultCfg := filepath.Join(lbPath("LBPCFG", "./config"), "EchoLox.cfg")
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

	log.Printf("EchoLox starting on port %d", cfg.Server.Port)
	if err := bridge.Run(cfg, *cfgPath); err != nil {
		log.Fatal(err)
	}
}

func lbPath(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}
