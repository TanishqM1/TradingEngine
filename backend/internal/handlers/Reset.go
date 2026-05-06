package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/TanishqM1/Orderbook/api"
	log "github.com/sirupsen/logrus"
)

// ResetResponse represents the response for the reset endpoint
type ResetResponse struct {
	Message      string `json:"message"`
	EnginesReset int    `json:"enginesReset"`
}

// Reset resets all orderbooks across all engines.
func Reset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// If not in distributed mode, use single engine
	if engineManager == nil || balancer == nil {
		resetSingleEngine(w)
		return
	}

	// Get all engine URLs
	engineURLs := balancer.GetAllEngineURLs()
	if len(engineURLs) == 0 {
		// No engines running
		json.NewEncoder(w).Encode(ResetResponse{
			Message:      "No engines running",
			EnginesReset: 0,
		})
		return
	}

	// Reset all engines in parallel
	client := &http.Client{Timeout: 5 * time.Second}
	var wg sync.WaitGroup
	resetCount := 0
	var countMu sync.Mutex

	for symbol, baseURL := range engineURLs {
		wg.Add(1)
		go func(sym, url string) {
			defer wg.Done()

			resp, err := client.Post(url+"/reset", "application/json", nil)
			if err != nil {
				log.Warnf("Failed to reset engine for %s: %v", sym, err)
				return
			}
			resp.Body.Close()

			if resp.StatusCode == 200 {
				countMu.Lock()
				resetCount++
				countMu.Unlock()
				log.Infof("Reset engine for %s", sym)
			}
		}(symbol, baseURL)
	}

	wg.Wait()

	json.NewEncoder(w).Encode(ResetResponse{
		Message:      "All engines reset",
		EnginesReset: resetCount,
	})

	log.Infof("Reset %d engines", resetCount)
}

// resetSingleEngine resets the default single engine
func resetSingleEngine(w http.ResponseWriter) {
	client := http.Client{Timeout: 5 * time.Second}

	cppServerURL := "http://localhost:6060/reset"

	log.Debugf("Forwarding reset request to C++ engine: %s", cppServerURL)

	cppReq, err := http.NewRequest("POST", cppServerURL, nil)
	if err != nil {
		log.Errorf("Failed to create C++ request: %v", err)
		api.HandleInternalError(w)
		return
	}

	cppResp, err := client.Do(cppReq)
	if err != nil {
		log.Errorf("Failed to connect to C++ engine at :6060. Is the C++ server running? Error: %v", err)
		api.HandleInternalError(w)
		return
	}
	defer cppResp.Body.Close()

	w.WriteHeader(cppResp.StatusCode)

	if _, err := io.Copy(w, cppResp.Body); err != nil {
		log.Errorf("Failed to copy proxy response body: %v", err)
	}
}
