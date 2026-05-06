package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/TanishqM1/Orderbook/api"
	"github.com/TanishqM1/Orderbook/internal/loadbalancer"
	log "github.com/sirupsen/logrus"
)

// Simulation handles the distributed simulation workflow:
// 1. Parse config from frontend
// 2. Spawn engines for each unique stock (one engine per symbol)
// 3. Generate random orders grouped by symbol
// 4. Send batches to each engine in parallel
// 5. Aggregate results and return with timing
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
	symbols := make([]string, 0, len(params.Stocks))
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
		symbols = append(symbols, stock.Symbol)
	}

	// Check if distributed mode is available
	if engineManager == nil || balancer == nil {
		log.Warn("Distributed mode not initialized, falling back to single engine")
		simulationSingleEngine(w, params)
		return
	}

	// Step 1: Spawn engines for all symbols in parallel
	log.Infof("Spawning engines for %d symbols: %v", len(symbols), symbols)
	engineInfos, err := engineManager.SpawnEnginesForSymbols(symbols)
	if err != nil {
		log.Errorf("Failed to spawn some engines: %v", err)
		// Continue with engines that were spawned successfully
	}

	if len(engineInfos) == 0 {
		api.HandleRequestError(w, fmt.Errorf("failed to spawn any engines"))
		return
	}

	// Register engines with load balancer
	mapping := engineManager.GetMapping()
	balancer.RegisterEngines(mapping)

	// Step 2: Reset all engines in parallel
	log.Info("Resetting all engines...")
	resetEnginesParallel(symbols)

	// Step 3: Generate orders grouped by symbol
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	ordersBySymbol := make(map[string][]loadbalancer.BatchOrder)

	for _, stock := range params.Stocks {
		var orders []loadbalancer.BatchOrder

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

			orders = append(orders, loadbalancer.BatchOrder{
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

			orders = append(orders, loadbalancer.BatchOrder{
				OrderId:   api.GetNextOrderId(),
				Book:      stock.Symbol,
				TradeType: "GTC",
				Side:      "SELL",
				Price:     price,
				Quantity:  quantity,
			})
		}

		ordersBySymbol[stock.Symbol] = orders
	}

	// Count total orders
	totalOrders := 0
	for _, orders := range ordersBySymbol {
		totalOrders += len(orders)
	}

	log.Infof("Sending %d orders across %d engines in parallel", totalOrders, len(ordersBySymbol))

	// Step 4: Send batches to engines in parallel and time it
	startTime := time.Now()

	batchResults, err := balancer.ForwardBatchParallel(ordersBySymbol)
	if err != nil {
		log.Warnf("Some batch requests failed: %v", err)
	}

	executionTime := time.Since(startTime)

	// Step 5: Aggregate results
	var results []api.StockResult
	processedCount := 0

	for _, stock := range params.Stocks {
		result := api.StockResult{
			Symbol: stock.Symbol,
		}

		if batchResp, exists := batchResults[stock.Symbol]; exists && batchResp != nil {
			processedCount += batchResp.ProcessedCount

			// Get the result for this symbol (there should only be one since each engine handles one symbol)
			if bookResult, ok := batchResp.Results[stock.Symbol]; ok {
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
		}

		results = append(results, result)
	}

	response := api.SimulationResponse{
		ExecutionTimeMs:      float64(executionTime.Microseconds()) / 1000.0,
		TotalOrdersProcessed: processedCount,
		Results:              results,
	}

	// Send response to frontend
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Errorf("Failed to encode simulation response: %v", err)
	}

	fmt.Printf("\nDistributed simulation complete: %d orders across %d engines in %.2fms\n",
		totalOrders, len(ordersBySymbol), response.ExecutionTimeMs)
}

// resetEnginesParallel resets all engines in parallel
func resetEnginesParallel(symbols []string) {
	var wg sync.WaitGroup
	for _, symbol := range symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			if _, err := balancer.ForwardReset(sym); err != nil {
				log.Warnf("Failed to reset engine for %s: %v", sym, err)
			}
		}(symbol)
	}
	wg.Wait()
}

// simulationSingleEngine is the fallback for when distributed mode is not available
func simulationSingleEngine(w http.ResponseWriter, params api.SimulationRequest) {
	log.Info("Running simulation in single-engine mode")

	client := http.Client{Timeout: 30 * time.Second}

	// Reset single engine
	resetResp, err := client.Post("http://localhost:6060/reset", "application/json", nil)
	if err != nil {
		log.Errorf("Failed to reset engine: %v", err)
		api.HandleInternalError(w)
		return
	}
	resetResp.Body.Close()

	// Generate all orders
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var orders []api.BatchOrder

	for _, stock := range params.Stocks {
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

	// Send batch
	batchRequest := api.BatchRequest{Orders: orders}
	batchBody, _ := json.Marshal(batchRequest)

	startTime := time.Now()
	resp, err := client.Post("http://localhost:6060/batch", "application/json", bytes.NewReader(batchBody))
	if err != nil {
		log.Errorf("Batch request failed: %v", err)
		api.HandleInternalError(w)
		return
	}
	defer resp.Body.Close()
	executionTime := time.Since(startTime)

	var cppResponse api.CppBatchResponse
	json.NewDecoder(resp.Body).Decode(&cppResponse)

	// Build response
	var results []api.StockResult
	for _, stock := range params.Stocks {
		result := api.StockResult{Symbol: stock.Symbol}
		if bookResult, exists := cppResponse.Results[stock.Symbol]; exists {
			result.TradesExecuted = bookResult.TradesExecuted
			result.VolumeTraded = bookResult.VolumeTraded
			result.RemainingBids = bookResult.RemainingBids
			result.RemainingAsks = bookResult.RemainingAsks
			result.BidLevels = bookResult.BidLevels
			result.AskLevels = bookResult.AskLevels
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
