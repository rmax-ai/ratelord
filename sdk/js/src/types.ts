export interface RatelordIntent {
  agentId: string;
  identityId: string;
  workloadId: string;
  scopeId: string;
  urgency?: 'high' | 'normal' | 'background';
  expectedCost?: number;
  durationHint?: number;
  clientContext?: Record<string, unknown>;
}

export interface RatelordDecision {
  allowed: boolean;
  intentId: string;
  status: 'approve' | 'approve_with_modifications' | 'deny_with_reason';
  modifications?: {
    waitSeconds?: number;
    identitySwitch?: string;
  };
  reason?: string;
}

export interface RatelordClientOptions {
  endpoint?: string; // default: http://127.0.0.1:8090
  timeout?: number;  // default: 1000ms
}
