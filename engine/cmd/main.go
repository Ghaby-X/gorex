package main

import (
	"fmt"
	"log/slog"

	"github.com/Ghaby-X/gorex/internal/config"
)

func main() {
	// define logger
	slog.Info("Starting gorex")

	// load config file
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		slog.Error("error loading config file", "error", err)
	}

	// printing out details
	fmt.Printf("Provider: %s (%s)\n", cfg.Provider.Name, cfg.Provider.Timeframe)
	for _, asset := range cfg.Assets {
		fmt.Printf("Trading %s with strategy: %s\n", asset.Symbol, asset.Strategy.Name)
		fmt.Printf("  Slow Period: %v\n", asset.Strategy.Params["slow_period"])
	}
}
