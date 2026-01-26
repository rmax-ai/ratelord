// web/src/features/identities/utils.ts

export interface Identity {
  id: string;
  kind: string;
  labels: Record<string, any>;
}

export interface IdentityNode extends Identity {
  children: IdentityNode[];
}

export function buildIdentityTree(identities: Identity[]): IdentityNode[] {
  const identityMap = new Map<string, IdentityNode>();
  const roots: IdentityNode[] = [];

  // First pass: create nodes
  for (const identity of identities) {
    const node: IdentityNode = { ...identity, children: [] };
    identityMap.set(identity.id, node);
  }

  // Second pass: build hierarchy
  for (const node of identityMap.values()) {
    let parentId: string | undefined;

    if (node.kind === 'scope' && node.labels.agent_id) {
      parentId = node.labels.agent_id;
    } else if (node.kind === 'pool' && node.labels.scope_id) {
      parentId = node.labels.scope_id;
    }

    if (parentId && identityMap.has(parentId)) {
      identityMap.get(parentId)!.children.push(node);
    } else {
      // If no parent or parent not found, treat as root
      roots.push(node);
    }
  }

  return roots;
}