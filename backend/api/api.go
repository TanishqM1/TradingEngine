package api

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

type Error struct {
	Code    int
	Message string
}

var OrderIdCounter uint64 = 0

func GetNextOrderId() uint64 {
	return atomic.AddUint64(&OrderIdCounter, 1)
}

// orders need type, side, price, quantity
type AddFields struct {
	TradeType string `json:"tradetype"` // GTILLCANCEL or FILLANDKILL
	Side      string `json:"side"`      // BUY or SELL
	Price     int    `json:"price"`     // INT
	Quantity  int    `json:"quantity"`  // INT
	Name      string `json:"name"`      // NAME
}

type CancelFields struct {
	OrderId int    `json:"orderID"` // OrderId
	Book    string `json:"name"`    // book
}

// Simulation configuration for a single stock
type StockSimConfig struct {
	Symbol      string `json:"symbol"`
	NumBids     int    `json:"numBids"`
	NumAsks     int    `json:"numAsks"`
	PriceMin    int    `json:"priceMin"`
	PriceMax    int    `json:"priceMax"`
	QuantityMin int    `json:"quantityMin"`
	QuantityMax int    `json:"quantityMax"`
}

// Full simulation request from frontend
type SimulationRequest struct {
	Stocks []StockSimConfig `json:"stocks"`
}

// Single stock result
type StockResult struct {
	Symbol         string `json:"symbol"`
	TradesExecuted int    `json:"tradesExecuted"`
	VolumeTraded   int64  `json:"volumeTraded"`
	RemainingBids  int    `json:"remainingBids"`
	RemainingAsks  int    `json:"remainingAsks"`
	BestBidPrice   *int   `json:"bestBidPrice"`
	BestAskPrice   *int   `json:"bestAskPrice"`
	BidLevels      int    `json:"bidLevels"`
	AskLevels      int    `json:"askLevels"`
}

// Full simulation response to frontend
type SimulationResponse struct {
	ExecutionTimeMs      float64       `json:"executionTimeMs"`
	TotalOrdersProcessed int           `json:"totalOrdersProcessed"`
	Results              []StockResult `json:"results"`
}

// Order for batch request to C++
type BatchOrder struct {
	OrderId   uint64 `json:"orderid"`
	Book      string `json:"book"`
	TradeType string `json:"tradetype"`
	Side      string `json:"side"`
	Price     int    `json:"price"`
	Quantity  int    `json:"quantity"`
}

// Batch request to C++ engine
type BatchRequest struct {
	Orders []BatchOrder `json:"orders"`
}

// C++ batch response - per book result
type CppBookResult struct {
	TradesExecuted int   `json:"tradesExecuted"`
	VolumeTraded   int64 `json:"volumeTraded"`
	RemainingBids  int   `json:"remainingBids"`
	RemainingAsks  int   `json:"remainingAsks"`
	BestBidPrice   int   `json:"bestBidPrice"`
	BestAskPrice   int   `json:"bestAskPrice"`
	BidLevels      int   `json:"bidLevels"`
	AskLevels      int   `json:"askLevels"`
}

// C++ batch response
type CppBatchResponse struct {
	ProcessedCount int                      `json:"processedCount"`
	Results        map[string]CppBookResult `json:"results"`
}

func writeError(w http.ResponseWriter, message string, code int) {
	resp := Error{
		Code:    code,
		Message: message,
	}

	// response header (error case)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	// json for output (using Encode)
	json.NewEncoder(w).Encode(resp)
}

// wrapper function for writeError. This lets us use it with two different use cases.

var (
	// error is not internal.
	HandleRequestError = func(w http.ResponseWriter, err error) {
		writeError(w, err.Error(), http.StatusBadRequest)
	}
	// internal error. we log it speerately!
	HandleInternalError = func(w http.ResponseWriter) {
		writeError(w, "An Unexpected Error Occured", http.StatusInternalServerError)
	}
)
