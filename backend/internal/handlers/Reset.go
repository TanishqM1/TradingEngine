package handlers

import (
	"io"
	"net/http"

	"github.com/TanishqM1/Orderbook/api"
	log "github.com/sirupsen/logrus"
)

// Reset proxies the POST request to the C++ engine to clear all orderbooks.
func Reset(w http.ResponseWriter, r *http.Request) {
	client := http.Client{}

	// C++ server URL for reset
	cppServerURL := "http://localhost:6060/reset"

	log.Debugf("Forwarding reset request to C++ engine: %s", cppServerURL)

	// Create a new POST request (empty body)
	cppReq, err := http.NewRequest("POST", cppServerURL, nil)
	if err != nil {
		log.Errorf("Failed to create C++ request: %v", err)
		api.HandleInternalError(w)
		return
	}

	// Execute the request
	cppResp, err := client.Do(cppReq)
	if err != nil {
		log.Errorf("Failed to connect to C++ engine at :6060. Is the C++ server running? Error: %v", err)
		api.HandleInternalError(w)
		return
	}
	defer cppResp.Body.Close()

	// Proxy the response back to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(cppResp.StatusCode)

	if _, err := io.Copy(w, cppResp.Body); err != nil {
		log.Errorf("Failed to copy proxy response body: %v", err)
	}
}
