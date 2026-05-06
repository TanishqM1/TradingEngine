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

func Trade(w http.ResponseWriter, r *http.Request) {
	var params = api.AddFields{}
	err := json.NewDecoder(r.Body).Decode(&params)

	if err != nil {
		log.Error(err)
		api.HandleRequestError(w, err)
		return
	}

	orderId := api.GetNextOrderId()

	urlValues := url.Values{}
	urlValues.Set("orderid", strconv.FormatUint(orderId, 10))
	urlValues.Set("tradetype", params.TradeType)
	urlValues.Set("side", params.Side)
	urlValues.Set("price", strconv.Itoa(params.Price))
	urlValues.Set("quantity", strconv.Itoa(params.Quantity))
	urlValues.Set("book", params.Name)

	log.Debugf("Processing trade request: %s", urlValues.Encode())

	// Try distributed mode first
	if balancer != nil {
		// Check if we have an engine for this symbol
		if _, exists := balancer.GetEngineURL(params.Name); exists {
			resp, err := balancer.ForwardTrade(urlValues)
			if err != nil {
				log.Errorf("Failed to forward trade via load balancer: %v", err)
				api.HandleInternalError(w)
				return
			}
			defer resp.Body.Close()

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)

			if _, err := io.Copy(w, resp.Body); err != nil {
				log.Errorf("Failed to proxy response body: %v", err)
			}

			fmt.Printf("\nProcessed order ID %d for %s via load balancer", orderId, params.Name)
			return
		}
	}

	// Fallback to single engine mode
	tradeSingleEngine(w, urlValues, orderId)
}

// tradeSingleEngine forwards trade to the default single engine
func tradeSingleEngine(w http.ResponseWriter, urlValues url.Values, orderId uint64) {
	reqBody := strings.NewReader(urlValues.Encode())
	client := http.Client{}

	cppServerURL := "http://localhost:6060/trade"

	log.Debugf("Forwarding trade request to C++ engine: %s with body: %s", cppServerURL, urlValues.Encode())

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
		log.Errorf("Failed to proxy response body: %v", err)
	}

	fmt.Printf("\nProcessed new order with ID: %d", orderId)
}
