// web/src/features/identities/components/IdentityTree.tsx

import { IdentityNode } from '../utils';

interface IdentityTreeProps {
  nodes: IdentityNode[];
  level?: number;
}

export function IdentityTree({ nodes, level = 0 }: IdentityTreeProps) {
  return (
    <ul style={{ marginLeft: `${level * 16}px` }}>
      {nodes.map((node) => (
        <li key={node.id} className="mb-2">
          <div className="font-medium">
            {node.kind}: {node.id}
          </div>
          {node.children.length > 0 && (
            <IdentityTree nodes={node.children} level={level + 1} />
          )}
        </li>
      ))}
    </ul>
  );
}