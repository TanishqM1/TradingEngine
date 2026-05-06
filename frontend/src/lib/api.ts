import { SimulationRequest, SimulationResponse } from './types';

const API_BASE = 'http://localhost:8000/order';

export async function runSimulation(request: SimulationRequest): Promise<SimulationResponse> {
  const response = await fetch(`${API_BASE}/simulation`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.Message || `Simulation failed with status ${response.status}`);
  }

  return response.json();
}

export async function resetEngine(): Promise<void> {
  const response = await fetch(`${API_BASE}/reset`, {
    method: 'POST',
  });

  if (!response.ok) {
    throw new Error('Failed to reset engine');
  }
}

export async function getStatus(): Promise<Record<string, unknown>> {
  const response = await fetch(`${API_BASE}/status`);

  if (!response.ok) {
    throw new Error('Failed to get status');
  }

  return response.json();
}
