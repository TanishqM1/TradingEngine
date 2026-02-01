package handlers

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/TanishqM1/Orderbook/api"
	pb "github.com/TanishqM1/Orderbook/internal/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Trade handles the gRPC Trade request
func (s *GRPCServer) Trade(ctx context.Context, req *pb.TradeRequest) (*pb.TradeResponse, error) {
	log.Debugf("Received Trade request: %+v", req)

	// Validate request
	if req.Tradetype == "" || req.Side == "" || req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "tradetype, side, and name fields are required")
	}
	if req.Price <= 0 || req.Quantity <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "price and quantity must be positive")
	}

	// Generate order ID
	orderId := api.GetNextOrderId()

	// Prepare request to C++ engine
	urlValues := url.Values{}
	urlValues.Set("orderid", strconv.FormatUint(orderId, 10))
	urlValues.Set("tradetype", req.Tradetype)
	urlValues.Set("side", req.Side)
	urlValues.Set("price", strconv.Itoa(int(req.Price)))
	urlValues.Set("quantity", strconv.Itoa(int(req.Quantity)))
	urlValues.Set("book", req.Name)

	reqBody := strings.NewReader(urlValues.Encode())
	client := http.Client{}
	cppServerURL := "http://localhost:6060/trade"

	log.Debugf("Forwarding trade request to C++ engine: %s with body: %s", cppServerURL, urlValues.Encode())

	// Create HTTP request to C++ engine
	httpReq, err := http.NewRequestWithContext(ctx, "POST", cppServerURL, reqBody)
	if err != nil {
		log.Errorf("Failed to create C++ request: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to create request to C++ engine")
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request to C++ engine
	cppResp, err := client.Do(httpReq)
	if err != nil {
		log.Errorf("Failed to connect to C++ engine at :6060. Is the C++ server running? Error: %v", err)
		return nil, status.Errorf(codes.Unavailable, "failed to connect to C++ engine: %v", err)
	}
	defer cppResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(cppResp.Body)
	if err != nil {
		log.Errorf("Failed to read response body: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to read C++ engine response")
	}

	log.Infof("Processed new order with ID: %d", orderId)

	return &pb.TradeResponse{
		OrderId:    orderId,
		StatusCode: int32(cppResp.StatusCode),
		Body:       string(respBody),
	}, nil
}
