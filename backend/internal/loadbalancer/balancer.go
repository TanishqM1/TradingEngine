package loadbalancer

import (
	"encoding/json"
	"hash/fnv"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Balancer routes requests to engine servers with minimal overhead
type Balancer struct {
	servers []string
	client  *http.Client
	mapping map[string]int // explicit stock symbol -> server index mapping
}

// New creates a balancer with an optimized HTTP client for minimal latency
func New(servers []string) *Balancer {
	// Tuned transport for maximum throughput and connection reuse
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   50 * time.Millisecond,
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
		servers: servers,
		client: &http.Client{
			Transport: tr,
			Timeout:   5 * time.Second,
		},
	}
}

// NewWithMapping creates a balancer with explicit stock-to-server mapping
func NewWithMapping(servers []string, mapping map[string]int) *Balancer {
	b := New(servers)
	b.mapping = mapping
	return b
}

// LoadMapping reads stock-to-server mapping from a JSON file
// Expected format: {"NVDA": 0, "AAPL": 1, "TSLA": 2, ...}
func LoadMapping(path string) (map[string]int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var mapping map[string]int
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, err
	}

	return mapping, nil
}

// SaveMapping writes stock-to-server mapping to a JSON file
func SaveMapping(path string, mapping map[string]int) error {
	data, err := json.MarshalIndent(mapping, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// BuildEvenMapping creates a balanced distribution of symbols across servers
// Uses round-robin assignment to distribute load evenly
func BuildEvenMapping(symbols []string, numServers int) map[string]int {
	mapping := make(map[string]int, len(symbols))
	for i, symbol := range symbols {
		mapping[symbol] = i % numServers
	}
	return mapping
}

// pickServer returns server index for a stock symbol
// First checks explicit mapping, then falls back to FNV1a hash
// This is O(1) - map lookup or hash + modulo
func (b *Balancer) pickServer(book string) int {
	if len(b.servers) == 0 {
		return -1
	}

	// Check explicit mapping first
	if b.mapping != nil {
		if idx, ok := b.mapping[book]; ok {
			if idx >= 0 && idx < len(b.servers) {
				return idx
			}
		}
	}

	// Fallback to hash-based routing
	h := fnv.New32a()
	h.Write([]byte(book))
	return int(h.Sum32()) % len(b.servers)
}

// FireTrade sends trade request to engine without waiting for response
// This is fire-and-forget for maximum speed - request is in flight immediately
func (b *Balancer) FireTrade(form url.Values) {
	book := form.Get("book")
	idx := b.pickServer(book)
	if idx < 0 {
		return // no servers available
	}

	target := b.servers[idx] + "/trade"

	// Fire in goroutine so we don't block at all
	go func() {
		req, err := http.NewRequest("POST", target, strings.NewReader(form.Encode()))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := b.client.Do(req)
		if err != nil {
			return
		}
		resp.Body.Close() // close immediately, don't read
	}()
}

// ForwardStatus sends status request and waits for response (synchronous)
func (b *Balancer) ForwardStatus() (*http.Response, error) {
	// For status, just hit first server (or implement round-robin if needed)
	if len(b.servers) == 0 {
		return nil, http.ErrServerClosed
	}

	target := b.servers[0] + "/status"
	return b.client.Get(target)
}

// ForwardCancel sends cancel request and waits for response
func (b *Balancer) ForwardCancel(form url.Values) (*http.Response, error) {
	book := form.Get("name")
	idx := b.pickServer(book)
	if idx < 0 {
		return nil, http.ErrServerClosed
	}

	target := b.servers[idx] + "/cancel"
	req, err := http.NewRequest("POST", target, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return b.client.Do(req)
}
