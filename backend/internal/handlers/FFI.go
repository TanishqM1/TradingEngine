package handlers

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/TanishqM1/Orderbook/api"
	log "github.com/sirupsen/logrus"
)

var wg = sync.WaitGroup{}

func Trade(w http.ResponseWriter, r *http.Request) {
	var params = api.Fields{}
	err := json.NewDecoder(r.Body).Decode(&params)
	// automatically parses the json. the json is in this schema:
	// Type     string `json:"type"`
	// Side     string `json::"side"`
	// Price    string `json::"price"`
	// Quantity string `json::"quantity"`

	if err != nil {
		log.Error(err)
		api.HandleRequestError(w, err)
		return
	}
	// params to send over.

	// now params has all the values from our JSON. We need to send this over to our C++ engine via FFI.

	// FLOW:

	// Start Engine
	// Invoke CreateOrderBook() and allocate all orderbook memory addresses
	// use name_ to resolve address of books
	// using the rest of the information and the book address, invoke AddOrder().
}
