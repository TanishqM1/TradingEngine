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

// Status retrieves the full Orderbook state from all engines.
// In distributed mode, it aggregates results from all running engines.
func Status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// If not in distributed mode, use single engine
	if engineManager == nil || balancer == nil {
		statusSingleEngine(w)
		return
	}

	// Get all engine URLs
	engineURLs := balancer.GetAllEngineURLs()
	if len(engineURLs) == 0 {
		// No engines running, return empty status
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
		return
	}

	// Query all engines in parallel
	client := &http.Client{Timeout: 5 * time.Second}
	results := make(map[string]json.RawMessage)
	var resultMu sync.Mutex
	var wg sync.WaitGroup

	for symbol, baseURL := range engineURLs {
		wg.Add(1)
		go func(sym, url string) {
			defer wg.Done()

			resp, err := client.Get(url + "/status")
			if err != nil {
				log.Warnf("Failed to get status from engine for %s: %v", sym, err)
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Warnf("Failed to read status response for %s: %v", sym, err)
				return
			}

			// The response should be a JSON object with the symbol as key
			// Each engine only has one book, so extract that
			var engineStatus map[string]json.RawMessage
			if err := json.Unmarshal(body, &engineStatus); err != nil {
				log.Warnf("Failed to parse status response for %s: %v", sym, err)
				return
			}

			resultMu.Lock()
			// Merge all books from this engine into results
			for book, data := range engineStatus {
				results[book] = data
			}
			resultMu.Unlock()
		}(symbol, baseURL)
	}

	wg.Wait()

	// Write aggregated results
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Errorf("Failed to encode aggregated status: %v", err)
	}
}

// statusSingleEngine is the fallback for single engine mode
func statusSingleEngine(w http.ResponseWriter) {
	client := http.Client{Timeout: 5 * time.Second}

	cppServerURL := "http://localhost:6060/status"
	log.Debugf("Forwarding status request to C++ engine: %s", cppServerURL)

	cppReq, err := http.NewRequest("GET", cppServerURL, nil)
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
