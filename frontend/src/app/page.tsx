'use client';

import { useState } from 'react';
import { SimulationResponse } from '@/lib/types';
import SimulationForm from '@/components/SimulationForm';
import ResultsDisplay from '@/components/ResultsDisplay';

export default function Home() {
  const [results, setResults] = useState<SimulationResponse | null>(null);

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white border-b border-gray-200">
        <div className="max-w-6xl mx-auto px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="h-9 w-9 bg-indigo-600 rounded-lg flex items-center justify-center">
              <svg className="h-5 w-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
              </svg>
            </div>
            <div>
              <h1 className="text-lg font-semibold text-gray-900">Trading Engine</h1>
              <p className="text-xs text-gray-500">CLOB Matching Simulator</p>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-6xl mx-auto px-6 py-8">
        <div className="flex flex-col lg:flex-row gap-8">
          {/* Left - Configuration */}
          <div className="w-full lg:w-80 flex-shrink-0">
            <SimulationForm onResults={setResults} />
          </div>

          {/* Right - Results */}
          <div className="flex-1 min-w-0">
            {results ? (
              <ResultsDisplay response={results} />
            ) : (
              <EmptyState />
            )}
          </div>
        </div>
      </main>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="bg-white rounded-lg border border-gray-200 p-12 text-center h-full flex flex-col items-center justify-center min-h-[400px]">
      <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mb-4">
        <svg className="w-8 h-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
        </svg>
      </div>
      <h3 className="text-lg font-medium text-gray-900 mb-2">No Results Yet</h3>
      <p className="text-sm text-gray-500 max-w-xs">
        Configure your simulation on the left and click "Run Simulation" to see the matching engine results.
      </p>
    </div>
  );
}
