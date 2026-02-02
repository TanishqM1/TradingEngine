# Stock-to-Server Load Balancer

## Architecture

The load balancer uses explicit stock→server mapping to distribute trading load based on trading frequency and volume.

### Current Setup (Example)

**Servers:** 4 engine instances
- Server 0 (`:6060`): High-frequency stocks (NVDA)
- Server 1 (`:6061`): High-frequency stocks (AAPL)
- Server 2 (`:6062`): Medium-frequency stocks (TSLA, MSFT)
- Server 3 (`:6063`): High-frequency stocks (GOOGL)

**Configuration:** `backend/config/stock_mapping.json`

```json
{
  "NVDA": 0,
  "AAPL": 1,
  "TSLA": 2,
  "MSFT": 2,
  "GOOGL": 3
}
```

## How It Works

1. **Explicit Mapping (Primary):** `pickServer()` first checks the `stock_mapping.json` file for exact symbol match
2. **Hash Fallback:** If symbol not in mapping, uses FNV-1a hash for deterministic routing
3. **O(1) Performance:** Map lookup or single hash operation per request

## Scaling to 500+ Stocks

### Option 1: Manual Mapping (Best Control)
Analyze trading data and manually assign stocks to servers:

```bash
# Generate even distribution as starting point
cd backend
go run scripts/generate_mapping.go \
  --symbols symbols.txt \
  --servers 50 \
  --output config/stock_mapping.json
```

Then adjust high-frequency stocks manually based on real data.

### Option 2: Auto-Balance Script
Use the `BuildEvenMapping()` helper in code:

```go
symbols := []string{"NVDA", "AAPL", "TSLA", /* ... 500 more ... */}
mapping := loadbalancer.BuildEvenMapping(symbols, 50) // 50 servers
loadbalancer.SaveMapping("config/stock_mapping.json", mapping)
```

### Option 3: Dynamic Rebalancing
Monitor per-server request rates and periodically regenerate mapping:
- Track requests/sec per stock
- Group stocks by trading frequency
- Distribute groups evenly across servers
- Hot-reload mapping (requires adding `Balancer.UpdateMapping()`)

## Testing

Start 4 mock engine servers:
```bash
for port in 6060 6061 6062 6063; do
  echo "Starting mock engine on :$port"
  python3 -c "
from http.server import BaseHTTPRequestHandler, HTTPServer
class H(BaseHTTPRequestHandler):
    def do_POST(self):
        l = int(self.headers.get('Content-Length',0))
        body = self.rfile.read(l).decode()
        print(f'[{$port}] {self.path} -> {body}')
        self.send_response(200); self.end_headers(); self.wfile.write(b'ok')
HTTPServer(('0.0.0.0', $port), H).serve_forever()
" &
done
```

Test with grpcurl:
```bash
# NVDA should go to server 0 (6060)
grpcurl -plaintext -d '{"tradetype":"GTILLCANCEL","side":"BUY","name":"NVDA","price":100,"quantity":1}' \
  localhost:50051 orderbook.OrderService/Trade

# TSLA should go to server 2 (6062)
grpcurl -plaintext -d '{"tradetype":"GTILLCANCEL","side":"BUY","name":"TSLA","price":200,"quantity":5}' \
  localhost:50051 orderbook.OrderService/Trade

# Unknown stock falls back to hash-based routing
grpcurl -plaintext -d '{"tradetype":"GTILLCANCEL","side":"BUY","name":"UNKNOWN","price":50,"quantity":2}' \
  localhost:50051 orderbook.OrderService/Trade
```

## Modifying Mapping

Edit `backend/config/stock_mapping.json` and restart the gRPC server. Changes take effect immediately on restart.

For production, implement hot-reload:
```go
// Add to Balancer
func (b *Balancer) UpdateMapping(mapping map[string]int) {
    // Use atomic.Value or sync.RWMutex for thread-safe updates
}
```

## Performance Notes

- **Mapping lookup:** O(1) map access (~10-20ns)
- **Hash fallback:** O(1) FNV-1a hash (~50ns)
- **No blocking:** Fire-and-forget keeps latency <2μs
- **Connection pooling:** 200 idle connections per server for reuse
