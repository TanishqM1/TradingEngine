#!/bin/bash

# Distributed Trading Engine Startup Script
# Compiles C++ Engine and starts Go API + Next.js Frontend
# C++ engines are spawned dynamically by Go backend (one per stock)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Source common shell profiles to get PATH (for Go, etc.)
[ -f "$HOME/.bashrc" ] && source "$HOME/.bashrc" 2>/dev/null || true
[ -f "$HOME/.bash_profile" ] && source "$HOME/.bash_profile" 2>/dev/null || true
[ -f "$HOME/.zshrc" ] && source "$HOME/.zshrc" 2>/dev/null || true
[ -f "$HOME/.profile" ] && source "$HOME/.profile" 2>/dev/null || true

# Add common Go paths
export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin:/opt/homebrew/bin"

# PIDs for cleanup
GO_PID=""
NEXT_PID=""

cleanup() {
    echo -e "\n${YELLOW}Shutting down services...${NC}"

    if [ -n "$NEXT_PID" ] && kill -0 "$NEXT_PID" 2>/dev/null; then
        echo -e "${BLUE}Stopping Next.js frontend...${NC}"
        kill "$NEXT_PID" 2>/dev/null || true
    fi

    if [ -n "$GO_PID" ] && kill -0 "$GO_PID" 2>/dev/null; then
        echo -e "${BLUE}Stopping Go API (this will also stop all C++ engines)...${NC}"
        kill "$GO_PID" 2>/dev/null || true
    fi

    # Kill any remaining C++ engines on ports 6060-6099
    echo -e "${BLUE}Cleaning up any remaining C++ engines...${NC}"
    for port in $(seq 6060 6099); do
        pid=$(lsof -ti:$port 2>/dev/null || true)
        if [ -n "$pid" ]; then
            kill $pid 2>/dev/null || true
        fi
    done

    echo -e "${GREEN}All services stopped.${NC}"
    exit 0
}

trap cleanup SIGINT SIGTERM

echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}  Distributed Trading Engine Startup${NC}"
echo -e "${CYAN}========================================${NC}"
echo ""
echo -e "${CYAN}Architecture:${NC}"
echo -e "  Frontend (Next.js) -> Go API -> Multiple C++ Engines"
echo -e "  Each stock gets its own dedicated C++ engine instance"
echo ""

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

# Check for g++
if ! command -v g++ &> /dev/null; then
    echo -e "${RED}Error: g++ not found. Please install a C++ compiler.${NC}"
    echo -e "  macOS: xcode-select --install"
    echo -e "  Ubuntu: sudo apt install g++"
    exit 1
fi

# Check for Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go not found in PATH.${NC}"
    echo -e "Please install Go from https://go.dev/dl/"
    echo -e "  macOS: brew install go"
    echo -e "  Ubuntu: sudo apt install golang-go"
    exit 1
fi

# Check for Node.js/npm
if ! command -v npm &> /dev/null; then
    echo -e "${RED}Error: npm not found. Please install Node.js.${NC}"
    echo -e "  https://nodejs.org/"
    echo -e "  macOS: brew install node"
    exit 1
fi

echo -e "${GREEN}All prerequisites found!${NC}"
echo ""

# Step 1: Compile C++ Engine
echo -e "${YELLOW}[1/3] Compiling C++ Engine...${NC}"
cd backend/engine

# Detect OS for compile flags
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    g++ -std=c++23 -O2 Server.cpp -o server
elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]] || [[ "$OSTYPE" == "win32" ]]; then
    # Windows
    g++ -std=c++23 -O2 Server.cpp -lws2_32 -o server.exe
else
    # Linux
    g++ -std=c++23 -O2 Server.cpp -pthread -o server
fi

echo -e "${GREEN}C++ Engine compiled successfully${NC}"
echo -e "${BLUE}Note: C++ engines will be spawned dynamically by Go API (one per stock)${NC}"

cd "$SCRIPT_DIR"

# Step 2: Start Go API
echo ""
echo -e "${YELLOW}[2/3] Starting Go API (Distributed Engine Manager) on port 8000...${NC}"
cd backend/cmd/api

# Set ENGINE_PATH to absolute path of the compiled engine
export ENGINE_PATH="$SCRIPT_DIR/backend/engine/server"
echo -e "${BLUE}ENGINE_PATH set to: $ENGINE_PATH${NC}"

go run main.go &
GO_PID=$!
sleep 2

if ! kill -0 "$GO_PID" 2>/dev/null; then
    echo -e "${RED}Failed to start Go API${NC}"
    cleanup
    exit 1
fi
echo -e "${GREEN}Go API started (PID: $GO_PID)${NC}"
echo -e "${BLUE}Go API will spawn C++ engines on ports 6060+ as needed${NC}"

cd "$SCRIPT_DIR"

# Step 3: Start Next.js Frontend
echo ""
echo -e "${YELLOW}[3/3] Starting Next.js Frontend on port 3000...${NC}"
cd frontend

# Install dependencies if node_modules doesn't exist
if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}Installing npm dependencies...${NC}"
    npm install
fi

npm run dev &
NEXT_PID=$!
sleep 3

if ! kill -0 "$NEXT_PID" 2>/dev/null; then
    echo -e "${RED}Failed to start Next.js Frontend${NC}"
    cleanup
    exit 1
fi
echo -e "${GREEN}Next.js Frontend started (PID: $NEXT_PID)${NC}"

cd "$SCRIPT_DIR"

# All services started
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}   All services started successfully!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "  ${BLUE}Go API:${NC}        http://localhost:8000"
echo -e "  ${BLUE}Frontend:${NC}      http://localhost:3000"
echo -e "  ${BLUE}C++ Engines:${NC}   Spawned dynamically on ports 6060+"
echo ""
echo -e "${CYAN}How it works:${NC}"
echo -e "  1. Configure stocks in the frontend"
echo -e "  2. Click 'Run Simulation'"
echo -e "  3. Go API spawns a dedicated C++ engine for each stock"
echo -e "  4. Orders are distributed across engines in parallel"
echo -e "  5. Results are aggregated and displayed"
echo ""
echo -e "${CYAN}API Endpoints:${NC}"
echo -e "  GET  /order/health   - Check health of all engines"
echo -e "  GET  /order/engines  - List all running engines"
echo -e "  POST /order/simulation - Run distributed simulation"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop all services${NC}"
echo ""

# Wait for any process to exit
wait
