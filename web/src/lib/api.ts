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
  identity_id: string;
  provider_id?: string;
  // Add more fields as needed
}

export interface Status {
  total_events: number;
  // Add more status fields as needed
}

export const api = {
  async fetchEvents(): Promise<Event[]> {
    const response = await fetch(`${API_BASE}/events`);
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
};