package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/TanishqM1/Orderbook/api"
	log "github.com/sirupsen/logrus"
)

// Simulation handles the full simulation workflow:
// 1. Parse config from frontend
// 2. Reset the C++ engine
// 3. Generate random orders
// 4. Send batch to C++ engine
// 5. Return results with timing
func Simulation(w http.ResponseWriter, r *http.Request) {
	// Parse simulation request from frontend
	var params api.SimulationRequest
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		log.Error(err)
		api.HandleRequestError(w, err)
		return
	}

	if len(params.Stocks) == 0 {
		api.HandleRequestError(w, fmt.Errorf("no stocks provided in simulation request"))
		return
	}

	// Validate stock configs
	for _, stock := range params.Stocks {
		if stock.Symbol == "" {
			api.HandleRequestError(w, fmt.Errorf("stock symbol cannot be empty"))
			return
		}
		if stock.PriceMin > stock.PriceMax {
			api.HandleRequestError(w, fmt.Errorf("priceMin cannot be greater than priceMax for %s", stock.Symbol))
			return
		}
		if stock.QuantityMin > stock.QuantityMax {
			api.HandleRequestError(w, fmt.Errorf("quantityMin cannot be greater than quantityMax for %s", stock.Symbol))
			return
		}
	}

	client := http.Client{}

	// Step 1: Reset the C++ engine
	resetReq, err := http.NewRequest("POST", "http://localhost:6060/reset", nil)
	if err != nil {
		log.Errorf("Failed to create reset request: %v", err)
		api.HandleInternalError(w)
		return
	}

	resetResp, err := client.Do(resetReq)
	if err != nil {
		log.Errorf("Failed to reset C++ engine: %v", err)
		api.HandleInternalError(w)
		return
	}
	resetResp.Body.Close()

	if resetResp.StatusCode != 200 {
		log.Errorf("C++ engine reset failed with status: %d", resetResp.StatusCode)
		api.HandleInternalError(w)
		return
	}

	// Step 2: Generate random orders
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var orders []api.BatchOrder
	symbolSet := make(map[string]bool) // Track which symbols we're simulating

	for _, stock := range params.Stocks {
		symbolSet[stock.Symbol] = true

		// Generate bid orders
		for i := 0; i < stock.NumBids; i++ {
			price := stock.PriceMin
			if stock.PriceMax > stock.PriceMin {
				price = stock.PriceMin + rng.Intn(stock.PriceMax-stock.PriceMin+1)
			}

			quantity := stock.QuantityMin
			if stock.QuantityMax > stock.QuantityMin {
				quantity = stock.QuantityMin + rng.Intn(stock.QuantityMax-stock.QuantityMin+1)
			}

			orders = append(orders, api.BatchOrder{
				OrderId:   api.GetNextOrderId(),
				Book:      stock.Symbol,
				TradeType: "GTC",
				Side:      "BUY",
				Price:     price,
				Quantity:  quantity,
			})
		}

		// Generate ask orders
		for i := 0; i < stock.NumAsks; i++ {
			price := stock.PriceMin
			if stock.PriceMax > stock.PriceMin {
				price = stock.PriceMin + rng.Intn(stock.PriceMax-stock.PriceMin+1)
			}

			quantity := stock.QuantityMin
			if stock.QuantityMax > stock.QuantityMin {
				quantity = stock.QuantityMin + rng.Intn(stock.QuantityMax-stock.QuantityMin+1)
			}

			orders = append(orders, api.BatchOrder{
				OrderId:   api.GetNextOrderId(),
				Book:      stock.Symbol,
				TradeType: "GTC",
				Side:      "SELL",
				Price:     price,
				Quantity:  quantity,
			})
		}
	}

	// Step 3: Send batch to C++ engine and time it
	batchRequest := api.BatchRequest{Orders: orders}
	batchBody, err := json.Marshal(batchRequest)
	if err != nil {
		log.Errorf("Failed to marshal batch request: %v", err)
		api.HandleInternalError(w)
		return
	}

	log.Debugf("Sending batch of %d orders to C++ engine", len(orders))

	startTime := time.Now()

	batchReq, err := http.NewRequest("POST", "http://localhost:6060/batch", bytes.NewReader(batchBody))
	if err != nil {
		log.Errorf("Failed to create batch request: %v", err)
		api.HandleInternalError(w)
		return
	}
	batchReq.Header.Set("Content-Type", "application/json")

	batchResp, err := client.Do(batchReq)
	if err != nil {
		log.Errorf("Failed to send batch to C++ engine: %v", err)
		api.HandleInternalError(w)
		return
	}
	defer batchResp.Body.Close()

	executionTime := time.Since(startTime)

	if batchResp.StatusCode != 200 {
		body, _ := io.ReadAll(batchResp.Body)
		log.Errorf("C++ batch processing failed with status %d: %s", batchResp.StatusCode, string(body))
		api.HandleInternalError(w)
		return
	}

	// Step 4: Parse C++ response
	var cppResponse api.CppBatchResponse
	err = json.NewDecoder(batchResp.Body).Decode(&cppResponse)
	if err != nil {
		log.Errorf("Failed to parse C++ batch response: %v", err)
		api.HandleInternalError(w)
		return
	}

	// Step 5: Transform to frontend response
	var results []api.StockResult
	for symbol := range symbolSet {
		bookResult, exists := cppResponse.Results[symbol]

		result := api.StockResult{
			Symbol: symbol,
		}

		if exists {
			result.TradesExecuted = bookResult.TradesExecuted
			result.VolumeTraded = bookResult.VolumeTraded
			result.RemainingBids = bookResult.RemainingBids
			result.RemainingAsks = bookResult.RemainingAsks
			result.BidLevels = bookResult.BidLevels
			result.AskLevels = bookResult.AskLevels

			// Convert -1 (no price) to nil for JSON
			if bookResult.BestBidPrice >= 0 {
				result.BestBidPrice = &bookResult.BestBidPrice
			}
			if bookResult.BestAskPrice >= 0 {
				result.BestAskPrice = &bookResult.BestAskPrice
			}
		}

		results = append(results, result)
	}

	response := api.SimulationResponse{
		ExecutionTimeMs:      float64(executionTime.Microseconds()) / 1000.0,
		TotalOrdersProcessed: cppResponse.ProcessedCount,
		Results:              results,
	}

	// Send response to frontend
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("Failed to encode simulation response: %v", err)
	}

	fmt.Printf("\nSimulation complete: %d orders processed in %.2fms\n",
		len(orders), response.ExecutionTimeMs)
}
