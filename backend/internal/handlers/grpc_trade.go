package handlers

import (
	"context"
	"net/url"
	"strconv"

	"github.com/TanishqM1/Orderbook/api"
	pb "github.com/TanishqM1/Orderbook/internal/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Trade handles the gRPC Trade request with minimal latency
// Critical path: validate -> generate ID -> fire request -> return
// Logging happens in background after request is already in flight
func (s *GRPCServer) Trade(ctx context.Context, req *pb.TradeRequest) (*pb.TradeResponse, error) {
	// Fast validation (only required fields, no expensive checks)
	if req.Tradetype == "" || req.Side == "" || req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "tradetype, side, and name fields are required")
	}
	if req.Price <= 0 || req.Quantity <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "price and quantity must be positive")
	}

	// Generate order ID (atomic increment - ~nanoseconds)
	orderId := api.GetNextOrderId()

	// Build form data (minimal allocations)
	form := url.Values{}
	form.Set("orderid", strconv.FormatUint(orderId, 10))
	form.Set("tradetype", req.Tradetype)
	form.Set("side", req.Side)
	form.Set("price", strconv.Itoa(int(req.Price)))
	form.Set("quantity", strconv.Itoa(int(req.Quantity)))
	form.Set("book", req.Name)

	// FIRE REQUEST IMMEDIATELY (non-blocking, request already in flight)
	s.balancer.FireTrade(form)

	// Log in background (after request is already sent)
	go func() {
		log.Infof("Trade order fired: ID=%d book=%s side=%s type=%s price=%d qty=%d",
			orderId, req.Name, req.Side, req.Tradetype, req.Price, req.Quantity)
	}()

	// Return success immediately
	return &pb.TradeResponse{
		OrderId:    orderId,
		StatusCode: 202, // HTTP 202 Accepted
		Body:       "Order queued",
	}, nil
}
