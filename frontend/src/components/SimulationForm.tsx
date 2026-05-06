'use client';

import { useState } from 'react';
import { StockSimConfig, SimulationResponse, defaultStockConfig } from '@/lib/types';
import { runSimulation } from '@/lib/api';
import StockConfigCard from './StockConfigCard';

interface SimulationFormProps {
  onResults: (response: SimulationResponse) => void;
}

export default function SimulationForm({ onResults }: SimulationFormProps) {
  const [stocks, setStocks] = useState<StockSimConfig[]>([
    { ...defaultStockConfig, symbol: 'TSLA' },
  ]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleStockChange = (index: number, config: StockSimConfig) => {
    const newStocks = [...stocks];
    newStocks[index] = config;
    setStocks(newStocks);
  };

  const handleRemoveStock = (index: number) => {
    if (stocks.length > 1) {
      setStocks(stocks.filter((_, i) => i !== index));
    }
  };

  const handleAddStock = () => {
    const usedSymbols = new Set(stocks.map(s => s.symbol));
    const defaultSymbols = ['AAPL', 'GOOG', 'MSFT', 'AMZN', 'META', 'NVDA'];
    const nextSymbol = defaultSymbols.find(s => !usedSymbols.has(s)) || '';
    setStocks([...stocks, { ...defaultStockConfig, symbol: nextSymbol }]);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    const emptySymbol = stocks.find((s) => !s.symbol.trim());
    if (emptySymbol) {
      setError('All stocks must have a symbol');
      return;
    }

    const symbols = stocks.map((s) => s.symbol);
    if (new Set(symbols).size !== symbols.length) {
      setError('Duplicate symbols not allowed');
      return;
    }

    for (const stock of stocks) {
      const numBids = Number(stock.numBids) || 0;
      const numAsks = Number(stock.numAsks) || 0;
      if (numBids === 0 && numAsks === 0) {
        setError(`${stock.symbol}: Add at least one order`);
        return;
      }
      if ((Number(stock.priceMin) || 0) > (Number(stock.priceMax) || 0)) {
        setError(`${stock.symbol}: Invalid price range`);
        return;
      }
      if ((Number(stock.quantityMin) || 0) > (Number(stock.quantityMax) || 0)) {
        setError(`${stock.symbol}: Invalid quantity range`);
        return;
      }
    }

    setLoading(true);
    try {
      const cleanedStocks = stocks.map(s => ({
        ...s,
        numBids: Number(s.numBids) || 0,
        numAsks: Number(s.numAsks) || 0,
        priceMin: Number(s.priceMin) || 1,
        priceMax: Number(s.priceMax) || 100,
        quantityMin: Number(s.quantityMin) || 1,
        quantityMax: Number(s.quantityMax) || 100,
      }));
      const response = await runSimulation({ stocks: cleanedStocks });
      onResults(response);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Simulation failed');
    } finally {
      setLoading(false);
    }
  };

  const totalOrders = stocks.reduce((sum, s) => sum + (Number(s.numBids) || 0) + (Number(s.numAsks) || 0), 0);

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-medium text-gray-700">Stocks</h2>
        <button
          type="button"
          onClick={handleAddStock}
          className="text-xs text-indigo-600 hover:text-indigo-700 font-medium"
        >
          + Add Stock
        </button>
      </div>

      <div className="space-y-3">
        {stocks.map((stock, index) => (
          <StockConfigCard
            key={index}
            config={stock}
            index={index}
            onChange={handleStockChange}
            onRemove={handleRemoveStock}
            canRemove={stocks.length > 1}
          />
        ))}
      </div>

      {error && (
        <div className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
          {error}
        </div>
      )}

      <button
        type="submit"
        disabled={loading || totalOrders === 0}
        className={`w-full py-2.5 rounded-lg text-sm font-medium transition-colors ${
          loading || totalOrders === 0
            ? 'bg-gray-200 text-gray-500 cursor-not-allowed'
            : 'bg-indigo-600 text-white hover:bg-indigo-700'
        }`}
      >
        {loading ? 'Running...' : `Run Simulation (${totalOrders} orders)`}
      </button>
    </form>
  );
}
