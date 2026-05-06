package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// EngineStatus represents the status of a single engine
type EngineStatus struct {
	Symbol  string `json:"symbol"`
	Port    int    `json:"port"`
	Healthy bool   `json:"healthy"`
	URL     string `json:"url"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status      string         `json:"status"`
	TotalEngines int           `json:"totalEngines"`
	HealthyEngines int         `json:"healthyEngines"`
	Engines     []EngineStatus `json:"engines"`
}

// Health checks the health of all running engines
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if engineManager == nil {
		// No distributed mode - check single engine
		resp, err := http.Get("http://localhost:6060/status")
		if err != nil {
			json.NewEncoder(w).Encode(HealthResponse{
				Status:         "degraded",
				TotalEngines:   1,
				HealthyEngines: 0,
				Engines: []EngineStatus{{
					Symbol:  "default",
					Port:    6060,
					Healthy: false,
					URL:     "http://localhost:6060",
				}},
			})
			return
		}
		resp.Body.Close()

		json.NewEncoder(w).Encode(HealthResponse{
			Status:         "healthy",
			TotalEngines:   1,
			HealthyEngines: 1,
			Engines: []EngineStatus{{
				Symbol:  "default",
				Port:    6060,
				Healthy: true,
				URL:     "http://localhost:6060",
			}},
		})
		return
	}

	// Distributed mode - check all engines
	healthResults := engineManager.HealthCheck()
	engines := engineManager.GetAllEngines()

	var statuses []EngineStatus
	healthyCount := 0

	for symbol, info := range engines {
		healthy := healthResults[symbol]
		if healthy {
			healthyCount++
		}

		statuses = append(statuses, EngineStatus{
			Symbol:  symbol,
			Port:    info.Port,
			Healthy: healthy,
			URL:     fmt.Sprintf("http://localhost:%d", info.Port),
		})
	}

	status := "healthy"
	if healthyCount == 0 && len(engines) > 0 {
		status = "unhealthy"
	} else if healthyCount < len(engines) {
		status = "degraded"
	}

	response := HealthResponse{
		Status:         status,
		TotalEngines:   len(engines),
		HealthyEngines: healthyCount,
		Engines:        statuses,
	}

	json.NewEncoder(w).Encode(response)
}

// EnginesResponse represents the response for the engines endpoint
type EnginesResponse struct {
	Count   int                    `json:"count"`
	Mapping map[string]int         `json:"mapping"`
	Engines []EngineStatus         `json:"engines"`
}

// Engines returns information about all registered engines
func Engines(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if engineManager == nil {
		json.NewEncoder(w).Encode(EnginesResponse{
			Count:   0,
			Mapping: map[string]int{},
			Engines: []EngineStatus{},
		})
		return
	}

	engines := engineManager.GetAllEngines()
	mapping := engineManager.GetMapping()
	healthResults := engineManager.HealthCheck()

	var statuses []EngineStatus
	for symbol, info := range engines {
		statuses = append(statuses, EngineStatus{
			Symbol:  symbol,
			Port:    info.Port,
			Healthy: healthResults[symbol],
			URL:     "http://localhost:" + string(rune(info.Port)),
		})
	}

	response := EnginesResponse{
		Count:   len(engines),
		Mapping: mapping,
		Engines: statuses,
	}

	log.Debugf("Engines endpoint: %d engines registered", len(engines))
	json.NewEncoder(w).Encode(response)
}
