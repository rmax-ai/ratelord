import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import GraphView from '../features/graph/GraphView';
import { AlertCircle, Loader2 } from 'lucide-react';

export default function GraphPage() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['graph'],
    queryFn: api.fetchGraph,
    refetchInterval: 5000, // Refresh every 5s
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full text-gray-400">
        <Loader2 className="w-8 h-8 animate-spin mr-2" />
        <span>Loading constraint graph...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-full text-red-400">
        <AlertCircle className="w-8 h-8 mr-2" />
        <span>Failed to load graph data</span>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col p-4 space-y-4">
      <h1 className="text-2xl font-bold text-white">Constraint Graph</h1>
      <div className="flex-1 min-h-0">
        <GraphView data={data!} />
      </div>
    </div>
  );
}
