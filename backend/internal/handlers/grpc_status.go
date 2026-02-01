package handlers

import (
	"context"
	"io"
	"net/http"

	pb "github.com/TanishqM1/Orderbook/internal/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Status handles the gRPC Status request
func (s *GRPCServer) Status(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
	log.Debug("Received Status request")

	client := http.Client{}
	cppServerURL := "http://localhost:6060/status"

	log.Debugf("Forwarding status request to C++ engine: %s", cppServerURL)

	// Create HTTP request to C++ engine
	httpReq, err := http.NewRequestWithContext(ctx, "GET", cppServerURL, nil)
	if err != nil {
		log.Errorf("Failed to create C++ request: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to create request to C++ engine")
	}

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

	log.Debug("Status request completed successfully")

	return &pb.StatusResponse{
		StatusCode: int32(cppResp.StatusCode),
		Body:       string(respBody),
	}, nil
}
