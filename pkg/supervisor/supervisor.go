package supervisor

import (
	"fmt"
	"log"
	"time"

	"github.com/ebravofm/railway-supervisor/pkg/railway"
)

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

type Supervisor struct {
	Config        Config
	RailwayClient *railway.Client
	Location      *time.Location
	State         map[string]bool
}

func NewSupervisor(config Config, client *railway.Client, loc *time.Location) *Supervisor {
	return &Supervisor{
		Config:        config,
		RailwayClient: client,
		Location:      loc,
		State:         make(map[string]bool),
	}
}

func (s *Supervisor) Start() {
	ticker := time.NewTicker(time.Duration(s.Config.CheckIntervalMinutes) * time.Minute)
	defer ticker.Stop()

	s.EvaluateAllRules() // Run immediately

	for range ticker.C {
		s.EvaluateAllRules()
	}
}

func (s *Supervisor) EvaluateAllRules() {
	now := time.Now().In(s.Location)
	currentTimeStr := now.Format("15:04")
	log.Printf("--- Evaluating Rules at %s ---", currentTimeStr)

	for _, rule := range s.Config.Rules {
		isSleepingTime := s.isTimeInWindow(currentTimeStr, rule.SleepWindow.Start, rule.SleepWindow.End)
		currentlyAsleep, exists := s.State[rule.Name]

		if isSleepingTime && (!exists || !currentlyAsleep) {
			log.Printf("[Rule: %s] Enforcing SLEEP.", rule.Name)
			if err := s.ToggleServerless(rule, true); err != nil {
				log.Printf("[Rule: %s] Error sleeping: %v", rule.Name, err)
			} else {
				s.State[rule.Name] = true
			}
		} else if !isSleepingTime && (!exists || currentlyAsleep) {
			log.Printf("[Rule: %s] Enforcing WAKE.", rule.Name)
			if err := s.ToggleServerless(rule, false); err != nil {
				log.Printf("[Rule: %s] Error waking: %v", rule.Name, err)
			} else {
				s.State[rule.Name] = false
			}
		} else {
			log.Printf("[Rule: %s] Expected asleep: %v, No action needed.", rule.Name, isSleepingTime)
		}
	}
}

func (s *Supervisor) isTimeInWindow(current, start, end string) bool {
	if start <= end {
		return current >= start && current < end
	}
	return current >= start || current < end
}

func (s *Supervisor) ToggleServerless(rule Rule, sleep bool) error {
	var serviceIDs []string

	if rule.ServiceID != nil && *rule.ServiceID != "" {
		serviceIDs = append(serviceIDs, *rule.ServiceID)
	} else {
		// Environment level toggle
		log.Printf("[Rule: %s] Fetching all services for environment %s", rule.Name, rule.EnvironmentID)
		fetchedServices, err := s.RailwayClient.FetchServicesForEnvironment(rule.EnvironmentID)
		if err != nil {
			return fmt.Errorf("failed to fetch services for environment: %v", err)
		}
		serviceIDs = fetchedServices
		log.Printf("[Rule: %s] Found %d services in environment %s", rule.Name, len(serviceIDs), rule.EnvironmentID)
	}

	for _, sID := range serviceIDs {
		// 1. Update instance
		log.Printf("[Rule: %s] Setting sleepApplication=%v for service %s", rule.Name, sleep, sID)
		if err := s.RailwayClient.ExecuteServiceInstanceUpdate(rule.EnvironmentID, sID, sleep); err != nil {
			log.Printf("Error updating service %s: %v", sID, err)
			continue
		}

		// 2. Deploy to apply
		if err := s.RailwayClient.ExecuteServiceInstanceDeploy(rule.EnvironmentID, sID); err != nil {
			log.Printf("Error deploying service %s: %v", sID, err)
		}
	}
	return nil
}
