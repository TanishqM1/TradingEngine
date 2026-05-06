package engine

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// EngineInfo holds information about a running engine instance
type EngineInfo struct {
	Symbol  string
	Port    int
	Process *os.Process
	Healthy bool
}

// Manager handles spawning and managing C++ engine processes
type Manager struct {
	mu           sync.RWMutex
	engines      map[string]*EngineInfo // symbol -> engine info
	nextPort     int
	basePort     int
	engineBinary string
	client       *http.Client
}

// NewManager creates a new engine manager
func NewManager(engineBinaryPath string) *Manager {
	return &Manager{
		engines:      make(map[string]*EngineInfo),
		basePort:     6060,
		nextPort:     6060,
		engineBinary: engineBinaryPath,
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

// GetOrSpawnEngine returns an existing engine for the symbol or spawns a new one
func (m *Manager) GetOrSpawnEngine(symbol string) (*EngineInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if engine already exists for this symbol
	if info, exists := m.engines[symbol]; exists {
		return info, nil
	}

	// Spawn a new engine
	port := m.nextPort
	m.nextPort++

	log.Infof("Spawning new C++ engine for %s on port %d", symbol, port)

	// Start the engine process with the port as an argument
	cmd := exec.Command(m.engineBinary, fmt.Sprintf("%d", port))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set working directory to where the binary is located
	cmd.Dir = filepath.Dir(m.engineBinary)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start engine for %s: %w", symbol, err)
	}

	info := &EngineInfo{
		Symbol:  symbol,
		Port:    port,
		Process: cmd.Process,
		Healthy: false,
	}

	m.engines[symbol] = info

	// Wait for engine to be ready
	if err := m.waitForEngine(port); err != nil {
		log.Warnf("Engine for %s may not be fully ready: %v", symbol, err)
	} else {
		info.Healthy = true
	}

	return info, nil
}

// SpawnEnginesForSymbols spawns engines for all symbols in parallel
func (m *Manager) SpawnEnginesForSymbols(symbols []string) (map[string]*EngineInfo, error) {
	var wg sync.WaitGroup
	results := make(map[string]*EngineInfo)
	var resultMu sync.Mutex
	var firstErr error
	var errMu sync.Mutex

	for _, symbol := range symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()

			info, err := m.GetOrSpawnEngine(sym)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				return
			}

			resultMu.Lock()
			results[sym] = info
			resultMu.Unlock()
		}(symbol)
	}

	wg.Wait()

	if firstErr != nil {
		return results, firstErr
	}

	return results, nil
}

// waitForEngine polls the engine until it responds or times out
func (m *Manager) waitForEngine(port int) error {
	maxAttempts := 50 // 5 seconds total (50 * 100ms)
	url := fmt.Sprintf("http://localhost:%d/status", port)

	for i := 0; i < maxAttempts; i++ {
		resp, err := m.client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				log.Infof("Engine on port %d is ready", port)
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("engine on port %d did not become ready in time", port)
}

// GetEngineForSymbol returns the engine info for a symbol (nil if not exists)
func (m *Manager) GetEngineForSymbol(symbol string) *EngineInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.engines[symbol]
}

// GetAllEngines returns all running engine infos
func (m *Manager) GetAllEngines() map[string]*EngineInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*EngineInfo, len(m.engines))
	for k, v := range m.engines {
		result[k] = v
	}
	return result
}

// GetEngineURL returns the base URL for an engine by symbol
func (m *Manager) GetEngineURL(symbol string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, exists := m.engines[symbol]
	if !exists {
		return "", fmt.Errorf("no engine for symbol %s", symbol)
	}

	return fmt.Sprintf("http://localhost:%d", info.Port), nil
}

// HealthCheck checks the health of all engines
func (m *Manager) HealthCheck() map[string]bool {
	m.mu.RLock()
	engines := make(map[string]*EngineInfo, len(m.engines))
	for k, v := range m.engines {
		engines[k] = v
	}
	m.mu.RUnlock()

	results := make(map[string]bool)
	var wg sync.WaitGroup
	var resultMu sync.Mutex

	for symbol, info := range engines {
		wg.Add(1)
		go func(sym string, eng *EngineInfo) {
			defer wg.Done()

			url := fmt.Sprintf("http://localhost:%d/status", eng.Port)
			resp, err := m.client.Get(url)

			healthy := false
			if err == nil {
				resp.Body.Close()
				healthy = resp.StatusCode == 200
			}

			resultMu.Lock()
			results[sym] = healthy
			resultMu.Unlock()

			// Update engine health status
			m.mu.Lock()
			if e, exists := m.engines[sym]; exists {
				e.Healthy = healthy
			}
			m.mu.Unlock()
		}(symbol, info)
	}

	wg.Wait()
	return results
}

// StopEngine stops a specific engine
func (m *Manager) StopEngine(symbol string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, exists := m.engines[symbol]
	if !exists {
		return fmt.Errorf("no engine for symbol %s", symbol)
	}

	if info.Process != nil {
		if err := info.Process.Kill(); err != nil {
			log.Warnf("Failed to kill engine process for %s: %v", symbol, err)
		}
	}

	delete(m.engines, symbol)
	log.Infof("Stopped engine for %s", symbol)
	return nil
}

// StopAllEngines stops all running engines
func (m *Manager) StopAllEngines() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for symbol, info := range m.engines {
		if info.Process != nil {
			if err := info.Process.Kill(); err != nil {
				log.Warnf("Failed to kill engine process for %s: %v", symbol, err)
			}
		}
		log.Infof("Stopped engine for %s", symbol)
	}

	m.engines = make(map[string]*EngineInfo)
	m.nextPort = m.basePort
}

// ResetEngine resets a specific engine's orderbook
func (m *Manager) ResetEngine(symbol string) error {
	m.mu.RLock()
	info, exists := m.engines[symbol]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no engine for symbol %s", symbol)
	}

	url := fmt.Sprintf("http://localhost:%d/reset", info.Port)
	resp, err := m.client.Post(url, "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to reset engine for %s: %w", symbol, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("reset failed for %s with status %d", symbol, resp.StatusCode)
	}

	return nil
}

// ResetAllEngines resets all engine orderbooks
func (m *Manager) ResetAllEngines() error {
	engines := m.GetAllEngines()
	var firstErr error

	for symbol := range engines {
		if err := m.ResetEngine(symbol); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			log.Warnf("Failed to reset engine for %s: %v", symbol, err)
		}
	}

	return firstErr
}

// GetMapping returns the current symbol to port mapping
func (m *Manager) GetMapping() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mapping := make(map[string]int, len(m.engines))
	for symbol, info := range m.engines {
		mapping[symbol] = info.Port
	}
	return mapping
}
