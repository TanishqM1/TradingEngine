# /Trade Endpoint
frontend -> gRPC

```
grpcurl -plaintext -d '{
  "tradetype":"GTILLCANCEL",
  "side":"BUY",
  "price":100,
  "quantity":10,
  "name":"AAPL"
}' localhost:50051 orderbook.OrderService/Trade
```

Go Backend to C++ Engine (through the load balancer)

```
curl -X POST "http://<engine-host>:<port>/trade" \
  -d "orderid=1" \
  -d "tradetype=GTILLCANCEL" \
  -d "side=BUY" \
  -d "price=100" \
  -d "quantity=10" \
  -d "book=AAPL"
```

Response to the frontend

```
{
  "order_id": 1,
  "status_code": 202,
  "body": "Order queued"
}
```


# /Cancel Endpoint

Frontend sends CancelRequest (orderid uint64, book string) gRPC

```
curl -X POST "http://<engine-host>:<port>/cancel" -d "orderid=1" -d "book=AAPL"
```