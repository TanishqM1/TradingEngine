package handlers

import (
	"github.com/TanishqM1/Orderbook/internal/loadbalancer"
	pb "github.com/TanishqM1/Orderbook/internal/pb"
)

// GRPCServer implements the OrderService gRPC server
type GRPCServer struct {
	pb.UnimplementedOrderServiceServer
	balancer *loadbalancer.Balancer
}

// NewGRPCServer creates a new gRPC server instance
func NewGRPCServer(balancer *loadbalancer.Balancer) *GRPCServer {
	return &GRPCServer{
		balancer: balancer,
	}
}
