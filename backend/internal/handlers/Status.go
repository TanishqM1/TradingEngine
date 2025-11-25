package handlers

import (
	"io"
	"net/http"

	"github.com/TanishqM1/Orderbook/api"
	log "github.com/sirupsen/logrus"
)

// Status proxies the GET request to the C++ engine to retrieve the full Orderbook state.
func Status(w http.ResponseWriter, r *http.Request) {
	client := http.Client{}

	// C++ server URL for status check
	cppServerURL := "http://localhost:6060/status"

	log.Debugf("Forwarding status request to C++ engine: %s", cppServerURL)

	// 1. Create a new GET request (no body needed)
	cppReq, err := http.NewRequest("GET", cppServerURL, nil)
	if err != nil {
		log.Errorf("Failed to create C++ request: %v", err)
		api.HandleInternalError(w)
		return
	}

	// 2. Reroute request to local C++ server
	cppResp, err := client.Do(cppReq)
	if err != nil {
		log.Errorf("Failed to connect to C++ engine at :6060. Is the C++ server running? Error: %v", err)
		api.HandleInternalError(w)
		return
	}
	defer cppResp.Body.Close()

	// 3. Proxy the response (status and body) back to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(cppResp.StatusCode)

	if _, err := io.Copy(w, cppResp.Body); err != nil {
		log.Errorf("Failed to copy proxy response body: %v", err)
	}
}
