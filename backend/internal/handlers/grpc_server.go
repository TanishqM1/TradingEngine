package handlers

import (
	pb "github.com/TanishqM1/Orderbook/internal/pb"
)

// GRPCServer implements the OrderService gRPC server
type GRPCServer struct {
	pb.UnimplementedOrderServiceServer
}

// NewGRPCServer creates a new gRPC server instance
func NewGRPCServer() *GRPCServer {
	return &GRPCServer{}
}
