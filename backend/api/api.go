package api

import (
	"encoding/json"
	"net/http"
)

type Error struct {
	Code    int
	Message string
}

// orders need type, side, price, quantity
type Fields struct {
	TradeType string `json:"tradetype"` // GTILLCANCEL or FILLANDKILL
	Side      string `json::"side"`     // BUY or SELL
	Price     int    `json::"price"`    // INT
	Quantity  int    `json::"quantity"` // INT
	Name      string `json::"name"`     // NAME
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
