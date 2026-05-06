'use client';

import { StockResult } from '@/lib/types';

interface StockResultCardProps {
  result: StockResult;
}

export default function StockResultCard({ result }: StockResultCardProps) {
  const totalRemaining = result.remainingBids + result.remainingAsks;
  const allMatched = totalRemaining === 0 && result.tradesExecuted > 0;
  const hasSpread = result.bestBidPrice !== null && result.bestAskPrice !== null;
  const spread = hasSpread ? result.bestAskPrice! - result.bestBidPrice! : null;

  return (
    <div className={`bg-white rounded-xl shadow-sm border overflow-hidden ${allMatched ? 'border-emerald-200' : 'border-gray-100'}`}>
      <div className="p-4">
        {/* Header Row */}
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-3">
            <div className={`h-10 w-10 rounded-lg flex items-center justify-center text-white font-bold text-lg ${
              allMatched
                ? 'bg-gradient-to-br from-emerald-500 to-green-600'
                : 'bg-gradient-to-br from-indigo-500 to-purple-600'
            }`}>
              {result.symbol.charAt(0)}
            </div>
            <div>
              <h3 className="text-lg font-bold text-gray-900">{result.symbol}</h3>
              {allMatched && <span className="text-xs text-emerald-600 font-medium">All matched!</span>}
            </div>
          </div>
          <div className="text-right">
            <div className="text-2xl font-bold text-gray-900">{result.tradesExecuted}</div>
            <div className="text-xs text-gray-500">trades</div>
          </div>
        </div>

        {/* Stats Row - Horizontal Layout */}
        <div className="grid grid-cols-5 gap-3 text-center">
          {/* Volume */}
          <div className="bg-blue-50 rounded-lg p-2">
            <div className="text-lg font-semibold text-blue-800">{result.volumeTraded.toLocaleString()}</div>
            <div className="text-xs text-blue-600">volume</div>
          </div>

          {/* Remaining Bids */}
          <div className="bg-green-50 rounded-lg p-2">
            <div className="text-lg font-semibold text-green-800">{result.remainingBids}</div>
            <div className="text-xs text-green-600">bids left</div>
          </div>

          {/* Remaining Asks */}
          <div className="bg-red-50 rounded-lg p-2">
            <div className="text-lg font-semibold text-red-800">{result.remainingAsks}</div>
            <div className="text-xs text-red-600">asks left</div>
          </div>

          {/* Best Bid */}
          <div className="bg-gray-50 rounded-lg p-2">
            <div className="text-lg font-semibold text-green-700">
              {result.bestBidPrice !== null ? `$${result.bestBidPrice}` : '—'}
            </div>
            <div className="text-xs text-gray-500">best bid</div>
          </div>

          {/* Best Ask */}
          <div className="bg-gray-50 rounded-lg p-2">
            <div className="text-lg font-semibold text-red-700">
              {result.bestAskPrice !== null ? `$${result.bestAskPrice}` : '—'}
            </div>
            <div className="text-xs text-gray-500">best ask</div>
          </div>
        </div>

        {/* Spread - Only show if exists */}
        {hasSpread && spread !== null && (
          <div className="mt-3 pt-3 border-t border-gray-100 flex items-center justify-between">
            <span className="text-sm text-gray-500">Spread</span>
            <span className="text-sm font-medium text-amber-700">
              ${spread} ({((spread / result.bestBidPrice!) * 100).toFixed(2)}%)
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
