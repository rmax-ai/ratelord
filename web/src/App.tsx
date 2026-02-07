import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import AppShell from './layouts/AppShell';
import Dashboard from './pages/Dashboard';
import History from './pages/History';
import IdentitiesPage from './pages/Identities';
import Cluster from './pages/Cluster';
import GraphPage from './pages/GraphPage';
import Simulation from './pages/Simulation';

function App() {
  const queryClient = new QueryClient();

  return (
    <QueryClientProvider client={queryClient}>
      <Router>
        <Routes>
          <Route path="/" element={<AppShell />}>
            <Route index element={<Dashboard />} />
            <Route path="history" element={<History />} />
            <Route path="identities" element={<IdentitiesPage />} />
            <Route path="cluster" element={<Cluster />} />
            <Route path="graph" element={<GraphPage />} />
            <Route path="simulate" element={<Simulation />} />
            {/* Add other routes here */}
          </Route>
        </Routes>
      </Router>
    </QueryClientProvider>
  );
}

export default App;