# Orderbook Mock Trading Engine: Low-Latency CLOB Simulator

This project is a high-performance, polyglot system designed to serve as a mock electronic trading engine, simulating the core components of a real financial exchange's Central Limit Order Book (CLOB). It uses a layered architecture to maximize speed in the core logic while maintaining flexibility in the API layer.

-----

## Technologies & Libraries

| Component | Technology | Role | Source |
| :--- | :--- | :--- | :--- |
| **Engine Core (C++)** | C++23, `std::unordered_map` | Business-critical low-latency order matching and multi-asset persistence. | |
| **Server Interface (C++)** | `httplib` | Provides the minimal, high-speed HTTP interface for the Go API to communicate with the engine process. | |
| **API Gateway (Go)** | Go (w/ `go-chi/chi`, `net/http`) | Routing, thread-safe `OrderID` generation, and proxying requests to the C++ engine. | |
| **Frontend (In Progress)** | TypeScript, Next.js, shadcn/ui | Visualization of market depth and trade history. | |

-----

## Core Engine Features & Concepts

The core engine models key market concepts and logic:

  * **Limit Order Management:** Handles creation, placement, and persistence of buy and sell orders.
  * **CLOB Structure:** Maintains separate internal data structures (Bids/Asks) for the book state.
  * **Priority Matching:** Orders are executed based on **Price–Time Priority** (best price first, then earliest submission time).
  * **Order Types:** Models **GTC** (Good-Till-Cancel) and **FAK** (Fill-And-Kill) orders.
  * **Real-World State:** Tracks important metrics like the total quantity at specific **Price Levels** and handles **Partially Filled Orders**.

-----

## Architecture & Roadmap

### 1\. C++ Engine Layer (The Core Logic)

  * **Status: DONE**
  * **Function:** Holds the active Orderbook state in RAM and executes the matching algorithm.
  * **Output:** Generates trade executions and new orderbook liquidity states.

### 2\. C++ Server Layer (The Performance Gateway)

  * **Status: Complete (Minimal HTTP Implemented)**
  * **Function:** Wraps the Engine via an `httplib` server to expose functionality on port 6060.
  * **Future:** Planning migration to gRPC for minimal-latency binary communication.

### 3\. Go Backend API (The External Interface)

  * **Status: DONE**
  * **Function:** Provides external RESTful endpoints on port 8000 for the Frontend. It handles request routing, generates thread-safe `OrderID`s, and proxies requests to the C++ Server.
  * **Why Go:** Excellent for high concurrency and robust traffic management (goroutines).

### 4\. Frontend UI (Visualization and Input)

  * **Status: In Progress**
  * **Function:** Renders the order book depth, trade history, and state visually for the end-user.

-----

## Setup & Run Instructions

To run the system, you must compile and start the C++ engine first, followed by the Go API.

### Phase 1: Run the C++ Orderbook Engine (Port 6060)

1.  **Navigate to the C++ Directory:**

    ```bash
    cd /S/Users/tanis/Desktop/Projects/OrderBook/backend/engine
    ```

2.  **Compile the Server:** This command compiles the entire merged engine and server logic.

    ```bash
    g++ -std=c++23 -O2 Server.cpp -lws2_32 -o server.exe
    ```

3.  **Run the Server:** Keep this console window **open and running**.

    ```bash
    ./server.exe
    ```

    *(Output will confirm listening on port 6060).*

### Phase 2: Run the Go API Proxy (Port 8000)

1.  **Open a NEW Console Window.**

2.  **Navigate to the Go API Directory:**

    ```bash
    cd /S/Users/tanis/Desktop/Projects/OrderBook/backend/cmd/api
    ```

3.  **Run the Go Server:**

    ```bash
    go run main.go
    ```

    *(Output will confirm listening on port 8000).*

-----

## API Testing Examples (Postman/cURL)

Direct all requests to the **Go API on Port 8000**. The Go API will handle ID generation and proxy the asset name as the `book` parameter.

### 1\. Place an Order (`POST /order/trade`)

This order creates the first resting **Buy Limit** order for TSLA, setting up the Bid side of the book.

  * **URL:** `http://localhost:8000/order/trade`
  * **Method:** `POST`
  * **Body (Raw JSON):**
    ```json
    {
        "tradetype": "GTC", 
        "side": "BUY", 
        "price": 100, 
        "quantity": 100, 
        "name": "TSLA" 
    }
    ```
  * **Expected Status:** `200 OK`

### 2\. Match an Order (`POST /order/trade`)

This order crosses the spread, immediately matching the resting Buy order from Step 1.

  * **URL:** `http://localhost:8000/order/trade`
  * **Method:** `POST`
  * **Body (Raw JSON):**
    ```json
    {
        "tradetype": "GTC", 
        "side": "SELL", 
        "price": 99, 
        "quantity": 50, 
        "name": "TSLA" 
    }
    ```
  * **Expected Status:** `200 OK`. The JSON response confirms a **Trade** occurred at $100. The original Buy order remains in the book with a remaining quantity of 50.

### 3\. Cancel a Resting Order (`POST /order/cancel`)

This removes the partially filled order from the book using the `OrderID` generated from the first `trade` call (which is likely **1**).

  * **URL:** `http://localhost:8000/order/cancel`
  * **Method:** `POST`
  * **Body (Raw JSON):**
    ```json
    {
        "orderid": 1,
        "book": "TSLA"
    }
    ```
  * **Expected Status:** `200 OK` (Message confirms successful removal).


### 4\. Retrieve Engine Status (`GET /order/status`)

This returns all information across all Orderbook's, stock information, and ask(s)/bid(s) at each level.

  * **URL:** `http://localhost:8000/order/status`
  * **Method:** `GET`
  * **Body (Raw JSON):**
    ```json
    {
      "AAPL": {
          "bids": [
              {
                  "type": "Bid",
                  "price": 100,
                  "quantity": 200
              }
          ],
          "asks": [],
          "size": 2
      },
      "GOOG": {
          "bids": [
              {
                  "type": "Bid",
                  "price": 100,
                  "quantity": 200
              }
          ],
          "asks": [],
          "size": 2
      },
      "TSLA": {
          "bids": [
              {
                  "type": "Bid",
                  "price": 100,
                  "quantity": 200
              }
          ],
          "asks": [],
          "size": 2
      }
  }
    ```
  * **Expected Status:** `200 OK` (Message confirms successful retrieval).

-----

## Attribution

The fundamental concepts and core algorithm structure for the Orderbook matching engine were derived from the open-source work of **CodingJesus**, with some additional features.
 Everything else including backend api, server functions, parsing logic, and frontend were built entirely from scratch.

  * [Orderbook Video](https://www.youtube.com/watch?v=XeLWe0Cx_Lg&t=1258s)