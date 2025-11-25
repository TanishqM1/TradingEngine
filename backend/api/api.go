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
	TradeType string `json::"tradetype"` // GTILLCANCEL or FILLANDKILL
	Side      string `json::"side"`      // BUY or SELL
	Price     int    `json::"price"`     // INT
	Quantity  int    `json::"quantity"`  // INT
	Name      string `json::"name"`      // NAME
}

type CancelFields struct {
	OrderId int    `json::orderID` // OrderId
	Book    string `json::name`    // book
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
