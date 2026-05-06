# Distributed Trading Engine

A high-performance CLOB (Central Limit Order Book) simulator with **distributed processing** - each stock runs on its own dedicated C++ engine instance for parallel order matching.

## Quick Start

```bash
./start.sh
```

Opens:
- **Frontend**: http://localhost:3000
- **API**: http://localhost:8000
- **C++ Engines**: Spawned dynamically on ports 6060+

## Architecture

```
Frontend (Next.js) → Go API → Multiple C++ Engines (1 per stock)
```

When you run a simulation with 4 stocks, 4 separate C++ engine processes are spawned and orders are distributed in parallel.

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/order/trade` | Place an order |
| POST | `/order/cancel` | Cancel an order |
| POST | `/order/simulation` | Run distributed simulation |
| POST | `/order/reset` | Reset all engines |
| GET | `/order/status` | Get orderbook state |
| GET | `/order/health` | Check engine health |
| GET | `/order/engines` | List running engines |

## Examples

**Place Order:**
```bash
curl -X POST http://localhost:8000/order/trade \
  -H "Content-Type: application/json" \
  -d '{"tradetype":"GTC","side":"BUY","price":100,"quantity":50,"name":"AAPL"}'
```

**Run Simulation:**
```bash
curl -X POST http://localhost:8000/order/simulation \
  -H "Content-Type: application/json" \
  -d '{"stocks":[{"symbol":"AAPL","numBids":100,"numAsks":100,"priceMin":100,"priceMax":200,"quantityMin":10,"quantityMax":100}]}'
```

**Check Health:**
```bash
curl http://localhost:8000/order/health
```

## Tech Stack

- **Frontend**: Next.js, React, Tailwind CSS
- **API Gateway**: Go (chi router)
- **Matching Engine**: C++ (httplib)

## Attribution

Core orderbook algorithm based on [CodingJesus](https://www.youtube.com/watch?v=XeLWe0Cx_Lg).
