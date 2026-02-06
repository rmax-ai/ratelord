import { RatelordClientOptions, RatelordDecision, RatelordIntent } from './types';

export class RatelordClient {
  private endpoint: string;
  private timeout: number;

  constructor(options?: RatelordClientOptions) {
    this.endpoint = options?.endpoint || 'http://127.0.0.1:8090';
    this.timeout = options?.timeout || 1000;
  }

  /**
   * Negotiate intent with the daemon.
   * - Blocks (awaits) if the daemon requests a wait.
   * - Returns allowed=false if daemon is unreachable (fail-closed).
   */
  async ask(intent: RatelordIntent): Promise<RatelordDecision> {
    try {
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

      if (!response.ok) {
        // Fail-closed on server error
        return {
          allowed: false,
          intentId: '',
          status: 'deny_with_reason',
          reason: `upstream_error: ${response.status} ${response.statusText}`
        };
      }

      const rawDecision = await response.json();
      
      // Map API (snake_case) to TS (camelCase)
      // The API returns fields like 'decision', 'intent_id', 'modifications'
      
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
        // Sleep for the requested duration
        // We log to stderr to be unobtrusive but helpful
        // console.error(`[ratelord] Rate limiting: waiting ${decision.modifications.waitSeconds}s...`);
        await new Promise(resolve => setTimeout(resolve, decision.modifications!.waitSeconds! * 1000));
      }

      return decision;

    } catch (error) {
      // Fail-closed on network error
      return {
        allowed: false,
        intentId: '',
        status: 'deny_with_reason',
        reason: error instanceof Error ? `daemon_unreachable: ${error.message}` : 'daemon_unreachable'
      };
    }
  }
}
