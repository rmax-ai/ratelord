const API_BASE = '/v1';

export interface Event {
  event_id: string;
  event_type: string;
  schema_version: number;
  ts_event: string; // ISO string
  ts_ingest: string;
  source: {
    origin_kind: string;
    origin_id: string;
    writer_id: string;
  };
  dimensions: {
    agent_id: string;
    identity_id: string;
    workload_id: string;
    scope_id: string;
  };
  correlation: {
    correlation_id: string;
    causation_id: string;
  };
  payload: Record<string, any>;
}

export interface Identity {
  id: string;
  kind: string;
  labels: Record<string, string>;
}

export interface ClusterNode {
  node_id: string;
  last_seen: string; // RFC3339
  status: string;
  metadata?: Record<string, any>;
}

export interface ClusterTopology {
  leader_id: string;
  nodes: ClusterNode[];
}

export interface GraphNode {
  id: string;
  type: string;
  label: string;
  properties?: Record<string, string>;
}

export interface GraphEdge {
  from_id: string;
  to_id: string;
  type: string;
}

export interface GraphData {
  nodes: Record<string, GraphNode>;
  edges: GraphEdge[];
}

export interface Status {
  total_events: number;
  // Add more status fields as needed
}

export const api = {
  async fetchEvents(params?: { from?: string; to?: string; limit?: number }): Promise<Event[]> {
    const searchParams = new URLSearchParams();
    if (params?.from) searchParams.append('from', params.from);
    if (params?.to) searchParams.append('to', params.to);
    if (params?.limit) searchParams.append('limit', params.limit.toString());

    const queryString = searchParams.toString();
    const url = queryString ? `${API_BASE}/events?${queryString}` : `${API_BASE}/events`;

    const response = await fetch(url);
    if (!response.ok) throw new Error('Failed to fetch events');
    return response.json();
  },

  async fetchStatus(): Promise<Status> {
    const response = await fetch(`${API_BASE}/status`);
    if (!response.ok) throw new Error('Failed to fetch status');
    return response.json();
  },

  async fetchIdentities(): Promise<Identity[]> {
    const response = await fetch(`${API_BASE}/identities`);
    if (!response.ok) throw new Error('Failed to fetch identities');
    return response.json();
  },

  async fetchClusterNodes(): Promise<ClusterTopology> {
    const response = await fetch(`${API_BASE}/cluster/nodes`);
    if (!response.ok) throw new Error('Failed to fetch cluster nodes');
    return response.json();
  },

  async fetchGraph(): Promise<GraphData> {
    const response = await fetch(`${API_BASE}/graph`);
    if (!response.ok) throw new Error('Failed to fetch graph');
    return response.json();
  },
};
