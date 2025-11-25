package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

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

	fmt.Println(params)

	if params.OrderId == 0 {
		api.HandleRequestError(w, fmt.Errorf("orderId field is required, and cannot be zero"))
	}

	URL_Values := url.Values{}
	URL_Values.Set("orderid", strconv.FormatUint(uint64(params.OrderId), 10))
	URL_Values.Set("book", params.Book)

	client := http.Client{}

	cppServerURL := fmt.Sprintf("http://localhost:6060/cancel%s", URL_Values.Encode())

	log.Debugf("Forwarding cancel request to C++ engine: %s", cppServerURL)

	cppReq, err := http.NewRequest("POST", cppServerURL, nil)
	if err != nil {
		log.Errorf("Failed to create C++ request: %v", err)
		api.HandleInternalError(w)
		return
	}

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
		log.Errorf("Failed to cop proxy response body: %v", err)
	}

	fmt.Printf("\nAttempted to Cancel Order: %d", params.OrderId)
}
