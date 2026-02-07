import { RatelordClient } from './client';
import { RatelordIntent } from './types';

// Mock global fetch
const mockFetch = jest.fn();
global.fetch = mockFetch;

describe('RatelordClient', () => {
  let client: RatelordClient;

  beforeEach(() => {
    mockFetch.mockReset();
    client = new RatelordClient({ endpoint: 'http://test-api' });
  });

  it('should send a valid intent and return decision', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        decision: 'approve',
        intent_id: 'test-intent-id',
      }),
    });

    const intent: RatelordIntent = {
      agentId: 'agent-1',
      identityId: 'id-1',
      workloadId: 'work-1',
      scopeId: 'scope-1',
    };

    const decision = await client.ask(intent);

    expect(mockFetch).toHaveBeenCalledWith(
      'http://test-api/v1/intent',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          agent_id: 'agent-1',
          identity_id: 'id-1',
          workload_id: 'work-1',
          scope_id: 'scope-1',
        }),
      })
    );

    expect(decision).toEqual({
      allowed: true,
      intentId: 'test-intent-id',
      status: 'approve',
      reason: undefined,
      modifications: undefined,
    });
  });

  it('should fail-closed if daemon returns 500', async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 500,
      statusText: 'Internal Server Error',
    });

    const intent: RatelordIntent = {
      agentId: 'agent-1',
      identityId: 'id-1',
      workloadId: 'work-1',
      scopeId: 'scope-1',
    };

    const decision = await client.ask(intent);

    expect(decision.allowed).toBe(false);
    expect(decision.status).toBe('deny_with_reason');
    expect(decision.reason).toContain('upstream_error');
  });

  it('should fail-closed if network throws', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    const intent: RatelordIntent = {
      agentId: 'agent-1',
      identityId: 'id-1',
      workloadId: 'work-1',
      scopeId: 'scope-1',
    };

    const decision = await client.ask(intent);

    expect(decision.allowed).toBe(false);
    expect(decision.status).toBe('deny_with_reason');
    expect(decision.reason).toContain('daemon_unreachable');
  });

  it('should retry on 500 error and succeed', async () => {
    // First two calls return 500, third succeeds
    mockFetch
      .mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
      })
      .mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          decision: 'approve',
          intent_id: 'retry-success-intent',
        }),
      });

    const client = new RatelordClient({ endpoint: 'http://test-api', maxRetries: 2, baseDelay: 1, maxDelay: 10 }); // fast for test
    const intent: RatelordIntent = {
      agentId: 'agent-1',
      identityId: 'id-1',
      workloadId: 'work-1',
      scopeId: 'scope-1',
    };

    const decision = await client.ask(intent);

    expect(mockFetch).toHaveBeenCalledTimes(3);
    expect(decision.allowed).toBe(true);
    expect(decision.intentId).toBe('retry-success-intent');
  });

  it('should fail after max retries', async () => {
    // Always return 500
    mockFetch.mockResolvedValue({
      ok: false,
      status: 500,
      statusText: 'Internal Server Error',
    });

    const client = new RatelordClient({ endpoint: 'http://test-api', maxRetries: 2, baseDelay: 1, maxDelay: 10 }); // fast for test
    const intent: RatelordIntent = {
      agentId: 'agent-1',
      identityId: 'id-1',
      workloadId: 'work-1',
      scopeId: 'scope-1',
    };

    const decision = await client.ask(intent);

    expect(mockFetch).toHaveBeenCalledTimes(3); // 0,1,2 attempts
    expect(decision.allowed).toBe(false);
    expect(decision.status).toBe('deny_with_reason');
    expect(decision.reason).toContain('upstream_error');
  });
});