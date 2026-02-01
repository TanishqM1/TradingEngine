package handlers

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	pb "github.com/TanishqM1/Orderbook/internal/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Cancel handles the gRPC Cancel request
func (s *GRPCServer) Cancel(ctx context.Context, req *pb.CancelRequest) (*pb.CancelResponse, error) {
	log.Debugf("Received Cancel request: %+v", req)

	// Validate request
	if req.Orderid == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "orderid field is required and cannot be zero")
	}
	if req.Book == "" {
		return nil, status.Errorf(codes.InvalidArgument, "book field is required")
	}

	// Prepare request to C++ engine
	urlValues := url.Values{}
	urlValues.Set("orderid", strconv.FormatUint(req.Orderid, 10))
	urlValues.Set("book", req.Book)

	reqBody := strings.NewReader(urlValues.Encode())
	client := http.Client{}
	cppServerURL := "http://localhost:6060/cancel"

	log.Debugf("Forwarding cancel request to C++ engine: %s with body: %s", cppServerURL, urlValues.Encode())

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

	log.Infof("Attempted to Cancel Order: %d", req.Orderid)

	return &pb.CancelResponse{
		StatusCode: int32(cppResp.StatusCode),
		Body:       string(respBody),
	}, nil
}
