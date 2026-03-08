package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config models
type SleepWindow struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type Rule struct {
	Name          string      `json:"name"`
	EnvironmentID string      `json:"environmentId"`
	ServiceID     *string     `json:"serviceId"`
	SleepWindow   SleepWindow `json:"sleepWindow"`
}

type Config struct {
	CheckIntervalMinutes int    `json:"checkIntervalMinutes"`
	Timezone             string `json:"timezone"`
	Rules                []Rule `json:"rules"`
}

// Global Variables
var (
	railwayToken string
	state        = make(map[string]bool) // Key: rule.Name, Value: isSleeping
)

func main() {
	// Load .env if it exists
	_ = godotenv.Load()

	railwayToken = os.Getenv("RAILWAY_TOKEN")
	if railwayToken == "" {
		log.Fatal("RAILWAY_TOKEN environment variable is required")
	}

	configJSON := os.Getenv("CONFIG_JSON")
	var config Config
	if configJSON != "" {
		if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
			log.Fatalf("Error parsing CONFIG_JSON: %v", err)
		}
	} else {
		// Try to read from config.json if CONFIG_JSON env is not set
		fileBytes, err := os.ReadFile("config.json")
		if err != nil {
			log.Fatalf("CONFIG_JSON env variable not set and config.json file not found/readable: %v", err)
		}
		if err := json.Unmarshal(fileBytes, &config); err != nil {
			log.Fatalf("Error parsing config.json: %v", err)
		}
	}

	if config.CheckIntervalMinutes <= 0 {
		config.CheckIntervalMinutes = 5 // default
	}

	loc, err := time.LoadLocation(config.Timezone)
	if err != nil {
		log.Fatalf("Invalid timezone '%s': %v", config.Timezone, err)
	}

	log.Printf("Starting Railway Supervisor. Timezone: %s, Interval: %d min, Rules: %d", config.Timezone, config.CheckIntervalMinutes, len(config.Rules))

	// Initial evaluation delay 0, then tick every interval
	ticker := time.NewTicker(time.Duration(config.CheckIntervalMinutes) * time.Minute)
	defer ticker.Stop()

	evaluateRules(config, loc) // Run immediately on startup

	for range ticker.C {
		evaluateRules(config, loc)
	}
}

func evaluateRules(config Config, loc *time.Location) {
	now := time.Now().In(loc)
	currentTimeStr := now.Format("15:04")
	log.Printf("--- Evaluating Rules at %s ---", currentTimeStr)

	for _, rule := range config.Rules {
		isSleepingTime := isTimeInWindow(currentTimeStr, rule.SleepWindow.Start, rule.SleepWindow.End)
		currentlyAsleep, exists := state[rule.Name]

		if !exists {
			// First time we evaluate this rule, we assume it's awake unless it's sleeping time.
			// Actually, to be safe and enforce state, let's just trigger the action on first run 
			// if we want it to sleep, or if we want it to be awake. But we don't know the actual state.
			// We will just force the state.
		}

		if isSleepingTime && (!exists || !currentlyAsleep) {
			log.Printf("[Rule: %s] Time %s is within sleep window (%s - %s). Enforcing SLEEP.", rule.Name, currentTimeStr, rule.SleepWindow.Start, rule.SleepWindow.End)
			if err := toggleServerless(rule, true); err != nil {
				log.Printf("[Rule: %s] Error sleeping: %v", rule.Name, err)
			} else {
				state[rule.Name] = true
			}
		} else if !isSleepingTime && (!exists || currentlyAsleep) {
			log.Printf("[Rule: %s] Time %s is outside sleep window (%s - %s). Enforcing WAKE.", rule.Name, currentTimeStr, rule.SleepWindow.Start, rule.SleepWindow.End)
			if err := toggleServerless(rule, false); err != nil {
				log.Printf("[Rule: %s] Error waking: %v", rule.Name, err)
			} else {
				state[rule.Name] = false
			}
		} else {
			// State matches expected schedule
			log.Printf("[Rule: %s] No action needed. Expected asleep: %v, Current state asleep: %v", rule.Name, isSleepingTime, state[rule.Name])
		}
	}
}

// isTimeInWindow checks if Current time ("HH:MM") is between Start and End.
// Handles crossing midnight (e.g., Start: 22:00, End: 06:00).
func isTimeInWindow(current, start, end string) bool {
	if start <= end {
		// Normal window: e.g. 02:00 to 08:00
		return current >= start && current < end
	}
	// Crossing midnight: e.g. 22:00 to 06:00
	return current >= start || current < end
}

// toggleServerless applies the change to one or multiple services.
func toggleServerless(rule Rule, sleep bool) error {
	var serviceIDs []string

	if rule.ServiceID != nil && *rule.ServiceID != "" {
		serviceIDs = append(serviceIDs, *rule.ServiceID)
	} else {
		// Need to fetch all services for the environment
		fetchedServices, err := fetchServicesForEnvironment(rule.EnvironmentID)
		if err != nil {
			return fmt.Errorf("failed to fetch services for environment: %v", err)
		}
		serviceIDs = fetchedServices
	}

	if len(serviceIDs) == 0 {
		return fmt.Errorf("no services found to toggle")
	}

	for _, sID := range serviceIDs {
		// 1. Update instance (set sleepApplication)
		log.Printf("[Rule: %s] Setting sleepApplication=%v for service %s in env %s", rule.Name, sleep, sID, rule.EnvironmentID)
		if err := executeServiceInstanceUpdate(rule.EnvironmentID, sID, sleep); err != nil {
			log.Printf("Error updating service %s: %v", sID, err)
			continue // try next
		}

		// 2. Deploy to apply changes
		log.Printf("[Rule: %s] Deploying service %s in env %s to apply changes", rule.Name, sID, rule.EnvironmentID)
		if err := executeServiceInstanceDeploy(rule.EnvironmentID, sID); err != nil {
			log.Printf("Error deploying service %s: %v", sID, err)
			continue
		}
	}

	return nil
}

// --- GraphQL Operations ---

func fetchServicesForEnvironment(envID string) ([]string, error) {
	query := `query($id: String!) { environment(id: $id) { serviceInstances { serviceId } } }`
	vars := map[string]interface{}{"id": envID}
	
	respData, err := executeGraphQL(query, vars)
	if err != nil {
		return nil, err
	}

	// Dynamic parsing of JSON response
	var result struct {
		Data struct {
			Environment struct {
				ServiceInstances []struct {
					ServiceID string `json:"serviceId"`
				} `json:"serviceInstances"`
			} `json:"environment"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("failed to parse environment response: %v", err)
	}

	var sIDs []string
	for _, instance := range result.Data.Environment.ServiceInstances {
		sIDs = append(sIDs, instance.ServiceID)
	}

	return sIDs, nil
}

func executeServiceInstanceUpdate(envID, serviceID string, sleep bool) error {
	query := `mutation($environmentId: String!, $serviceId: String!, $input: ServiceInstanceUpdateInput!) {
		serviceInstanceUpdate(environmentId: $environmentId, serviceId: $serviceId, input: $input)
	}`
	
	vars := map[string]interface{}{
		"environmentId": envID,
		"serviceId":     serviceID,
		"input": map[string]interface{}{
			"sleepApplication": sleep,
		},
	}

	_, err := executeGraphQL(query, vars)
	return err
}

func executeServiceInstanceDeploy(envID, serviceID string) error {
	query := `mutation($environmentId: String!, $serviceId: String!) {
		serviceInstanceDeployV2(environmentId: $environmentId, serviceId: $serviceId)
	}`
	
	vars := map[string]interface{}{
		"environmentId": envID,
		"serviceId":     serviceID,
	}

	_, err := executeGraphQL(query, vars)
	return err
}

func executeGraphQL(query string, variables map[string]interface{}) ([]byte, error) {
	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://backboard.railway.com/graphql/v2", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+railwayToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Check for GraphQL errors
	var gqlError struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(bodyBytes, &gqlError); err == nil && len(gqlError.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL Exception: %s (Raw: %s)", gqlError.Errors[0].Message, string(bodyBytes))
	}

	return bodyBytes, nil
}
