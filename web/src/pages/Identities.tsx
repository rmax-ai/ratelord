import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import { buildIdentityTree } from '../features/identities/utils';
import { IdentityTree } from '../features/identities/components/IdentityTree';

const IdentitiesPage = () => {
  const { data: identities, isLoading, error } = useQuery({
    queryKey: ['identities'],
    queryFn: api.fetchIdentities,
  });

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error loading identities</div>;

  const tree = identities ? buildIdentityTree(identities) : [];

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Identity Explorer</h1>

      {tree.length > 0 ? (
        <div className="bg-white p-4 rounded shadow">
          <h2 className="text-lg font-semibold mb-4">Identity Hierarchy</h2>
          <IdentityTree nodes={tree} />
        </div>
      ) : (
        <div className="bg-white p-4 rounded shadow">
          <p>No identities found.</p>
        </div>
      )}
    </div>
  );
};

export default IdentitiesPage;