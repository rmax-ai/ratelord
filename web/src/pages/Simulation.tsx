import { useState } from 'react';
import { 
  Play, 
  Square, 
  Plus, 
  Trash2 
} from 'lucide-react';

interface AgentConfig {
  id: string;
  name: string;
  type: 'greedy' | 'periodic' | 'poisson';
  rps: number;
}

interface ScenarioConfig {
  name: string;
  durationSeconds: number;
  agents: AgentConfig[];
}

const Simulation = () => {
  const [scenario, setScenario] = useState<ScenarioConfig>({
    name: 'New Scenario',
    durationSeconds: 60,
    agents: [
      { id: '1', name: 'Web Crawler', type: 'greedy', rps: 10 },
      { id: '2', name: 'Background Job', type: 'periodic', rps: 1 }
    ]
  });

  const [isRunning, setIsRunning] = useState(false);
  const [simulationResult, setSimulationResult] = useState<string | null>(null);

  const handleStart = () => {
    setIsRunning(true);
    setSimulationResult(null);
    // TODO: Call backend API to start simulation
    setTimeout(() => {
      setIsRunning(false);
      setSimulationResult('Simulation completed successfully. 95% of intents approved.');
    }, 2000);
  };

  const addAgent = () => {
    const newAgent: AgentConfig = {
      id: Math.random().toString(36).substr(2, 9),
      name: `Agent ${scenario.agents.length + 1}`,
      type: 'periodic',
      rps: 5
    };
    setScenario({ ...scenario, agents: [...scenario.agents, newAgent] });
  };

  const removeAgent = (id: string) => {
    setScenario({ ...scenario, agents: scenario.agents.filter(a => a.id !== id) });
  };

  const updateAgent = (id: string, field: keyof AgentConfig, value: any) => {
    setScenario({
      ...scenario,
      agents: scenario.agents.map(a => a.id === id ? { ...a, [field]: value } : a)
    });
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Simulation Lab</h1>
        <div className="flex space-x-2">
          {!isRunning ? (
            <button
              onClick={handleStart}
              className="flex items-center px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700"
            >
              <Play className="h-5 w-5 mr-2" />
              Run Simulation
            </button>
          ) : (
            <button
              disabled
              className="flex items-center px-4 py-2 bg-gray-400 text-white rounded cursor-not-allowed"
            >
              <Square className="h-5 w-5 mr-2" />
              Running...
            </button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Configuration Panel */}
        <div className="lg:col-span-1 space-y-6">
          <div className="bg-white p-4 rounded shadow">
            <h2 className="text-lg font-semibold mb-4">Scenario Settings</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700">Scenario Name</label>
                <input
                  type="text"
                  value={scenario.name}
                  onChange={(e) => setScenario({ ...scenario, name: e.target.value })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm p-2 border"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">Duration (seconds)</label>
                <input
                  type="number"
                  value={scenario.durationSeconds}
                  onChange={(e) => setScenario({ ...scenario, durationSeconds: parseInt(e.target.value) })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm p-2 border"
                />
              </div>
            </div>
          </div>

          <div className="bg-white p-4 rounded shadow">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-lg font-semibold">Agents</h2>
              <button
                onClick={addAgent}
                className="p-1 rounded-full hover:bg-gray-100"
                title="Add Agent"
              >
                <Plus className="h-5 w-5 text-gray-600" />
              </button>
            </div>
            
            <div className="space-y-4">
              {scenario.agents.map((agent) => (
                <div key={agent.id} className="border rounded p-3 relative bg-gray-50">
                  <button
                    onClick={() => removeAgent(agent.id)}
                    className="absolute top-2 right-2 text-red-500 hover:text-red-700"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                  <div className="space-y-2">
                    <input
                      type="text"
                      value={agent.name}
                      onChange={(e) => updateAgent(agent.id, 'name', e.target.value)}
                      className="block w-full text-sm font-medium bg-transparent border-b border-gray-300 focus:border-indigo-500 focus:outline-none"
                    />
                    <div className="flex space-x-2">
                      <select
                        value={agent.type}
                        onChange={(e) => updateAgent(agent.id, 'type', e.target.value)}
                        className="block w-1/2 text-sm rounded border-gray-300 p-1"
                      >
                        <option value="greedy">Greedy</option>
                        <option value="periodic">Periodic</option>
                        <option value="poisson">Poisson</option>
                      </select>
                      <div className="flex items-center w-1/2">
                        <input
                          type="number"
                          value={agent.rps}
                          onChange={(e) => updateAgent(agent.id, 'rps', parseInt(e.target.value))}
                          className="w-full text-sm rounded border-gray-300 p-1"
                        />
                        <span className="ml-1 text-xs text-gray-500">RPS</span>
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Results / Visualization Panel */}
        <div className="lg:col-span-2 space-y-6">
          <div className="bg-white p-4 rounded shadow h-96 flex flex-col items-center justify-center text-gray-500 border-2 border-dashed border-gray-200">
            {isRunning ? (
              <div className="text-center">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600 mx-auto mb-4"></div>
                <p>Running simulation...</p>
              </div>
            ) : simulationResult ? (
              <div className="text-center text-green-600">
                <p className="text-xl font-semibold mb-2">Success</p>
                <p>{simulationResult}</p>
                <p className="text-sm text-gray-400 mt-4">(Visualization charts will appear here)</p>
              </div>
            ) : (
              <p>Configure a scenario and click "Run Simulation"</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Simulation;
