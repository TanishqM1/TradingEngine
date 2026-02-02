package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/TanishqM1/Orderbook/internal/handlers"
	"github.com/TanishqM1/Orderbook/internal/loadbalancer"
	pb "github.com/TanishqM1/Orderbook/internal/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// in this file, I setup the logger and start the gRPC server.

func main() {
	log.SetReportCaller(true)

	// Create TCP listener for gRPC
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen on port 50051: %v", err)
	}

	// Initialize load balancer with engine servers
	engineServers := []string{
		"http://localhost:6060", // Server 0: NVDA
		"http://localhost:6061", // Server 1: AAPL
		"http://localhost:6062", // Server 2: TSLA, MSFT
		"http://localhost:6063", // Server 3: GOOGL
	}

	// Load stock-to-server mapping from config file
	mappingPath := "../../config/stock_mapping.json"
	var balancer *loadbalancer.Balancer

	if mapping, err := loadbalancer.LoadMapping(mappingPath); err == nil {
		balancer = loadbalancer.NewWithMapping(engineServers, mapping)
		log.Infof("Load balancer initialized with %d servers and %d explicit stock mappings",
			len(engineServers), len(mapping))
	} else {
		// Fallback to hash-based routing if mapping file not found
		balancer = loadbalancer.New(engineServers)
		log.Warnf("Stock mapping file not found (%s), using hash-based routing: %v", mappingPath, err)
		log.Infof("Load balancer initialized with %d servers (hash-based routing)", len(engineServers))
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register our service with load balancer
	pb.RegisterOrderServiceServer(grpcServer, handlers.NewGRPCServer(balancer))

	// Register reflection service (useful for tools like grpcurl)
	reflection.Register(grpcServer)

	fmt.Println("Starting gRPC Trading Engine API on :50051")
	log.Info("gRPC server listening on :50051")

	// Start server in a goroutine
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down gRPC server...")
	grpcServer.GracefulStop()
	log.Info("gRPC server stopped")
}
