'use client';

import { useState } from 'react';
import { StockSimConfig, SimulationResponse, defaultStockConfig } from '@/lib/types';
import { runSimulation } from '@/lib/api';

interface SimulationFormProps {
  onResults: (response: SimulationResponse) => void;
}

export default function SimulationForm({ onResults }: SimulationFormProps) {
  const [stocks, setStocks] = useState<StockSimConfig[]>([
    { ...defaultStockConfig, symbol: 'TSLA' },
  ]);
  const [activeIndex, setActiveIndex] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const activeStock = stocks[activeIndex];

  const handleConfigChange = (field: keyof StockSimConfig, value: string | number) => {
    const newStocks = [...stocks];
    newStocks[activeIndex] = { ...newStocks[activeIndex], [field]: value };
    setStocks(newStocks);
  };

  const handleNumberChange = (field: keyof StockSimConfig, value: string) => {
    if (value === '') {
      handleConfigChange(field, '' as unknown as number);
    } else {
      const num = parseInt(value, 10);
      if (!isNaN(num)) {
        handleConfigChange(field, num);
      }
    }
  };

  const handleRemoveStock = (index: number) => {
    if (stocks.length > 1) {
      const newStocks = stocks.filter((_, i) => i !== index);
      setStocks(newStocks);
      // Adjust active index if needed
      if (activeIndex >= newStocks.length) {
        setActiveIndex(newStocks.length - 1);
      } else if (activeIndex > index) {
        setActiveIndex(activeIndex - 1);
      }
    }
  };

  const handleAddStock = () => {
    const usedSymbols = new Set(stocks.map(s => s.symbol));
    const defaultSymbols = ['AAPL', 'GOOG', 'MSFT', 'AMZN', 'META', 'NVDA', 'NFLX', 'AMD', 'INTC', 'CRM'];
    const nextSymbol = defaultSymbols.find(s => !usedSymbols.has(s)) || '';
    const newStocks = [...stocks, { ...defaultStockConfig, symbol: nextSymbol }];
    setStocks(newStocks);
    setActiveIndex(newStocks.length - 1); // Select the new stock
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

  const getValue = (value: number | string): string => {
    if (value === '' || value === undefined || value === null) return '';
    return String(value);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {/* Stock Tabs */}
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div className="flex items-center border-b border-gray-200 bg-gray-50 px-2 py-2 gap-1 flex-wrap">
          {stocks.map((stock, index) => (
            <button
              key={index}
              type="button"
              onClick={() => setActiveIndex(index)}
              className={`group relative px-3 py-1.5 text-sm font-medium rounded-md transition-all ${
                activeIndex === index
                  ? 'bg-indigo-600 text-white shadow-sm'
                  : 'bg-white text-gray-600 hover:bg-gray-100 border border-gray-200'
              }`}
            >
              {stock.symbol || 'NEW'}
              {stocks.length > 1 && (
                <span
                  onClick={(e) => {
                    e.stopPropagation();
                    handleRemoveStock(index);
                  }}
                  className={`ml-1.5 inline-flex items-center justify-center w-4 h-4 rounded-full text-xs leading-none hover:bg-red-500 hover:text-white transition-colors ${
                    activeIndex === index ? 'text-indigo-200 hover:text-white' : 'text-gray-400'
                  }`}
                >
                  x
                </span>
              )}
            </button>
          ))}
          <button
            type="button"
            onClick={handleAddStock}
            className="px-2 py-1.5 text-sm text-indigo-600 hover:text-indigo-700 hover:bg-indigo-50 rounded-md transition-colors"
          >
            +
          </button>
        </div>

        {/* Active Stock Configuration */}
        <div className="p-4 space-y-4">
          {/* Symbol Input */}
          <div>
            <label className="block text-xs text-gray-500 mb-1">Symbol</label>
            <input
              type="text"
              value={activeStock.symbol}
              onChange={(e) => handleConfigChange('symbol', e.target.value.toUpperCase())}
              placeholder="AAPL"
              maxLength={5}
              className="w-full px-3 py-2 text-sm font-bold text-gray-900 bg-white border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 placeholder:text-gray-400 uppercase"
            />
          </div>

          {/* Orders Row */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs text-gray-500 mb-1">Bids</label>
              <input
                type="number"
                value={getValue(activeStock.numBids)}
                onChange={(e) => handleNumberChange('numBids', e.target.value)}
                placeholder="100"
                className="w-full px-3 py-2 text-sm text-gray-900 bg-white border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 placeholder:text-gray-400"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">Asks</label>
              <input
                type="number"
                value={getValue(activeStock.numAsks)}
                onChange={(e) => handleNumberChange('numAsks', e.target.value)}
                placeholder="100"
                className="w-full px-3 py-2 text-sm text-gray-900 bg-white border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 placeholder:text-gray-400"
              />
            </div>
          </div>

          {/* Price Row */}
          <div>
            <label className="block text-xs text-gray-500 mb-1">Price Range ($)</label>
            <div className="grid grid-cols-2 gap-3">
              <input
                type="number"
                value={getValue(activeStock.priceMin)}
                onChange={(e) => handleNumberChange('priceMin', e.target.value)}
                placeholder="Min"
                className="w-full px-3 py-2 text-sm text-gray-900 bg-white border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 placeholder:text-gray-400"
              />
              <input
                type="number"
                value={getValue(activeStock.priceMax)}
                onChange={(e) => handleNumberChange('priceMax', e.target.value)}
                placeholder="Max"
                className="w-full px-3 py-2 text-sm text-gray-900 bg-white border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 placeholder:text-gray-400"
              />
            </div>
          </div>

          {/* Quantity Row */}
          <div>
            <label className="block text-xs text-gray-500 mb-1">Quantity Range</label>
            <div className="grid grid-cols-2 gap-3">
              <input
                type="number"
                value={getValue(activeStock.quantityMin)}
                onChange={(e) => handleNumberChange('quantityMin', e.target.value)}
                placeholder="Min"
                className="w-full px-3 py-2 text-sm text-gray-900 bg-white border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 placeholder:text-gray-400"
              />
              <input
                type="number"
                value={getValue(activeStock.quantityMax)}
                onChange={(e) => handleNumberChange('quantityMax', e.target.value)}
                placeholder="Max"
                className="w-full px-3 py-2 text-sm text-gray-900 bg-white border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 placeholder:text-gray-400"
              />
            </div>
          </div>
        </div>

        {/* Footer with summary */}
        <div className="px-4 py-2 bg-gray-50 border-t border-gray-200 text-xs text-gray-500 flex justify-between">
          <span>{stocks.length} stock{stocks.length > 1 ? 's' : ''} configured</span>
          <span>{totalOrders} total orders</span>
        </div>
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
