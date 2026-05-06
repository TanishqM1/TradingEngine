'use client';

import { SimulationResponse } from '@/lib/types';

interface ResultsDisplayProps {
  response: SimulationResponse | null;
}

export default function ResultsDisplay({ response }: ResultsDisplayProps) {
  if (!response) return null;

  const totalTrades = response.results.reduce((sum, r) => sum + r.tradesExecuted, 0);
  const totalVolume = response.results.reduce((sum, r) => sum + r.volumeTraded, 0);
  const totalRemaining = response.results.reduce((sum, r) => sum + r.remainingBids + r.remainingAsks, 0);
  const throughput = Math.round(response.totalOrdersProcessed / (response.executionTimeMs / 1000));
  const numEngines = response.results.length;

  return (
    <div className="space-y-6">
      {/* Performance Summary */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-medium text-gray-500">Performance Summary</h2>
          {numEngines > 1 && (
            <span className="text-xs text-indigo-600 bg-indigo-50 px-2 py-1 rounded-full">
              {numEngines} distributed engines
            </span>
          )}
        </div>

        {/* Main Metric */}
        <div className="text-center mb-6">
          <div className="text-4xl font-bold text-gray-900">{response.executionTimeMs.toFixed(2)} ms</div>
          <div className="text-sm text-gray-500 mt-1">Execution Time</div>
        </div>

        {/* Stats Grid */}
        <div className="grid grid-cols-5 gap-4 text-center">
          <div>
            <div className="text-xl font-semibold text-gray-900">{response.totalOrdersProcessed.toLocaleString()}</div>
            <div className="text-xs text-gray-500">Orders</div>
          </div>
          <div>
            <div className="text-xl font-semibold text-gray-900">{totalTrades.toLocaleString()}</div>
            <div className="text-xs text-gray-500">Trades</div>
          </div>
          <div>
            <div className="text-xl font-semibold text-gray-900">{totalVolume.toLocaleString()}</div>
            <div className="text-xs text-gray-500">Volume</div>
          </div>
          <div>
            <div className="text-xl font-semibold text-gray-900">{throughput.toLocaleString()}</div>
            <div className="text-xs text-gray-500">Orders/sec</div>
          </div>
          <div>
            <div className="text-xl font-semibold text-indigo-600">{numEngines}</div>
            <div className="text-xs text-gray-500">Engines</div>
          </div>
        </div>
      </div>

      {/* Results by Stock */}
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200">
          <h2 className="text-sm font-medium text-gray-500">Results by Stock (1 Engine per Stock)</h2>
        </div>

        {/* Table */}
        <table className="w-full">
          <thead>
            <tr className="bg-gray-50 text-xs text-gray-500 uppercase tracking-wider">
              <th className="px-6 py-3 text-left font-medium">Symbol</th>
              <th className="px-6 py-3 text-right font-medium">Trades</th>
              <th className="px-6 py-3 text-right font-medium">Volume</th>
              <th className="px-6 py-3 text-right font-medium">Bids Left</th>
              <th className="px-6 py-3 text-right font-medium">Asks Left</th>
              <th className="px-6 py-3 text-right font-medium">Best Bid</th>
              <th className="px-6 py-3 text-right font-medium">Best Ask</th>
              <th className="px-6 py-3 text-right font-medium">Spread</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {response.results.map((result) => {
              const spread = result.bestBidPrice !== null && result.bestAskPrice !== null
                ? result.bestAskPrice - result.bestBidPrice
                : null;
              const allMatched = result.remainingBids === 0 && result.remainingAsks === 0 && result.tradesExecuted > 0;

              return (
                <tr key={result.symbol} className={allMatched ? 'bg-green-50' : ''}>
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-gray-900">{result.symbol}</span>
                      {allMatched && (
                        <span className="text-xs text-green-600 bg-green-100 px-1.5 py-0.5 rounded">matched</span>
                      )}
                    </div>
                  </td>
                  <td className="px-6 py-4 text-right text-sm text-gray-900">{result.tradesExecuted}</td>
                  <td className="px-6 py-4 text-right text-sm text-gray-900">{result.volumeTraded.toLocaleString()}</td>
                  <td className="px-6 py-4 text-right text-sm">
                    <span className={result.remainingBids > 0 ? 'text-green-600' : 'text-gray-400'}>
                      {result.remainingBids}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-right text-sm">
                    <span className={result.remainingAsks > 0 ? 'text-red-600' : 'text-gray-400'}>
                      {result.remainingAsks}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-right text-sm">
                    {result.bestBidPrice !== null ? (
                      <span className="text-green-600">${result.bestBidPrice}</span>
                    ) : (
                      <span className="text-gray-400">—</span>
                    )}
                  </td>
                  <td className="px-6 py-4 text-right text-sm">
                    {result.bestAskPrice !== null ? (
                      <span className="text-red-600">${result.bestAskPrice}</span>
                    ) : (
                      <span className="text-gray-400">—</span>
                    )}
                  </td>
                  <td className="px-6 py-4 text-right text-sm">
                    {spread !== null ? (
                      <span className="text-gray-900">${spread}</span>
                    ) : (
                      <span className="text-gray-400">—</span>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>

        {/* Summary Footer */}
        <div className="px-6 py-3 bg-gray-50 border-t border-gray-200 text-xs text-gray-500 flex justify-between">
          <span>
            {totalRemaining === 0 ? (
              <span className="text-green-600">All orders matched successfully</span>
            ) : (
              <span>{totalRemaining} orders remaining in orderbook</span>
            )}
          </span>
          <span className="text-indigo-600">
            Processed in parallel across {numEngines} engine{numEngines > 1 ? 's' : ''}
          </span>
        </div>
      </div>
    </div>
  );
}
