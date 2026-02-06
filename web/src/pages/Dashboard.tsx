import { useQuery } from '@tanstack/react-query';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { api } from '../lib/api';

// Mock data for chart since backend may not be ready
const mockChartData = [
  { name: 'Jan', events: 400 },
  { name: 'Feb', events: 300 },
  { name: 'Mar', events: 600 },
  { name: 'Apr', events: 800 },
  { name: 'May', events: 500 },
];

const Dashboard = () => {
  const { data: events, isLoading, error } = useQuery(['events'], () => api.fetchEvents());

  const totalEvents = events?.length || 0;

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error loading data</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Dashboard</h1>

      {/* Metric Card */}
      <div className="bg-white p-4 rounded shadow mb-6">
        <h2 className="text-lg font-semibold">Total Events</h2>
        <p className="text-3xl font-bold">{totalEvents}</p>
      </div>

      {/* Chart */}
      <div className="bg-white p-4 rounded shadow">
        <h2 className="text-lg font-semibold mb-4">Event Volume</h2>
        <ResponsiveContainer width="100%" height={300}>
          <LineChart data={mockChartData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="name" />
            <YAxis />
            <Tooltip />
            <Line type="monotone" dataKey="events" stroke="#8884d8" />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};

export default Dashboard;