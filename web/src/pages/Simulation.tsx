import { useState } from 'react';
import { 
  Play, 
  Plus, 
  Trash2,
  AlertTriangle
} from 'lucide-react';

interface AgentConfig {
  name: string;
  count: number;
  identity_id: string;
  scope_id: string;
  priority: string;
  behavior: 'greedy' | 'periodic' | 'poisson';
  rate: number;
  burst?: number;
  jitter?: number;
}

interface ScenarioConfig {
  name: string;
  description: string;
  duration: number; // nanoseconds
  agents: AgentConfig[];
  invariants?: Invariant[];
}

interface Invariant {
  metric: string;
  condition: string;
  value: number;
  scope: string;
}

interface SimulationResult {
  scenario_name: string;
  duration: number;
  total_requests: number;
  total_approved: number;
  total_denied: number;
  total_modified: number;
  total_errors: number;
  agent_stats: Record<string, {
    requests: number;
    approved: number;
    denied: number;
    modified: number;
    errors: number;
  }>;
  invariants: {
    metric: string;
    scope: string;
    expected: string;
    actual: string;
    passed: boolean;
  }[];
  success: boolean;
}

const Simulation = () => {
  // Local state for the form (simplified view)
  const [formState, setFormState] = useState({
    name: 'Lab Simulation',
    durationSeconds: 10,
    agents: [
      { id: '1', name: 'Web Crawler', type: 'greedy', rps: 10 },
      { id: '2', name: 'Background Job', type: 'periodic', rps: 2 }
    ]
  });

  const [isRunning, setIsRunning] = useState(false);
  const [result, setResult] = useState<SimulationResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleStart = async () => {
    setIsRunning(true);
    setResult(null);
    setError(null);

    // Map form state to API payload
    const payload: ScenarioConfig = {
      name: formState.name,
      description: "Generated from Web UI",
      duration: formState.durationSeconds * 1_000_000_000,
      agents: formState.agents.map(a => ({
        name: a.name,
        count: 1,
        identity_id: `sim-${a.name.toLowerCase().replace(/\s+/g, '-')}`,
        scope_id: 'default',
        priority: 'normal',
        behavior: a.type as any,
        rate: a.rps,
        burst: 0,
        jitter: 0
      }))
    };

    try {
      const response = await fetch('/v1/simulation', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          // Assuming auth is handled by cookie or global header, or add here if needed
          // For now, in dev mode, we might need a token if auth is strict.
          // Let's assume standard auth header injection or open access for localhost
          'Authorization': `Bearer ${localStorage.getItem('token') || ''}`
        },
        body: JSON.stringify(payload)
      });

      if (!response.ok) {
        throw new Error(`Simulation failed: ${response.statusText}`);
      }

      const data = await response.json();
      setResult(data);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsRunning(false);
    }
  };

  const addAgent = () => {
    const newAgent = {
      id: Math.random().toString(36).substr(2, 9),
      name: `Agent ${formState.agents.length + 1}`,
      type: 'periodic',
      rps: 5
    };
    setFormState({ ...formState, agents: [...formState.agents, newAgent] });
  };

  const removeAgent = (id: string) => {
    setFormState({ ...formState, agents: formState.agents.filter(a => a.id !== id) });
  };

  const updateAgent = (id: string, field: string, value: any) => {
    setFormState({
      ...formState,
      agents: formState.agents.map(a => a.id === id ? { ...a, [field]: value } : a)
    });
  };

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Simulation Lab</h1>
          <p className="text-sm text-gray-500">Design and run synthetic workloads against the daemon</p>
        </div>
        <div className="flex space-x-2">
          {!isRunning ? (
            <button
              onClick={handleStart}
              className="flex items-center px-4 py-2 bg-indigo-600 text-white rounded hover:bg-indigo-700 shadow-sm transition-colors"
            >
              <Play className="h-4 w-4 mr-2" />
              Run Simulation
            </button>
          ) : (
            <button
              disabled
              className="flex items-center px-4 py-2 bg-gray-400 text-white rounded cursor-not-allowed"
            >
              <div className="animate-spin h-4 w-4 mr-2 border-2 border-white border-t-transparent rounded-full"></div>
              Running...
            </button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Configuration Panel */}
        <div className="lg:col-span-4 space-y-6">
          <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
            <h2 className="text-sm font-semibold text-gray-900 uppercase tracking-wider mb-4">Scenario Settings</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700">Scenario Name</label>
                <input
                  type="text"
                  value={formState.name}
                  onChange={(e) => setFormState({ ...formState, name: e.target.value })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm p-2 border"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">Duration (seconds)</label>
                <div className="mt-1 relative rounded-md shadow-sm">
                  <input
                    type="number"
                    min="1"
                    max="60"
                    value={formState.durationSeconds}
                    onChange={(e) => setFormState({ ...formState, durationSeconds: parseInt(e.target.value) })}
                    className="block w-full rounded-md border-gray-300 focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm p-2 border"
                  />
                  <div className="absolute inset-y-0 right-0 pr-3 flex items-center pointer-events-none">
                    <span className="text-gray-500 sm:text-sm">sec</span>
                  </div>
                </div>
                <p className="mt-1 text-xs text-gray-500">Max 60s for lab runs.</p>
              </div>
            </div>
          </div>

          <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-sm font-semibold text-gray-900 uppercase tracking-wider">Agents</h2>
              <button
                onClick={addAgent}
                className="p-1 rounded-full hover:bg-gray-100 text-indigo-600"
                title="Add Agent"
              >
                <Plus className="h-5 w-5" />
              </button>
            </div>
            
            <div className="space-y-4">
              {formState.agents.map((agent) => (
                <div key={agent.id} className="border rounded-md p-4 relative bg-gray-50 hover:border-indigo-300 transition-colors">
                  <button
                    onClick={() => removeAgent(agent.id)}
                    className="absolute top-2 right-2 text-gray-400 hover:text-red-500"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                  <div className="space-y-3">
                    <input
                      type="text"
                      value={agent.name}
                      onChange={(e) => updateAgent(agent.id, 'name', e.target.value)}
                      className="block w-full text-sm font-medium bg-transparent border-b border-gray-300 focus:border-indigo-500 focus:outline-none pb-1"
                      placeholder="Agent Name"
                    />
                    <div className="grid grid-cols-2 gap-2">
                      <div>
                        <label className="block text-xs text-gray-500 mb-1">Behavior</label>
                        <select
                          value={agent.type}
                          onChange={(e) => updateAgent(agent.id, 'type', e.target.value)}
                          className="block w-full text-xs rounded border-gray-300 py-1"
                        >
                          <option value="greedy">Greedy</option>
                          <option value="periodic">Periodic</option>
                          <option value="poisson">Poisson</option>
                        </select>
                      </div>
                      <div>
                        <label className="block text-xs text-gray-500 mb-1">Rate (RPS)</label>
                        <input
                          type="number"
                          value={agent.rps}
                          onChange={(e) => updateAgent(agent.id, 'rps', parseInt(e.target.value))}
                          className="block w-full text-xs rounded border-gray-300 py-1 pl-2"
                        />
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Results / Visualization Panel */}
        <div className="lg:col-span-8 space-y-6">
          <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200 min-h-[500px]">
            <h2 className="text-lg font-semibold mb-6">Results</h2>
            
            {isRunning ? (
              <div className="flex flex-col items-center justify-center h-64 text-gray-500">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600 mb-4"></div>
                <p className="font-medium">Simulating traffic...</p>
                <p className="text-sm mt-2">Watch the "Dashboard" or "Graph" tab for real-time updates.</p>
              </div>
            ) : error ? (
              <div className="bg-red-50 border border-red-200 rounded-md p-4 text-red-700 flex items-start">
                <AlertTriangle className="h-5 w-5 mr-3 mt-0.5" />
                <div>
                  <h3 className="font-medium">Simulation Error</h3>
                  <p className="text-sm mt-1">{error}</p>
                </div>
              </div>
            ) : result ? (
              <div className="space-y-6">
                {/* Summary Stats */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div className="bg-gray-50 p-4 rounded-lg">
                    <p className="text-xs text-gray-500 uppercase">Requests</p>
                    <p className="text-2xl font-bold text-gray-900">{result.total_requests}</p>
                  </div>
                  <div className="bg-green-50 p-4 rounded-lg">
                    <p className="text-xs text-green-600 uppercase">Approved</p>
                    <p className="text-2xl font-bold text-green-700">{result.total_approved}</p>
                  </div>
                  <div className="bg-red-50 p-4 rounded-lg">
                    <p className="text-xs text-red-600 uppercase">Denied</p>
                    <p className="text-2xl font-bold text-red-700">{result.total_denied}</p>
                  </div>
                  <div className="bg-yellow-50 p-4 rounded-lg">
                    <p className="text-xs text-yellow-600 uppercase">Modified</p>
                    <p className="text-2xl font-bold text-yellow-700">{result.total_modified}</p>
                  </div>
                </div>

                {/* Agent Breakdown */}
                <div>
                  <h3 className="text-sm font-medium text-gray-700 mb-3">Agent Breakdown</h3>
                  <div className="overflow-x-auto">
                    <table className="min-w-full divide-y divide-gray-200">
                      <thead className="bg-gray-50">
                        <tr>
                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Agent</th>
                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Requests</th>
                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Rate</th>
                        </tr>
                      </thead>
                      <tbody className="bg-white divide-y divide-gray-200">
                        {Object.entries(result.agent_stats).map(([name, stats]) => (
                          <tr key={name}>
                            <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{name}</td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                              <div className="flex flex-col">
                                <span>Total: {stats.requests}</span>
                                <span className="text-green-600 text-xs">{stats.approved} OK</span>
                                <span className="text-red-600 text-xs">{stats.denied} Denied</span>
                              </div>
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                              {((stats.approved / stats.requests) * 100).toFixed(1)}% Approved
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center h-64 text-gray-400 border-2 border-dashed border-gray-200 rounded-lg">
                <Play className="h-12 w-12 mb-2 opacity-50" />
                <p>Configure a scenario and run it to see results.</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Simulation;

