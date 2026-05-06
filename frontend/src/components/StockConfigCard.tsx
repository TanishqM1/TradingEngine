'use client';

import { StockSimConfig } from '@/lib/types';

interface StockConfigCardProps {
  config: StockSimConfig;
  index: number;
  onChange: (index: number, config: StockSimConfig) => void;
  onRemove: (index: number) => void;
  canRemove: boolean;
}

export default function StockConfigCard({
  config,
  index,
  onChange,
  onRemove,
  canRemove,
}: StockConfigCardProps) {
  const handleNumberChange = (field: keyof StockSimConfig, value: string) => {
    if (value === '') {
      onChange(index, { ...config, [field]: '' as unknown as number });
    } else {
      const num = parseInt(value, 10);
      if (!isNaN(num)) {
        onChange(index, { ...config, [field]: num });
      }
    }
  };

  const getValue = (value: number | string): string => {
    if (value === '' || value === undefined || value === null) return '';
    return String(value);
  };

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4 shadow-sm overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <input
          type="text"
          value={config.symbol}
          onChange={(e) => onChange(index, { ...config, symbol: e.target.value.toUpperCase() })}
          placeholder="SYM"
          maxLength={5}
          className="text-sm font-bold text-gray-900 bg-white border border-gray-300 rounded-md px-2 py-1.5 w-20 focus:outline-none focus:ring-2 focus:ring-indigo-500 placeholder:text-gray-400 uppercase"
        />
        {canRemove && (
          <button
            onClick={() => onRemove(index)}
            className="text-gray-400 hover:text-red-500 transition-colors p-1"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        )}
      </div>

      {/* Orders Row */}
      <div className="grid grid-cols-2 gap-2 mb-3">
        <div>
          <label className="block text-xs text-gray-500 mb-1">Bids</label>
          <input
            type="number"
            value={getValue(config.numBids)}
            onChange={(e) => handleNumberChange('numBids', e.target.value)}
            placeholder="100"
            className="w-full px-2 py-1.5 text-sm text-gray-900 bg-white border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 placeholder:text-gray-400"
          />
        </div>
        <div>
          <label className="block text-xs text-gray-500 mb-1">Asks</label>
          <input
            type="number"
            value={getValue(config.numAsks)}
            onChange={(e) => handleNumberChange('numAsks', e.target.value)}
            placeholder="100"
            className="w-full px-2 py-1.5 text-sm text-gray-900 bg-white border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 placeholder:text-gray-400"
          />
        </div>
      </div>

      {/* Price Row */}
      <div className="mb-3">
        <label className="block text-xs text-gray-500 mb-1">Price ($)</label>
        <div className="grid grid-cols-2 gap-2">
          <input
            type="number"
            value={getValue(config.priceMin)}
            onChange={(e) => handleNumberChange('priceMin', e.target.value)}
            placeholder="Min"
            className="w-full px-2 py-1.5 text-sm text-gray-900 bg-white border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 placeholder:text-gray-400"
          />
          <input
            type="number"
            value={getValue(config.priceMax)}
            onChange={(e) => handleNumberChange('priceMax', e.target.value)}
            placeholder="Max"
            className="w-full px-2 py-1.5 text-sm text-gray-900 bg-white border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 placeholder:text-gray-400"
          />
        </div>
      </div>

      {/* Quantity Row */}
      <div>
        <label className="block text-xs text-gray-500 mb-1">Quantity</label>
        <div className="grid grid-cols-2 gap-2">
          <input
            type="number"
            value={getValue(config.quantityMin)}
            onChange={(e) => handleNumberChange('quantityMin', e.target.value)}
            placeholder="Min"
            className="w-full px-2 py-1.5 text-sm text-gray-900 bg-white border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 placeholder:text-gray-400"
          />
          <input
            type="number"
            value={getValue(config.quantityMax)}
            onChange={(e) => handleNumberChange('quantityMax', e.target.value)}
            placeholder="Max"
            className="w-full px-2 py-1.5 text-sm text-gray-900 bg-white border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 placeholder:text-gray-400"
          />
        </div>
      </div>
    </div>
  );
}
