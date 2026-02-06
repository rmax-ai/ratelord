import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

const Cluster = () => {
  const { data: topology, isLoading, error } = useQuery({
    queryKey: ['clusterNodes'],
    queryFn: api.fetchClusterNodes,
  });

  if (isLoading) return <div className="p-6">Loading...</div>;
  if (error) return <div className="p-6">Error loading cluster data</div>;

  const formatLastSeen = (lastSeen: string) => {
    const date = new Date(lastSeen);
    return date.toLocaleString();
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Cluster Topology</h1>

      <div className="bg-white dark:bg-gray-800 p-4 rounded shadow">
        <table className="min-w-full table-auto">
          <thead>
            <tr className="bg-gray-50 dark:bg-gray-700">
              <th className="px-4 py-2 text-left">Node ID</th>
              <th className="px-4 py-2 text-left">Address</th>
              <th className="px-4 py-2 text-left">Role</th>
              <th className="px-4 py-2 text-left">State</th>
              <th className="px-4 py-2 text-left">Last Seen</th>
            </tr>
          </thead>
          <tbody>
            {topology?.nodes.map((node) => (
              <tr
                key={node.id}
                className={`border-t ${node.id === topology.leader_id ? 'bg-yellow-50 dark:bg-yellow-900' : ''}`}
              >
                <td className="px-4 py-2">{node.id}</td>
                <td className="px-4 py-2">{node.addr}</td>
                <td className="px-4 py-2">{node.id === topology.leader_id ? 'Leader' : 'Follower'}</td>
                <td className="px-4 py-2">{node.state}</td>
                <td className="px-4 py-2">{formatLastSeen(node.last_seen)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default Cluster;