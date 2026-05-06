package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/TanishqM1/Orderbook/api"
	log "github.com/sirupsen/logrus"
)

func Cancel(w http.ResponseWriter, r *http.Request) {
	var params = api.CancelFields{}
	err := json.NewDecoder(r.Body).Decode(&params)

	if err != nil {
		log.Error(err)
		api.HandleRequestError(w, err)
		return
	}

	fmt.Printf("\nCancel request: %v", params)

	if params.OrderId == 0 {
		api.HandleRequestError(w, fmt.Errorf("orderId field is required, and cannot be zero"))
		return
	}

	urlValues := url.Values{}
	urlValues.Set("orderid", strconv.FormatUint(uint64(params.OrderId), 10))
	urlValues.Set("book", params.Book)

	log.Debugf("Processing cancel request: %s", urlValues.Encode())

	// Try distributed mode first
	if balancer != nil {
		if _, exists := balancer.GetEngineURL(params.Book); exists {
			resp, err := balancer.ForwardCancel(urlValues)
			if err != nil {
				log.Errorf("Failed to forward cancel via load balancer: %v", err)
				api.HandleInternalError(w)
				return
			}
			defer resp.Body.Close()

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)

			if _, err := io.Copy(w, resp.Body); err != nil {
				log.Errorf("Failed to proxy response body: %v", err)
			}

			fmt.Printf("\nCancelled order %d for %s via load balancer", params.OrderId, params.Book)
			return
		}
	}

	// Fallback to single engine mode
	cancelSingleEngine(w, urlValues, params.OrderId)
}

// cancelSingleEngine forwards cancel to the default single engine
func cancelSingleEngine(w http.ResponseWriter, urlValues url.Values, orderId int) {
	reqBody := strings.NewReader(urlValues.Encode())
	client := http.Client{}

	cppServerURL := "http://localhost:6060/cancel"

	log.Debugf("Forwarding cancel request to C++ engine: %s with body: %s", cppServerURL, urlValues.Encode())

	cppReq, err := http.NewRequest("POST", cppServerURL, reqBody)
	if err != nil {
		log.Errorf("Failed to create C++ request: %v", err)
		api.HandleInternalError(w)
		return
	}

	cppReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	cppResp, err := client.Do(cppReq)

	if err != nil {
		log.Errorf("Failed to connect to C++ engine at :6060. Is the C++ server running? Error: %v", err)
		api.HandleInternalError(w)
		return
	}
	defer cppResp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(cppResp.StatusCode)

	if _, err := io.Copy(w, cppResp.Body); err != nil {
		log.Errorf("Failed to copy proxy response body: %v", err)
	}

	fmt.Printf("\nAttempted to Cancel Order: %d", orderId)
}
