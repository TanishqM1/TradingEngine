package loadbalancer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Balancer routes requests to engine servers based on symbol mapping
type Balancer struct {
	mu      sync.RWMutex
	mapping map[string]string // symbol -> base URL (e.g., "AAPL" -> "http://localhost:6060")
	client  *http.Client
}

// New creates a new load balancer with an optimized HTTP client
func New() *Balancer {
	// Tuned transport for maximum throughput and connection reuse
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   100 * time.Millisecond,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   200,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true, // avoid decompression overhead
	}

	return &Balancer{
		mapping: make(map[string]string),
		client: &http.Client{
			Transport: tr,
			Timeout:   5 * time.Second,
		},
	}
}

// RegisterEngine registers a symbol to an engine URL
func (b *Balancer) RegisterEngine(symbol string, baseURL string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.mapping[symbol] = baseURL
	log.Infof("Registered engine for %s at %s", symbol, baseURL)
}

// RegisterEngines registers multiple symbols to their engine URLs
func (b *Balancer) RegisterEngines(mapping map[string]int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for symbol, port := range mapping {
		b.mapping[symbol] = fmt.Sprintf("http://localhost:%d", port)
	}
}

// UnregisterEngine removes a symbol from the mapping
func (b *Balancer) UnregisterEngine(symbol string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.mapping, symbol)
	log.Infof("Unregistered engine for %s", symbol)
}

// GetEngineURL returns the engine URL for a symbol
func (b *Balancer) GetEngineURL(symbol string) (string, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	url, exists := b.mapping[symbol]
	return url, exists
}

// GetAllEngineURLs returns all registered engine URLs
func (b *Balancer) GetAllEngineURLs() map[string]string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make(map[string]string, len(b.mapping))
	for k, v := range b.mapping {
		result[k] = v
	}
	return result
}

// ForwardTrade sends a trade request to the appropriate engine
func (b *Balancer) ForwardTrade(form url.Values) (*http.Response, error) {
	book := form.Get("book")

	b.mu.RLock()
	baseURL, exists := b.mapping[book]
	b.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no engine registered for symbol %s", book)
	}

	target := baseURL + "/trade"
	req, err := http.NewRequest("POST", target, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return b.client.Do(req)
}

// FireTrade sends a trade request without waiting for response (fire-and-forget)
func (b *Balancer) FireTrade(form url.Values) {
	book := form.Get("book")

	b.mu.RLock()
	baseURL, exists := b.mapping[book]
	b.mu.RUnlock()

	if !exists {
		log.Warnf("No engine registered for symbol %s", book)
		return
	}

	target := baseURL + "/trade"

	// Fire in goroutine so we don't block
	go func() {
		req, err := http.NewRequest("POST", target, strings.NewReader(form.Encode()))
		if err != nil {
			log.Warnf("Failed to create trade request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := b.client.Do(req)
		if err != nil {
			log.Warnf("Failed to send trade to engine: %v", err)
			return
		}
		resp.Body.Close()
	}()
}

// ForwardCancel sends a cancel request to the appropriate engine
func (b *Balancer) ForwardCancel(form url.Values) (*http.Response, error) {
	book := form.Get("book")

	b.mu.RLock()
	baseURL, exists := b.mapping[book]
	b.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no engine registered for symbol %s", book)
	}

	target := baseURL + "/cancel"
	req, err := http.NewRequest("POST", target, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return b.client.Do(req)
}

// ForwardStatus gets status from a specific engine
func (b *Balancer) ForwardStatus(symbol string) (*http.Response, error) {
	b.mu.RLock()
	baseURL, exists := b.mapping[symbol]
	b.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no engine registered for symbol %s", symbol)
	}

	return b.client.Get(baseURL + "/status")
}

// ForwardReset resets a specific engine
func (b *Balancer) ForwardReset(symbol string) (*http.Response, error) {
	b.mu.RLock()
	baseURL, exists := b.mapping[symbol]
	b.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no engine registered for symbol %s", symbol)
	}

	return b.client.Post(baseURL+"/reset", "application/json", nil)
}

// BatchOrder represents an order in a batch request
type BatchOrder struct {
	OrderId   uint64 `json:"orderid"`
	Book      string `json:"book"`
	TradeType string `json:"tradetype"`
	Side      string `json:"side"`
	Price     int    `json:"price"`
	Quantity  int    `json:"quantity"`
}

// BatchRequest is the request format for the batch endpoint
type BatchRequest struct {
	Orders []BatchOrder `json:"orders"`
}

// BatchResult represents the result from a batch request
type BatchResult struct {
	TradesExecuted int   `json:"tradesExecuted"`
	VolumeTraded   int64 `json:"volumeTraded"`
	RemainingBids  int   `json:"remainingBids"`
	RemainingAsks  int   `json:"remainingAsks"`
	BestBidPrice   int   `json:"bestBidPrice"`
	BestAskPrice   int   `json:"bestAskPrice"`
	BidLevels      int   `json:"bidLevels"`
	AskLevels      int   `json:"askLevels"`
}

// BatchResponse is the response format from the batch endpoint
type BatchResponse struct {
	ProcessedCount int                    `json:"processedCount"`
	Results        map[string]BatchResult `json:"results"`
}

// ForwardBatch sends a batch of orders to the appropriate engine
// Since each engine handles one symbol, we route directly
func (b *Balancer) ForwardBatch(symbol string, orders []BatchOrder) (*BatchResponse, error) {
	b.mu.RLock()
	baseURL, exists := b.mapping[symbol]
	b.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no engine registered for symbol %s", symbol)
	}

	batchReq := BatchRequest{Orders: orders}
	body, err := json.Marshal(batchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch request: %w", err)
	}

	resp, err := b.client.Post(baseURL+"/batch", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to send batch request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("batch request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var batchResp BatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return nil, fmt.Errorf("failed to decode batch response: %w", err)
	}

	return &batchResp, nil
}

// ForwardBatchParallel sends batches to multiple engines in parallel
func (b *Balancer) ForwardBatchParallel(ordersBySymbol map[string][]BatchOrder) (map[string]*BatchResponse, error) {
	results := make(map[string]*BatchResponse)
	var resultMu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex

	for symbol, orders := range ordersBySymbol {
		wg.Add(1)
		go func(sym string, ords []BatchOrder) {
			defer wg.Done()

			resp, err := b.ForwardBatch(sym, ords)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				log.Warnf("Batch request failed for %s: %v", sym, err)
				return
			}

			resultMu.Lock()
			results[sym] = resp
			resultMu.Unlock()
		}(symbol, orders)
	}

	wg.Wait()

	return results, firstErr
}

// HealthCheck checks if an engine is healthy
func (b *Balancer) HealthCheck(symbol string) bool {
	resp, err := b.ForwardStatus(symbol)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// HealthCheckAll checks health of all registered engines
func (b *Balancer) HealthCheckAll() map[string]bool {
	engines := b.GetAllEngineURLs()
	results := make(map[string]bool)
	var resultMu sync.Mutex
	var wg sync.WaitGroup

	for symbol := range engines {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			healthy := b.HealthCheck(sym)
			resultMu.Lock()
			results[sym] = healthy
			resultMu.Unlock()
		}(symbol)
	}

	wg.Wait()
	return results
}
