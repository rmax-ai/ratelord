import { Link, Outlet } from 'react-router-dom';
import { Home, History, Users, Play } from 'lucide-react';

const AppShell = () => {
  return (
    <div className="flex h-screen bg-gray-100">
      {/* Sidebar */}
      <div className="w-64 bg-white shadow-md">
        <div className="p-4">
          <h1 className="text-xl font-bold text-gray-800">Ratelord</h1>
        </div>
        <nav className="mt-4">
          <Link
            to="/"
            className="flex items-center px-4 py-2 text-gray-700 hover:bg-gray-200"
          >
            <Home className="mr-2 h-5 w-5" />
            Dashboard
          </Link>
          <Link
            to="/history"
            className="flex items-center px-4 py-2 text-gray-700 hover:bg-gray-200"
          >
            <History className="mr-2 h-5 w-5" />
            History
          </Link>
          <Link
            to="/identities"
            className="flex items-center px-4 py-2 text-gray-700 hover:bg-gray-200"
          >
            <Users className="mr-2 h-5 w-5" />
            Identities
          </Link>
          <Link
            to="/simulate"
            className="flex items-center px-4 py-2 text-gray-700 hover:bg-gray-200"
          >
            <Play className="mr-2 h-5 w-5" />
            Simulate
          </Link>
        </nav>
      </div>
      {/* Content Area */}
      <div className="flex-1 overflow-auto">
        <Outlet />
      </div>
    </div>
  );
};

export default AppShell;