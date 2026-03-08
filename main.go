package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"railway-supervisor/pkg/railway"
	"railway-supervisor/pkg/supervisor"
)

func main() {
	// Load .env if it exists
	_ = godotenv.Load()

	railwayToken := os.Getenv("RAILWAY_TOKEN")
	if railwayToken == "" {
		log.Fatal("RAILWAY_TOKEN environment variable is required")
	}

	configJSON := os.Getenv("CONFIG_JSON")
	var config supervisor.Config
	if configJSON != "" {
		if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
			log.Fatalf("Error parsing CONFIG_JSON: %v", err)
		}
	} else {
		fileBytes, err := os.ReadFile("config.json")
		if err != nil {
			log.Fatalf("CONFIG_JSON env variable not set and config.json file not found/readable: %v", err)
		}
		if err := json.Unmarshal(fileBytes, &config); err != nil {
			log.Fatalf("Error parsing config.json: %v", err)
		}
	}

	if config.CheckIntervalMinutes <= 0 {
		config.CheckIntervalMinutes = 5
	}

	loc, err := time.LoadLocation(config.Timezone)
	if err != nil {
		log.Fatalf("Invalid timezone '%s': %v", config.Timezone, err)
	}

	client := railway.NewClient(railwayToken)
	s := supervisor.NewSupervisor(config, client, loc)

	log.Printf("Starting Railway Supervisor. Timezone: %s, Interval: %d min, Rules: %d", config.Timezone, config.CheckIntervalMinutes, len(config.Rules))
	s.Start()
}
