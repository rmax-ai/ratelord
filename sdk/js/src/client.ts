import { RatelordClientOptions, RatelordDecision, RatelordIntent } from './types';

export class RatelordClient {
  private endpoint: string;
  private timeout: number;
  private maxRetries: number;
  private baseDelay: number;
  private maxDelay: number;

  constructor(options?: RatelordClientOptions) {
    this.endpoint = options?.endpoint || 'http://127.0.0.1:8090';
    this.timeout = options?.timeout || 1000;
    this.maxRetries = options?.maxRetries ?? 3;
    this.baseDelay = options?.baseDelay ?? 50;
    this.maxDelay = options?.maxDelay ?? 1000;
  }

  /**
   * Negotiate intent with the daemon.
   * - Blocks (awaits) if the daemon requests a wait.
   * - Returns allowed=false if the daemon is unreachable (fail-closed).
   */
  async ask(intent: RatelordIntent): Promise<RatelordDecision> {
    // Validate inputs
    if (!intent.agentId || !intent.scopeId || !intent.workloadId || !intent.identityId) {
      throw new Error("Invalid intent: missing required fields");
    }

    // Map TS (camelCase) to API (snake_case)
    const payload = {
      agent_id: intent.agentId,
      identity_id: intent.identityId,
      workload_id: intent.workloadId,
      scope_id: intent.scopeId,
      urgency: intent.urgency,
      expected_cost: intent.expectedCost,
      duration_hint: intent.durationHint,
      client_context: intent.clientContext,
    };

    for (let attempt = 0; attempt <= this.maxRetries; attempt++) {
      try {
        const controller = new AbortController();
        const id = setTimeout(() => controller.abort(), this.timeout);

        const response = await fetch(`${this.endpoint}/v1/intent`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(payload),
          signal: controller.signal,
        });

        clearTimeout(id);

        if (response.ok) {
          const rawDecision = await response.json();
          
          // Map API (snake_case) to TS (camelCase)
          const decision: RatelordDecision = {
            allowed: rawDecision.decision === 'approve' || rawDecision.decision === 'approve_with_modifications',
            intentId: rawDecision.intent_id,
            status: rawDecision.decision,
            reason: rawDecision.reason,
            modifications: rawDecision.modifications ? {
              waitSeconds: rawDecision.modifications.wait_seconds,
              identitySwitch: rawDecision.modifications.identity_switch
            } : undefined
          };

          // Auto-Wait Logic
          if (decision.modifications?.waitSeconds && decision.modifications.waitSeconds > 0) {
            await new Promise(resolve => setTimeout(resolve, decision.modifications!.waitSeconds! * 1000));
          }

          return decision;
        } else {
          // Check if client error (4xx), fail immediately
          if (response.status >= 400 && response.status < 500) {
            return {
              allowed: false,
              intentId: '',
              status: 'deny_with_reason',
              reason: `upstream_error: ${response.status} ${response.statusText}`
            };
          }
          // Server error (5xx), retry if attempts left
          if (attempt < this.maxRetries) {
            const delay = Math.min(this.maxDelay, this.baseDelay * Math.pow(2, attempt));
            const jitter = delay * 0.2 * Math.random();
            await new Promise(resolve => setTimeout(resolve, delay + jitter));
            continue;
          } else {
            return {
              allowed: false,
              intentId: '',
              status: 'deny_with_reason',
              reason: `upstream_error: ${response.status} ${response.statusText}`
            };
          }
        }
      } catch (error) {
        // Network error, retry if attempts left
        if (attempt < this.maxRetries) {
          const delay = Math.min(this.maxDelay, this.baseDelay * Math.pow(2, attempt));
          const jitter = delay * 0.2 * Math.random();
          await new Promise(resolve => setTimeout(resolve, delay + jitter));
          continue;
        } else {
          return {
            allowed: false,
            intentId: '',
            status: 'deny_with_reason',
            reason: error instanceof Error ? `daemon_unreachable: ${error.message}` : 'daemon_unreachable'
          };
        }
      }
    }
    // Should not reach here
    throw new Error('Unexpected error in retry loop');
  }
}
