// Type definitions for the Trading Engine Simulation

export interface StockSimConfig {
  symbol: string;
  numBids: number;
  numAsks: number;
  priceMin: number;
  priceMax: number;
  quantityMin: number;
  quantityMax: number;
}

export interface SimulationRequest {
  stocks: StockSimConfig[];
}

export interface StockResult {
  symbol: string;
  tradesExecuted: number;
  volumeTraded: number;
  remainingBids: number;
  remainingAsks: number;
  bestBidPrice: number | null;
  bestAskPrice: number | null;
  bidLevels: number;
  askLevels: number;
}

export interface SimulationResponse {
  executionTimeMs: number;
  totalOrdersProcessed: number;
  results: StockResult[];
}

// Default values for a new stock config
export const defaultStockConfig: StockSimConfig = {
  symbol: '',
  numBids: 100,
  numAsks: 100,
  priceMin: 100,
  priceMax: 200,
  quantityMin: 10,
  quantityMax: 100,
};
