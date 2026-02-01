package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/TanishqM1/Orderbook/internal/handlers"
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

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register our service
	pb.RegisterOrderServiceServer(grpcServer, handlers.NewGRPCServer())

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
