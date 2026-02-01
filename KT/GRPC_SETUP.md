# gRPC Migration Setup Instructions

## 1. Install Required Tools

### Install protoc compiler
```bash
# macOS
brew install protobuf

# Verify installation
protoc --version
```

### Install Go protoc plugins
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Make sure $GOPATH/bin is in your PATH
export PATH="$PATH:$(go env GOPATH)/bin"
```

## 2. Install Go Dependencies

```bash
cd backend
go get google.golang.org/grpc
go get google.golang.org/protobuf
go mod tidy
```

## 3. Generate gRPC Code from Proto File

```bash
# Run from the backend directory
cd /Users/tanishq/Work/TradingEngine/backend

# Generate Go code
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/orderbook.proto
```

This will generate two files:
- `internal/pb/orderbook.pb.go` (message types)
- `internal/pb/orderbook_grpc.pb.go` (service interface and client/server code)

## 4. Build and Run

```bash
# Build the server
cd /Users/tanishq/Work/TradingEngine/backend
go build -o server cmd/api/main.go

# Run the server
./server
```

The gRPC server will start on `:50051`

## 5. Testing with grpcurl

Install grpcurl for testing:
```bash
brew install grpcurl
```

### Test Trade endpoint:
```bash
grpcurl -plaintext -d '{
  "tradetype": "GTILLCANCEL",
  "side": "BUY",
  "price": 100,
  "quantity": 10,
  "name": "AAPL"
}' localhost:50051 orderbook.OrderService/Trade
```

### Test Cancel endpoint:
```bash
grpcurl -plaintext -d '{
  "orderid": 1,
  "book": "AAPL"
}' localhost:50051 orderbook.OrderService/Cancel
```

### Test Status endpoint:
```bash
grpcurl -plaintext -d '{}' localhost:50051 orderbook.OrderService/Status
```

### List available services:
```bash
grpcurl -plaintext localhost:50051 list
```

### Describe a service:
```bash
grpcurl -plaintext localhost:50051 describe orderbook.OrderService
```

## 6. Client Code Example (Go)

```go
package main

import (
    "context"
    "log"
    "time"

    pb "github.com/TanishqM1/Orderbook/internal/pb"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func main() {
    // Connect to server
    conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewOrderServiceClient(conn)
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()

    // Place a trade
    tradeResp, err := client.Trade(ctx, &pb.TradeRequest{
        Tradetype: "GTILLCANCEL",
        Side:      "BUY",
        Price:     100,
        Quantity:  10,
        Name:      "AAPL",
    })
    if err != nil {
        log.Fatalf("Trade failed: %v", err)
    }
    log.Printf("Trade response: OrderID=%d, Status=%d, Body=%s", 
        tradeResp.OrderId, tradeResp.StatusCode, tradeResp.Body)
}
```

## Architecture Overview

The new gRPC architecture maintains the decoupled structure:

- `proto/orderbook.proto` - Service and message definitions
- `internal/pb/` - Generated gRPC code
- `internal/handlers/grpc_server.go` - Server struct and factory
- `internal/handlers/grpc_trade.go` - Trade RPC implementation
- `internal/handlers/grpc_cancel.go` - Cancel RPC implementation
- `internal/handlers/grpc_status.go` - Status RPC implementation
- `cmd/api/main.go` - gRPC server startup

Each endpoint is still in its own file, maintaining separation of concerns while using strongly-typed gRPC messages instead of JSON.
